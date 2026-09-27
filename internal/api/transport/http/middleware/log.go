package middleware

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func RequestID() fiber.Handler {
	return func(c *fiber.Ctx) error {
		id := c.Get("X-Request-ID")
		if id == "" {
			id = uuid.NewString()
		}
		c.Set("X-Request-ID", id)
		c.Locals("request_id", id)
		return c.Next()
	}
}

func Log(log *zap.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		err := c.Next()
		log.Info("http",
			zap.String("method", c.Method()),
			zap.String("path", c.Path()),
			zap.Int("status", c.Response().StatusCode()),
			zap.String("request_id", c.Get("X-Request-ID")),
		)
		return err
	}
}
