package task

import (
	"strings"
	"time"

	"github.com/google/uuid"

	"kaiban/internal/api/domain/column"
)

type Artifacts struct {
	JiraIssue     string `json:"jira_issue"`
	ConfluenceURL string `json:"confluence_url"`
	GitLabRepo    string `json:"gitlab_repo"`
	GitHubRepo    string `json:"github_repo"`
}

// BudgetOverride is optional per-task limit override (nil fields inherit).
type BudgetOverride struct {
	MaxTokens    *int     `json:"max_tokens,omitempty"`
	MaxCostUSD   *float64 `json:"max_cost_usd,omitempty"`
	MaxWallSec   *int     `json:"max_wall_sec,omitempty"`
	MaxToolCalls *int     `json:"max_tool_calls,omitempty"`
	MaxLLMSteps  *int     `json:"max_llm_steps,omitempty"`
}

type ContextData struct {
	ExtraInstructions string                       `json:"extra_instructions"`
	RetryNotes        []string                     `json:"retry_notes"`
	Artifacts         Artifacts                    `json:"artifacts"`
	StageOutputs      map[string]map[string]string `json:"stage_outputs"`
	Budget            BudgetOverride               `json:"budget"`
}

type Task struct {
	ID              uuid.UUID         `json:"id"`
	Title           string            `json:"title"`
	Description     string            `json:"description"`
	Variables       map[string]string `json:"variables"`
	ColumnID        uuid.UUID         `json:"column_id"`
	ExecutionStatus ExecutionStatus   `json:"execution_status"`
	GitBranch       string            `json:"git_branch"`
	GitPRURL        string            `json:"git_pr_url"`
	GitPushStatus   string            `json:"git_push_status"`
	GitPRStatus     string            `json:"git_pr_status"`
	CurrentReport   string            `json:"current_report"`
	ContextData     ContextData       `json:"context_data"`
	CreatedBy       uuid.UUID         `json:"created_by"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
	ArchivedAt      *time.Time        `json:"archived_at"`
}

func New(title, description string, variables map[string]string, artifacts Artifacts, columnID, createdBy uuid.UUID) (*Task, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, ErrTitleRequired
	}
	if variables == nil {
		variables = map[string]string{}
	}
	now := time.Now().UTC()
	return &Task{
		ID:              uuid.New(),
		Title:           title,
		Description:     description,
		Variables:       variables,
		ColumnID:        columnID,
		ExecutionStatus: StatusIdle,
		ContextData: ContextData{
			Artifacts:    artifacts,
			StageOutputs: map[string]map[string]string{},
		},
		CreatedBy: createdBy,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (t *Task) StageOutput(columnID uuid.UUID) map[string]string {
	if t.ContextData.StageOutputs == nil {
		return map[string]string{}
	}
	out := t.ContextData.StageOutputs[columnID.String()]
	if out == nil {
		return map[string]string{}
	}
	return out
}

func (t *Task) SetStageOutput(columnID uuid.UUID, values map[string]string) {
	if t.ContextData.StageOutputs == nil {
		t.ContextData.StageOutputs = map[string]map[string]string{}
	}
	cp := map[string]string{}
	for k, v := range values {
		cp[k] = v
	}
	t.ContextData.StageOutputs[columnID.String()] = cp
	t.UpdatedAt = time.Now().UTC()
}

func (t *Task) IsArchived() bool {
	return t.ArchivedAt != nil
}

func (t *Task) Archive() error {
	if t.IsArchived() {
		return ErrAlreadyArchived
	}
	if t.ExecutionStatus == StatusQueued || t.ExecutionStatus == StatusRunning {
		return ErrAlreadyRunning
	}
	now := time.Now().UTC()
	t.ArchivedAt = &now
	t.UpdatedAt = now
	return nil
}

func (t *Task) Unarchive() error {
	if !t.IsArchived() {
		return ErrNotArchived
	}
	t.ArchivedAt = nil
	t.UpdatedAt = time.Now().UTC()
	return nil
}

func (t *Task) CanRun() error {
	if t.IsArchived() {
		return ErrArchived
	}
	switch t.ExecutionStatus {
	case StatusIdle, StatusFailed, StatusSucceeded:
		return nil
	case StatusQueued, StatusRunning:
		return ErrAlreadyRunning
	default:
		return ErrInvalidTransition
	}
}

func (t *Task) MarkQueued() error {
	if err := t.CanRun(); err != nil {
		return err
	}
	t.ExecutionStatus = StatusQueued
	t.UpdatedAt = time.Now().UTC()
	return nil
}

func (t *Task) MarkRunning() error {
	if t.ExecutionStatus != StatusQueued && t.ExecutionStatus != StatusIdle {
		return ErrInvalidTransition
	}
	t.ExecutionStatus = StatusRunning
	t.UpdatedAt = time.Now().UTC()
	return nil
}

func (t *Task) MarkSucceeded(report string) error {
	if t.ExecutionStatus != StatusRunning && t.ExecutionStatus != StatusQueued {
		return ErrInvalidTransition
	}
	t.CurrentReport = report
	t.ExecutionStatus = StatusSucceeded
	t.UpdatedAt = time.Now().UTC()
	return nil
}

func (t *Task) MarkFailed(report string) error {
	if t.ExecutionStatus != StatusRunning && t.ExecutionStatus != StatusQueued {
		return ErrInvalidTransition
	}
	t.CurrentReport = report
	t.ExecutionStatus = StatusFailed
	t.UpdatedAt = time.Now().UTC()
	return nil
}

func (t *Task) Approve(isLast bool, nextColumnID uuid.UUID) error {
	if t.IsArchived() {
		return ErrArchived
	}
	if t.ExecutionStatus != StatusSucceeded {
		return ErrNotSucceeded
	}
	if isLast {
		t.ExecutionStatus = StatusDone
	} else {
		if nextColumnID == uuid.Nil {
			return ErrInvalidTransition
		}
		t.ColumnID = nextColumnID
		t.ExecutionStatus = StatusIdle
	}
	t.UpdatedAt = time.Now().UTC()
	return nil
}

func (t *Task) ReturnTo(current, target *column.Column, comment string) error {
	if t.IsArchived() {
		return ErrArchived
	}
	if strings.TrimSpace(comment) == "" {
		return ErrCommentRequired
	}
	if t.ExecutionStatus == StatusRunning || t.ExecutionStatus == StatusQueued {
		return ErrAlreadyRunning
	}
	if target == nil || current == nil {
		return ErrReturnTarget
	}
	if target.OrderIndex >= current.OrderIndex {
		return ErrReturnTarget
	}
	t.ColumnID = target.ID
	t.ExecutionStatus = StatusIdle
	t.CurrentReport = ""
	t.ContextData.RetryNotes = append(t.ContextData.RetryNotes, strings.TrimSpace(comment))
	t.UpdatedAt = time.Now().UTC()
	return nil
}

func (t *Task) RetryCurrent(comment string) error {
	if t.IsArchived() {
		return ErrArchived
	}
	if strings.TrimSpace(comment) == "" {
		return ErrCommentRequired
	}
	if t.ExecutionStatus == StatusRunning || t.ExecutionStatus == StatusQueued {
		return ErrAlreadyRunning
	}
	t.ExecutionStatus = StatusIdle
	t.ContextData.RetryNotes = append(t.ContextData.RetryNotes, strings.TrimSpace(comment))
	t.UpdatedAt = time.Now().UTC()
	return nil
}
