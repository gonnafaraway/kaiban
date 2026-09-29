package llm

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

func TestOpenCodeZenLiveChat(t *testing.T) {
	if os.Getenv("KAIBAN_SKIP_LIVE_LLM") == "1" {
		t.Skip("KAIBAN_SKIP_LIVE_LLM=1")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	client := NewOpenAI(45 * time.Second)
	msg, err := client.Chat(ctx, "https://opencode.ai/zen/v1", "", "space-bunny-free", []ChatMessage{
		{Role: "user", Content: "Reply with exactly the word OK and nothing else."},
	}, nil)
	if err != nil {
		t.Fatalf("opencode zen chat: %v", err)
	}
	got := msg.Content
	if !strings.Contains(got, "OK") {
		t.Fatalf("unexpected reply %q", msg.Content)
	}
}
