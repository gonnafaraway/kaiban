// Package config owns everything a user configures: settings, integrations,
// MCP servers and the portable config bundle.
package config

import (
	"fmt"

	"kaiban/internal/api/domain/mcpserver"
	"kaiban/internal/api/usecase/core"
)

// Manager reads and writes the application configuration.
type Manager struct {
	*core.Deps
}

func New(d *core.Deps) *Manager { return &Manager{Deps: d} }

func parseMCPStatus(s string) (mcpserver.Status, error) {
	st := mcpserver.Status(s)
	switch st {
	case mcpserver.StatusDisabled, mcpserver.StatusEnabled, mcpserver.StatusError:
		return st, nil
	default:
		return "", fmt.Errorf("invalid mcp status %q", s)
	}
}
