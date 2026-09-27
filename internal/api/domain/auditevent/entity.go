package auditevent

import (
	"time"

	"github.com/google/uuid"
)

type ActorType string

const (
	ActorUser   ActorType = "user"
	ActorAgent  ActorType = "agent"
	ActorSystem ActorType = "system"
)

type Event struct {
	ID        uuid.UUID      `json:"id"`
	TaskID    *uuid.UUID     `json:"task_id"`
	ActorType ActorType      `json:"actor_type"`
	ActorID   string         `json:"actor_id"`
	Action    string         `json:"action"`
	Payload   map[string]any `json:"payload"`
	CreatedAt time.Time      `json:"created_at"`
}
