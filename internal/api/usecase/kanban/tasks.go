package kanban

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	"kaiban/internal/api/domain/auditevent"
	"kaiban/internal/api/domain/column"
	"kaiban/internal/api/domain/job"
	"kaiban/internal/api/domain/task"
	"kaiban/internal/api/integration"
	httptransport "kaiban/internal/api/transport/http"
)

func (u *Board) ListTasks(ctx context.Context) ([]*task.Task, error) {
	items, err := u.Repo.Tasks.List(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "list tasks")
	}
	return items, nil
}

func (u *Board) ListArchived(ctx context.Context) ([]ArchivedTask, error) {
	items, err := u.Repo.Tasks.ListArchived(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "list archive")
	}
	out := make([]ArchivedTask, 0, len(items))
	for _, t := range items {
		reports, err := u.Repo.Tasks.ListReports(ctx, t.ID)
		if err != nil {
			return nil, errors.Wrap(err, "list reports")
		}
		out = append(out, ArchivedTask{Task: t, Reports: reports})
	}
	return out, nil
}

func (u *Board) GetTask(ctx context.Context, id uuid.UUID) (*task.Task, []task.Report, error) {
	t, err := u.Repo.Tasks.Get(ctx, id)
	if err != nil {
		return nil, nil, errors.Wrap(err, "get task")
	}
	reports, err := u.Repo.Tasks.ListReports(ctx, id)
	if err != nil {
		return t, nil, errors.Wrap(err, "list reports")
	}
	return t, reports, nil
}

func (u *Board) CreateTask(ctx context.Context, title, desc string, vars map[string]string, artifacts task.Artifacts) (*task.Task, error) {
	col, err := u.Repo.Columns.First(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "first column")
	}
	me, err := u.Repo.Users.GetLocal(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "get local user")
	}
	t, err := task.New(title, desc, vars, artifacts, col.ID, me.ID)
	if err != nil {
		return nil, errors.Wrap(err, "new")
	}
	t.GitBranch = "kaiban/task-" + t.ID.String()[:8]
	st, _ := u.Repo.Settings.Get(ctx)
	repoURL, gitTok, useGitHub := u.ResolveGitRemote(ctx, st, artifacts)
	defaultBranch := "main"
	if st != nil && st.GitDefaultBranch != "" {
		defaultBranch = st.GitDefaultBranch
	}
	if repoURL != "" {
		if useGitHub {
			repoURL = integration.AuthenticatedGitHubURL(repoURL, gitTok)
		} else {
			repoURL = integration.AuthenticatedGitURL(repoURL, gitTok)
		}
		_ = integration.EnsureTaskBranch(ctx, u.GitWorkDir, repoURL, defaultBranch, t.GitBranch)
	}
	if err := u.Repo.Tasks.Create(ctx, t); err != nil {
		return nil, errors.Wrap(err, "create")
	}
	_ = u.Audit(ctx, &t.ID, auditevent.ActorUser, me.Login, auditevent.ActionTaskCreated, map[string]any{
		"title": title, "artifacts": artifacts,
	})
	u.Publish(httptransport.EventTaskCreated, t)
	return t, nil
}

func (u *Board) PatchTask(ctx context.Context, id uuid.UUID, title, desc *string, vars map[string]string, extra *string, artifacts *task.Artifacts, budget *task.BudgetOverride) (*task.Task, error) {
	t, err := u.Repo.Tasks.Get(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "get task")
	}
	if t.IsArchived() {
		return nil, task.ErrArchived
	}
	if t.ExecutionStatus == task.StatusQueued || t.ExecutionStatus == task.StatusRunning {
		return nil, task.ErrAlreadyRunning
	}
	if title != nil {
		t.Title = *title
	}
	if desc != nil {
		t.Description = *desc
	}
	if vars != nil {
		t.Variables = vars
	}
	if extra != nil {
		t.ContextData.ExtraInstructions = *extra
	}
	if artifacts != nil {
		t.ContextData.Artifacts = *artifacts
	}
	if budget != nil {
		t.ContextData.Budget = *budget
	}
	t.UpdatedAt = time.Now().UTC()
	if err := u.Repo.Tasks.Update(ctx, t); err != nil {
		return nil, errors.Wrap(err, "update")
	}
	u.Publish(httptransport.EventTaskUpdated, t)
	return t, nil
}

