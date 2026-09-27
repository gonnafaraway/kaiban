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

// Action is a canonical audit action name.
type Action string

const (
	ActionTaskCreated    Action = "task.created"
	ActionTaskArchived   Action = "task.archived"
	ActionTaskUnarchived Action = "task.unarchived"
	ActionUserApproved   Action = "user.approved"
	ActionUserRetried    Action = "user.retried"
	ActionUserReturned   Action = "user.returned"
	ActionAgentStarted   Action = "agent.started"
	ActionAgentToolCall  Action = "agent.tool_call"
	ActionAgentFailed    Action = "agent.failed"
	ActionAgentComplete  Action = "agent.completed"
)

type Event struct {
	ID        uuid.UUID      `json:"id"`
	TaskID    *uuid.UUID     `json:"task_id"`
	ActorType ActorType      `json:"actor_type"`
	ActorID   string         `json:"actor_id"`
	Action    Action         `json:"action"`
	Payload   map[string]any `json:"payload"`
	CreatedAt time.Time      `json:"created_at"`
}
