package handlers

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"kaiban/internal/api/domain/agentrun"
	"kaiban/internal/api/domain/task"
	httptransport "kaiban/internal/api/transport/http"
)

func (d Deps) listTasks(c *fiber.Ctx) error {
	items, err := d.UC.ListTasks(c.Context())
	if err != nil {
		return httptransport.JSONError(c, 500, "internal", err.Error())
	}
	if items == nil {
		items = []*task.Task{}
	}
	return c.JSON(items)
}

func (d Deps) listArchive(c *fiber.Ctx) error {
	items, err := d.UC.ListArchived(c.Context())
	if err != nil {
		return httptransport.JSONError(c, 500, "internal", err.Error())
	}
	out := make([]fiber.Map, 0, len(items))
	for _, item := range items {
		out = append(out, taskPayload(item.Task, item.Reports))
	}
	return c.JSON(out)
}

func (d Deps) createTask(c *fiber.Ctx) error {
	var body struct {
		Title       string            `json:"title"`
		Description string            `json:"description"`
		Variables   map[string]string `json:"variables"`
		Artifacts   task.Artifacts    `json:"artifacts"`
	}
	if err := c.BodyParser(&body); err != nil {
		return httptransport.JSONError(c, 400, "bad_request", err.Error())
	}
	t, err := d.UC.CreateTask(c.Context(), body.Title, body.Description, body.Variables, body.Artifacts)
	if err != nil {
		return mapDomainErr(c, err)
	}
	return c.Status(201).JSON(t)
}

func (d Deps) getTask(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	t, reports, err := d.UC.GetTask(c.Context(), id)
	if err != nil {
		return httptransport.JSONError(c, 404, "not_found", err.Error())
	}
	return c.JSON(taskPayload(t, reports))
}

func (d Deps) taskDiff(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	out, err := d.UC.TaskDiff(c.Context(), id)
	if err != nil {
		return httptransport.JSONError(c, 404, "not_found", err.Error())
	}
	return c.JSON(out)
}

func (d Deps) taskDiffRaw(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	raw, err := d.UC.TaskDiffRaw(c.Context(), id)
	if err != nil {
		return httptransport.JSONError(c, 400, "bad_request", err.Error())
	}
	c.Set("Content-Type", "text/plain; charset=utf-8")
	return c.SendString(raw)
}

func (d Deps) patchTask(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	var body struct {
		Title             *string              `json:"title"`
		Description       *string              `json:"description"`
		Variables         map[string]string    `json:"variables"`
		ExtraInstructions *string              `json:"extra_instructions"`
		Artifacts         *task.Artifacts      `json:"artifacts"`
		Budget            *task.BudgetOverride `json:"budget"`
	}
	if err := c.BodyParser(&body); err != nil {
		return httptransport.JSONError(c, 400, "bad_request", err.Error())
	}
	t, err := d.UC.PatchTask(c.Context(), id, body.Title, body.Description, body.Variables, body.ExtraInstructions, body.Artifacts, body.Budget)
	if err != nil {
		return mapDomainErr(c, err)
	}
	return c.JSON(t)
}

func (d Deps) taskEvents(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	ev, err := d.UC.Events(c.Context(), id)
	if err != nil {
		return httptransport.JSONError(c, 500, "internal", err.Error())
	}
	return c.JSON(ev)
}

func (d Deps) listRuns(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	runs, err := d.UC.ListRuns(c.Context(), id)
	if err != nil {
		return httptransport.JSONError(c, 500, "internal", err.Error())
	}
	if runs == nil {
		runs = []*agentrun.Run{}
	}
	return c.JSON(runs)
}

func (d Deps) getRun(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	run, err := d.UC.GetRun(c.Context(), id)
	if err != nil {
		return httptransport.JSONError(c, 404, "not_found", err.Error())
	}
	return c.JSON(run)
}

func (d Deps) run(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	t, err := d.UC.Run(c.Context(), id)
	if err != nil {
		return mapDomainErr(c, err)
	}
	return c.JSON(t)
}

func (d Deps) approve(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	var body struct {
		Comment string `json:"comment"`
	}
	if err := parseOptionalBody(c, &body); err != nil {
		return err
	}
	t, err := d.UC.Approve(c.Context(), id, body.Comment)
	if err != nil {
		if errors.Is(err, task.ErrNotSucceeded) {
			return httptransport.JSONError(c, 409, "conflict", err.Error())
		}
		return mapDomainErr(c, err)
	}
	return c.JSON(t)
}

func (d Deps) retry(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	var body struct {
		Comment string `json:"comment"`
	}
	if err := parseOptionalBody(c, &body); err != nil {
		return err
	}
	t, err := d.UC.Retry(c.Context(), id, body.Comment)
	if err != nil {
		return mapDomainErr(c, err)
	}
	return c.JSON(t)
}

func (d Deps) returnTo(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	var body struct {
		ColumnID uuid.UUID `json:"column_id"`
		Comment  string    `json:"comment"`
	}
	if err := c.BodyParser(&body); err != nil {
		return httptransport.JSONError(c, 400, "bad_request", err.Error())
	}
	t, err := d.UC.ReturnTo(c.Context(), id, body.ColumnID, body.Comment)
	if err != nil {
		return mapDomainErr(c, err)
	}
	return c.JSON(t)
}

func (d Deps) archive(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	t, err := d.UC.Archive(c.Context(), id)
	if err != nil {
		return mapDomainErr(c, err)
	}
	return c.JSON(t)
}

func (d Deps) unarchive(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}
	t, err := d.UC.Unarchive(c.Context(), id)
	if err != nil {
		return mapDomainErr(c, err)
	}
	return c.JSON(t)
}
