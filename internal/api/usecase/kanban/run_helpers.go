package kanban

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	"kaiban/internal/api/domain/agentrun"
	"kaiban/internal/api/domain/column"
	"kaiban/internal/api/domain/settings"
	"kaiban/internal/api/domain/task"
)

type activeRun struct {
	run               *agentrun.Run
	seq               int
	budget            settings.Budget
	priceIn, priceOut float64
	started           time.Time
}

func (u *UseCase) startRun(ctx context.Context, t *task.Task, col *column.Column, st *settings.Settings, systemPrompt, packHash string) (*activeRun, error) {
	sum := sha256.Sum256([]byte(systemPrompt))
	r := &agentrun.Run{
		ID:              uuid.New(),
		TaskID:          t.ID,
		ColumnID:        col.ID,
		Status:          agentrun.StatusRunning,
		Model:           st.LLMModel,
		PromptHash:      hex.EncodeToString(sum[:8]),
		ContextPackHash: packHash,
		StartedAt:       time.Now().UTC(),
	}
	if err := u.Repo.AgentRuns.Create(ctx, r); err != nil {
		return nil, errors.Wrap(err, "create")
	}
	b := st.BaseBudget()
	b = settings.ApplyBudgetOverride(b, col.Budget.MaxTokens, col.Budget.MaxCostUSD, col.Budget.MaxWallSec, col.Budget.MaxToolCalls, col.Budget.MaxLLMSteps)
	tb := t.ContextData.Budget
	b = settings.ApplyBudgetOverride(b, tb.MaxTokens, tb.MaxCostUSD, tb.MaxWallSec, tb.MaxToolCalls, tb.MaxLLMSteps)
	return &activeRun{
		run: r, budget: b,
		priceIn: st.PriceInputPer1K, priceOut: st.PriceOutputPer1K,
		started: time.Now(),
	}, nil
}

func (u *UseCase) runLog(ctx context.Context, ar *activeRun, taskID uuid.UUID, kind, message string, extra map[string]any) {
	u.agentLog(taskID, kind, message, extra)
	if ar == nil || u.Repo.AgentRuns == nil {
		return
	}
	ar.seq++
	e := &agentrun.Event{
		ID: uuid.New(), RunID: ar.run.ID, Seq: ar.seq, Kind: kind, Message: message,
		Payload: extra, CreatedAt: time.Now().UTC(),
	}
	if e.Payload == nil {
		e.Payload = map[string]any{}
	}
	_ = u.Repo.AgentRuns.AddEvent(ctx, e)
}

func (ar *activeRun) addTokens(in, out int) {
	ar.run.TokensIn += in
	ar.run.TokensOut += out
	ar.run.CostUSD = settings.EstimateCostUSD(ar.run.TokensIn, ar.run.TokensOut, ar.priceIn, ar.priceOut)
}

func (ar *activeRun) checkBudget() (stop bool, reason string) {
	if ar.run.LLMSteps >= ar.budget.MaxLLMSteps {
		return true, fmt.Sprintf("max_llm_steps=%d", ar.budget.MaxLLMSteps)
	}
	if ar.run.ToolCalls >= ar.budget.MaxToolCalls {
		return true, fmt.Sprintf("max_tool_calls=%d", ar.budget.MaxToolCalls)
	}
	if ar.run.TokensIn+ar.run.TokensOut >= ar.budget.MaxTokens {
		return true, fmt.Sprintf("max_tokens=%d", ar.budget.MaxTokens)
	}
	if ar.run.CostUSD >= ar.budget.MaxCostUSD {
		return true, fmt.Sprintf("max_cost_usd=%.4f", ar.budget.MaxCostUSD)
	}
	if time.Since(ar.started) >= time.Duration(ar.budget.MaxWallSec)*time.Second {
		return true, fmt.Sprintf("max_wall_sec=%d", ar.budget.MaxWallSec)
	}
	return false, ""
}

func (u *UseCase) finishRun(ctx context.Context, ar *activeRun, status agentrun.Status, reason string) {
	if ar == nil || u.Repo.AgentRuns == nil {
		return
	}
	now := time.Now().UTC()
	ar.run.Status = status
	ar.run.StopReason = reason
	ar.run.FinishedAt = &now
	ar.run.CostUSD = settings.EstimateCostUSD(ar.run.TokensIn, ar.run.TokensOut, ar.priceIn, ar.priceOut)
	_ = u.Repo.AgentRuns.Update(ctx, ar.run)
}

func (u *UseCase) ListRuns(ctx context.Context, taskID uuid.UUID) ([]*agentrun.Run, error) {
	items, err := u.Repo.AgentRuns.ListByTask(ctx, taskID)
	if err != nil {
		return nil, errors.Wrap(err, "list agent runs")
	}
	return items, nil
}

func (u *UseCase) GetRun(ctx context.Context, runID uuid.UUID) (*agentrun.Run, error) {
	r, err := u.Repo.AgentRuns.Get(ctx, runID)
	if err != nil {
		return nil, errors.Wrap(err, "get agent run")
	}
	events, err := u.Repo.AgentRuns.ListEvents(ctx, runID)
	if err != nil {
		return nil, errors.Wrap(err, "list run events")
	}
	r.Events = events
	return r, nil
}

func estimateMessagesTokens(msgs []string) int {
	n := 0
	for _, m := range msgs {
		n += settings.EstimateTokens(m)
	}
	return n
}
