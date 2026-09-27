package handlers

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"kaiban/internal/api/domain/task"
	httptransport "kaiban/internal/api/transport/http"
)

func parseID(c *fiber.Ctx) (uuid.UUID, error) {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return uuid.Nil, httptransport.JSONError(c, 400, "bad_request", "invalid id")
	}
	return id, nil
}

func parseOptionalBody(c *fiber.Ctx, out any) error {
	if len(c.Body()) == 0 {
		return nil
	}
	if err := c.BodyParser(out); err != nil {
		return httptransport.JSONError(c, 400, "bad_request", err.Error())
	}
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
