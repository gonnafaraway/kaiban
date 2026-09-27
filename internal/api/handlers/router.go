package handlers

import (
	"bufio"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"kaiban/internal/api/domain/agentrun"
	"kaiban/internal/api/domain/column"
	domint "kaiban/internal/api/domain/integration"
	"kaiban/internal/api/domain/mcpserver"
	"kaiban/internal/api/domain/settings"
	"kaiban/internal/api/domain/task"
	httptransport "kaiban/internal/api/transport/http"
	"kaiban/internal/api/usecase/kanban"
	"kaiban/internal/api/validator"
)

type Deps struct {
	UC  *kanban.UseCase
	Hub *httptransport.Hub
}

func Register(app *fiber.App, d Deps) {
	app.Get("/health", func(c *fiber.Ctx) error { return c.JSON(fiber.Map{"status": "ok"}) })
	app.Get("/api/v1/events", d.sse)
	v1 := app.Group("/api/v1")
	v1.Get("/me", d.me)
	v1.Get("/settings", d.getSettings)
	v1.Put("/settings", d.putSettings)
	v1.Get("/settings/export", d.exportSettings)
	v1.Post("/settings/import", d.importSettings)
	v1.Get("/columns", d.listColumns)
	v1.Post("/columns", d.createColumn)
	v1.Put("/columns/reorder", d.reorder)
	v1.Patch("/columns/:id", d.patchColumn)
	v1.Delete("/columns/:id", d.deleteColumn)
	v1.Post("/columns/:id/reset-overlay", d.resetOverlay)
	v1.Post("/columns/:id/restore-default-prompt", d.restoreDefault)
	v1.Get("/tasks", d.listTasks)
	v1.Get("/archive", d.listArchive)
	v1.Post("/tasks", d.createTask)
	v1.Get("/tasks/:id", d.getTask)
	v1.Patch("/tasks/:id", d.patchTask)
	v1.Get("/tasks/:id/events", d.taskEvents)
	v1.Get("/tasks/:id/runs", d.listRuns)
	v1.Get("/tasks/:id/diff", d.taskDiff)
	v1.Get("/tasks/:id/diff/raw", d.taskDiffRaw)
	v1.Get("/runs/:id", d.getRun)
	v1.Post("/tasks/:id/run", d.run)
	v1.Post("/tasks/:id/approve", d.approve)
	v1.Post("/tasks/:id/return", d.ret)
	v1.Post("/tasks/:id/retry", d.retry)
	v1.Post("/tasks/:id/archive", d.archive)
	v1.Post("/tasks/:id/unarchive", d.unarchive)
	v1.Get("/integrations", d.listInt)
	v1.Post("/integrations", d.createInt)
	v1.Patch("/integrations/:id", d.patchInt)
	v1.Delete("/integrations/:id", d.deleteInt)
	v1.Post("/integrations/:id/test", d.testInt)
	v1.Get("/mcp-servers", d.listMCP)
	v1.Post("/mcp-servers", d.createMCP)
	v1.Patch("/mcp-servers/:id", d.patchMCP)
	v1.Delete("/mcp-servers/:id", d.deleteMCP)
	v1.Post("/mcp-servers/:id/test", d.testMCP)
}

func (d Deps) me(c *fiber.Ctx) error {
	u, err := d.UC.Me(c.Context())
	if err != nil {
		return httptransport.JSONError(c, 500, "internal", err.Error())
	}
	return c.JSON(u)
}

func (d Deps) getSettings(c *fiber.Ctx) error {
	s, err := d.UC.GetSettings(c.Context())
	if err != nil {
		return httptransport.JSONError(c, 500, "internal", err.Error())
	}
	return c.JSON(maskSettings(s))
}

func (d Deps) putSettings(c *fiber.Ctx) error {
	var body settings.Settings
	if err := c.BodyParser(&body); err != nil {
		return httptransport.JSONError(c, 400, "bad_request", err.Error())
	}
	s, err := d.UC.UpdateSettings(c.Context(), &body)
	if err != nil {
		return httptransport.JSONError(c, 400, "bad_request", err.Error())
	}
	return c.JSON(maskSettings(s))
}

func (d Deps) exportSettings(c *fiber.Ctx) error {
	bundle, err := d.UC.ExportConfig(c.Context())
	if err != nil {
		return httptransport.JSONError(c, 500, "internal", err.Error())
	}
	return c.JSON(bundle)
}

func (d Deps) importSettings(c *fiber.Ctx) error {
	var body kanban.ConfigBundle
	if err := c.BodyParser(&body); err != nil {
		return httptransport.JSONError(c, 400, "bad_request", err.Error())
	}
	result, err := d.UC.ImportConfig(c.Context(), &body)
	if err != nil {
		return httptransport.JSONError(c, 400, "bad_request", err.Error())
	}
	return c.JSON(result)
}

