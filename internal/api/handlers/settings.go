package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/pkg/errors"

	"kaiban/internal/api/domain/settings"
	"kaiban/internal/api/usecase/config"
)

func (d Deps) getSettings(c *fiber.Ctx) error {
	s, err := d.UC.GetSettings(c.Context())
	if err != nil {
		return respondError(c, err)
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
		return respondError(c, err)
	}
	return c.JSON(maskSettings(s))
}

func (d Deps) exportSettings(c *fiber.Ctx) error {
	bundle, err := d.UC.ExportConfig(c.Context())
	if err != nil {
		return respondError(c, err)
	}
	return c.JSON(bundle)
}

func (d Deps) importSettings(c *fiber.Ctx) error {
	var body config.Bundle
	if err := parseBody(c, &body); err != nil {
		return respondError(c, errors.Wrap(err, "import settings"))
	}
	result, err := d.UC.ImportConfig(c.Context(), &body)
	if err != nil {
		return respondError(c, err)
	}
	return c.JSON(result)
}

func (d Deps) listLLMModels(c *fiber.Ctx) error {
	provider := c.Query("provider")
	if provider == "" {
		provider = settings.ProviderOpenAI
	}
	return c.JSON(llmModelsResponse{
		Provider: settings.NormalizeProvider(provider),
		Models:   d.UC.ListLLMModels(provider),
	})
}

func (d Deps) testLLM(c *fiber.Ctx) error {
	var body settings.Settings
	if err := parseOptionalBody(c, &body); err != nil {
		return respondError(c, errors.Wrap(err, "test llm"))
	}
	res, err := d.UC.TestLLM(c.Context(), &body)
	if err != nil {
		return respondError(c, err)
	}
	return c.JSON(res)
}
