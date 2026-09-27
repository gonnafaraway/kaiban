package pg

import (
	"context"
	"time"

	"github.com/google/uuid"

	"kaiban/internal/api/domain/auditevent"
)

func (s *Store) AddAudit(ctx context.Context, e *auditevent.Event) error {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	if e.CreatedAt.IsZero() {
		e.CreatedAt = time.Now().UTC()
	}
	payload, err := marshalJSON(e.Payload, "audit payload")
	if err != nil {
		return err
	}
	_, err = s.db.Pool.Exec(ctx, `INSERT INTO audit_events (id,task_id,actor_type,actor_id,action,payload,created_at) VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		e.ID, e.TaskID, string(e.ActorType), e.ActorID, e.Action, payload, e.CreatedAt)
	return err
}

func (s *Store) ListAudit(ctx context.Context, taskID uuid.UUID) ([]*auditevent.Event, error) {
	rows, err := s.db.Pool.Query(ctx, `SELECT id,task_id,actor_type,actor_id,action,payload,created_at FROM audit_events WHERE task_id=$1 ORDER BY created_at`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*auditevent.Event
	for rows.Next() {
		e := &auditevent.Event{Payload: map[string]any{}}
		var at string
		var payload []byte
		if err := rows.Scan(&e.ID, &e.TaskID, &at, &e.ActorID, &e.Action, &payload, &e.CreatedAt); err != nil {
			return nil, err
		}
		e.ActorType = auditevent.ActorType(at)
		if err := unmarshalJSON(payload, &e.Payload, "audit payload"); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
