package agent

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"kaiban/internal/api/domain/agentrun"
	"kaiban/internal/api/domain/auditevent"
	"kaiban/internal/api/domain/column"
	"kaiban/internal/api/domain/task"
	httptransport "kaiban/internal/api/transport/http"
)

// agentOutcome describes how a finished run is recorded.
type agentOutcome struct {
	runStatus    agentrun.Status
	stopReason   string
	auditAction  auditevent.Action
	auditPayload map[string]any
}

// finalizeAgent validates the stage contract, records the run outcome and
// persists task state. It returns a typed error when the run did not succeed.
func (u *Runner) finalizeAgent(ctx context.Context, rc *agentRunContext, report, stopReason string) error {
	u.reloadTask(ctx, rc)
	t, col, ar := rc.task, rc.column, rc.run

	if contractErr := column.ValidateOutputs(col.OutputFields, t.StageOutput(col.ID)); contractErr != nil {
		report += "\n\n## Stage contract\n- FAIL: " + contractErr.Error()
		u.runLog(ctx, ar, t.ID, "contract", "FAIL: "+contractErr.Error(), nil)
		reason := "contract: " + contractErr.Error()
		if stopReason != "" {
			reason = stopReason + "; " + reason
		}
		if err := t.MarkFailed(report); err != nil {
			return errors.Wrap(err, "mark failed")
		}
		if err := u.persistAgentOutcome(ctx, rc, report, agentOutcome{
			runStatus:   agentrun.StatusFailed,
			stopReason:  reason,
			auditAction: auditevent.ActionAgentFailed,
			auditPayload: map[string]any{
				"error": reason, "run_id": ar.run.ID.String(),
			},
		}); err != nil {
			return err
		}
		return fmt.Errorf("%w: %s", task.ErrContractInvalid, reason)
	}

	if stopReason != "" {
		report += "\n\n## Budget\n- Stopped: " + stopReason
		if err := t.MarkFailed(report); err != nil {
			return errors.Wrap(err, "mark failed")
		}
		if err := u.persistAgentOutcome(ctx, rc, report, agentOutcome{
			runStatus:   agentrun.StatusStopped,
			stopReason:  stopReason,
			auditAction: auditevent.ActionAgentFailed,
			auditPayload: map[string]any{
				"error": stopReason, "run_id": ar.run.ID.String(), "budget_stop": true,
			},
		}); err != nil {
			return err
		}
		u.runLog(ctx, ar, t.ID, "status", "Прогон остановлен по бюджету", map[string]any{
			"run_id": ar.run.ID.String(), "cost_usd": ar.run.CostUSD, "tokens": ar.run.TokensIn + ar.run.TokensOut,
			"reason": stopReason,
		})
		return fmt.Errorf("%w: %s", task.ErrBudgetExceeded, stopReason)
	}

	if err := t.MarkSucceeded(report); err != nil {
		return errors.Wrap(err, "mark succeeded")
	}
	if err := u.persistAgentOutcome(ctx, rc, report, agentOutcome{
		runStatus:   agentrun.StatusSucceeded,
		stopReason:  "completed",
		auditAction: auditevent.ActionAgentComplete,
		auditPayload: map[string]any{
			"report":    report,
			"column_id": t.ColumnID.String(),
			"column":    col.Name,
			"run_id":    ar.run.ID.String(),
		},
	}); err != nil {
		return err
	}
	u.runLog(ctx, ar, t.ID, "status", "Прогон завершён", map[string]any{
		"run_id": ar.run.ID.String(), "cost_usd": ar.run.CostUSD, "tokens": ar.run.TokensIn + ar.run.TokensOut,
	})
	return nil
}

// reloadTask re-reads the task so stage outputs submitted by tools survive,
// while keeping the git sync fields produced by the sync stage.
func (u *Runner) reloadTask(ctx context.Context, rc *agentRunContext) {
	prURL, pushSt, prSt := rc.task.GitPRURL, rc.task.GitPushStatus, rc.task.GitPRStatus
	fresh, err := u.Repo.Tasks.Get(ctx, rc.task.ID)
	if err != nil {
		u.runLog(ctx, rc.run, rc.task.ID, "error", "Не удалось перечитать задачу: "+err.Error(), nil)
		return
	}
	fresh.GitPRURL, fresh.GitPushStatus, fresh.GitPRStatus = prURL, pushSt, prSt
	rc.task = fresh
}

// persistAgentOutcome closes the run, stores the report and publishes the result.
// Run/report/task writes are critical and abort on failure; audit is best effort.
func (u *Runner) persistAgentOutcome(ctx context.Context, rc *agentRunContext, report string, out agentOutcome) error {
	t, ar := rc.task, rc.run
	if err := u.finishRun(ctx, ar, out.runStatus, out.stopReason); err != nil {
		return errors.Wrap(err, "finish run")
	}
	if err := u.Repo.Tasks.AddReport(ctx, t.ID, t.ColumnID, report); err != nil {
		return errors.Wrap(err, "add report")
	}
	if err := u.Repo.Tasks.Update(ctx, t); err != nil {
		return errors.Wrap(err, "update task")
	}
	if err := u.Audit(ctx, &t.ID, auditevent.ActorAgent, "agent", out.auditAction, out.auditPayload); err != nil {
		u.runLog(ctx, ar, t.ID, "error", "Audit "+string(out.auditAction)+": "+err.Error(), nil)
	}
	u.Publish(httptransport.EventTaskUpdated, t)
	u.Publish(httptransport.EventAgentFinished, map[string]any{"task_id": t.ID, "run_id": ar.run.ID.String()})
	return nil
}

// failAgentRun records a hard failure (LLM error) without storing a stage report.
func (u *Runner) failAgentRun(ctx context.Context, rc *agentRunContext, cause error) error {
	t, ar := rc.task, rc.run
	if err := u.finishRun(ctx, ar, agentrun.StatusFailed, cause.Error()); err != nil {
		return errors.Wrap(err, "finish run")
	}
	if err := t.MarkFailed(cause.Error()); err != nil {
		return errors.Wrap(err, "mark failed")
	}
	if err := u.Repo.Tasks.Update(ctx, t); err != nil {
		return errors.Wrap(err, "update task")
	}
	if err := u.Audit(ctx, &t.ID, auditevent.ActorAgent, "agent", auditevent.ActionAgentFailed, map[string]any{
		"error": cause.Error(), "run_id": ar.run.ID.String(),
	}); err != nil {
		u.runLog(ctx, ar, t.ID, "error", "Audit agent.failed: "+err.Error(), nil)
	}
	u.Publish(httptransport.EventTaskUpdated, t)
	u.runLog(ctx, ar, t.ID, "error", "Ошибка LLM: "+cause.Error(), nil)
	return nil
}
