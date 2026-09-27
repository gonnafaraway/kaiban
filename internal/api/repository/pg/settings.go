package pg

import (
	"context"

	"kaiban/internal/api/domain/settings"
)

func (s *Store) GetSettings(ctx context.Context) (*settings.Settings, error) {
	st := &settings.Settings{ContextPack: []settings.ContextPackItem{}}
	var pack []byte
	err := s.db.Pool.QueryRow(ctx, `
		SELECT id,COALESCE(llm_provider,'openai'),llm_base_url,llm_api_key,llm_model,git_repo_url,git_default_branch,locale,
			COALESCE(max_tokens,200000), COALESCE(max_cost_usd,5), COALESCE(max_wall_sec,1800),
			COALESCE(max_tool_calls,80), COALESCE(max_llm_steps,40),
			COALESCE(price_input_per_1k,0.002), COALESCE(price_output_per_1k,0.008),
			COALESCE(context_pack, '[]'::jsonb),
			updated_at
		FROM app_settings LIMIT 1`).
		Scan(&st.ID, &st.LLMProvider, &st.LLMBaseURL, &st.LLMAPIKey, &st.LLMModel, &st.GitRepoURL, &st.GitDefaultBranch, &st.Locale,
			&st.MaxTokens, &st.MaxCostUSD, &st.MaxWallSec, &st.MaxToolCalls, &st.MaxLLMSteps,
			&st.PriceInputPer1K, &st.PriceOutputPer1K, &pack, &st.UpdatedAt)
	if err != nil {
		return nil, mapNoRows(err, "scan settings")
	}
	if err := unmarshalJSON(pack, &st.ContextPack, "context pack"); err != nil {
		return nil, err
	}
	if st.ContextPack == nil {
		st.ContextPack = []settings.ContextPackItem{}
	}
	if st.LLMProvider == "" {
		st.LLMProvider = settings.InferProviderFromURL(st.LLMBaseURL)
	} else {
		st.LLMProvider = settings.NormalizeProvider(st.LLMProvider)
	}
	return st, nil
}

func (s *Store) UpdateSettings(ctx context.Context, st *settings.Settings) error {
	pack, err := marshalJSON(st.ContextPack, "context pack")
	if err != nil {
		return err
	}
	_, err = s.db.Pool.Exec(ctx, `
		UPDATE app_settings SET llm_provider=$2,llm_base_url=$3,llm_api_key=$4,llm_model=$5,git_repo_url=$6,git_default_branch=$7,locale=$8,
			max_tokens=$9,max_cost_usd=$10,max_wall_sec=$11,max_tool_calls=$12,max_llm_steps=$13,
			price_input_per_1k=$14,price_output_per_1k=$15,context_pack=$16,updated_at=now()
		WHERE id=$1`,
		st.ID, st.LLMProvider, st.LLMBaseURL, st.LLMAPIKey, st.LLMModel, st.GitRepoURL, st.GitDefaultBranch, st.Locale,
		st.MaxTokens, st.MaxCostUSD, st.MaxWallSec, st.MaxToolCalls, st.MaxLLMSteps,
		st.PriceInputPer1K, st.PriceOutputPer1K, pack)
	return err
}
