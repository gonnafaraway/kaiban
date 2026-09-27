package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"kaiban/internal/api/domain/column"
	httptransport "kaiban/internal/api/transport/http"
	"kaiban/internal/api/validator"
)

func (d Deps) listColumns(c *fiber.Ctx) error {
	cols, err := d.UC.ListColumns(c.Context())
	if err != nil {
		return httptransport.JSONError(c, 500, "internal", err.Error())
	}
	return c.JSON(cols)
}

func (d Deps) createColumn(c *fiber.Ctx) error {
	var body struct {
		Name         string `json:"name"`
		SystemPrompt string `json:"system_prompt_template"`
		OrderIndex   int    `json:"order_index"`
	}
	if err := c.BodyParser(&body); err != nil {
		return httptransport.JSONError(c, 400, "bad_request", err.Error())
	}
	if err := validator.Required(body.Name); err != nil {
		return httptransport.JSONError(c, 400, "bad_request", err.Error())
	}
	col, err := d.UC.CreateColumn(c.Context(), body.Name, body.SystemPrompt, body.OrderIndex)
	if err != nil {
		return httptransport.JSONError(c, 400, "bad_request", err.Error())
	}
	return c.Status(201).JSON(col)
}

func (d Deps) patchColumn(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	var body struct {
		Name                 *string               `json:"name"`
		NameI18n             map[string]string     `json:"name_i18n"`
		SystemPromptTemplate *string               `json:"system_prompt_template"`
		UserCustomPrompt     *string               `json:"user_custom_prompt"`
		OutputFields         *[]column.OutputField `json:"output_fields"`
		Budget               *column.BudgetLimits  `json:"budget"`
		RequiresGitDiff      *bool                 `json:"requires_git_diff"`
	}
	if err := c.BodyParser(&body); err != nil {
		return httptransport.JSONError(c, 400, "bad_request", err.Error())
	}
	col, err := d.UC.PatchColumn(c.Context(), id, body.Name, body.SystemPromptTemplate, body.UserCustomPrompt, body.NameI18n, body.OutputFields, body.Budget, body.RequiresGitDiff)
	if err != nil {
		return httptransport.JSONError(c, 400, "bad_request", err.Error())
	}
	return c.JSON(col)
}

func (d Deps) deleteColumn(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	if err := d.UC.DeleteColumn(c.Context(), id); err != nil {
		return httptransport.JSONError(c, 409, "conflict", err.Error())
	}
	return c.SendStatus(204)
}

func (d Deps) resetOverlay(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	col, err := d.UC.ResetOverlay(c.Context(), id)
	if err != nil {
		return httptransport.JSONError(c, 400, "bad_request", err.Error())
	}
	return c.JSON(col)
}

func (d Deps) restoreDefault(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	col, err := d.UC.RestoreDefault(c.Context(), id)
	if err != nil {
		return httptransport.JSONError(c, 400, "bad_request", err.Error())
	}
	return c.JSON(col)
}

func (d Deps) reorder(c *fiber.Ctx) error {
	var body []struct {
		ID         uuid.UUID `json:"id"`
		OrderIndex int       `json:"order_index"`
	}
	if err := c.BodyParser(&body); err != nil {
		return httptransport.JSONError(c, 400, "bad_request", err.Error())
	}
	ids := make([]uuid.UUID, len(body))
	for i, b := range body {
		ids[i] = b.ID
	}
	if err := d.UC.ReorderColumns(c.Context(), ids); err != nil {
		return httptransport.JSONError(c, 400, "bad_request", err.Error())
	}
	return c.JSON(fiber.Map{"ok": true})
}
