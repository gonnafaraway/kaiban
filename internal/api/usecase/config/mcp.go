package config

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	"kaiban/internal/api/domain/mcpserver"
	"kaiban/internal/api/integration"
)

func (u *Manager) ListMCP(ctx context.Context) ([]*mcpserver.Server, error) {
	items, err := u.Repo.MCP.List(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "list mcp servers")
	}
	return items, nil
}

func (u *Manager) UpsertMCP(ctx context.Context, id *uuid.UUID, name, endpoint string, headers map[string]string, status string) (*mcpserver.Server, error) {
	now := time.Now().UTC()
	var st mcpserver.Status
	if status != "" {
		parsed, err := parseMCPStatus(status)
		if err != nil {
			return nil, errors.Wrap(err, "parse mcp status")
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
	s := &mcpserver.Server{
		ID: uuid.New(), Name: name, Endpoint: endpoint, Headers: headers,
		Capabilities: map[string]any{}, Status: mcpserver.StatusDisabled, CreatedAt: now, UpdatedAt: now,
	}
	if status != "" {
		s.Status = st
	}
	if err := u.Repo.MCP.Create(ctx, s); err != nil {
		return nil, errors.Wrap(err, "create")
	}
	return s, nil
}

func (u *Manager) DeleteMCP(ctx context.Context, id uuid.UUID) error {
	if err := u.Repo.MCP.Delete(ctx, id); err != nil {
		return errors.Wrap(err, "delete mcp server")
	}
	return nil
}

// TestMCP handshakes with the server and stores the reported capabilities.
func (u *Manager) TestMCP(ctx context.Context, id uuid.UUID) (*mcpserver.Server, error) {
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
