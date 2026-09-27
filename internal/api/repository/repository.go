package repository

import (
	"context"
	"errors"
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
)

// ErrNotFound is returned when a requested row does not exist. Storage drivers
// must map their own "no rows" errors to it.
var ErrNotFound = errors.New("not found")

type ColumnRepository interface {
	List(ctx context.Context) ([]*column.Column, error)
	Get(ctx context.Context, id uuid.UUID) (*column.Column, error)
	First(ctx context.Context) (*column.Column, error)
	Next(ctx context.Context, currentOrder int) (*column.Column, error)
	Create(ctx context.Context, c *column.Column) error
	Update(ctx context.Context, c *column.Column) error
	Delete(ctx context.Context, id uuid.UUID) error
	Reorder(ctx context.Context, ids []uuid.UUID) error
	CountTasks(ctx context.Context, columnID uuid.UUID) (int, error)
}

type TaskRepository interface {
	List(ctx context.Context) ([]*task.Task, error)
	ListArchived(ctx context.Context) ([]*task.Task, error)
	Get(ctx context.Context, id uuid.UUID) (*task.Task, error)
	Create(ctx context.Context, t *task.Task) error
	Update(ctx context.Context, t *task.Task) error
	AddReport(ctx context.Context, taskID, columnID uuid.UUID, report string) error
	ListReports(ctx context.Context, taskID uuid.UUID) ([]task.Report, error)
}

type UserRepository interface {
	GetLocal(ctx context.Context) (*user.User, error)
}

type JobRepository interface {
	Enqueue(ctx context.Context, j *job.Job) error
	Lease(ctx context.Context, n int, lease time.Duration) ([]*job.Job, error)
	RenewLease(ctx context.Context, id uuid.UUID, lease time.Duration) error
	Complete(ctx context.Context, id uuid.UUID, status job.Status, lastErr string) error
	CancelQueuedForTask(ctx context.Context, taskID uuid.UUID) error
}

type AuditRepository interface {
	Add(ctx context.Context, e *auditevent.Event) error
	ListByTask(ctx context.Context, taskID uuid.UUID) ([]*auditevent.Event, error)
}

type SettingsRepository interface {
	Get(ctx context.Context) (*settings.Settings, error)
	Update(ctx context.Context, s *settings.Settings) error
}

type IntegrationRepository interface {
	List(ctx context.Context) ([]*domint.Integration, error)
	Get(ctx context.Context, id uuid.UUID) (*domint.Integration, error)
	Create(ctx context.Context, i *domint.Integration) error
	Update(ctx context.Context, i *domint.Integration) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type MCPRepository interface {
	List(ctx context.Context) ([]*mcpserver.Server, error)
	Get(ctx context.Context, id uuid.UUID) (*mcpserver.Server, error)
	Create(ctx context.Context, s *mcpserver.Server) error
	Update(ctx context.Context, s *mcpserver.Server) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type AgentRunRepository interface {
	Create(ctx context.Context, r *agentrun.Run) error
	Update(ctx context.Context, r *agentrun.Run) error
	AddEvent(ctx context.Context, e *agentrun.Event) error
	ListByTask(ctx context.Context, taskID uuid.UUID) ([]*agentrun.Run, error)
	Get(ctx context.Context, id uuid.UUID) (*agentrun.Run, error)
	ListEvents(ctx context.Context, runID uuid.UUID) ([]agentrun.Event, error)
}

type Repository struct {
	Columns      ColumnRepository
	Tasks        TaskRepository
	Users        UserRepository
	Jobs         JobRepository
	Audit        AuditRepository
	Settings     SettingsRepository
	Integrations IntegrationRepository
	MCP          MCPRepository
	AgentRuns    AgentRunRepository
}
