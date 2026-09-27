package pg

import (
	"context"
	"time"

	"github.com/google/uuid"

	"kaiban/internal/api/domain/job"
)

const maxJobAttempts = 8

func (s *Store) EnqueueJob(ctx context.Context, j *job.Job) error {
	j.ID = uuid.New()
	j.Status = job.StatusQueued
	j.CreatedAt = time.Now().UTC()
	j.UpdatedAt = j.CreatedAt
	_, err := s.db.Pool.Exec(ctx, `INSERT INTO jobs (id,task_id,column_id,status,attempt,created_at,updated_at) VALUES ($1,$2,$3,$4,0,$5,$6)`,
		j.ID, j.TaskID, j.ColumnID, string(j.Status), j.CreatedAt, j.UpdatedAt)
	return err
}

func (s *Store) LeaseJobs(ctx context.Context, n int, lease time.Duration) ([]*job.Job, error) {
	// Permanently fail jobs that exceeded attempt budget; reset stuck tasks.
	if _, err := s.db.Pool.Exec(ctx, `
		WITH exhausted AS (
			UPDATE jobs SET status='failed', last_error='max attempts exceeded', updated_at=now()
			WHERE (status='queued' OR (status='running' AND leased_until < now()))
			  AND attempt >= $1
			RETURNING task_id
		)
		UPDATE tasks SET execution_status='failed', updated_at=now()
		WHERE id IN (SELECT task_id FROM exhausted)
		  AND execution_status IN ('queued','running')`, maxJobAttempts); err != nil {
		return nil, err
	}

	tx, err := s.db.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	rows, err := tx.Query(ctx, `
		SELECT id, task_id, status FROM jobs
		WHERE (status='queued' OR (status='running' AND leased_until < now()))
		  AND attempt < $1
		ORDER BY created_at
		FOR UPDATE SKIP LOCKED
		LIMIT $2`, maxJobAttempts, n)
	if err != nil {
		return nil, err
	}
	type cand struct {
		id, taskID uuid.UUID
		oldStatus  string
	}
	var cands []cand
	for rows.Next() {
		var c cand
		if err := rows.Scan(&c.id, &c.taskID, &c.oldStatus); err != nil {
			rows.Close()
			return nil, err
		}
		cands = append(cands, c)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var out []*job.Job
	for _, c := range cands {
		if c.oldStatus == "running" {
			// Dead worker reclaim: put task back to queued so MarkRunning succeeds.
			if _, err := tx.Exec(ctx, `
				UPDATE tasks SET execution_status='queued', updated_at=now()
				WHERE id=$1 AND execution_status='running'`, c.taskID); err != nil {
				return nil, err
			}
		}
		j := &job.Job{}
		var st string
		err := tx.QueryRow(ctx, `
			UPDATE jobs SET status='running', attempt=attempt+1,
				leased_until=now() + make_interval(secs => $2), updated_at=now()
			WHERE id=$1
			RETURNING id, task_id, column_id, status, attempt, COALESCE(last_error,''), leased_until, created_at, updated_at`,
			c.id, int(lease.Seconds()),
		).Scan(&j.ID, &j.TaskID, &j.ColumnID, &st, &j.Attempt, &j.LastError, &j.LeasedUntil, &j.CreatedAt, &j.UpdatedAt)
		if err != nil {
			return nil, mapNoRows(err, "lease job")
		}
		j.Status = job.Status(st)
		out = append(out, j)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) RenewJobLease(ctx context.Context, id uuid.UUID, lease time.Duration) error {
	_, err := s.db.Pool.Exec(ctx, `
		UPDATE jobs SET leased_until=now() + make_interval(secs => $2), updated_at=now()
		WHERE id=$1 AND status='running'`, id, int(lease.Seconds()))
	return err
}

func (s *Store) CompleteJob(ctx context.Context, id uuid.UUID, status job.Status, lastErr string) error {
	_, err := s.db.Pool.Exec(ctx, `UPDATE jobs SET status=$2, last_error=$3, updated_at=now() WHERE id=$1`, id, string(status), lastErr)
	return err
}

func (s *Store) CancelQueuedForTask(ctx context.Context, taskID uuid.UUID) error {
	_, err := s.db.Pool.Exec(ctx, `UPDATE jobs SET status='canceled', updated_at=now() WHERE task_id=$1 AND status='queued'`, taskID)
	return err
}
