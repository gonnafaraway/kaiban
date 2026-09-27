package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/pkg/errors"

	"kaiban/internal/api/domain/agentrun"
	"kaiban/internal/api/domain/task"
)

type createTaskBody struct {
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Variables   map[string]string `json:"variables"`
	Artifacts   task.Artifacts    `json:"artifacts"`
}

type patchTaskBody struct {
	Title             *string              `json:"title"`
	Description       *string              `json:"description"`
	Variables         map[string]string    `json:"variables"`
	ExtraInstructions *string              `json:"extra_instructions"`
	Artifacts         *task.Artifacts      `json:"artifacts"`
	Budget            *task.BudgetOverride `json:"budget"`
}

type commentBody struct {
	Comment string `json:"comment"`
}

type returnTaskBody struct {
	ColumnID uuid.UUID `json:"column_id"`
	Comment  string    `json:"comment"`
}

func (d Deps) listTasks(c *fiber.Ctx) error {
	items, err := d.UC.ListTasks(c.Context())
	if err != nil {
		return respondError(c, err)
	}
	if items == nil {
		items = []*task.Task{}
	}
	return c.JSON(items)
}

func (d Deps) listArchive(c *fiber.Ctx) error {
	items, err := d.UC.ListArchived(c.Context())
	if err != nil {
		return respondError(c, err)
	}
	out := make([]taskResponse, 0, len(items))
	for _, item := range items {
		out = append(out, taskPayload(item.Task, item.Reports))
	}
	return c.JSON(out)
}

func (d Deps) createTask(c *fiber.Ctx) error {
	var body createTaskBody
	if err := parseBody(c, &body); err != nil {
		return respondError(c, errors.Wrap(err, "create task"))
	}
	t, err := d.UC.CreateTask(c.Context(), body.Title, body.Description, body.Variables, body.Artifacts)
	if err != nil {
		return respondError(c, err)
	}
	return c.Status(201).JSON(t)
}

func (d Deps) getTask(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return respondError(c, errors.Wrap(err, "get task"))
	}
	t, reports, err := d.UC.GetTask(c.Context(), id)
	if err != nil {
		return respondError(c, err)
	}
	return c.JSON(taskPayload(t, reports))
}

func (d Deps) taskDiff(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return respondError(c, errors.Wrap(err, "task diff"))
	}
	out, err := d.UC.TaskDiff(c.Context(), id)
	if err != nil {
		return respondError(c, err)
	}
	return c.JSON(out)
}

func (d Deps) taskDiffRaw(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return respondError(c, errors.Wrap(err, "task diff raw"))
	}
	raw, err := d.UC.TaskDiffRaw(c.Context(), id)
	if err != nil {
		return respondError(c, err)
	}
	c.Set("Content-Type", "text/plain; charset=utf-8")
	return c.SendString(raw)
}

func (d Deps) patchTask(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return respondError(c, errors.Wrap(err, "patch task"))
	}
	var body patchTaskBody
	if err := parseBody(c, &body); err != nil {
		return respondError(c, errors.Wrap(err, "patch task"))
	}
	t, err := d.UC.PatchTask(c.Context(), id, body.Title, body.Description, body.Variables, body.ExtraInstructions, body.Artifacts, body.Budget)
	if err != nil {
		return respondError(c, err)
	}
	return c.JSON(t)
}

func (d Deps) taskEvents(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return respondError(c, errors.Wrap(err, "task events"))
	}
	ev, err := d.UC.Events(c.Context(), id)
	if err != nil {
		return respondError(c, err)
	}
	return c.JSON(ev)
}

func (d Deps) listRuns(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return respondError(c, errors.Wrap(err, "list task runs"))
	}
	runs, err := d.UC.ListRuns(c.Context(), id)
	if err != nil {
		return respondError(c, err)
	}
	if runs == nil {
		runs = []*agentrun.Run{}
	}
	return c.JSON(runs)
}

func (d Deps) getRun(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return respondError(c, errors.Wrap(err, "get run"))
	}
	run, err := d.UC.GetRun(c.Context(), id)
	if err != nil {
		return respondError(c, err)
	}
	return c.JSON(run)
}

func (d Deps) run(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return respondError(c, errors.Wrap(err, "run task"))
	}
	t, err := d.UC.Run(c.Context(), id)
	if err != nil {
		return respondError(c, err)
	}
	return c.JSON(t)
}

func (d Deps) approve(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return respondError(c, errors.Wrap(err, "approve task"))
	}
	var body commentBody
	if err := parseOptionalBody(c, &body); err != nil {
		return respondError(c, errors.Wrap(err, "approve task"))
	}
	t, err := d.UC.Approve(c.Context(), id, body.Comment)
	if err != nil {
		return respondError(c, err)
	}
	return c.JSON(t)
}

func (d Deps) retry(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return respondError(c, errors.Wrap(err, "retry task"))
	}
	var body commentBody
	if err := parseOptionalBody(c, &body); err != nil {
		return respondError(c, errors.Wrap(err, "retry task"))
	}
	t, err := d.UC.Retry(c.Context(), id, body.Comment)
	if err != nil {
		return respondError(c, err)
	}
	return c.JSON(t)
}

func (d Deps) returnTo(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return respondError(c, errors.Wrap(err, "return task"))
	}
	var body returnTaskBody
	if err := parseBody(c, &body); err != nil {
		return respondError(c, errors.Wrap(err, "return task"))
	}
	t, err := d.UC.ReturnTo(c.Context(), id, body.ColumnID, body.Comment)
	if err != nil {
		return respondError(c, err)
	}
	return c.JSON(t)
}

func (d Deps) archive(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return respondError(c, errors.Wrap(err, "archive task"))
	}
	t, err := d.UC.Archive(c.Context(), id)
	if err != nil {
		return respondError(c, err)
	}
	return c.JSON(t)
}

func (d Deps) unarchive(c *fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return respondError(c, errors.Wrap(err, "unarchive task"))
	}
	t, err := d.UC.Unarchive(c.Context(), id)
	if err != nil {
		return respondError(c, err)
	}
	return c.JSON(t)
}
