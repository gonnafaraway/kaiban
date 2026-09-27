// Package core holds the wiring and cross-cutting helpers shared by the board,
// agent and config use cases.
package core

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	"kaiban/internal/api/domain/auditevent"
	domint "kaiban/internal/api/domain/integration"
	"kaiban/internal/api/domain/settings"
	"kaiban/internal/api/domain/task"
	"kaiban/internal/api/integration"
	"kaiban/internal/api/integration/llm"
	"kaiban/internal/api/repository"
	httptransport "kaiban/internal/api/transport/http"
)

// Publisher fans an event out to connected clients.
type Publisher interface {
	Publish(event string, payload any)
}

// Deps is everything a use case service needs from the outside world.
type Deps struct {
	Repo       *repository.Repository
	LLM        llm.LLM
	GitWorkDir string
	Bus        Publisher
}

func (d *Deps) Audit(ctx context.Context, taskID *uuid.UUID, actor auditevent.ActorType, actorID string, action auditevent.Action, payload map[string]any) error {
	if payload == nil {
		payload = map[string]any{}
	}
	return errors.Wrap(d.Repo.Audit.Add(ctx, &auditevent.Event{
		TaskID: taskID, ActorType: actor, ActorID: actorID, Action: action, Payload: payload,
	}), "audit")
}

func (d *Deps) Publish(event string, payload any) {
	if d.Bus != nil {
		d.Bus.Publish(event, payload)
	}
}

func (d *Deps) AgentLog(taskID uuid.UUID, kind, message string, extra map[string]any) {
	p := map[string]any{
		"task_id": taskID.String(),
		"kind":    kind,
		"message": message,
		"ts":      time.Now().UTC(),
	}
	for k, v := range extra {
		p[k] = v
	}
	d.Publish(httptransport.EventAgentLog, p)
}

// IntegrationByType returns the enabled integration of that type, or nil.
func (d *Deps) IntegrationByType(ctx context.Context, typ domint.Type) *domint.Integration {
	items, err := d.Repo.Integrations.List(ctx)
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

func (d *Deps) LocalActor(ctx context.Context) string {
	me, err := d.Repo.Users.GetLocal(ctx)
	if err != nil || me == nil {
		return "local"
	}
	return me.Login
}

// ResolveGitRemote picks the GitHub repo artifact, then GitLab, then the settings URL.
func (d *Deps) ResolveGitRemote(ctx context.Context, st *settings.Settings, arts task.Artifacts) (repoURL, token string, useGitHub bool) {
	if arts.GitHubRepo != "" {
		repoURL = arts.GitHubRepo
		useGitHub = true
	} else if arts.GitLabRepo != "" {
		repoURL = arts.GitLabRepo
	} else if st != nil && st.GitRepoURL != "" {
		repoURL = st.GitRepoURL
		useGitHub = integration.IsGitHubHost(repoURL)
	}
	if repoURL == "" {
		return "", "", false
	}
	if useGitHub {
		if gh := d.IntegrationByType(ctx, domint.TypeGitHub); gh != nil {
			_, token = IntegrationCreds(gh)
		}
	} else if gl := d.IntegrationByType(ctx, domint.TypeGitLab); gl != nil {
		_, token = IntegrationCreds(gl)
	}
	return repoURL, token, useGitHub
}

func IntegrationCreds(i *domint.Integration) (email, token string) {
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
