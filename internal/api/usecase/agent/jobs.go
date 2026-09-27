package agent

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	"kaiban/internal/api/domain/job"
)

// LeaseJobs claims up to n queued jobs for the caller for the lease window.
func (u *Runner) LeaseJobs(ctx context.Context, n int, lease time.Duration) ([]*job.Job, error) {
	jobs, err := u.Repo.Jobs.Lease(ctx, n, lease)
	if err != nil {
		return nil, errors.Wrap(err, "lease jobs")
	}
	return jobs, nil
}

// RenewJobLease extends the lease of a job still being worked on.
func (u *Runner) RenewJobLease(ctx context.Context, id uuid.UUID, lease time.Duration) error {
	return errors.Wrap(u.Repo.Jobs.RenewLease(ctx, id, lease), "renew job lease")
}

// CompleteJob records the terminal status of a job.
func (u *Runner) CompleteJob(ctx context.Context, id uuid.UUID, status job.Status, lastErr string) error {
	return errors.Wrap(u.Repo.Jobs.Complete(ctx, id, status, lastErr), "complete job")
}
