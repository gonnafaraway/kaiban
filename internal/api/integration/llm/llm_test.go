package llm

import (
	"strings"
	"testing"
)

func TestParseChatStream_content(t *testing.T) {
	raw := strings.Join([]string{
		`data: {"choices":[{"delta":{"role":"assistant","content":"hel"}}]}`,
		`data: {"choices":[{"delta":{"content":"lo"}}]}`,
		`data: [DONE]`,
		"",
	}, "\n")
	msg, err := parseChatStream(strings.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	if msg.Content != "hello" {
		t.Fatalf("got %q", msg.Content)
	}
}

func TestParseChatStream_toolCalls(t *testing.T) {
	raw := strings.Join([]string{
		`data: {"choices":[{"delta":{"tool_calls":[{"index":0,"id":"c1","type":"function","function":{"name":"git_status","arguments":""}}]}}]}`,
		`data: {"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"path\":\".\"}"}}]}}]}`,
		`data: [DONE]`,
		"",
	}, "\n")
	msg, err := parseChatStream(strings.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	if len(msg.ToolCalls) != 1 || msg.ToolCalls[0].Function.Name != "git_status" {
		t.Fatalf("%+v", msg.ToolCalls)
	}
	if msg.ToolCalls[0].Function.Arguments != `{"path":"."}` {
		t.Fatalf("args %q", msg.ToolCalls[0].Function.Arguments)
	}
}

func TestNewOpenAIFallsBackToDefaultTimeout(t *testing.T) {
	if got := NewOpenAI(0).HTTP.Timeout; got != DefaultTimeout {
		t.Fatalf("timeout %s", got)
	}
}
