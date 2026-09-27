package kanban

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"kaiban/internal/api/domain/auditevent"
	domint "kaiban/internal/api/domain/integration"
	"kaiban/internal/api/domain/mcpserver"
)

func (u *UseCase) audit(ctx context.Context, taskID *uuid.UUID, actor auditevent.ActorType, actorID, action string, payload map[string]any) error {
	if payload == nil {
		payload = map[string]any{}
	}
	return u.Repo.Audit.Add(ctx, &auditevent.Event{TaskID: taskID, ActorType: actor, ActorID: actorID, Action: action, Payload: payload})
}

func (u *UseCase) publish(ev string, payload any) {
	if u.Bus != nil {
		u.Bus.Publish(ev, payload)
	}
}

func (u *UseCase) agentLog(taskID uuid.UUID, kind, message string, extra map[string]any) {
	p := map[string]any{
		"task_id": taskID.String(),
		"kind":    kind,
		"message": message,
		"ts":      time.Now().UTC(),
	}
	for k, v := range extra {
		p[k] = v
	}
	u.publish("agent.log", p)
}

func (u *UseCase) integrationByType(ctx context.Context, typ domint.Type) *domint.Integration {
	items, err := u.Repo.Integrations.List(ctx)
	if err != nil {
		return nil
	}
	for _, i := range items {
		if i.Type == typ && i.Status == domint.StatusEnabled {
			return i
		}
	}
	return nil
}

func (u *UseCase) localActor(ctx context.Context) string {
	me, err := u.Repo.Users.GetLocal(ctx)
	if err != nil || me == nil {
		return "local"
	}
	return me.Login
}

func integrationCreds(i *domint.Integration) (email, token string) {
	if i == nil {
		return "", ""
	}
	email = i.Credentials["email"]
	token = i.Credentials["api_token"]
	if token == "" {
		token = i.Credentials["token"]
	}
	return email, token
}

func parseMCPStatus(s string) (mcpserver.Status, error) {
	st := mcpserver.Status(strings.TrimSpace(s))
	switch st {
	case mcpserver.StatusDisabled, mcpserver.StatusEnabled, mcpserver.StatusError:
		return st, nil
	default:
		return "", fmt.Errorf("invalid mcp status %q", s)
	}
}

func truncateRunes(s string, n int) string {
	if n <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "..."
}