func (u *Board) Run(ctx context.Context, id uuid.UUID) (*task.Task, error) {
	t, err := u.Repo.Tasks.Get(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "get task")
	}
	prevStatus := t.ExecutionStatus
	if err := t.MarkQueued(); err != nil {
		return nil, errors.Wrap(err, "mark queued")
	}
	if err := u.Repo.Tasks.Update(ctx, t); err != nil {
		return nil, errors.Wrap(err, "update")
	}
	if err := u.Repo.Jobs.Enqueue(ctx, &job.Job{TaskID: t.ID, ColumnID: t.ColumnID}); err != nil {
		t.ExecutionStatus = prevStatus
		t.UpdatedAt = time.Now().UTC()
		if rollbackErr := u.Repo.Tasks.Update(ctx, t); rollbackErr != nil {
			return nil, errors.Wrapf(err, "enqueue job (status rollback failed: %v)", rollbackErr)
		}
		return nil, errors.Wrap(err, "enqueue job")
	}
	_ = u.Audit(ctx, &t.ID, auditevent.ActorUser, u.LocalActor(ctx), auditevent.ActionAgentStarted, nil)
	u.Publish(httptransport.EventTaskUpdated, t)
	u.AgentLog(t.ID, "status", "Задача в очереди, ожидает воркер", nil)
	return t, nil
}

func (u *Board) Approve(ctx context.Context, id uuid.UUID, comment string) (*task.Task, error) {
	t, err := u.Repo.Tasks.Get(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "get task")
	}
	cur, err := u.Repo.Columns.Get(ctx, t.ColumnID)
	if err != nil {
		return nil, errors.Wrap(err, "get column")
	}
	if err := column.ValidateOutputs(cur.OutputFields, t.StageOutput(t.ColumnID)); err != nil {
		return nil, fmt.Errorf("%w: %s", task.ErrContractInvalid, err.Error())
	}
	if cur.RequiresGitDiff {
		st, err := u.Repo.Settings.Get(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "get settings")
		}
		repoURL, _, _ := u.ResolveGitRemote(ctx, st, t.ContextData.Artifacts)
		refreshErr := integration.RefreshTaskWorktree(ctx, u.GitWorkDir, repoURL, t.GitBranch, st.GitDefaultBranch)
		sum, err := integration.TaskDiffSummary(ctx, u.GitWorkDir, t.GitBranch, st.GitDefaultBranch)
		if err != nil || sum == nil || sum.Empty {
			msg := "no changes vs " + st.GitDefaultBranch
			if err != nil {
				msg = err.Error()
			}
			// A stale worktree is the usual reason for an empty diff — say so.
			if refreshErr != nil {
				msg += " (worktree refresh failed: " + refreshErr.Error() + ")"
			}
			return nil, fmt.Errorf("%w: %s", task.ErrGitDiffRequired, msg)
		}
	}
	next, err := u.Repo.Columns.Next(ctx, cur.OrderIndex)
	if err != nil {
		return nil, errors.Wrap(err, "next")
	}
	isLast := next == nil
	var nextID uuid.UUID
	if next != nil {
		nextID = next.ID
	}
	if err := t.Approve(isLast, nextID); err != nil {
		return nil, errors.Wrap(err, "approve task")
	}
	if err := u.Repo.Tasks.Update(ctx, t); err != nil {
		return nil, errors.Wrap(err, "update")
	}
	_ = u.Audit(ctx, &t.ID, auditevent.ActorUser, u.LocalActor(ctx), auditevent.ActionUserApproved, map[string]any{"comment": comment})
	u.Publish(httptransport.EventTaskUpdated, t)
	return t, nil
}

