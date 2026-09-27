package handlers

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"kaiban/internal/api/domain/mcpserver"
	httptransport "kaiban/internal/api/transport/http"
)

func maskMCP(s *mcpserver.Server) fiber.Map {
	h := map[string]string{}
	for k, v := range s.Headers {
		if strings.EqualFold(k, "Authorization") && len(v) > 4 {
			h[k] = "****" + v[len(v)-4:]
		} else {
			h[k] = v
		}
	}
	return fiber.Map{
		"id": s.ID, "name": s.Name, "endpoint": s.Endpoint, "headers": h,
		"capabilities": s.Capabilities, "status": s.Status, "last_error": s.LastError,
		"created_at": s.CreatedAt, "updated_at": s.UpdatedAt,
	}
}

func (d Deps) listMCP(c *fiber.Ctx) error {
	items, err := d.UC.ListMCP(c.Context())
	if err != nil {
		return httptransport.JSONError(c, 500, "internal", err.Error())
	}
	out := make([]fiber.Map, 0, len(items))
	for _, i := range items {
		out = append(out, maskMCP(i))
	}
	return c.JSON(out)
}

func (d Deps) createMCP(c *fiber.Ctx) error {
	var body struct {
		Name     string            `json:"name"`
		Endpoint string            `json:"endpoint"`
		Headers  map[string]string `json:"headers"`
		Status   string            `json:"status"`
	}
	if err := c.BodyParser(&body); err != nil {
		return httptransport.JSONError(c, 400, "bad_request", err.Error())
	}
	item, err := d.UC.UpsertMCP(c.Context(), nil, body.Name, body.Endpoint, body.Headers, body.Status)
	if err != nil {
		return httptransport.JSONError(c, 400, "bad_request", err.Error())
	}
	return c.Status(201).JSON(maskMCP(item))
}

func (d Deps) patchMCP(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	var body struct {
		Name     string            `json:"name"`
		Endpoint string            `json:"endpoint"`
		Headers  map[string]string `json:"headers"`
		Status   string            `json:"status"`
	}
	if err := c.BodyParser(&body); err != nil {
		return httptransport.JSONError(c, 400, "bad_request", err.Error())
	}
	item, err := d.UC.UpsertMCP(c.Context(), &id, body.Name, body.Endpoint, body.Headers, body.Status)
	if err != nil {
		return httptransport.JSONError(c, 400, "bad_request", err.Error())
	}
	return c.JSON(maskMCP(item))
}

func (d Deps) deleteMCP(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	if err := d.UC.DeleteMCP(c.Context(), id); err != nil {
		return httptransport.JSONError(c, 400, "bad_request", err.Error())
	}
	return c.SendStatus(204)
}

func (d Deps) testMCP(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	item, err := d.UC.TestMCP(c.Context(), id)
	if err != nil {
		return httptransport.JSONError(c, 400, "bad_request", err.Error())
	}
	return c.JSON(maskMCP(item))
}
