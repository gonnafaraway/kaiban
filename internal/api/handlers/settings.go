package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/pkg/errors"

	"kaiban/internal/api/domain/settings"
	"kaiban/internal/api/usecase/kanban"
)

func (d Deps) getSettings(c *fiber.Ctx) error {
	s, err := d.UC.GetSettings(c.Context())
	if err != nil {
		return respondError(c, errors.Wrap(err, "get settings"))
	}
	return c.JSON(maskSettings(s))
}

func (d Deps) putSettings(c *fiber.Ctx) error {
	var body settings.Settings
	if err := parseBody(c, &body); err != nil {
		return respondError(c, errors.Wrap(err, "update settings"))
	}
	s, err := d.UC.UpdateSettings(c.Context(), &body)
	if err != nil {
		return respondError(c, errors.Wrap(err, "update settings"))
	}
	return c.JSON(maskSettings(s))
}

func (d Deps) exportSettings(c *fiber.Ctx) error {
	bundle, err := d.UC.ExportConfig(c.Context())
	if err != nil {
		return respondError(c, errors.Wrap(err, "export settings"))
	}
	return c.JSON(bundle)
}

func (d Deps) importSettings(c *fiber.Ctx) error {
	var body kanban.ConfigBundle
	if err := parseBody(c, &body); err != nil {
		return respondError(c, errors.Wrap(err, "import settings"))
	}
	result, err := d.UC.ImportConfig(c.Context(), &body)
	if err != nil {
		return respondError(c, errors.Wrap(err, "import settings"))
	}
	return c.JSON(result)
}

func (d Deps) listLLMModels(c *fiber.Ctx) error {
	provider := c.Query("provider")
	if provider == "" {
		provider = settings.ProviderOpenAI
	}
	return c.JSON(fiber.Map{"provider": settings.NormalizeProvider(provider), "models": d.UC.ListLLMModels(provider)})
}

func (d Deps) testLLM(c *fiber.Ctx) error {
	var body settings.Settings
	_ = parseOptionalBody(c, &body)
	res, err := d.UC.TestLLM(c.Context(), &body)
	if err != nil {
		return respondError(c, errors.Wrap(err, "test llm"))
	}
	return c.JSON(res)
}

func maskSettings(s *settings.Settings) fiber.Map {
	provider := s.LLMProvider
	if provider == "" {
		provider = settings.InferProviderFromURL(s.LLMBaseURL)
	}
	return fiber.Map{
		"id": s.ID, "llm_provider": provider, "llm_base_url": s.LLMBaseURL, "llm_api_key": s.MaskedKey(),
		"llm_model": s.LLMModel, "git_repo_url": s.GitRepoURL,
		"git_default_branch": s.GitDefaultBranch, "locale": s.Locale,
		"max_tokens": s.MaxTokens, "max_cost_usd": s.MaxCostUSD, "max_wall_sec": s.MaxWallSec,
		"max_tool_calls": s.MaxToolCalls, "max_llm_steps": s.MaxLLMSteps,
		"price_input_per_1k": s.PriceInputPer1K, "price_output_per_1k": s.PriceOutputPer1K,
		"context_pack": s.ContextPack,
		"updated_at":   s.UpdatedAt,
	}
}
