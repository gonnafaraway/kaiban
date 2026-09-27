package integration

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/pkg/errors"
)

// mcpTimeout bounds MCP handshakes and tool calls.
const mcpTimeout = 20 * time.Second

type mcpToolDescriptor struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

type mcpToolsListResult struct {
	Tools []mcpToolDescriptor `json:"tools"`
}

type mcpRPCError struct {
	Message string `json:"message"`
}

type mcpToolsListResponse struct {
	Result mcpToolsListResult `json:"result"`
	Error  *mcpRPCError       `json:"error"`
}

// HandshakeMCP runs JSON-RPC initialize + tools/list (streamable HTTP / JSON).
func HandshakeMCP(ctx context.Context, endpoint string, headers map[string]string) (map[string]any, []Tool, error) {
	client := httpClient(mcpTimeout)
	initBody := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]any{},
			"clientInfo":      map[string]any{"name": "kaiban", "version": "0.1.0"},
		},
	}
	if _, err := mcpPost(ctx, client, endpoint, headers, initBody); err != nil {
		return nil, nil, err
	}
	listBody := map[string]any{"jsonrpc": "2.0", "id": 2, "method": "tools/list", "params": map[string]any{}}
	raw, err := mcpPost(ctx, client, endpoint, headers, listBody)
	if err != nil {
		return nil, nil, err
	}
	var parsed mcpToolsListResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return map[string]any{"raw": string(raw)}, nil, nil
	}
	if parsed.Error != nil {
		return nil, nil, errors.Errorf("mcp: %s", parsed.Error.Message)
	}
	var tools []Tool
	caps := map[string]any{"tools": parsed.Result.Tools}
	for _, t := range parsed.Result.Tools {
		schema := t.InputSchema
		if schema == nil {
			schema = map[string]any{"type": "object"}
		}
		tools = append(tools, mcpTool{name: t.Name, desc: t.Description, schema: schema, endpoint: endpoint, headers: headers, client: client})
	}
	return caps, tools, nil
}

type mcpTool struct {
	name, desc, endpoint string
	schema               map[string]any
	headers              map[string]string
	client               *http.Client
}

func (m mcpTool) Name() string               { return m.name }
func (m mcpTool) Description() string        { return m.desc }
func (m mcpTool) Parameters() map[string]any { return m.schema }
func (m mcpTool) Call(ctx context.Context, args string) (string, error) {
	var arg any
	if args == "" {
		args = "{}"
	}
	if err := json.Unmarshal([]byte(args), &arg); err != nil {
		return "", errors.Wrap(err, "decode mcp tool args")
	}
	body := map[string]any{
		"jsonrpc": "2.0", "id": 3, "method": "tools/call",
		"params": map[string]any{"name": m.name, "arguments": arg},
	}
	raw, err := mcpPost(ctx, m.client, m.endpoint, m.headers, body)
	if err != nil {
		return "", errors.Wrap(err, "mcp tool call")
	}
	return string(raw), nil
}

func mcpPost(ctx context.Context, client *http.Client, endpoint string, headers map[string]string, body any) ([]byte, error) {
	b, err := marshalJSON(body, "mcp request")
	if err != nil {
		return nil, err
	}
	req, err := newJSONRequest(ctx, http.MethodPost, endpoint, headers, b)
	if err != nil {
		return nil, errors.Wrap(err, "build mcp request")
	}
	req.Header.Set("Accept", "application/json, text/event-stream")
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "mcp http request")
	}
	defer func() { _ = resp.Body.Close() }()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return nil, errors.Wrap(err, "read mcp body")
	}
	if resp.StatusCode >= 300 {
		return nil, errors.Errorf("mcp http %d: %s", resp.StatusCode, string(data))
	}
	return data, nil
}
