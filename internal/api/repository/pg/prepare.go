package pg

import (
	"context"
	"time"

	"github.com/google/uuid"

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
	"kaiban/internal/api/storage"
)

func PrepareRepository(st *storage.Storage) *repository.Repository {
	s := New(st.Postgres)
	return &repository.Repository{
		Columns:      columnRepo{s},
		Tasks:        taskRepo{s},
		Users:        userRepo{s},
		Jobs:         jobRepo{s},
		Audit:        auditRepo{s},
		Settings:     settingsRepo{s},
		Integrations: intRepo{s},
		MCP:          mcpRepo{s},
		AgentRuns:    agentRunRepo{s},
	}
}

type columnRepo struct{ s *Store }

func (r columnRepo) List(ctx context.Context) ([]*column.Column, error) {
	return r.s.ListColumns(ctx)
}
func (r columnRepo) Get(ctx context.Context, id uuid.UUID) (*column.Column, error) {
	return r.s.GetColumn(ctx, id)
}
func (r columnRepo) First(ctx context.Context) (*column.Column, error) { return r.s.FirstColumn(ctx) }
func (r columnRepo) Next(ctx context.Context, currentOrder int) (*column.Column, error) {
	return r.s.NextColumn(ctx, currentOrder)
}
func (r columnRepo) Create(ctx context.Context, c *column.Column) error {
	return r.s.CreateColumn(ctx, c)
}
func (r columnRepo) Update(ctx context.Context, c *column.Column) error {
	return r.s.UpdateColumn(ctx, c)
}
func (r columnRepo) Delete(ctx context.Context, id uuid.UUID) error { return r.s.DeleteColumn(ctx, id) }
func (r columnRepo) Reorder(ctx context.Context, ids []uuid.UUID) error {
	return r.s.ReorderColumns(ctx, ids)
}
func (r columnRepo) CountTasks(ctx context.Context, columnID uuid.UUID) (int, error) {
	return r.s.CountTasks(ctx, columnID)
}

type taskRepo struct{ s *Store }

func (r taskRepo) List(ctx context.Context) ([]*task.Task, error) { return r.s.ListTasks(ctx) }
func (r taskRepo) ListArchived(ctx context.Context) ([]*task.Task, error) {
	return r.s.ListArchivedTasks(ctx)
}
func (r taskRepo) Get(ctx context.Context, id uuid.UUID) (*task.Task, error) {
	return r.s.GetTask(ctx, id)
}
func (r taskRepo) Create(ctx context.Context, t *task.Task) error { return r.s.CreateTask(ctx, t) }
func (r taskRepo) Update(ctx context.Context, t *task.Task) error { return r.s.UpdateTask(ctx, t) }
func (r taskRepo) AddReport(ctx context.Context, taskID, columnID uuid.UUID, report string) error {
	return r.s.AddReport(ctx, taskID, columnID, report)
}
func (r taskRepo) ListReports(ctx context.Context, taskID uuid.UUID) ([]task.Report, error) {
	return r.s.ListReports(ctx, taskID)
}

type userRepo struct{ s *Store }

func (r userRepo) GetLocal(ctx context.Context) (*user.User, error) { return r.s.GetLocalUser(ctx) }

type jobRepo struct{ s *Store }

func (r jobRepo) Enqueue(ctx context.Context, j *job.Job) error { return r.s.EnqueueJob(ctx, j) }
func (r jobRepo) Lease(ctx context.Context, n int, lease time.Duration) ([]*job.Job, error) {
	return r.s.LeaseJobs(ctx, n, lease)
}
func (r jobRepo) RenewLease(ctx context.Context, id uuid.UUID, lease time.Duration) error {
	return r.s.RenewJobLease(ctx, id, lease)
}
func (r jobRepo) Complete(ctx context.Context, id uuid.UUID, status job.Status, lastErr string) error {
	return r.s.CompleteJob(ctx, id, status, lastErr)
}
func (r jobRepo) CancelQueuedForTask(ctx context.Context, taskID uuid.UUID) error {
	return r.s.CancelQueuedForTask(ctx, taskID)
}

type auditRepo struct{ s *Store }

func (r auditRepo) Add(ctx context.Context, e *auditevent.Event) error { return r.s.AddAudit(ctx, e) }
func (r auditRepo) ListByTask(ctx context.Context, taskID uuid.UUID) ([]*auditevent.Event, error) {
	return r.s.ListAudit(ctx, taskID)
}

type settingsRepo struct{ s *Store }

func (r settingsRepo) Get(ctx context.Context) (*settings.Settings, error) {
	return r.s.GetSettings(ctx)
}
func (r settingsRepo) Update(ctx context.Context, st *settings.Settings) error {
	return r.s.UpdateSettings(ctx, st)
}

type intRepo struct{ s *Store }

func (r intRepo) List(ctx context.Context) ([]*domint.Integration, error) {
	return r.s.ListIntegrations(ctx)
}
func (r intRepo) Get(ctx context.Context, id uuid.UUID) (*domint.Integration, error) {
	return r.s.GetIntegration(ctx, id)
}
func (r intRepo) Create(ctx context.Context, i *domint.Integration) error {
	return r.s.CreateIntegration(ctx, i)
}
func (r intRepo) Update(ctx context.Context, i *domint.Integration) error {
	return r.s.UpdateIntegration(ctx, i)
}
func (r intRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return r.s.DeleteIntegration(ctx, id)
}

type mcpRepo struct{ s *Store }

func (r mcpRepo) List(ctx context.Context) ([]*mcpserver.Server, error) { return r.s.ListMCP(ctx) }
func (r mcpRepo) Get(ctx context.Context, id uuid.UUID) (*mcpserver.Server, error) {
	return r.s.GetMCP(ctx, id)
}
func (r mcpRepo) Create(ctx context.Context, m *mcpserver.Server) error { return r.s.CreateMCP(ctx, m) }
func (r mcpRepo) Update(ctx context.Context, m *mcpserver.Server) error { return r.s.UpdateMCP(ctx, m) }
func (r mcpRepo) Delete(ctx context.Context, id uuid.UUID) error        { return r.s.DeleteMCP(ctx, id) }

type agentRunRepo struct{ s *Store }

func (r agentRunRepo) Create(ctx context.Context, run *agentrun.Run) error {
	return r.s.CreateAgentRun(ctx, run)
}
func (r agentRunRepo) Update(ctx context.Context, run *agentrun.Run) error {
	return r.s.UpdateAgentRun(ctx, run)
}
func (r agentRunRepo) AddEvent(ctx context.Context, e *agentrun.Event) error {
	return r.s.AddAgentRunEvent(ctx, e)
}
func (r agentRunRepo) ListByTask(ctx context.Context, taskID uuid.UUID) ([]*agentrun.Run, error) {
	return r.s.ListAgentRunsByTask(ctx, taskID)
}
func (r agentRunRepo) Get(ctx context.Context, id uuid.UUID) (*agentrun.Run, error) {
	return r.s.GetAgentRun(ctx, id)
}
func (r agentRunRepo) ListEvents(ctx context.Context, runID uuid.UUID) ([]agentrun.Event, error) {
	return r.s.ListAgentRunEvents(ctx, runID)
}

func ApplySchema(ctx context.Context, st *storage.Storage, e *env.Env) error {
	s := New(st.Postgres)
	if err := s.ApplySchema(ctx); err != nil {
		return err
	}
	if err := s.SeedLLM(ctx, e.LLMBaseURL, e.LLMAPIKey, e.LLMModel); err != nil {
		return err
	}
	return s.SeedIntegrations(ctx, e)
}
