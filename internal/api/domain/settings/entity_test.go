package settings_test

import (
	"strings"
	"testing"

	"kaiban/internal/api/domain/settings"
)

func TestBuildContextPackText(t *testing.T) {
	st := &settings.Settings{
		ContextPack: []settings.ContextPackItem{
			{Title: "B", Body: "second", Enabled: true, Order: 2},
			{Title: "A", Body: "first", Enabled: true, Order: 1},
			{Title: "Off", Body: "hidden", Enabled: false, Order: 0},
		},
	}
	text, hash := st.BuildContextPackText(12000)
	if hash == "" {
		t.Fatal("expected hash")
	}
	if !strings.Contains(text, "## A") || !strings.Contains(text, "first") {
		t.Fatalf("missing A: %q", text)
	}
	if strings.Contains(text, "hidden") {
		t.Fatal("disabled doc leaked")
	}
	if idxA, idxB := strings.Index(text, "## A"), strings.Index(text, "## B"); idxA > idxB {
		t.Fatal("order wrong")
	}
}

func TestBuildContextPackTextCap(t *testing.T) {
	st := &settings.Settings{
		ContextPack: []settings.ContextPackItem{
			{Title: "Big", Body: strings.Repeat("x", 500), Enabled: true},
		},
	}
	text, _ := st.BuildContextPackText(50)
	if len([]rune(text)) > 60 {
		t.Fatalf("not capped: %d", len([]rune(text)))
	}
}

func TestApplyBudgetOverrideTightensOnly(t *testing.T) {
	base := settings.Budget{MaxTokens: 1000, MaxCostUSD: 5, MaxWallSec: 600, MaxToolCalls: 40, MaxLLMSteps: 20}
	raise := 5000
	tokens := 100
	wall := 100
	tools := 10
	steps := 5
	costRaise := 99.0
	costLower := 1.0
	got := settings.ApplyBudgetOverride(base, &raise, &costRaise, &raise, &raise, &raise)
	if got.MaxTokens != 1000 || got.MaxCostUSD != 5 || got.MaxWallSec != 600 || got.MaxToolCalls != 40 || got.MaxLLMSteps != 20 {
		t.Fatalf("raise leaked: %+v", got)
	}
	got = settings.ApplyBudgetOverride(base, &tokens, &costLower, &wall, &tools, &steps)
	if got.MaxTokens != 100 || got.MaxCostUSD != 1 || got.MaxWallSec != 100 || got.MaxToolCalls != 10 || got.MaxLLMSteps != 5 {
		t.Fatalf("tighten failed: %+v", got)
	}
}
