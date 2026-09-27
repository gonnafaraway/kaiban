package job

import (
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusQueued    Status = "queued"
	StatusRunning   Status = "running"
	StatusSucceeded Status = "succeeded"
	StatusFailed    Status = "failed"
	StatusCanceled  Status = "canceled"
)

type Job struct {
	ID          uuid.UUID
	TaskID      uuid.UUID
	ColumnID    uuid.UUID
	Status      Status
	Attempt     int
	LastError   string
	LeasedUntil *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
