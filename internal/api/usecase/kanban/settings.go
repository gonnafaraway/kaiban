package kanban

import (
	"context"
	"strings"

	"kaiban/internal/api/domain/settings"
)

func (u *UseCase) GetSettings(ctx context.Context) (*settings.Settings, error) {
	return u.Repo.Settings.Get(ctx)
}

func (u *UseCase) UpdateSettings(ctx context.Context, in *settings.Settings) (*settings.Settings, error) {
	cur, err := u.Repo.Settings.Get(ctx)
	if err != nil {
		return nil, err
	}
	if in.LLMBaseURL != "" {
		cur.LLMBaseURL = in.LLMBaseURL
	}
	if in.LLMAPIKey != "" && !strings.Contains(in.LLMAPIKey, "*") {
		cur.LLMAPIKey = in.LLMAPIKey
	}
	if in.LLMModel != "" {
		cur.LLMModel = in.LLMModel
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
		return nil, err
	}
	return cur, nil
}
