package handlers

import (
	"github.com/gofiber/fiber/v2"

	domint "kaiban/internal/api/domain/integration"
	httptransport "kaiban/internal/api/transport/http"
)

func (d Deps) listInt(c *fiber.Ctx) error {
	items, err := d.UC.ListIntegrations(c.Context())
	if err != nil {
		return httptransport.JSONError(c, 500, "internal", err.Error())
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
	var body struct {
		Type        string            `json:"type"`
		Name        string            `json:"name"`
		BaseURL     string            `json:"base_url"`
		Credentials map[string]string `json:"credentials"`
		Status      string            `json:"status"`
	}
	if err := c.BodyParser(&body); err != nil {
		return httptransport.JSONError(c, 400, "bad_request", err.Error())
	}
	item, err := d.UC.UpsertIntegration(c.Context(), nil, body.Type, body.Name, body.BaseURL, body.Credentials, body.Status)
	if err != nil {
		return httptransport.JSONError(c, 400, "bad_request", err.Error())
	}
	return c.Status(201).JSON(maskInt(item))
}

func (d Deps) patchInt(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	var body struct {
		Type        string            `json:"type"`
		Name        string            `json:"name"`
		BaseURL     string            `json:"base_url"`
		Credentials map[string]string `json:"credentials"`
		Status      string            `json:"status"`
	}
	if err := c.BodyParser(&body); err != nil {
		return httptransport.JSONError(c, 400, "bad_request", err.Error())
	}
	item, err := d.UC.UpsertIntegration(c.Context(), &id, body.Type, body.Name, body.BaseURL, body.Credentials, body.Status)
	if err != nil {
		return httptransport.JSONError(c, 400, "bad_request", err.Error())
	}
	return c.JSON(maskInt(item))
}

func (d Deps) deleteInt(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	if err := d.UC.DeleteIntegration(c.Context(), id); err != nil {
		return httptransport.JSONError(c, 400, "bad_request", err.Error())
	}
	return c.SendStatus(204)
}

func (d Deps) testInt(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	item, err := d.UC.TestIntegration(c.Context(), id)
	if err != nil {
		return httptransport.JSONError(c, 400, "bad_request", err.Error())
	}
	return c.JSON(maskInt(item))
}