func (u *Board) Retry(ctx context.Context, id uuid.UUID, comment string) (*task.Task, error) {
	t, err := u.Repo.Tasks.Get(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "get task")
	}
	if err := t.RetryCurrent(comment); err != nil {
		return nil, errors.Wrap(err, "retry current")
	}
	if err := u.Repo.Jobs.CancelQueuedForTask(ctx, t.ID); err != nil {
		return nil, errors.Wrap(err, "cancel queued jobs")
	}
	if err := u.Repo.Tasks.Update(ctx, t); err != nil {
		return nil, errors.Wrap(err, "update")
	}
	_ = u.Audit(ctx, &t.ID, auditevent.ActorUser, u.LocalActor(ctx), auditevent.ActionUserRetried, map[string]any{"comment": comment})
	u.Publish(httptransport.EventTaskUpdated, t)
	return t, nil
}

func (u *Board) ReturnTo(ctx context.Context, id, columnID uuid.UUID, comment string) (*task.Task, error) {
	t, err := u.Repo.Tasks.Get(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "get task")
	}
	cur, err := u.Repo.Columns.Get(ctx, t.ColumnID)
	if err != nil {
		return nil, errors.Wrap(err, "get column")
	}
	target, err := u.Repo.Columns.Get(ctx, columnID)
	if err != nil {
		return nil, errors.Wrap(err, "get column")
	}
	if err := t.ReturnTo(cur, target, comment); err != nil {
		return nil, errors.Wrap(err, "return task")
	}
	if err := u.Repo.Jobs.CancelQueuedForTask(ctx, t.ID); err != nil {
		return nil, errors.Wrap(err, "cancel queued jobs")
	}
	if err := u.Repo.Tasks.Update(ctx, t); err != nil {
		return nil, errors.Wrap(err, "update")
	}
	_ = u.Audit(ctx, &t.ID, auditevent.ActorUser, u.LocalActor(ctx), auditevent.ActionUserReturned, map[string]any{"comment": comment, "column_id": columnID.String()})
	u.Publish(httptransport.EventTaskUpdated, t)
	return t, nil
}

func (u *Board) Archive(ctx context.Context, id uuid.UUID) (*task.Task, error) {
	t, err := u.Repo.Tasks.Get(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "get task")
	}
	if err := t.Archive(); err != nil {
		return nil, errors.Wrap(err, "archive task")
	}
	if err := u.Repo.Jobs.CancelQueuedForTask(ctx, t.ID); err != nil {
		return nil, errors.Wrap(err, "cancel queued jobs")
	}
	if err := u.Repo.Tasks.Update(ctx, t); err != nil {
		return nil, errors.Wrap(err, "update")
	}
	_ = u.Audit(ctx, &t.ID, auditevent.ActorUser, u.LocalActor(ctx), auditevent.ActionTaskArchived, nil)
	u.Publish(httptransport.EventTaskUpdated, t)
	return t, nil
}

func (u *Board) Unarchive(ctx context.Context, id uuid.UUID) (*task.Task, error) {
	t, err := u.Repo.Tasks.Get(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "get task")
	}
	if err := t.Unarchive(); err != nil {
		return nil, errors.Wrap(err, "unarchive task")
	}
	if err := u.Repo.Tasks.Update(ctx, t); err != nil {
		return nil, errors.Wrap(err, "update")
	}
	_ = u.Audit(ctx, &t.ID, auditevent.ActorUser, u.LocalActor(ctx), auditevent.ActionTaskUnarchived, nil)
	u.Publish(httptransport.EventTaskUpdated, t)
	return t, nil
}

func (u *Board) Events(ctx context.Context, taskID uuid.UUID) ([]*auditevent.Event, error) {
	items, err := u.Repo.Audit.ListByTask(ctx, taskID)
	if err != nil {
		return nil, errors.Wrap(err, "list audit events")
	}
	return items, nil
}
