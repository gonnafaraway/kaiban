package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/pkg/errors"

	"kaiban/internal/api/domain/column"
	"kaiban/internal/api/validator"
)

type createColumnBody struct {
	Name         string `json:"name"`
	SystemPrompt string `json:"system_prompt_template"`
	OrderIndex   int    `json:"order_index"`
}

type patchColumnBody struct {
	Name                 *string               `json:"name"`
	NameI18n             map[string]string     `json:"name_i18n"`
	SystemPromptTemplate *string               `json:"system_prompt_template"`
	UserCustomPrompt     *string               `json:"user_custom_prompt"`
	OutputFields         *[]column.OutputField `json:"output_fields"`
	Budget               *column.BudgetLimits  `json:"budget"`
	RequiresGitDiff      *bool                 `json:"requires_git_diff"`
}

type reorderColumnItem struct {
	ID         uuid.UUID `json:"id"`
	OrderIndex int       `json:"order_index"`
}

func (d Deps) listColumns(c *fiber.Ctx) error {
	cols, err := d.UC.ListColumns(c.Context())
	if err != nil {
		return respondError(c, errors.Wrap(err, "list columns"))
	}
	return c.JSON(cols)
}

func (d Deps) createColumn(c *fiber.Ctx) error {
	var body createColumnBody
	if err := parseBody(c, &body); err != nil {
		return respondError(c, errors.Wrap(err, "create column"))
	}
	if err := validator.Required(body.Name); err != nil {
		return respondError(c, errors.Wrap(column.ErrNameRequired, "create column: name"))
	}
	col, err := d.UC.CreateColumn(c.Context(), body.Name, body.SystemPrompt, body.OrderIndex)
	if err != nil {
		return respondError(c, errors.Wrap(err, "create column"))
	}
	return c.Status(201).JSON(col)
}

func (d Deps) patchColumn(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return respondError(c, errors.Wrap(err, "patch column"))
	}
	var body patchColumnBody
	if err := parseBody(c, &body); err != nil {
		return respondError(c, errors.Wrap(err, "patch column"))
	}
	col, err := d.UC.PatchColumn(c.Context(), id, body.Name, body.SystemPromptTemplate, body.UserCustomPrompt, body.NameI18n, body.OutputFields, body.Budget, body.RequiresGitDiff)
	if err != nil {
		return respondError(c, errors.Wrap(err, "patch column"))
	}
	return c.JSON(col)
}

func (d Deps) deleteColumn(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return respondError(c, errors.Wrap(err, "delete column"))
	}
	if err := d.UC.DeleteColumn(c.Context(), id); err != nil {
		return respondError(c, errors.Wrap(err, "delete column"))
	}
	return c.SendStatus(204)
}

func (d Deps) resetOverlay(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return respondError(c, errors.Wrap(err, "reset column overlay"))
	}
	col, err := d.UC.ResetOverlay(c.Context(), id)
	if err != nil {
		return respondError(c, errors.Wrap(err, "reset column overlay"))
	}
	return c.JSON(col)
}

func (d Deps) restoreDefault(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return respondError(c, errors.Wrap(err, "restore column default prompt"))
	}
	col, err := d.UC.RestoreDefault(c.Context(), id)
	if err != nil {
		return respondError(c, errors.Wrap(err, "restore column default prompt"))
	}
	return c.JSON(col)
}

func (d Deps) reorder(c *fiber.Ctx) error {
	var body []reorderColumnItem
	if err := parseBody(c, &body); err != nil {
		return respondError(c, errors.Wrap(err, "reorder columns"))
	}
	ids := make([]uuid.UUID, len(body))
	for i, b := range body {
		ids[i] = b.ID
	}
	if err := d.UC.ReorderColumns(c.Context(), ids); err != nil {
		return respondError(c, errors.Wrap(err, "reorder columns"))
	}
	return c.JSON(fiber.Map{"ok": true})
}
