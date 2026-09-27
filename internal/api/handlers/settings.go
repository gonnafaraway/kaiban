package handlers

import (
	"github.com/gofiber/fiber/v2"

	"kaiban/internal/api/domain/settings"
	httptransport "kaiban/internal/api/transport/http"
	"kaiban/internal/api/usecase/kanban"
)

func (d Deps) getSettings(c *fiber.Ctx) error {
	s, err := d.UC.GetSettings(c.Context())
	if err != nil {
		return httptransport.JSONError(c, 500, "internal", err.Error())
	}
	return c.JSON(maskSettings(s))
}

func (d Deps) putSettings(c *fiber.Ctx) error {
	var body settings.Settings
	if err := c.BodyParser(&body); err != nil {
		return httptransport.JSONError(c, 400, "bad_request", err.Error())
	}
	s, err := d.UC.UpdateSettings(c.Context(), &body)
	if err != nil {
		return httptransport.JSONError(c, 400, "bad_request", err.Error())
	}
	return c.JSON(maskSettings(s))
}

func (d Deps) exportSettings(c *fiber.Ctx) error {
	bundle, err := d.UC.ExportConfig(c.Context())
	if err != nil {
		return httptransport.JSONError(c, 500, "internal", err.Error())
	}
	return c.JSON(bundle)
}

func (d Deps) importSettings(c *fiber.Ctx) error {
	var body kanban.ConfigBundle
	if err := c.BodyParser(&body); err != nil {
		return httptransport.JSONError(c, 400, "bad_request", err.Error())
	}
	result, err := d.UC.ImportConfig(c.Context(), &body)
	if err != nil {
		return httptransport.JSONError(c, 400, "bad_request", err.Error())
	}
	return c.JSON(result)
}

func maskSettings(s *settings.Settings) fiber.Map {
	return fiber.Map{
		"id": s.ID, "llm_base_url": s.LLMBaseURL, "llm_api_key": s.MaskedKey(),
		"llm_model": s.LLMModel, "git_repo_url": s.GitRepoURL,
		"git_default_branch": s.GitDefaultBranch, "locale": s.Locale,
		"max_tokens": s.MaxTokens, "max_cost_usd": s.MaxCostUSD, "max_wall_sec": s.MaxWallSec,
		"max_tool_calls": s.MaxToolCalls, "max_llm_steps": s.MaxLLMSteps,
		"price_input_per_1k": s.PriceInputPer1K, "price_output_per_1k": s.PriceOutputPer1K,
		"context_pack": s.ContextPack,
		"updated_at":   s.UpdatedAt,
	}
}
