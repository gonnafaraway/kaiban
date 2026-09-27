package pg

import (
	"context"

	"github.com/google/uuid"

	"kaiban/internal/api/domain/task"
)

const taskCols = `id,title,description,variables,column_id,execution_status,COALESCE(git_branch,''),COALESCE(git_pr_url,''),COALESCE(git_push_status,''),COALESCE(git_pr_status,''),current_report,context_data,created_by,created_at,updated_at,archived_at`

func (s *Store) ListTasks(ctx context.Context) ([]*task.Task, error) {
	rows, err := s.db.Pool.Query(ctx, `SELECT `+taskCols+` FROM tasks WHERE archived_at IS NULL ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]*task.Task, 0)
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *Store) ListArchivedTasks(ctx context.Context) ([]*task.Task, error) {
	rows, err := s.db.Pool.Query(ctx, `SELECT `+taskCols+` FROM tasks WHERE archived_at IS NOT NULL ORDER BY archived_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*task.Task
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *Store) GetTask(ctx context.Context, id uuid.UUID) (*task.Task, error) {
	row := s.db.Pool.QueryRow(ctx, `SELECT `+taskCols+` FROM tasks WHERE id=$1`, id)
	return scanTask(row)
}

func (s *Store) CreateTask(ctx context.Context, t *task.Task) error {
	vars, err := marshalJSON(t.Variables, "task variables")
	if err != nil {
		return err
	}
	ctxd, err := marshalJSON(t.ContextData, "task context_data")
	if err != nil {
		return err
	}
	var branch any
	if t.GitBranch != "" {
		branch = t.GitBranch
	}
	_, err = s.db.Pool.Exec(ctx, `INSERT INTO tasks (id,title,description,variables,column_id,execution_status,git_branch,git_pr_url,git_push_status,git_pr_status,current_report,context_data,created_by,created_at,updated_at,archived_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)`,
		t.ID, t.Title, t.Description, vars, t.ColumnID, string(t.ExecutionStatus), branch, t.GitPRURL, t.GitPushStatus, t.GitPRStatus, t.CurrentReport, ctxd, t.CreatedBy, t.CreatedAt, t.UpdatedAt, t.ArchivedAt)
	return err
}

func (s *Store) UpdateTask(ctx context.Context, t *task.Task) error {
	vars, err := marshalJSON(t.Variables, "task variables")
	if err != nil {
		return err
	}
	ctxd, err := marshalJSON(t.ContextData, "task context_data")
	if err != nil {
		return err
	}
	var branch any
	if t.GitBranch != "" {
		branch = t.GitBranch
	}
	_, err = s.db.Pool.Exec(ctx, `UPDATE tasks SET title=$2,description=$3,variables=$4,column_id=$5,execution_status=$6,git_branch=$7,git_pr_url=$8,git_push_status=$9,git_pr_status=$10,current_report=$11,context_data=$12,updated_at=$13,archived_at=$14 WHERE id=$1`,
		t.ID, t.Title, t.Description, vars, t.ColumnID, string(t.ExecutionStatus), branch, t.GitPRURL, t.GitPushStatus, t.GitPRStatus, t.CurrentReport, ctxd, t.UpdatedAt, t.ArchivedAt)
	return err
}

func (s *Store) AddReport(ctx context.Context, taskID, columnID uuid.UUID, report string) error {
	_, err := s.db.Pool.Exec(ctx, `INSERT INTO task_column_reports (task_id,column_id,report_md) VALUES ($1,$2,$3)`, taskID, columnID, report)
	return err
}

func (s *Store) ListReports(ctx context.Context, taskID uuid.UUID) ([]task.Report, error) {
	rows, err := s.db.Pool.Query(ctx, `SELECT column_id, report_md, created_at FROM task_column_reports WHERE task_id=$1 ORDER BY created_at`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []task.Report
	for rows.Next() {
		var r task.Report
		if err := rows.Scan(&r.ColumnID, &r.ReportMD, &r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func scanTask(row rowScanner) (*task.Task, error) {
	t := &task.Task{Variables: map[string]string{}}
	var vars, ctxd []byte
	var status string
	err := row.Scan(&t.ID, &t.Title, &t.Description, &vars, &t.ColumnID, &status, &t.GitBranch, &t.GitPRURL, &t.GitPushStatus, &t.GitPRStatus, &t.CurrentReport, &ctxd, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt, &t.ArchivedAt)
	if err != nil {
		return nil, mapNoRows(err, "scan task")
	}
	t.ExecutionStatus = task.ExecutionStatus(status)
	if err := unmarshalJSON(vars, &t.Variables, "task variables"); err != nil {
		return nil, err
	}
	if err := unmarshalJSON(ctxd, &t.ContextData, "task context_data"); err != nil {
		return nil, err
	}
	if t.Variables == nil {
		t.Variables = map[string]string{}
	}
	return t, nil
}
