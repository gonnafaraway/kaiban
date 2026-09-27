package kanban

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"kaiban/internal/api/domain/auditevent"
	"kaiban/internal/api/domain/column"
	"kaiban/internal/api/domain/job"
	"kaiban/internal/api/domain/task"
	"kaiban/internal/api/integration"
	"kaiban/internal/api/repository"
)

func (u *UseCase) ListTasks(ctx context.Context) ([]*task.Task, error) {
	return u.Repo.Tasks.List(ctx)
}

func (u *UseCase) ListArchived(ctx context.Context) ([]ArchivedTask, error) {
	items, err := u.Repo.Tasks.ListArchived(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]ArchivedTask, 0, len(items))
	for _, t := range items {
		reports, err := u.Repo.Tasks.ListReports(ctx, t.ID)
		if err != nil {
			return nil, err
		}
		out = append(out, ArchivedTask{Task: t, Reports: reports})
	}
	return out, nil
}

func (u *UseCase) GetTask(ctx context.Context, id uuid.UUID) (*task.Task, []repository.Report, error) {
	t, err := u.Repo.Tasks.Get(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	reports, err := u.Repo.Tasks.ListReports(ctx, id)
	if err != nil {
		return t, nil, err
	}
	return t, reports, nil
}

func (u *UseCase) CreateTask(ctx context.Context, title, desc string, vars map[string]string, artifacts task.Artifacts) (*task.Task, error) {
	col, err := u.Repo.Columns.First(ctx)
	if err != nil {
		return nil, err
	}
	me, err := u.Repo.Users.GetLocal(ctx)
	if err != nil {
		return nil, err
	}
	t, err := task.New(title, desc, vars, artifacts, col.ID, me.ID)
	if err != nil {
		return nil, err
	}
	t.GitBranch = "kaiban/task-" + t.ID.String()[:8]
	st, _ := u.Repo.Settings.Get(ctx)
	repoURL, gitTok, useGitHub := u.resolveGitRemote(ctx, st, artifacts)
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
		return nil, err
	}
	_ = u.audit(ctx, &t.ID, auditevent.ActorUser, me.Login, "task.created", map[string]any{
		"title": title, "artifacts": artifacts,
	})
	u.publish("task.created", t)
	return t, nil
}

func (u *UseCase) PatchTask(ctx context.Context, id uuid.UUID, title, desc *string, vars map[string]string, extra *string, artifacts *task.Artifacts, budget *task.BudgetOverride) (*task.Task, error) {
	t, err := u.Repo.Tasks.Get(ctx, id)
	if err != nil {
		return nil, err
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
		return nil, err
	}
	u.publish("task.updated", t)
	return t, nil
}

func (u *UseCase) Run(ctx context.Context, id uuid.UUID) (*task.Task, error) {
	t, err := u.Repo.Tasks.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	prevStatus := t.ExecutionStatus
	if err := t.MarkQueued(); err != nil {
		return nil, err
	}
	if err := u.Repo.Tasks.Update(ctx, t); err != nil {
		return nil, err
	}
	if err := u.Repo.Jobs.Enqueue(ctx, &job.Job{TaskID: t.ID, ColumnID: t.ColumnID}); err != nil {
		t.ExecutionStatus = prevStatus
		t.UpdatedAt = time.Now().UTC()
		_ = u.Repo.Tasks.Update(ctx, t)
		return nil, err
	}
	_ = u.audit(ctx, &t.ID, auditevent.ActorUser, u.localActor(ctx), "agent.started", nil)
	u.publish("task.updated", t)
	u.agentLog(t.ID, "status", "Задача в очереди, ожидает воркер", nil)
	return t, nil
}

