package httptransport

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"go.uber.org/zap"

	"kaiban/internal/api/transport/http/middleware"
)

func NewApp(log *zap.Logger) *fiber.App {
	app := fiber.New(fiber.Config{DisableStartupMessage: true, StreamRequestBody: true})
	app.Use(recover.New())
	app.Use(cors.New(cors.Config{AllowOrigins: "*", AllowHeaders: "*"}))
	app.Use(middleware.RequestID())
	app.Use(middleware.Log(log))
	return app
}

func JSONError(c *fiber.Ctx, status int, code, msg string) error {
	return c.Status(status).JSON(fiber.Map{"error": fiber.Map{"code": code, "message": msg}})
}
