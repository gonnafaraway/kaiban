package mcpserver

import (
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusDisabled Status = "disabled"
	StatusEnabled  Status = "enabled"
	StatusError    Status = "error"
)

type Server struct {
	ID           uuid.UUID
	Name         string
	Endpoint     string
	Headers      map[string]string
	Capabilities map[string]any
	Status       Status
	LastError    string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
