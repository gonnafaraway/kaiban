package kanban

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	domint "kaiban/internal/api/domain/integration"
	"kaiban/internal/api/domain/mcpserver"
	"kaiban/internal/api/integration"
)

func (u *UseCase) ListIntegrations(ctx context.Context) ([]*domint.Integration, error) {
	items, err := u.Repo.Integrations.List(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "list integrations")
	}
	return items, nil
}

func (u *UseCase) UpsertIntegration(ctx context.Context, id *uuid.UUID, typ, name, base string, cred map[string]string, status string) (*domint.Integration, error) {
	now := time.Now().UTC()
	var item *domint.Integration
	var err error
	if id != nil {
		item, err = u.Repo.Integrations.Get(ctx, *id)
		if err != nil {
			return nil, errors.Wrap(err, "get integration")
		}
		item.Name, item.BaseURL, item.Type = name, base, domint.Type(typ)
		if len(cred) > 0 {
			if item.Credentials == nil {
				item.Credentials = map[string]string{}
			}
			for k, v := range cred {
				if !strings.Contains(v, "*") {
					item.Credentials[k] = v
				}
			}
		}
		if status != "" {
			item.Status = domint.Status(status)
		}
		item.UpdatedAt = now
		if err := u.Repo.Integrations.Update(ctx, item); err != nil {
			return nil, errors.Wrap(err, "update")
		}
		return item, nil
	}
	item = &domint.Integration{
		ID: uuid.New(), Type: domint.Type(typ), Name: name, BaseURL: base,
		Credentials: cred, Status: domint.StatusDisabled, CreatedAt: now, UpdatedAt: now,
	}
	if status != "" {
		item.Status = domint.Status(status)
	}
	if err := u.Repo.Integrations.Create(ctx, item); err != nil {
		return nil, errors.Wrap(err, "create")
	}
	return item, nil
}

func (u *UseCase) DeleteIntegration(ctx context.Context, id uuid.UUID) error {
	if err := u.Repo.Integrations.Delete(ctx, id); err != nil {
		return errors.Wrap(err, "delete integration")
	}
	return nil
}

func (u *UseCase) TestIntegration(ctx context.Context, id uuid.UUID) (*domint.Integration, error) {
	item, err := u.Repo.Integrations.Get(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "get integration")
	}
	item.UpdatedAt = time.Now().UTC()
	if item.BaseURL == "" {
		item.Status = domint.StatusError
		item.LastError = "base_url is empty"
	} else {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, item.BaseURL, nil)
		if err != nil {
			item.Status = domint.StatusError
			item.LastError = err.Error()
		} else {
			email, tok := integrationCreds(item)
			switch item.Type {
			case domint.TypeJira, domint.TypeConfluence:
				if email != "" || tok != "" {
					for k, v := range integration.AtlassianAuth(email, tok) {
						req.Header.Set(k, v)
					}
				}
			case domint.TypeGitLab:
				if tok != "" {
					for k, v := range integration.GitLabAuth(tok) {
						req.Header.Set(k, v)
					}
				}
			case domint.TypeGitHub:
				if tok != "" {
					for k, v := range integration.GitHubAuth(tok) {
						req.Header.Set(k, v)
					}
				}
			}
			client := &http.Client{Timeout: 10 * time.Second}
			resp, err := client.Do(req)
			if err != nil {
				item.Status = domint.StatusError
				item.LastError = err.Error()
			} else {
				_ = resp.Body.Close()
				if resp.StatusCode >= 400 {
					item.Status = domint.StatusError
					item.LastError = fmt.Sprintf("HTTP %d", resp.StatusCode)
				} else {
					item.Status = domint.StatusEnabled
					item.LastError = ""
				}
			}
		}
	}
	if err := u.Repo.Integrations.Update(ctx, item); err != nil {
		return nil, errors.Wrap(err, "update")
	}
	return item, nil
}

func (u *UseCase) ListMCP(ctx context.Context) ([]*mcpserver.Server, error) {
	items, err := u.Repo.MCP.List(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "list mcp servers")
	}
	return items, nil
}

func (u *UseCase) UpsertMCP(ctx context.Context, id *uuid.UUID, name, endpoint string, headers map[string]string, status string) (*mcpserver.Server, error) {
	now := time.Now().UTC()
	var st mcpserver.Status
	if status != "" {
		parsed, err := parseMCPStatus(status)
		if err != nil {
			return nil, errors.Wrap(err, "parse mcpstatus")
		}
		st = parsed
	}
	if id != nil {
		s, err := u.Repo.MCP.Get(ctx, *id)
		if err != nil {
			return nil, errors.Wrap(err, "get mcp server")
		}
		if name != "" {
			s.Name = name
		}
		if endpoint != "" {
			s.Endpoint = endpoint
		}
		if headers != nil {
			s.Headers = headers
		}
		if status != "" {
			s.Status = st
		}
		s.UpdatedAt = now
		if err := u.Repo.MCP.Update(ctx, s); err != nil {
			return nil, errors.Wrap(err, "update")
		}
		return s, nil
	}
	s := &mcpserver.Server{ID: uuid.New(), Name: name, Endpoint: endpoint, Headers: headers, Capabilities: map[string]any{}, Status: mcpserver.StatusDisabled, CreatedAt: now, UpdatedAt: now}
	if status != "" {
		s.Status = st
	}
	if err := u.Repo.MCP.Create(ctx, s); err != nil {
		return nil, errors.Wrap(err, "create")
	}
	return s, nil
}

func (u *UseCase) DeleteMCP(ctx context.Context, id uuid.UUID) error {
	if err := u.Repo.MCP.Delete(ctx, id); err != nil {
		return errors.Wrap(err, "delete mcp server")
	}
	return nil
}

func (u *UseCase) TestMCP(ctx context.Context, id uuid.UUID) (*mcpserver.Server, error) {
	s, err := u.Repo.MCP.Get(ctx, id)
	if err != nil {
		return nil, errors.Wrap(err, "get mcp server")
	}
	caps, _, err := integration.HandshakeMCP(ctx, s.Endpoint, s.Headers)
	if err != nil {
		s.Status = mcpserver.StatusError
		s.LastError = err.Error()
	} else {
		s.Status = mcpserver.StatusEnabled
		s.LastError = ""
		s.Capabilities = caps
	}
	s.UpdatedAt = time.Now().UTC()
	if err := u.Repo.MCP.Update(ctx, s); err != nil {
		return nil, errors.Wrap(err, "update")
	}
	return s, nil
}
