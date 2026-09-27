package agent

import (
	"context"
	"strings"

	"github.com/pkg/errors"

	domint "kaiban/internal/api/domain/integration"
	"kaiban/internal/api/domain/mcpserver"
	"kaiban/internal/api/domain/settings"
	"kaiban/internal/api/domain/task"
	"kaiban/internal/api/integration"
	"kaiban/internal/api/integration/llm"
	"kaiban/internal/api/usecase/core"
)

// collectTools builds the tool set for a run: git workspace, enabled integrations, enabled MCP servers.
// MCP handshake failures are logged and skipped; repository failures abort the run.
func (u *Runner) collectTools(ctx context.Context, ar *activeRun, st *settings.Settings, t *task.Task) ([]integration.Tool, error) {
	var tools []integration.Tool
	repoURL, gitTok, useGitHub := u.ResolveGitRemote(ctx, st, t.ContextData.Artifacts)
	if repoURL != "" {
		tools = append(tools, integration.GitTool{
			WorkDir: u.GitWorkDir, RepoURL: repoURL, Branch: t.GitBranch, Token: gitTok, GitHubAuth: useGitHub,
		})
	}
	ints, err := u.Repo.Integrations.List(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "list integrations")
	}
	for _, i := range ints {
		if i.Status != domint.StatusEnabled {
			continue
		}
		email, tok := core.IntegrationCreds(i)
		switch i.Type {
		case domint.TypeJira:
			tools = append(tools, integration.JiraTools(i.BaseURL, email, tok)...)
		case domint.TypeConfluence:
			tools = append(tools, integration.ConfluenceTools(i.BaseURL, email, tok)...)
		case domint.TypeGitLab:
			tools = append(tools, integration.GitLabTools(i.BaseURL, tok)...)
		case domint.TypeGitHub:
			tools = append(tools, integration.GitHubTools(i.BaseURL, tok)...)
		}
	}
	mcps, err := u.Repo.MCP.List(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "list mcp servers")
	}
	for _, m := range mcps {
		if m.Status != mcpserver.StatusEnabled {
			continue
		}
		_, mcpTools, err := integration.HandshakeMCP(ctx, m.Endpoint, m.Headers)
		if err != nil {
			u.runLog(ctx, ar, t.ID, "tool", "MCP «"+m.Name+"» недоступен: "+err.Error(), map[string]any{
				"mcp": m.Name, "endpoint": m.Endpoint,
			})
			continue
		}
		for _, tl := range mcpTools {
			tools = append(tools, prefixTool{"mcp_" + slug(m.Name) + "_" + tl.Name(), tl})
		}
	}
	return tools, nil
}

// prefixTool namespaces an MCP tool so two servers can expose the same name.
type prefixTool struct {
	name  string
	inner integration.Tool
}

func (p prefixTool) Name() string               { return p.name }
func (p prefixTool) Description() string        { return p.inner.Description() }
func (p prefixTool) Parameters() map[string]any { return p.inner.Parameters() }

// Call delegates verbatim: the loop feeds the tool error text back to the model.
func (p prefixTool) Call(ctx context.Context, args string) (string, error) {
	return p.inner.Call(ctx, args)
}

func slug(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, " ", "_")
	return s
}

func toSpecs(tools []integration.Tool) []llm.ToolSpec {
	var out []llm.ToolSpec
	for _, t := range tools {
		var s llm.ToolSpec
		s.Type = "function"
		s.Function.Name = t.Name()
		s.Function.Description = t.Description()
		s.Function.Parameters = t.Parameters()
		if s.Function.Parameters == nil {
			s.Function.Parameters = map[string]any{"type": "object"}
		}
		out = append(out, s)
	}
	return out
}
