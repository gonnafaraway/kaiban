package settings

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

// ContextPackItem is an editable project document injected into agent prompts.
type ContextPackItem struct {
	Title   string `json:"title"`
	Body    string `json:"body"`
	Enabled bool   `json:"enabled"`
	Order   int    `json:"order"`
}

type Settings struct {
	ID               uuid.UUID `json:"id"`
	LLMBaseURL       string    `json:"llm_base_url"`
	LLMAPIKey        string    `json:"llm_api_key"`
	LLMModel         string    `json:"llm_model"`
	GitRepoURL       string    `json:"git_repo_url"`
	GitDefaultBranch string    `json:"git_default_branch"`
	Locale           string    `json:"locale"`

	MaxTokens    int     `json:"max_tokens"`
	MaxCostUSD   float64 `json:"max_cost_usd"`
	MaxWallSec   int     `json:"max_wall_sec"`
	MaxToolCalls int     `json:"max_tool_calls"`
	MaxLLMSteps  int     `json:"max_llm_steps"`

	PriceInputPer1K  float64 `json:"price_input_per_1k"`
	PriceOutputPer1K float64 `json:"price_output_per_1k"`

	ContextPack []ContextPackItem `json:"context_pack"`

	UpdatedAt time.Time `json:"updated_at"`
}

func NormalizeContextPack(items []ContextPackItem) []ContextPackItem {
	out := make([]ContextPackItem, 0, len(items))
	for i, it := range items {
		title := strings.TrimSpace(it.Title)
		body := strings.TrimSpace(it.Body)
		if title == "" && body == "" {
			continue
		}
		if title == "" {
			title = "doc"
		}
		order := it.Order
		if order == 0 {
			order = i
		}
		out = append(out, ContextPackItem{Title: title, Body: body, Enabled: it.Enabled, Order: order})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Order < out[j].Order })
	return out
}

// BuildContextPackText returns enabled docs (capped) and a short hash of the packed text.
func (s *Settings) BuildContextPackText(maxRunes int) (text, hash string) {
	if maxRunes <= 0 {
		maxRunes = 12000
	}
	items := NormalizeContextPack(s.ContextPack)
	var b strings.Builder
	for _, it := range items {
		if !it.Enabled {
			continue
		}
		if b.Len() > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString("## ")
		b.WriteString(it.Title)
		b.WriteString("\n")
		b.WriteString(it.Body)
	}
	text = b.String()
	if utf8.RuneCountInString(text) > maxRunes {
		runes := []rune(text)
		text = string(runes[:maxRunes]) + "\n…"
	}
	if text == "" {
		return "", ""
	}
	sum := sha256.Sum256([]byte(text))
	hash = hex.EncodeToString(sum[:8])
	return text, hash
}

// ContextPackTitles lists enabled document titles (for report footer).
func (s *Settings) ContextPackTitles() []string {
	items := NormalizeContextPack(s.ContextPack)
	var out []string
	for _, it := range items {
		if it.Enabled {
			out = append(out, it.Title)
		}
	}
	return out
}

func (s Settings) MaskedKey() string {
	v := s.LLMAPIKey
	if v == "" {
		return ""
	}
	if len(v) <= 4 {
		return "****"
	}
	return "****" + v[len(v)-4:]
}

// Budget is the effective limit after settings <- column <- task (tighten-only).
type Budget struct {
	MaxTokens    int
	MaxCostUSD   float64
	MaxWallSec   int
	MaxToolCalls int
	MaxLLMSteps  int
}

func (s *Settings) BaseBudget() Budget {
	b := Budget{
		MaxTokens:    s.MaxTokens,
		MaxCostUSD:   s.MaxCostUSD,
		MaxWallSec:   s.MaxWallSec,
		MaxToolCalls: s.MaxToolCalls,
		MaxLLMSteps:  s.MaxLLMSteps,
	}
	if b.MaxTokens <= 0 {
		b.MaxTokens = 200000
	}
	if b.MaxCostUSD <= 0 {
		b.MaxCostUSD = 5
	}
	if b.MaxWallSec <= 0 {
		b.MaxWallSec = 1800
	}
	if b.MaxToolCalls <= 0 {
		b.MaxToolCalls = 80
	}
	if b.MaxLLMSteps <= 0 {
		b.MaxLLMSteps = 40
	}
	return b
}

// ApplyBudgetOverride only tightens limits relative to b (never raises the ceiling).
func ApplyBudgetOverride(b Budget, maxTokens *int, maxCost *float64, maxWall, maxTools, maxSteps *int) Budget {
	if maxTokens != nil && *maxTokens > 0 && *maxTokens < b.MaxTokens {
		b.MaxTokens = *maxTokens
	}
	if maxCost != nil && *maxCost > 0 && *maxCost < b.MaxCostUSD {
		b.MaxCostUSD = *maxCost
	}
	if maxWall != nil && *maxWall > 0 && *maxWall < b.MaxWallSec {
		b.MaxWallSec = *maxWall
	}
	if maxTools != nil && *maxTools > 0 && *maxTools < b.MaxToolCalls {
		b.MaxToolCalls = *maxTools
	}
	if maxSteps != nil && *maxSteps > 0 && *maxSteps < b.MaxLLMSteps {
		b.MaxLLMSteps = *maxSteps
	}
	return b
}

func EstimateCostUSD(tokensIn, tokensOut int, priceInPer1K, priceOutPer1K float64) float64 {
	return (float64(tokensIn)/1000.0)*priceInPer1K + (float64(tokensOut)/1000.0)*priceOutPer1K
}

func EstimateTokens(s string) int {
	if s == "" {
		return 0
	}
	n := len(s) / 4
	if n < 1 {
		return 1
	}
	return n
}
