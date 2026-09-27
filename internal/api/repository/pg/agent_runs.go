package pg

import (
	"context"

	"github.com/google/uuid"

	"kaiban/internal/api/domain/agentrun"
)

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
	payload, err := marshalJSON(e.Payload, "run event payload")
	if err != nil {
		return err
	}
	_, err = s.db.Pool.Exec(ctx, `
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
		if err := unmarshalJSON(payload, &e.Payload, "run event payload"); err != nil {
			return nil, err
		}
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
		return nil, mapNoRows(err, "scan agent run")
	}
	r.Status = agentrun.Status(status)
	return r, nil
}
