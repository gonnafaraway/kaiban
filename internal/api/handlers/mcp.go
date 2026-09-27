package handlers

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/pkg/errors"

	"kaiban/internal/api/domain/mcpserver"
)

type mcpBody struct {
	Name     string            `json:"name"`
	Endpoint string            `json:"endpoint"`
	Headers  map[string]string `json:"headers"`
	Status   string            `json:"status"`
}

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
		return respondError(c, errors.Wrap(err, "list mcp servers"))
	}
	out := make([]fiber.Map, 0, len(items))
	for _, i := range items {
		out = append(out, maskMCP(i))
	}
	return c.JSON(out)
}

func (d Deps) createMCP(c *fiber.Ctx) error {
	var body mcpBody
	if err := parseBody(c, &body); err != nil {
		return respondError(c, errors.Wrap(err, "create mcp server"))
	}
	item, err := d.UC.UpsertMCP(c.Context(), nil, body.Name, body.Endpoint, body.Headers, body.Status)
	if err != nil {
		return respondError(c, errors.Wrap(err, "create mcp server"))
	}
	return c.Status(201).JSON(maskMCP(item))
}

func (d Deps) patchMCP(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return respondError(c, errors.Wrap(err, "patch mcp server"))
	}
	var body mcpBody
	if err := parseBody(c, &body); err != nil {
		return respondError(c, errors.Wrap(err, "patch mcp server"))
	}
	item, err := d.UC.UpsertMCP(c.Context(), &id, body.Name, body.Endpoint, body.Headers, body.Status)
	if err != nil {
		return respondError(c, errors.Wrap(err, "patch mcp server"))
	}
	return c.JSON(maskMCP(item))
}

func (d Deps) deleteMCP(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return respondError(c, errors.Wrap(err, "delete mcp server"))
	}
	if err := d.UC.DeleteMCP(c.Context(), id); err != nil {
		return respondError(c, errors.Wrap(err, "delete mcp server"))
	}
	return c.SendStatus(204)
}

func (d Deps) testMCP(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return respondError(c, errors.Wrap(err, "test mcp server"))
	}
	item, err := d.UC.TestMCP(c.Context(), id)
	if err != nil {
		return respondError(c, errors.Wrap(err, "test mcp server"))
	}
	return c.JSON(maskMCP(item))
}