func maskSettings(s *settings.Settings) fiber.Map {
	return fiber.Map{
		"id": s.ID, "llm_base_url": s.LLMBaseURL, "llm_api_key": s.MaskedKey(),
		"llm_model": s.LLMModel, "git_repo_url": s.GitRepoURL,
		"git_default_branch": s.GitDefaultBranch, "locale": s.Locale,
		"max_tokens": s.MaxTokens, "max_cost_usd": s.MaxCostUSD, "max_wall_sec": s.MaxWallSec,
		"max_tool_calls": s.MaxToolCalls, "max_llm_steps": s.MaxLLMSteps,
		"price_input_per_1k": s.PriceInputPer1K, "price_output_per_1k": s.PriceOutputPer1K,
		"context_pack": s.ContextPack,
		"updated_at":   s.UpdatedAt,
	}
}

func (d Deps) listColumns(c *fiber.Ctx) error {
	cols, err := d.UC.ListColumns(c.Context())
	if err != nil {
		return httptransport.JSONError(c, 500, "internal", err.Error())
	}
	return c.JSON(cols)
}

func (d Deps) createColumn(c *fiber.Ctx) error {
	var body struct {
		Name         string `json:"name"`
		SystemPrompt string `json:"system_prompt_template"`
		OrderIndex   int    `json:"order_index"`
	}
	if err := c.BodyParser(&body); err != nil {
		return httptransport.JSONError(c, 400, "bad_request", err.Error())
	}
	if err := validator.Required(body.Name); err != nil {
		return httptransport.JSONError(c, 400, "bad_request", err.Error())
	}
	col, err := d.UC.CreateColumn(c.Context(), body.Name, body.SystemPrompt, body.OrderIndex)
	if err != nil {
		return httptransport.JSONError(c, 400, "bad_request", err.Error())
	}
	return c.Status(201).JSON(col)
}

func (d Deps) patchColumn(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httptransport.JSONError(c, 400, "bad_request", "invalid id")
	}
	var body struct {
		Name                 *string               `json:"name"`
		NameI18n             map[string]string     `json:"name_i18n"`
		SystemPromptTemplate *string               `json:"system_prompt_template"`
		UserCustomPrompt     *string               `json:"user_custom_prompt"`
		OutputFields         *[]column.OutputField `json:"output_fields"`
		Budget               *column.BudgetLimits  `json:"budget"`
		RequiresGitDiff      *bool                 `json:"requires_git_diff"`
	}
	if err := c.BodyParser(&body); err != nil {
		return httptransport.JSONError(c, 400, "bad_request", err.Error())
	}
	col, err := d.UC.PatchColumn(c.Context(), id, body.Name, body.SystemPromptTemplate, body.UserCustomPrompt, body.NameI18n, body.OutputFields, body.Budget, body.RequiresGitDiff)
	if err != nil {
		return httptransport.JSONError(c, 400, "bad_request", err.Error())
	}
	return c.JSON(col)
}

func (d Deps) deleteColumn(c *fiber.Ctx) error {
	id, _ := uuid.Parse(c.Params("id"))
	if err := d.UC.DeleteColumn(c.Context(), id); err != nil {
		return httptransport.JSONError(c, 409, "conflict", err.Error())
	}
	return c.SendStatus(204)
}

func (d Deps) resetOverlay(c *fiber.Ctx) error {
	id, _ := uuid.Parse(c.Params("id"))
	col, err := d.UC.ResetOverlay(c.Context(), id)
	if err != nil {
		return httptransport.JSONError(c, 400, "bad_request", err.Error())
	}
	return c.JSON(col)
}

func (d Deps) restoreDefault(c *fiber.Ctx) error {
	id, _ := uuid.Parse(c.Params("id"))
	col, err := d.UC.RestoreDefault(c.Context(), id)
	if err != nil {
		return httptransport.JSONError(c, 400, "bad_request", err.Error())
	}
	return c.JSON(col)
}

func (d Deps) reorder(c *fiber.Ctx) error {
	var body []struct {
		ID         uuid.UUID `json:"id"`
		OrderIndex int       `json:"order_index"`
	}
	if err := c.BodyParser(&body); err != nil {
		return httptransport.JSONError(c, 400, "bad_request", err.Error())
	}
	ids := make([]uuid.UUID, len(body))
	for i, b := range body {
		ids[i] = b.ID
	}
	if err := d.UC.ReorderColumns(c.Context(), ids); err != nil {
		return httptransport.JSONError(c, 400, "bad_request", err.Error())
	}
	return c.JSON(fiber.Map{"ok": true})
}

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
	id, _ := uuid.Parse(c.Params("id"))
	t, reports, err := d.UC.GetTask(c.Context(), id)
	if err != nil {
		return httptransport.JSONError(c, 404, "not_found", err.Error())
	}
	return c.JSON(taskPayload(t, reports))
}

