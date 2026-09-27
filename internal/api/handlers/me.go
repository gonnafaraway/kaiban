package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/pkg/errors"
)

func (d Deps) me(c *fiber.Ctx) error {
	u, err := d.UC.Me(c.Context())
	if err != nil {
		return respondError(c, errors.Wrap(err, "get me"))
	}
	return c.JSON(u)
}
