package agent

import (
	"context"

	"github.com/pkg/errors"

	"kaiban/internal/api/domain/job"
)

// ExecuteAgentJob runs one column agent for a task: prepare context, drive the
// LLM/tool loop, sync artifacts, then record the outcome.
func (u *Runner) ExecuteAgentJob(ctx context.Context, j *job.Job) error {
	rc, err := u.prepareAgentRun(ctx, j)
	if err != nil {
		return errors.Wrap(err, "prepare agent run")
	}
	loop, err := u.runAgentLoop(ctx, rc)
	if err != nil {
		return errors.Wrap(err, "run agent loop")
	}
	notes := u.syncArtifacts(ctx, rc, loop.report)
	report := u.appendReportSections(ctx, rc, loop.report, notes)
	if err := u.finalizeAgent(ctx, rc, report, loop.stopReason); err != nil {
		return errors.Wrap(err, "finalize agent")
	}
	return nil
}
