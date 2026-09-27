package handlers

import (
	"github.com/gofiber/fiber/v2"

	httptransport "kaiban/internal/api/transport/http"
	"kaiban/internal/api/usecase/kanban"
)

type Deps struct {
	UC  *kanban.UseCase
	Hub *httptransport.Hub
}

func Register(app *fiber.App, d Deps) {
	app.Get("/health", func(c *fiber.Ctx) error { return c.JSON(fiber.Map{"status": "ok"}) })

	v1 := app.Group("/api/v1")
	v1.Get("/events", d.sse)
	v1.Get("/me", d.me)

	settings := v1.Group("/settings")
	settings.Get("", d.getSettings)
	settings.Put("", d.putSettings)
	settings.Get("/export", d.exportSettings)
	settings.Post("/import", d.importSettings)

	columns := v1.Group("/columns")
	columns.Get("", d.listColumns)
	columns.Post("", d.createColumn)
	columns.Put("/reorder", d.reorder)
	columns.Patch("/:id", d.patchColumn)
	columns.Delete("/:id", d.deleteColumn)
	columns.Post("/:id/reset-overlay", d.resetOverlay)
	columns.Post("/:id/restore-default-prompt", d.restoreDefault)

	tasks := v1.Group("/tasks")
	tasks.Get("", d.listTasks)
	tasks.Post("", d.createTask)
	tasks.Get("/:id", d.getTask)
	tasks.Patch("/:id", d.patchTask)
	tasks.Get("/:id/events", d.taskEvents)
	tasks.Get("/:id/runs", d.listRuns)
	tasks.Get("/:id/diff", d.taskDiff)
	tasks.Get("/:id/diff/raw", d.taskDiffRaw)
	tasks.Post("/:id/run", d.run)
	tasks.Post("/:id/approve", d.approve)
	tasks.Post("/:id/return", d.returnTo)
	tasks.Post("/:id/retry", d.retry)
	tasks.Post("/:id/archive", d.archive)
	tasks.Post("/:id/unarchive", d.unarchive)

	v1.Get("/archive", d.listArchive)
	v1.Get("/runs/:id", d.getRun)

	integrations := v1.Group("/integrations")
	integrations.Get("", d.listInt)
	integrations.Post("", d.createInt)
	integrations.Patch("/:id", d.patchInt)
	integrations.Delete("/:id", d.deleteInt)
	integrations.Post("/:id/test", d.testInt)

	mcp := v1.Group("/mcp-servers")
	mcp.Get("", d.listMCP)
	mcp.Post("", d.createMCP)
	mcp.Patch("/:id", d.patchMCP)
	mcp.Delete("/:id", d.deleteMCP)
	mcp.Post("/:id/test", d.testMCP)
}