func (d Deps) taskDiff(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httptransport.JSONError(c, 400, "bad_request", "invalid id")
	}
	out, err := d.UC.TaskDiff(c.Context(), id)
	if err != nil {
		return httptransport.JSONError(c, 404, "not_found", err.Error())
	}
	return c.JSON(out)
}

func (d Deps) taskDiffRaw(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httptransport.JSONError(c, 400, "bad_request", "invalid id")
	}
	raw, err := d.UC.TaskDiffRaw(c.Context(), id)
	if err != nil {
		return httptransport.JSONError(c, 400, "bad_request", err.Error())
	}
	c.Set("Content-Type", "text/plain; charset=utf-8")
	return c.SendString(raw)
}

func (d Deps) patchTask(c *fiber.Ctx) error {
	id, _ := uuid.Parse(c.Params("id"))
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
	id, _ := uuid.Parse(c.Params("id"))
	ev, err := d.UC.Events(c.Context(), id)
	if err != nil {
		return httptransport.JSONError(c, 500, "internal", err.Error())
	}
	return c.JSON(ev)
}

func (d Deps) listRuns(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httptransport.JSONError(c, 400, "bad_request", "invalid id")
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
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return httptransport.JSONError(c, 400, "bad_request", "invalid id")
	}
	run, err := d.UC.GetRun(c.Context(), id)
	if err != nil {
		return httptransport.JSONError(c, 404, "not_found", err.Error())
	}
	return c.JSON(run)
}

func (d Deps) run(c *fiber.Ctx) error {
	id, _ := uuid.Parse(c.Params("id"))
	t, err := d.UC.Run(c.Context(), id)
	if err != nil {
		return mapDomainErr(c, err)
	}
	return c.JSON(t)
}

func (d Deps) approve(c *fiber.Ctx) error {
	id, _ := uuid.Parse(c.Params("id"))
	var body struct {
		Comment string `json:"comment"`
	}
	_ = c.BodyParser(&body)
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
	id, _ := uuid.Parse(c.Params("id"))
	var body struct {
		Comment string `json:"comment"`
	}
	_ = c.BodyParser(&body)
	t, err := d.UC.Retry(c.Context(), id, body.Comment)
	if err != nil {
		return mapDomainErr(c, err)
	}
	return c.JSON(t)
}

func (d Deps) ret(c *fiber.Ctx) error {
	id, _ := uuid.Parse(c.Params("id"))
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
	id, _ := uuid.Parse(c.Params("id"))
	t, err := d.UC.Archive(c.Context(), id)
	if err != nil {
		return mapDomainErr(c, err)
	}
	return c.JSON(t)
}

func (d Deps) unarchive(c *fiber.Ctx) error {
	id, _ := uuid.Parse(c.Params("id"))
	t, err := d.UC.Unarchive(c.Context(), id)
	if err != nil {
		return mapDomainErr(c, err)
	}
	return c.JSON(t)
}

func (d Deps) listInt(c *fiber.Ctx) error {
	items, err := d.UC.ListIntegrations(c.Context())
	if err != nil {
		return httptransport.JSONError(c, 500, "internal", err.Error())
	}
	out := make([]fiber.Map, 0, len(items))
	for _, i := range items {
		out = append(out, maskInt(i))
	}
	return c.JSON(out)
}

func maskInt(i *domint.Integration) fiber.Map {
	return fiber.Map{
		"id": i.ID, "type": i.Type, "name": i.Name, "base_url": i.BaseURL,
		"credentials": i.MaskedCredentials(), "status": i.Status, "last_error": i.LastError,
		"created_at": i.CreatedAt, "updated_at": i.UpdatedAt,
	}
}

func (d Deps) createInt(c *fiber.Ctx) error {
	var body struct {
		Type        string            `json:"type"`
		Name        string            `json:"name"`
		BaseURL     string            `json:"base_url"`
		Credentials map[string]string `json:"credentials"`
		Status      string            `json:"status"`
	}
	if err := c.BodyParser(&body); err != nil {
		return httptransport.JSONError(c, 400, "bad_request", err.Error())
	}
	item, err := d.UC.UpsertIntegration(c.Context(), nil, body.Type, body.Name, body.BaseURL, body.Credentials, body.Status)
	if err != nil {
		return httptransport.JSONError(c, 400, "bad_request", err.Error())
	}
	return c.Status(201).JSON(maskInt(item))
}

