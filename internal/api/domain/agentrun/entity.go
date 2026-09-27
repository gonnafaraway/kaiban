package agentrun

import (
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusRunning   Status = "running"
	StatusSucceeded Status = "succeeded"
	StatusFailed    Status = "failed"
	StatusStopped   Status = "stopped"
)

type Run struct {
	ID              uuid.UUID  `json:"id"`
	TaskID          uuid.UUID  `json:"task_id"`
	ColumnID        uuid.UUID  `json:"column_id"`
	Status          Status     `json:"status"`
	Model           string     `json:"model"`
	PromptHash      string     `json:"prompt_hash"`
	ContextPackHash string     `json:"context_pack_hash"`
	StopReason      string     `json:"stop_reason"`
	TokensIn        int        `json:"tokens_in"`
	TokensOut       int        `json:"tokens_out"`
	CostUSD         float64    `json:"cost_usd"`
	ToolCalls       int        `json:"tool_calls"`
	LLMSteps        int        `json:"llm_steps"`
	StartedAt       time.Time  `json:"started_at"`
	FinishedAt      *time.Time `json:"finished_at,omitempty"`
	Events          []Event    `json:"events,omitempty"`
}

type Event struct {
	ID        uuid.UUID      `json:"id"`
	RunID     uuid.UUID      `json:"run_id"`
	Seq       int            `json:"seq"`
	Kind      string         `json:"kind"`
	Message   string         `json:"message"`
	Payload   map[string]any `json:"payload,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
}
