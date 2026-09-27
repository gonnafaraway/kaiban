package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/pkg/errors"

	domint "kaiban/internal/api/domain/integration"
)

type integrationBody struct {
	Type        string            `json:"type"`
	Name        string            `json:"name"`
	BaseURL     string            `json:"base_url"`
	Credentials map[string]string `json:"credentials"`
	Status      string            `json:"status"`
}

func (d Deps) listInt(c *fiber.Ctx) error {
	items, err := d.UC.ListIntegrations(c.Context())
	if err != nil {
		return respondError(c, errors.Wrap(err, "list integrations"))
	}
	out := make([]fiber.Map, 0, len(items))
	for _, i := range items {
		out = append(out, maskInt(i))
	}
	return c.JSON(out)
}

func maskInt(i *domint.Integration) fiber.Map {
	return fiber.Map{
		"id": i.ID, "type": i.Type, "name": i.Name, "base_url": i.BaseURL,
		"credentials": i.MaskedCredentials(), "status": i.Status, "last_error": i.LastError,
		"created_at": i.CreatedAt, "updated_at": i.UpdatedAt,
	}
}

func (d Deps) createInt(c *fiber.Ctx) error {
	var body integrationBody
	if err := parseBody(c, &body); err != nil {
		return respondError(c, errors.Wrap(err, "create integration"))
	}
	item, err := d.UC.UpsertIntegration(c.Context(), nil, body.Type, body.Name, body.BaseURL, body.Credentials, body.Status)
	if err != nil {
		return respondError(c, errors.Wrap(err, "create integration"))
	}
	return c.Status(201).JSON(maskInt(item))
}

func (d Deps) patchInt(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return respondError(c, errors.Wrap(err, "patch integration"))
	}
	var body integrationBody
	if err := parseBody(c, &body); err != nil {
		return respondError(c, errors.Wrap(err, "patch integration"))
	}
	item, err := d.UC.UpsertIntegration(c.Context(), &id, body.Type, body.Name, body.BaseURL, body.Credentials, body.Status)
	if err != nil {
		return respondError(c, errors.Wrap(err, "patch integration"))
	}
	return c.JSON(maskInt(item))
}

func (d Deps) deleteInt(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return respondError(c, errors.Wrap(err, "delete integration"))
	}
	if err := d.UC.DeleteIntegration(c.Context(), id); err != nil {
		return respondError(c, errors.Wrap(err, "delete integration"))
	}
	return c.SendStatus(204)
}

func (d Deps) testInt(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return respondError(c, errors.Wrap(err, "test integration"))
	}
	item, err := d.UC.TestIntegration(c.Context(), id)
	if err != nil {
		return respondError(c, errors.Wrap(err, "test integration"))
	}
	return c.JSON(maskInt(item))
}
