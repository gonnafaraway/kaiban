package handlers

import (
	"github.com/gofiber/fiber/v2"
)

func (d Deps) me(c *fiber.Ctx) error {
	u, err := d.UC.Me(c.Context())
	if err != nil {
		return respondError(c, err)
	}
	return c.JSON(u)
}
