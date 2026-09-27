package kanban

import (
	"context"
	"strings"

	"github.com/pkg/errors"

	"kaiban/internal/api/domain/settings"
	"kaiban/internal/api/integration"
)

func (u *UseCase) GetSettings(ctx context.Context) (*settings.Settings, error) {
	st, err := u.Repo.Settings.Get(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "get settings")
	}
	return st, nil
}

func (u *UseCase) UpdateSettings(ctx context.Context, in *settings.Settings) (*settings.Settings, error) {
	cur, err := u.Repo.Settings.Get(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "get settings")
	}
	if in.LLMProvider != "" {
		settings.ApplyProvider(cur, in.LLMProvider)
	}
	if in.LLMBaseURL != "" {
		cur.LLMBaseURL = in.LLMBaseURL
		if in.LLMProvider == "" {
			cur.LLMProvider = settings.InferProviderFromURL(in.LLMBaseURL)
		}
	}
	if in.LLMAPIKey != "" && !strings.Contains(in.LLMAPIKey, "*") {
		cur.LLMAPIKey = in.LLMAPIKey
	}
	if in.LLMModel != "" {
		cur.LLMModel = in.LLMModel
	}
	if cur.LLMProvider == "" {
		cur.LLMProvider = settings.InferProviderFromURL(cur.LLMBaseURL)
	}
	cur.GitRepoURL = in.GitRepoURL
	if in.GitDefaultBranch != "" {
		cur.GitDefaultBranch = in.GitDefaultBranch
	}
	if in.Locale == "ru" || in.Locale == "en" {
		cur.Locale = in.Locale
	}
	if in.MaxTokens > 0 {
		cur.MaxTokens = in.MaxTokens
	}
	if in.MaxCostUSD > 0 {
		cur.MaxCostUSD = in.MaxCostUSD
	}
	if in.MaxWallSec > 0 {
		cur.MaxWallSec = in.MaxWallSec
	}
	if in.MaxToolCalls > 0 {
		cur.MaxToolCalls = in.MaxToolCalls
	}
	if in.MaxLLMSteps > 0 {
		cur.MaxLLMSteps = in.MaxLLMSteps
	}
	if in.PriceInputPer1K > 0 {
		cur.PriceInputPer1K = in.PriceInputPer1K
	}
	if in.PriceOutputPer1K > 0 {
		cur.PriceOutputPer1K = in.PriceOutputPer1K
	}
	if in.ContextPack != nil {
		cur.ContextPack = settings.NormalizeContextPack(in.ContextPack)
	}
	if err := u.Repo.Settings.Update(ctx, cur); err != nil {
		return nil, errors.Wrap(err, "update settings")
	}
	return cur, nil
}

// LLMModelOption is a selectable model for the settings UI.
type LLMModelOption struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ListLLMModels returns models for a provider preset.
func (u *UseCase) ListLLMModels(provider string) []LLMModelOption {
	switch settings.NormalizeProvider(provider) {
	case settings.ProviderOpenCode:
		ids := settings.OpenCodeChatModels()
		out := make([]LLMModelOption, 0, len(ids))
		for _, id := range ids {
			out = append(out, LLMModelOption{ID: id, Name: id})
		}
		return out
	default:
		return nil
	}
}

// TestLLMResult is a short probe against the configured LLM endpoint.
type TestLLMResult struct {
	OK       bool   `json:"ok"`
	Provider string `json:"provider"`
	Model    string `json:"model"`
	BaseURL  string `json:"base_url"`
	Reply    string `json:"reply,omitempty"`
	Error    string `json:"error,omitempty"`
}

// TestLLM sends a tiny chat request using current settings (or optional overrides).
func (u *UseCase) TestLLM(ctx context.Context, override *settings.Settings) (*TestLLMResult, error) {
	st, err := u.Repo.Settings.Get(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "get settings")
	}
	if override != nil {
		if override.LLMProvider != "" {
			settings.ApplyProvider(st, override.LLMProvider)
		}
		if override.LLMBaseURL != "" {
			st.LLMBaseURL = override.LLMBaseURL
		}
		if override.LLMAPIKey != "" && !strings.Contains(override.LLMAPIKey, "*") {
			st.LLMAPIKey = override.LLMAPIKey
		}
		if override.LLMModel != "" {
			st.LLMModel = override.LLMModel
		}
	}
	if st.LLMProvider == "" {
		st.LLMProvider = settings.InferProviderFromURL(st.LLMBaseURL)
	}
	if u.LLM == nil {
		return &TestLLMResult{OK: false, Provider: st.LLMProvider, Model: st.LLMModel, BaseURL: st.LLMBaseURL, Error: "llm client not configured"}, nil
	}
	msg, err := u.LLM.Chat(ctx, st.LLMBaseURL, st.LLMAPIKey, st.LLMModel, []integration.ChatMessage{
		{Role: "user", Content: "Reply with exactly the word OK and nothing else."},
	}, nil)
	if err != nil {
		return &TestLLMResult{
			OK: false, Provider: st.LLMProvider, Model: st.LLMModel, BaseURL: st.LLMBaseURL, Error: err.Error(),
		}, nil
	}
	return &TestLLMResult{
		OK: true, Provider: st.LLMProvider, Model: st.LLMModel, BaseURL: st.LLMBaseURL, Reply: msg.Content,
	}, nil
}
