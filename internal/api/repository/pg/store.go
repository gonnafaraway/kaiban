package pg

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"kaiban/internal/api/domain/agentrun"
	"kaiban/internal/api/domain/auditevent"
	"kaiban/internal/api/domain/column"
	domint "kaiban/internal/api/domain/integration"
	"kaiban/internal/api/domain/job"
	"kaiban/internal/api/domain/mcpserver"
	"kaiban/internal/api/domain/settings"
	"kaiban/internal/api/domain/task"
	"kaiban/internal/api/domain/user"
	"kaiban/internal/api/env"
	"kaiban/internal/api/repository"
	"kaiban/internal/api/storage/postgres"
	"kaiban/migrations"
)

type Store struct {
	db *postgres.Client
}

func New(db *postgres.Client) *Store { return &Store{db: db} }

func (s *Store) ApplySchema(ctx context.Context) error {
	if _, err := s.db.Pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			filename TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`); err != nil {
		return err
	}
	entries, err := fs.Glob(migrations.FS, "api/*.sql")
	if err != nil {
		return err
	}
	sort.Strings(entries)

	var ledgerCount int
	if err := s.db.Pool.QueryRow(ctx, `SELECT count(*) FROM schema_migrations`).Scan(&ledgerCount); err != nil {
		return err
	}
	if ledgerCount == 0 {
		var hasColumns bool
		if err := s.db.Pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM information_schema.tables
				WHERE table_schema='public' AND table_name='columns'
			)`).Scan(&hasColumns); err != nil {
			return err
		}
		if hasColumns {
			// Pre-ledger DB: stamp only baseline schema files, then apply later
			// migrations (they use IF NOT EXISTS). Avoids skipping 002+.
			for _, name := range entries {
				base := migrationBase(name)
				if !isBaselineMigration(base) {
					continue
				}
				if _, err := s.db.Pool.Exec(ctx, `INSERT INTO schema_migrations (filename) VALUES ($1) ON CONFLICT DO NOTHING`, base); err != nil {
					return err
				}
			}
		}
	}

	for _, name := range entries {
		base := migrationBase(name)
		var exists bool
		if err := s.db.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE filename=$1)`, base).Scan(&exists); err != nil {
			return err
		}
		if exists {
			continue
		}
		sql, err := migrations.FS.ReadFile(name)
		if err != nil {
			return err
		}
		tx, err := s.db.Pool.Begin(ctx)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, string(sql)); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("%s: %w", base, err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (filename) VALUES ($1)`, base); err != nil {
			_ = tx.Rollback(ctx)
			return err
		}
		if err := tx.Commit(ctx); err != nil {
			return err
		}
	}
	return nil
}

func migrationBase(name string) string {
	if i := strings.LastIndex(name, "/"); i >= 0 {
		return name[i+1:]
	}
	return name
}

func isBaselineMigration(base string) bool {
	return strings.HasPrefix(base, "000_") || strings.HasPrefix(base, "001_")
}

func (s *Store) SeedLLM(ctx context.Context, base, key, model string) error {
	if key == "" {
		return nil
	}
	_, err := s.db.Pool.Exec(ctx, `
		UPDATE app_settings
		SET llm_base_url = $1, llm_api_key = $2, llm_model = $3, updated_at = now()
		WHERE llm_api_key = '' OR llm_api_key IS NULL
	`, base, key, model)
	return err
}

func (s *Store) SeedIntegrations(ctx context.Context, e *env.Env) error {
	if e == nil {
		return nil
	}
	type seed struct {
		id, typ, name, base, email, token string
	}
	items := []seed{
		{"00000000-0000-4000-8000-000000000201", "jira", "Jira", e.JiraURL, e.JiraEmail, e.JiraToken},
		{"00000000-0000-4000-8000-000000000202", "confluence", "Confluence", e.ConfluenceURL, e.ConfluenceEmail, e.ConfluenceToken},
		{"00000000-0000-4000-8000-000000000203", "gitlab", "GitLab", e.GitLabURL, "", e.GitLabToken},
		{"00000000-0000-4000-8000-000000000204", "github", "GitHub", e.GitHubURL, "", e.GitHubToken},
	}
	for _, it := range items {
		if it.base == "" || it.token == "" {
			continue
		}
		cred, _ := json.Marshal(map[string]string{
			"email":     it.email,
			"api_token": it.token,
			"token":     it.token,
		})
		_, err := s.db.Pool.Exec(ctx, `
			INSERT INTO integrations (id, type, name, base_url, credentials, status, last_error, created_at, updated_at)
			SELECT $1::uuid, $2, $3, $4, $5::jsonb, 'enabled', '', now(), now()
			WHERE NOT EXISTS (SELECT 1 FROM integrations WHERE type = $2)
		`, it.id, it.typ, it.name, it.base, cred)
		if err != nil {
			return err
		}
	}
	return nil
}

