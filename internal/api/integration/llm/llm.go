// Package llm speaks the OpenAI-compatible /chat/completions protocol.
package llm

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/pkg/errors"

	"kaiban/internal/api/textutil"
)

// DefaultTimeout is used when the caller passes a non-positive timeout.
const DefaultTimeout = 10 * time.Minute

// errorBodySnippet is how much of a failing response we quote back to the caller.
const errorBodySnippet = 800

type ChatMessage struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
}

type ToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type ToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Function ToolCallFunction `json:"function"`
}

type ToolSpecFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

type ToolSpec struct {
	Type     string           `json:"type"`
	Function ToolSpecFunction `json:"function"`
}

type LLM interface {
	Chat(ctx context.Context, baseURL, apiKey, model string, messages []ChatMessage, tools []ToolSpec) (ChatMessage, error)
}

type OpenAIClient struct {
	HTTP *http.Client
}

type llmAPIError struct {
	Message string `json:"message"`
}

type chatJSONChoice struct {
	Message ChatMessage `json:"message"`
}

type chatJSONResponse struct {
	Choices []chatJSONChoice `json:"choices"`
	Error   llmAPIError      `json:"error"`
}

// NewOpenAI builds a client for long-running agent calls; timeout comes from config.
func NewOpenAI(timeout time.Duration) *OpenAIClient {
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	return &OpenAIClient{HTTP: &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   20 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout:   20 * time.Second,
			ResponseHeaderTimeout: timeout,
			ExpectContinueTimeout: 1 * time.Second,
			ForceAttemptHTTP2:     true,
		},
	}}
}

func (c *OpenAIClient) Chat(ctx context.Context, baseURL, apiKey, model string, messages []ChatMessage, tools []ToolSpec) (ChatMessage, error) {
	baseURL = apiBase(baseURL)
	body := map[string]any{
		"model":    model,
		"messages": messages,
		"stream":   true,
	}
	if len(tools) > 0 {
		body["tools"] = tools
		body["tool_choice"] = "auto"
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return ChatMessage{}, errors.Wrap(err, "marshal llm request")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/chat/completions", bytes.NewReader(raw))
	if err != nil {
		return ChatMessage{}, errors.Wrap(err, "build llm request")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return ChatMessage{}, errors.Wrap(err, "llm http request")
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 32*1024))
		return ChatMessage{}, errors.Errorf("llm http %d: %s", resp.StatusCode, textutil.TruncateRunes(string(data), errorBodySnippet))
	}
	br := bufio.NewReader(resp.Body)
	head, _ := br.Peek(8)
	ct := resp.Header.Get("Content-Type")
	if strings.Contains(ct, "text/event-stream") || bytes.HasPrefix(bytes.TrimSpace(head), []byte("data:")) {
		msg, err := parseChatStream(br)
		if err != nil {
			return ChatMessage{}, errors.Wrap(err, "parse llm stream")
		}
		return msg, nil
	}
	data, err := io.ReadAll(br)
	if err != nil {
		return ChatMessage{}, errors.Wrap(err, "read llm body")
	}
	msg, err := parseChatJSON(data)
	if err != nil {
		return ChatMessage{}, errors.Wrap(err, "parse llm json")
	}
	return msg, nil
}

func parseChatJSON(data []byte) (ChatMessage, error) {
	var parsed chatJSONResponse
	if err := json.Unmarshal(data, &parsed); err != nil {
		return ChatMessage{}, errors.Wrap(err, "decode llm json")
	}
	if parsed.Error.Message != "" {
		return ChatMessage{}, errors.Errorf("llm: %s", parsed.Error.Message)
	}
	if len(parsed.Choices) == 0 {
		return ChatMessage{}, errors.New("llm: empty choices")
	}
	return parsed.Choices[0].Message, nil
}

type streamToolCallDelta struct {
	Index    int              `json:"index"`
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Function ToolCallFunction `json:"function"`
}

type streamDelta struct {
	Content   string                `json:"content"`
	Role      string                `json:"role"`
	ToolCalls []streamToolCallDelta `json:"tool_calls"`
}

type streamChoice struct {
	Delta        streamDelta `json:"delta"`
	FinishReason string      `json:"finish_reason"`
}

type streamChunk struct {
	Choices []streamChoice `json:"choices"`
	Error   llmAPIError    `json:"error"`
}

func parseChatStream(r io.Reader) (ChatMessage, error) {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	var msg ChatMessage
	msg.Role = "assistant"
	calls := map[int]*ToolCall{}
	sawData := false
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "[DONE]" {
			break
		}
		sawData = true
		var chunk streamChunk
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			continue
		}
		if chunk.Error.Message != "" {
			return ChatMessage{}, errors.Errorf("llm: %s", chunk.Error.Message)
		}
		if len(chunk.Choices) == 0 {
			continue
		}
		delta := chunk.Choices[0].Delta
		if delta.Role != "" {
			msg.Role = delta.Role
		}
		msg.Content += delta.Content
		for _, tc := range delta.ToolCalls {
			cur, ok := calls[tc.Index]
			if !ok {
				cur = &ToolCall{ID: tc.ID, Type: tc.Type}
				if cur.Type == "" {
					cur.Type = "function"
				}
				calls[tc.Index] = cur
			}
			if tc.ID != "" {
				cur.ID = tc.ID
			}
			if tc.Type != "" {
				cur.Type = tc.Type
			}
			if tc.Function.Name != "" {
				cur.Function.Name += tc.Function.Name
			}
			if tc.Function.Arguments != "" {
				cur.Function.Arguments += tc.Function.Arguments
			}
		}
	}
	if err := sc.Err(); err != nil {
		return ChatMessage{}, errors.Wrap(err, "read llm stream")
	}
	if !sawData {
		return ChatMessage{}, errors.New("llm: empty stream")
	}
	if len(calls) > 0 {
		last := -1
		for i := range calls {
			if i > last {
				last = i
			}
		}
		msg.ToolCalls = make([]ToolCall, last+1)
		for i, tc := range calls {
			if tc != nil {
				msg.ToolCalls[i] = *tc
			}
		}
	}
	if msg.Content == "" && len(msg.ToolCalls) == 0 {
		return ChatMessage{}, errors.New("llm: empty choices")
	}
	return msg, nil
}

// apiBase strips trailing slashes before joining paths.
func apiBase(base string) string {
	for len(base) > 0 && base[len(base)-1] == '/' {
		base = base[:len(base)-1]
	}
	return base
}