func (u *UseCase) Approve(ctx context.Context, id uuid.UUID, comment string) (*task.Task, error) {
	t, err := u.Repo.Tasks.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	cur, err := u.Repo.Columns.Get(ctx, t.ColumnID)
	if err != nil {
		return nil, err
	}
	if err := column.ValidateOutputs(cur.OutputFields, t.StageOutput(t.ColumnID)); err != nil {
		return nil, fmt.Errorf("%w: %s", task.ErrContractInvalid, err.Error())
	}
	if cur.RequiresGitDiff {
		st, err := u.Repo.Settings.Get(ctx)
		if err != nil {
			return nil, err
		}
		repoURL, _, _ := u.resolveGitRemote(ctx, st, t.ContextData.Artifacts)
		_ = integration.RefreshTaskWorktree(ctx, u.GitWorkDir, repoURL, t.GitBranch, st.GitDefaultBranch)
		sum, err := integration.TaskDiffSummary(ctx, u.GitWorkDir, t.GitBranch, st.GitDefaultBranch)
		if err != nil || sum == nil || sum.Empty {
			msg := "no changes vs " + st.GitDefaultBranch
			if err != nil {
				msg = err.Error()
			}
			return nil, fmt.Errorf("%w: %s", task.ErrGitDiffRequired, msg)
		}
	}
	next, err := u.Repo.Columns.Next(ctx, cur.OrderIndex)
	if err != nil {
		return nil, err
	}
	isLast := next == nil
	var nextID uuid.UUID
	if next != nil {
		nextID = next.ID
	}
	if err := t.Approve(isLast, nextID); err != nil {
		return nil, err
	}
	if err := u.Repo.Tasks.Update(ctx, t); err != nil {
		return nil, err
	}
	_ = u.audit(ctx, &t.ID, auditevent.ActorUser, u.localActor(ctx), "user.approved", map[string]any{"comment": comment})
	u.publish("task.updated", t)
	return t, nil
}

func (u *UseCase) Retry(ctx context.Context, id uuid.UUID, comment string) (*task.Task, error) {
	t, err := u.Repo.Tasks.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := t.RetryCurrent(comment); err != nil {
		return nil, err
	}
	_ = u.Repo.Jobs.CancelQueuedForTask(ctx, t.ID)
	if err := u.Repo.Tasks.Update(ctx, t); err != nil {
		return nil, err
	}
	_ = u.audit(ctx, &t.ID, auditevent.ActorUser, u.localActor(ctx), "user.retried", map[string]any{"comment": comment})
	u.publish("task.updated", t)
	return t, nil
}

func (u *UseCase) ReturnTo(ctx context.Context, id, columnID uuid.UUID, comment string) (*task.Task, error) {
	t, err := u.Repo.Tasks.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	cur, err := u.Repo.Columns.Get(ctx, t.ColumnID)
	if err != nil {
		return nil, err
	}
	target, err := u.Repo.Columns.Get(ctx, columnID)
	if err != nil {
		return nil, err
	}
	if err := t.ReturnTo(cur, target, comment); err != nil {
		return nil, err
	}
	_ = u.Repo.Jobs.CancelQueuedForTask(ctx, t.ID)
	if err := u.Repo.Tasks.Update(ctx, t); err != nil {
		return nil, err
	}
	_ = u.audit(ctx, &t.ID, auditevent.ActorUser, u.localActor(ctx), "user.returned", map[string]any{"comment": comment, "column_id": columnID.String()})
	u.publish("task.updated", t)
	return t, nil
}

func (u *UseCase) Archive(ctx context.Context, id uuid.UUID) (*task.Task, error) {
	t, err := u.Repo.Tasks.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := t.Archive(); err != nil {
		return nil, err
	}
	_ = u.Repo.Jobs.CancelQueuedForTask(ctx, t.ID)
	if err := u.Repo.Tasks.Update(ctx, t); err != nil {
		return nil, err
	}
	_ = u.audit(ctx, &t.ID, auditevent.ActorUser, u.localActor(ctx), "task.archived", nil)
	u.publish("task.updated", t)
	return t, nil
}

func (u *UseCase) Unarchive(ctx context.Context, id uuid.UUID) (*task.Task, error) {
	t, err := u.Repo.Tasks.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := t.Unarchive(); err != nil {
		return nil, err
	}
	if err := u.Repo.Tasks.Update(ctx, t); err != nil {
		return nil, err
	}
	_ = u.audit(ctx, &t.ID, auditevent.ActorUser, u.localActor(ctx), "task.unarchived", nil)
	u.publish("task.updated", t)
	return t, nil
}

func (u *UseCase) Events(ctx context.Context, taskID uuid.UUID) ([]*auditevent.Event, error) {
	return u.Repo.Audit.ListByTask(ctx, taskID)
}