// Columns

const columnCols = `id, name, name_i18n, system_prompt_default, system_prompt_template, user_custom_prompt,
	COALESCE(output_fields, '[]'::jsonb),
	max_tokens, max_cost_usd, max_wall_sec, max_tool_calls, max_llm_steps,
	COALESCE(requires_git_diff, false),
	order_index, created_at, updated_at`

func (s *Store) ListColumns(ctx context.Context) ([]*column.Column, error) {
	rows, err := s.db.Pool.Query(ctx, `SELECT `+columnCols+` FROM columns ORDER BY order_index`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*column.Column
	for rows.Next() {
		c, err := scanColumn(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) GetColumn(ctx context.Context, id uuid.UUID) (*column.Column, error) {
	row := s.db.Pool.QueryRow(ctx, `SELECT `+columnCols+` FROM columns WHERE id=$1`, id)
	return scanColumn(row)
}

func (s *Store) FirstColumn(ctx context.Context) (*column.Column, error) {
	row := s.db.Pool.QueryRow(ctx, `SELECT `+columnCols+` FROM columns ORDER BY order_index ASC LIMIT 1`)
	return scanColumn(row)
}

func (s *Store) NextColumn(ctx context.Context, currentOrder int) (*column.Column, error) {
	row := s.db.Pool.QueryRow(ctx, `SELECT `+columnCols+` FROM columns WHERE order_index > $1 ORDER BY order_index ASC LIMIT 1`, currentOrder)
	c, err := scanColumn(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return c, err
}

func (s *Store) CreateColumn(ctx context.Context, c *column.Column) error {
	i18n, _ := json.Marshal(c.NameI18n)
	fields, _ := json.Marshal(c.OutputFields)
	_, err := s.db.Pool.Exec(ctx, `INSERT INTO columns (id,name,name_i18n,system_prompt_default,system_prompt_template,user_custom_prompt,output_fields,max_tokens,max_cost_usd,max_wall_sec,max_tool_calls,max_llm_steps,requires_git_diff,order_index,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)`,
		c.ID, c.Name, i18n, c.SystemPromptDefault, c.SystemPromptTemplate, c.UserCustomPrompt, fields,
		c.Budget.MaxTokens, c.Budget.MaxCostUSD, c.Budget.MaxWallSec, c.Budget.MaxToolCalls, c.Budget.MaxLLMSteps,
		c.RequiresGitDiff, c.OrderIndex, c.CreatedAt, c.UpdatedAt)
	return err
}

func (s *Store) UpdateColumn(ctx context.Context, c *column.Column) error {
	i18n, _ := json.Marshal(c.NameI18n)
	fields, _ := json.Marshal(c.OutputFields)
	_, err := s.db.Pool.Exec(ctx, `UPDATE columns SET name=$2, name_i18n=$3, system_prompt_default=$4, system_prompt_template=$5, user_custom_prompt=$6, output_fields=$7, max_tokens=$8, max_cost_usd=$9, max_wall_sec=$10, max_tool_calls=$11, max_llm_steps=$12, requires_git_diff=$13, order_index=$14, updated_at=$15 WHERE id=$1`,
		c.ID, c.Name, i18n, c.SystemPromptDefault, c.SystemPromptTemplate, c.UserCustomPrompt, fields,
		c.Budget.MaxTokens, c.Budget.MaxCostUSD, c.Budget.MaxWallSec, c.Budget.MaxToolCalls, c.Budget.MaxLLMSteps,
		c.RequiresGitDiff, c.OrderIndex, c.UpdatedAt)
	return err
}

func (s *Store) DeleteColumn(ctx context.Context, id uuid.UUID) error {
	_, err := s.db.Pool.Exec(ctx, `DELETE FROM columns WHERE id=$1`, id)
	return err
}

func (s *Store) ReorderColumns(ctx context.Context, ids []uuid.UUID) error {
	tx, err := s.db.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	for i, id := range ids {
		if _, err := tx.Exec(ctx, `UPDATE columns SET order_index=$2, updated_at=now() WHERE id=$1`, id, i); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (s *Store) CountTasks(ctx context.Context, columnID uuid.UUID) (int, error) {
	var n int
	err := s.db.Pool.QueryRow(ctx, `SELECT count(*) FROM tasks WHERE column_id=$1`, columnID).Scan(&n)
	return n, err
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanColumn(row rowScanner) (*column.Column, error) {
	c := &column.Column{NameI18n: map[string]string{}, OutputFields: []column.OutputField{}}
	var i18n, fields []byte
	err := row.Scan(
		&c.ID, &c.Name, &i18n, &c.SystemPromptDefault, &c.SystemPromptTemplate, &c.UserCustomPrompt,
		&fields,
		&c.Budget.MaxTokens, &c.Budget.MaxCostUSD, &c.Budget.MaxWallSec, &c.Budget.MaxToolCalls, &c.Budget.MaxLLMSteps,
		&c.RequiresGitDiff,
		&c.OrderIndex, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(i18n, &c.NameI18n)
	_ = json.Unmarshal(fields, &c.OutputFields)
	if c.OutputFields == nil {
		c.OutputFields = []column.OutputField{}
	}
	return c, nil
}

// Tasks

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
	vars, _ := json.Marshal(t.Variables)
	ctxd, _ := json.Marshal(t.ContextData)
	var branch any
	if t.GitBranch != "" {
		branch = t.GitBranch
	}
	_, err := s.db.Pool.Exec(ctx, `INSERT INTO tasks (id,title,description,variables,column_id,execution_status,git_branch,git_pr_url,git_push_status,git_pr_status,current_report,context_data,created_by,created_at,updated_at,archived_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)`,
		t.ID, t.Title, t.Description, vars, t.ColumnID, string(t.ExecutionStatus), branch, t.GitPRURL, t.GitPushStatus, t.GitPRStatus, t.CurrentReport, ctxd, t.CreatedBy, t.CreatedAt, t.UpdatedAt, t.ArchivedAt)
	return err
}

func (s *Store) UpdateTask(ctx context.Context, t *task.Task) error {
	vars, _ := json.Marshal(t.Variables)
	ctxd, _ := json.Marshal(t.ContextData)
	var branch any
	if t.GitBranch != "" {
		branch = t.GitBranch
	}
	_, err := s.db.Pool.Exec(ctx, `UPDATE tasks SET title=$2,description=$3,variables=$4,column_id=$5,execution_status=$6,git_branch=$7,git_pr_url=$8,git_push_status=$9,git_pr_status=$10,current_report=$11,context_data=$12,updated_at=$13,archived_at=$14 WHERE id=$1`,
		t.ID, t.Title, t.Description, vars, t.ColumnID, string(t.ExecutionStatus), branch, t.GitPRURL, t.GitPushStatus, t.GitPRStatus, t.CurrentReport, ctxd, t.UpdatedAt, t.ArchivedAt)
	return err
}

func (s *Store) AddReport(ctx context.Context, taskID, columnID uuid.UUID, report string) error {
	_, err := s.db.Pool.Exec(ctx, `INSERT INTO task_column_reports (task_id,column_id,report_md) VALUES ($1,$2,$3)`, taskID, columnID, report)
	return err
}

func (s *Store) ListReports(ctx context.Context, taskID uuid.UUID) ([]repository.Report, error) {
	rows, err := s.db.Pool.Query(ctx, `SELECT column_id, report_md, created_at FROM task_column_reports WHERE task_id=$1 ORDER BY created_at`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []repository.Report
	for rows.Next() {
		var r repository.Report
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
		return nil, err
	}
	t.ExecutionStatus = task.ExecutionStatus(status)
	_ = json.Unmarshal(vars, &t.Variables)
	_ = json.Unmarshal(ctxd, &t.ContextData)
	if t.Variables == nil {
		t.Variables = map[string]string{}
	}
	return t, nil
}

func (s *Store) GetLocalUser(ctx context.Context) (*user.User, error) {
	u := &user.User{}
	err := s.db.Pool.QueryRow(ctx, `SELECT id, login, display_name, created_at FROM users WHERE login='local'`).
		Scan(&u.ID, &u.Login, &u.DisplayName, &u.CreatedAt)
	return u, err
}

func (s *Store) EnqueueJob(ctx context.Context, j *job.Job) error {
	j.ID = uuid.New()
	j.Status = job.StatusQueued
	j.CreatedAt = time.Now().UTC()
	j.UpdatedAt = j.CreatedAt
	_, err := s.db.Pool.Exec(ctx, `INSERT INTO jobs (id,task_id,column_id,status,attempt,created_at,updated_at) VALUES ($1,$2,$3,$4,0,$5,$6)`,
		j.ID, j.TaskID, j.ColumnID, string(j.Status), j.CreatedAt, j.UpdatedAt)
	return err
}

const maxJobAttempts = 8

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
			return nil, err
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

func (s *Store) AddAudit(ctx context.Context, e *auditevent.Event) error {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	if e.CreatedAt.IsZero() {
		e.CreatedAt = time.Now().UTC()
	}
	payload, _ := json.Marshal(e.Payload)
	_, err := s.db.Pool.Exec(ctx, `INSERT INTO audit_events (id,task_id,actor_type,actor_id,action,payload,created_at) VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		e.ID, e.TaskID, string(e.ActorType), e.ActorID, e.Action, payload, e.CreatedAt)
	return err
}

func (s *Store) ListAudit(ctx context.Context, taskID uuid.UUID) ([]*auditevent.Event, error) {
	rows, err := s.db.Pool.Query(ctx, `SELECT id,task_id,actor_type,actor_id,action,payload,created_at FROM audit_events WHERE task_id=$1 ORDER BY created_at`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*auditevent.Event
	for rows.Next() {
		e := &auditevent.Event{Payload: map[string]any{}}
		var at string
		var payload []byte
		if err := rows.Scan(&e.ID, &e.TaskID, &at, &e.ActorID, &e.Action, &payload, &e.CreatedAt); err != nil {
			return nil, err
		}
		e.ActorType = auditevent.ActorType(at)
		_ = json.Unmarshal(payload, &e.Payload)
		out = append(out, e)
	}
	return out, rows.Err()
}

func (s *Store) GetSettings(ctx context.Context) (*settings.Settings, error) {
	st := &settings.Settings{ContextPack: []settings.ContextPackItem{}}
	var pack []byte
	err := s.db.Pool.QueryRow(ctx, `
		SELECT id,llm_base_url,llm_api_key,llm_model,git_repo_url,git_default_branch,locale,
			COALESCE(max_tokens,200000), COALESCE(max_cost_usd,5), COALESCE(max_wall_sec,1800),
			COALESCE(max_tool_calls,80), COALESCE(max_llm_steps,40),
			COALESCE(price_input_per_1k,0.002), COALESCE(price_output_per_1k,0.008),
			COALESCE(context_pack, '[]'::jsonb),
			updated_at
		FROM app_settings LIMIT 1`).
		Scan(&st.ID, &st.LLMBaseURL, &st.LLMAPIKey, &st.LLMModel, &st.GitRepoURL, &st.GitDefaultBranch, &st.Locale,
			&st.MaxTokens, &st.MaxCostUSD, &st.MaxWallSec, &st.MaxToolCalls, &st.MaxLLMSteps,
			&st.PriceInputPer1K, &st.PriceOutputPer1K, &pack, &st.UpdatedAt)
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(pack, &st.ContextPack)
	if st.ContextPack == nil {
		st.ContextPack = []settings.ContextPackItem{}
	}
	return st, nil
}

func (s *Store) UpdateSettings(ctx context.Context, st *settings.Settings) error {
	pack, _ := json.Marshal(st.ContextPack)
	_, err := s.db.Pool.Exec(ctx, `
		UPDATE app_settings SET llm_base_url=$2,llm_api_key=$3,llm_model=$4,git_repo_url=$5,git_default_branch=$6,locale=$7,
			max_tokens=$8,max_cost_usd=$9,max_wall_sec=$10,max_tool_calls=$11,max_llm_steps=$12,
			price_input_per_1k=$13,price_output_per_1k=$14,context_pack=$15,updated_at=now()
		WHERE id=$1`,
		st.ID, st.LLMBaseURL, st.LLMAPIKey, st.LLMModel, st.GitRepoURL, st.GitDefaultBranch, st.Locale,
		st.MaxTokens, st.MaxCostUSD, st.MaxWallSec, st.MaxToolCalls, st.MaxLLMSteps,
		st.PriceInputPer1K, st.PriceOutputPer1K, pack)
	return err
}

func (s *Store) CreateAgentRun(ctx context.Context, r *agentrun.Run) error {
	_, err := s.db.Pool.Exec(ctx, `
		INSERT INTO agent_runs (id,task_id,column_id,status,model,prompt_hash,context_pack_hash,stop_reason,tokens_in,tokens_out,cost_usd,tool_calls,llm_steps,started_at,finished_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`,
		r.ID, r.TaskID, r.ColumnID, string(r.Status), r.Model, r.PromptHash, r.ContextPackHash, r.StopReason,
		r.TokensIn, r.TokensOut, r.CostUSD, r.ToolCalls, r.LLMSteps, r.StartedAt, r.FinishedAt)
	return err
}

func (s *Store) UpdateAgentRun(ctx context.Context, r *agentrun.Run) error {
	_, err := s.db.Pool.Exec(ctx, `
		UPDATE agent_runs SET status=$2, stop_reason=$3, tokens_in=$4, tokens_out=$5, cost_usd=$6,
			tool_calls=$7, llm_steps=$8, finished_at=$9 WHERE id=$1`,
		r.ID, string(r.Status), r.StopReason, r.TokensIn, r.TokensOut, r.CostUSD, r.ToolCalls, r.LLMSteps, r.FinishedAt)
	return err
}

func (s *Store) AddAgentRunEvent(ctx context.Context, e *agentrun.Event) error {
	payload, _ := json.Marshal(e.Payload)
	_, err := s.db.Pool.Exec(ctx, `
		INSERT INTO agent_run_events (id,run_id,seq,kind,message,payload,created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		e.ID, e.RunID, e.Seq, e.Kind, e.Message, payload, e.CreatedAt)
	return err
}

func (s *Store) ListAgentRunsByTask(ctx context.Context, taskID uuid.UUID) ([]*agentrun.Run, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT id,task_id,column_id,status,model,prompt_hash,COALESCE(context_pack_hash,''),stop_reason,tokens_in,tokens_out,cost_usd,tool_calls,llm_steps,started_at,finished_at
		FROM agent_runs WHERE task_id=$1 ORDER BY started_at DESC`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*agentrun.Run
	for rows.Next() {
		r, err := scanAgentRun(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) GetAgentRun(ctx context.Context, id uuid.UUID) (*agentrun.Run, error) {
	row := s.db.Pool.QueryRow(ctx, `
		SELECT id,task_id,column_id,status,model,prompt_hash,COALESCE(context_pack_hash,''),stop_reason,tokens_in,tokens_out,cost_usd,tool_calls,llm_steps,started_at,finished_at
		FROM agent_runs WHERE id=$1`, id)
	return scanAgentRun(row)
}

func (s *Store) ListAgentRunEvents(ctx context.Context, runID uuid.UUID) ([]agentrun.Event, error) {
	rows, err := s.db.Pool.Query(ctx, `
		SELECT id,run_id,seq,kind,message,payload,created_at FROM agent_run_events WHERE run_id=$1 ORDER BY seq`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []agentrun.Event
	for rows.Next() {
		var e agentrun.Event
		var payload []byte
		if err := rows.Scan(&e.ID, &e.RunID, &e.Seq, &e.Kind, &e.Message, &payload, &e.CreatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(payload, &e.Payload)
		if e.Payload == nil {
			e.Payload = map[string]any{}
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func scanAgentRun(row rowScanner) (*agentrun.Run, error) {
	r := &agentrun.Run{}
	var status string
	err := row.Scan(&r.ID, &r.TaskID, &r.ColumnID, &status, &r.Model, &r.PromptHash, &r.ContextPackHash, &r.StopReason,
		&r.TokensIn, &r.TokensOut, &r.CostUSD, &r.ToolCalls, &r.LLMSteps, &r.StartedAt, &r.FinishedAt)
	if err != nil {
		return nil, err
	}
	r.Status = agentrun.Status(status)
	return r, nil
}

func (s *Store) ListIntegrations(ctx context.Context) ([]*domint.Integration, error) {
	rows, err := s.db.Pool.Query(ctx, `SELECT id,type,name,base_url,credentials,status,COALESCE(last_error,''),created_at,updated_at FROM integrations ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*domint.Integration
	for rows.Next() {
		i, err := scanInt(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

func (s *Store) GetIntegration(ctx context.Context, id uuid.UUID) (*domint.Integration, error) {
	row := s.db.Pool.QueryRow(ctx, `SELECT id,type,name,base_url,credentials,status,COALESCE(last_error,''),created_at,updated_at FROM integrations WHERE id=$1`, id)
	return scanInt(row)
}

func (s *Store) CreateIntegration(ctx context.Context, i *domint.Integration) error {
	cred, _ := json.Marshal(i.Credentials)
	_, err := s.db.Pool.Exec(ctx, `INSERT INTO integrations (id,type,name,base_url,credentials,status,last_error,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		i.ID, string(i.Type), i.Name, i.BaseURL, cred, string(i.Status), i.LastError, i.CreatedAt, i.UpdatedAt)
	return err
}

func (s *Store) UpdateIntegration(ctx context.Context, i *domint.Integration) error {
	cred, _ := json.Marshal(i.Credentials)
	_, err := s.db.Pool.Exec(ctx, `UPDATE integrations SET type=$2,name=$3,base_url=$4,credentials=$5,status=$6,last_error=$7,updated_at=$8 WHERE id=$1`,
		i.ID, string(i.Type), i.Name, i.BaseURL, cred, string(i.Status), i.LastError, i.UpdatedAt)
	return err
}

func (s *Store) DeleteIntegration(ctx context.Context, id uuid.UUID) error {
	_, err := s.db.Pool.Exec(ctx, `DELETE FROM integrations WHERE id=$1`, id)
	return err
}

func scanInt(row rowScanner) (*domint.Integration, error) {
	i := &domint.Integration{Credentials: map[string]string{}}
	var typ, st string
	var cred []byte
	err := row.Scan(&i.ID, &typ, &i.Name, &i.BaseURL, &cred, &st, &i.LastError, &i.CreatedAt, &i.UpdatedAt)
	if err != nil {
		return nil, err
	}
	i.Type = domint.Type(typ)
	i.Status = domint.Status(st)
	_ = json.Unmarshal(cred, &i.Credentials)
	return i, nil
}

func (s *Store) ListMCP(ctx context.Context) ([]*mcpserver.Server, error) {
	rows, err := s.db.Pool.Query(ctx, `SELECT id,name,endpoint,headers,capabilities,status,COALESCE(last_error,''),created_at,updated_at FROM mcp_servers ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*mcpserver.Server
	for rows.Next() {
		m, err := scanMCP(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *Store) GetMCP(ctx context.Context, id uuid.UUID) (*mcpserver.Server, error) {
	row := s.db.Pool.QueryRow(ctx, `SELECT id,name,endpoint,headers,capabilities,status,COALESCE(last_error,''),created_at,updated_at FROM mcp_servers WHERE id=$1`, id)
	return scanMCP(row)
}

func (s *Store) CreateMCP(ctx context.Context, m *mcpserver.Server) error {
	h, _ := json.Marshal(m.Headers)
	c, _ := json.Marshal(m.Capabilities)
	_, err := s.db.Pool.Exec(ctx, `INSERT INTO mcp_servers (id,name,endpoint,headers,capabilities,status,last_error,created_at,updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		m.ID, m.Name, m.Endpoint, h, c, string(m.Status), m.LastError, m.CreatedAt, m.UpdatedAt)
	return err
}

func (s *Store) UpdateMCP(ctx context.Context, m *mcpserver.Server) error {
	h, _ := json.Marshal(m.Headers)
	c, _ := json.Marshal(m.Capabilities)
	_, err := s.db.Pool.Exec(ctx, `UPDATE mcp_servers SET name=$2,endpoint=$3,headers=$4,capabilities=$5,status=$6,last_error=$7,updated_at=$8 WHERE id=$1`,
		m.ID, m.Name, m.Endpoint, h, c, string(m.Status), m.LastError, m.UpdatedAt)
	return err
}

func (s *Store) DeleteMCP(ctx context.Context, id uuid.UUID) error {
	_, err := s.db.Pool.Exec(ctx, `DELETE FROM mcp_servers WHERE id=$1`, id)
	return err
}

func scanMCP(row rowScanner) (*mcpserver.Server, error) {
	m := &mcpserver.Server{Headers: map[string]string{}, Capabilities: map[string]any{}}
	var h, c []byte
	var st string
	err := row.Scan(&m.ID, &m.Name, &m.Endpoint, &h, &c, &st, &m.LastError, &m.CreatedAt, &m.UpdatedAt)
	if err != nil {
		return nil, err
	}
	m.Status = mcpserver.Status(st)
	_ = json.Unmarshal(h, &m.Headers)
	_ = json.Unmarshal(c, &m.Capabilities)
	return m, nil
}
