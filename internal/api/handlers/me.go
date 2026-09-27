package handlers

import (
	"github.com/gofiber/fiber/v2"

	httptransport "kaiban/internal/api/transport/http"
)

func (d Deps) me(c *fiber.Ctx) error {
	u, err := d.UC.Me(c.Context())
	if err != nil {
		return httptransport.JSONError(c, 500, "internal", err.Error())
	}
	return c.JSON(u)
}