func (d Deps) patchInt(c *fiber.Ctx) error {
	id, _ := uuid.Parse(c.Params("id"))
	var body struct {
		Type        string            `json:"type"`
		Name        string            `json:"name"`
		BaseURL     string            `json:"base_url"`
		Credentials map[string]string `json:"credentials"`
		Status      string            `json:"status"`
	}
	_ = c.BodyParser(&body)
	item, err := d.UC.UpsertIntegration(c.Context(), &id, body.Type, body.Name, body.BaseURL, body.Credentials, body.Status)
	if err != nil {
		return httptransport.JSONError(c, 400, "bad_request", err.Error())
	}
	return c.JSON(maskInt(item))
}

func (d Deps) deleteInt(c *fiber.Ctx) error {
	id, _ := uuid.Parse(c.Params("id"))
	if err := d.UC.DeleteIntegration(c.Context(), id); err != nil {
		return httptransport.JSONError(c, 400, "bad_request", err.Error())
	}
	return c.SendStatus(204)
}

func (d Deps) testInt(c *fiber.Ctx) error {
	id, _ := uuid.Parse(c.Params("id"))
	item, err := d.UC.TestIntegration(c.Context(), id)
	if err != nil {
		return httptransport.JSONError(c, 400, "bad_request", err.Error())
	}
	return c.JSON(maskInt(item))
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
	id, _ := uuid.Parse(c.Params("id"))
	var body struct {
		Name     string            `json:"name"`
		Endpoint string            `json:"endpoint"`
		Headers  map[string]string `json:"headers"`
		Status   string            `json:"status"`
	}
	_ = c.BodyParser(&body)
	item, err := d.UC.UpsertMCP(c.Context(), &id, body.Name, body.Endpoint, body.Headers, body.Status)
	if err != nil {
		return httptransport.JSONError(c, 400, "bad_request", err.Error())
	}
	return c.JSON(maskMCP(item))
}

func (d Deps) deleteMCP(c *fiber.Ctx) error {
	id, _ := uuid.Parse(c.Params("id"))
	if err := d.UC.DeleteMCP(c.Context(), id); err != nil {
		return httptransport.JSONError(c, 400, "bad_request", err.Error())
	}
	return c.SendStatus(204)
}

func (d Deps) testMCP(c *fiber.Ctx) error {
	id, _ := uuid.Parse(c.Params("id"))
	item, err := d.UC.TestMCP(c.Context(), id)
	if err != nil {
		return httptransport.JSONError(c, 400, "bad_request", err.Error())
	}
	return c.JSON(maskMCP(item))
}

func (d Deps) sse(c *fiber.Ctx) error {
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	ch := d.Hub.Subscribe()
	defer d.Hub.Unsubscribe(ch)
	c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case msg, ok := <-ch:
				if !ok {
					return
				}
				var wrap map[string]any
				_ = json.Unmarshal(msg, &wrap)
				ev, _ := wrap["event"].(string)
				_, _ = w.WriteString("event: " + ev + "\n")
				_, _ = w.WriteString("data: " + string(msg) + "\n\n")
				_ = w.Flush()
			case <-ticker.C:
				_, _ = w.WriteString(": ping\n\n")
				_ = w.Flush()
			}
		}
	})
	return nil
}

func mapDomainErr(c *fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, task.ErrTitleRequired), errors.Is(err, task.ErrCommentRequired), errors.Is(err, task.ErrReturnTarget), errors.Is(err, task.ErrInvalidTransition), errors.Is(err, task.ErrNotArchived), errors.Is(err, task.ErrContractInvalid), errors.Is(err, task.ErrGitDiffRequired):
		return httptransport.JSONError(c, 400, "bad_request", err.Error())
	case errors.Is(err, task.ErrAlreadyRunning), errors.Is(err, task.ErrNotSucceeded), errors.Is(err, task.ErrForwardMove), errors.Is(err, task.ErrArchived), errors.Is(err, task.ErrAlreadyArchived):
		return httptransport.JSONError(c, 409, "conflict", err.Error())
	default:
		return httptransport.JSONError(c, 500, "internal", err.Error())
	}
}

func taskPayload(t *task.Task, reports any) fiber.Map {
	return fiber.Map{
		"id": t.ID, "title": t.Title, "description": t.Description, "variables": t.Variables,
		"column_id": t.ColumnID, "execution_status": t.ExecutionStatus, "git_branch": t.GitBranch,
		"git_pr_url": t.GitPRURL, "git_push_status": t.GitPushStatus, "git_pr_status": t.GitPRStatus,
		"current_report": t.CurrentReport, "context_data": t.ContextData, "created_by": t.CreatedBy,
		"created_at": t.CreatedAt, "updated_at": t.UpdatedAt, "archived_at": t.ArchivedAt, "reports": reports,
	}
}
