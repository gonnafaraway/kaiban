package kanban

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/pkg/errors"

	"kaiban/internal/api/domain/column"
	domint "kaiban/internal/api/domain/integration"
	"kaiban/internal/api/domain/mcpserver"
	"kaiban/internal/api/domain/settings"
)

const configBundleVersion = 1

// ConfigBundle is a portable full-app settings snapshot (no tasks / runs / jobs).
type ConfigBundle struct {
	Version      int                   `json:"version"`
	ExportedAt   time.Time             `json:"exported_at"`
	Settings     *settings.Settings    `json:"settings"`
	Integrations []ExportedIntegration `json:"integrations"`
	MCPServers   []ExportedMCPServer   `json:"mcp_servers"`
	Columns      []ExportedColumn      `json:"columns"`
}

type ExportedIntegration struct {
	ID          uuid.UUID         `json:"id"`
	Type        string            `json:"type"`
	Name        string            `json:"name"`
	BaseURL     string            `json:"base_url"`
	Credentials map[string]string `json:"credentials"`
	Status      string            `json:"status"`
}

type ExportedMCPServer struct {
	ID           uuid.UUID         `json:"id"`
	Name         string            `json:"name"`
	Endpoint     string            `json:"endpoint"`
	Headers      map[string]string `json:"headers"`
	Capabilities map[string]any    `json:"capabilities"`
	Status       string            `json:"status"`
}

type ExportedColumn struct {
	ID                   uuid.UUID            `json:"id"`
	Name                 string               `json:"name"`
	NameI18n             map[string]string    `json:"name_i18n"`
	SystemPromptDefault  string               `json:"system_prompt_default"`
	SystemPromptTemplate string               `json:"system_prompt_template"`
	UserCustomPrompt     *string              `json:"user_custom_prompt"`
	OutputFields         []column.OutputField `json:"output_fields"`
	Budget               column.BudgetLimits  `json:"budget"`
	RequiresGitDiff      bool                 `json:"requires_git_diff"`
	OrderIndex           int                  `json:"order_index"`
}

// ImportResult describes what import did (replace-all for integrations/MCP/columns when provided).
type ImportResult struct {
	Mode                 string `json:"mode"`
	Version              int    `json:"version"`
	SettingsUpdated      bool   `json:"settings_updated"`
	Integrations         int    `json:"integrations"`
	MCPServers           int    `json:"mcp_servers"`
	Columns              int    `json:"columns"`
	ReplacedIntegrations bool   `json:"replaced_integrations"`
	ReplacedMCP          bool   `json:"replaced_mcp_servers"`
	ReplacedColumns      bool   `json:"replaced_columns"`
}

func (u *UseCase) ExportConfig(ctx context.Context) (*ConfigBundle, error) {
	st, err := u.Repo.Settings.Get(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "get settings")
	}
	ints, err := u.Repo.Integrations.List(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "list integrations")
	}
	mcps, err := u.Repo.MCP.List(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "list mcp servers")
	}
	cols, err := u.Repo.Columns.List(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "list columns")
	}

	out := &ConfigBundle{
		Version:      configBundleVersion,
		ExportedAt:   time.Now().UTC(),
		Settings:     st,
		Integrations: make([]ExportedIntegration, 0, len(ints)),
		MCPServers:   make([]ExportedMCPServer, 0, len(mcps)),
		Columns:      make([]ExportedColumn, 0, len(cols)),
	}
	for _, i := range ints {
		cred := i.Credentials
		if cred == nil {
			cred = map[string]string{}
		}
		out.Integrations = append(out.Integrations, ExportedIntegration{
			ID: i.ID, Type: string(i.Type), Name: i.Name, BaseURL: i.BaseURL,
			Credentials: cred, Status: string(i.Status),
		})
	}
	for _, m := range mcps {
		headers := m.Headers
		if headers == nil {
			headers = map[string]string{}
		}
		caps := m.Capabilities
		if caps == nil {
			caps = map[string]any{}
		}
		out.MCPServers = append(out.MCPServers, ExportedMCPServer{
			ID: m.ID, Name: m.Name, Endpoint: m.Endpoint,
			Headers: headers, Capabilities: caps, Status: string(m.Status),
		})
	}
	for _, c := range cols {
		fields := c.OutputFields
		if fields == nil {
			fields = []column.OutputField{}
		}
		nameI18n := c.NameI18n
		if nameI18n == nil {
			nameI18n = map[string]string{}
		}
		out.Columns = append(out.Columns, ExportedColumn{
			ID: c.ID, Name: c.Name, NameI18n: nameI18n,
			SystemPromptDefault: c.SystemPromptDefault, SystemPromptTemplate: c.SystemPromptTemplate,
			UserCustomPrompt: c.UserCustomPrompt, OutputFields: fields, Budget: c.Budget,
			RequiresGitDiff: c.RequiresGitDiff, OrderIndex: c.OrderIndex,
		})
	}
	return out, nil
}

func (u *UseCase) ImportConfig(ctx context.Context, in *ConfigBundle) (*ImportResult, error) {
	if in == nil {
		return nil, errors.New("empty config")
	}
	if in.Version != 0 && in.Version != configBundleVersion {
		return nil, fmt.Errorf("unsupported config version %d", in.Version)
	}

	settingsUpdated := false
	if in.Settings != nil {
		if _, err := u.UpdateSettings(ctx, in.Settings); err != nil {
			return nil, errors.Wrap(err, "import settings")
		}
		settingsUpdated = true
	}

	replacedInt := false
	if in.Integrations != nil {
		if err := u.replaceIntegrations(ctx, in.Integrations); err != nil {
			return nil, errors.Wrap(err, "replace integrations")
		}
		replacedInt = true
	}
	replacedMCP := false
	if in.MCPServers != nil {
		if err := u.replaceMCP(ctx, in.MCPServers); err != nil {
			return nil, errors.Wrap(err, "replace mcp")
		}
		replacedMCP = true
	}
	replacedCols := false
	if in.Columns != nil {
		if err := u.replaceColumns(ctx, in.Columns); err != nil {
			return nil, errors.Wrap(err, "replace columns")
		}
		replacedCols = true
	}

	bundle, err := u.ExportConfig(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "config bundle")
	}
	return &ImportResult{
		Mode:                 "replace_all",
		Version:              configBundleVersion,
		SettingsUpdated:      settingsUpdated,
		Integrations:         len(bundle.Integrations),
		MCPServers:           len(bundle.MCPServers),
		Columns:              len(bundle.Columns),
		ReplacedIntegrations: replacedInt,
		ReplacedMCP:          replacedMCP,
		ReplacedColumns:      replacedCols,
	}, nil
}

func (u *UseCase) replaceIntegrations(ctx context.Context, items []ExportedIntegration) error {
	existing, err := u.Repo.Integrations.List(ctx)
	if err != nil {
		return errors.Wrap(err, "list integrations")
	}
	for _, e := range existing {
		if err := u.Repo.Integrations.Delete(ctx, e.ID); err != nil {
			return errors.Wrap(err, "delete")
		}
	}
	now := time.Now().UTC()
	for _, item := range items {
		typ := item.Type
		name := item.Name
		if typ == "" || name == "" {
			return errors.New("integration requires type and name")
		}
		id := item.ID
		if id == uuid.Nil {
			id = uuid.New()
		}
		status := domint.Status(item.Status)
		if status == "" {
			status = domint.StatusDisabled
		}
		cred := scrubSecretMap(item.Credentials)
		if err := u.Repo.Integrations.Create(ctx, &domint.Integration{
			ID: id, Type: domint.Type(typ), Name: name, BaseURL: item.BaseURL,
			Credentials: cred, Status: status, CreatedAt: now, UpdatedAt: now,
		}); err != nil {
			return errors.Wrap(err, "create integration")
		}
	}
	return nil
}

func (u *UseCase) replaceMCP(ctx context.Context, items []ExportedMCPServer) error {
	existing, err := u.Repo.MCP.List(ctx)
	if err != nil {
		return errors.Wrap(err, "list mcp servers")
	}
	for _, e := range existing {
		if err := u.Repo.MCP.Delete(ctx, e.ID); err != nil {
			return errors.Wrap(err, "delete mcp server")
		}
	}
	now := time.Now().UTC()
	for _, item := range items {
		name := item.Name
		endpoint := item.Endpoint
		if name == "" || endpoint == "" {
			return errors.New("mcp server requires name and endpoint")
		}
		id := item.ID
		if id == uuid.Nil {
			id = uuid.New()
		}
		status := mcpserver.Status(item.Status)
		if status == "" {
			status = mcpserver.StatusDisabled
		} else {
			parsed, err := parseMCPStatus(string(status))
			if err != nil {
				return errors.Wrap(err, "parse mcp status")
			}
			status = parsed
		}
		headers := scrubSecretMap(item.Headers)
		caps := item.Capabilities
		if caps == nil {
			caps = map[string]any{}
		}
		if err := u.Repo.MCP.Create(ctx, &mcpserver.Server{
			ID: id, Name: name, Endpoint: endpoint, Headers: headers, Capabilities: caps,
			Status: status, CreatedAt: now, UpdatedAt: now,
		}); err != nil {
			return errors.Wrap(err, "create mcp server")
		}
	}
	return nil
}

func (u *UseCase) replaceColumns(ctx context.Context, items []ExportedColumn) error {
	existing, err := u.Repo.Columns.List(ctx)
	if err != nil {
		return errors.Wrap(err, "list columns")
	}
	keep := make(map[uuid.UUID]struct{}, len(items))
	now := time.Now().UTC()

	for i, item := range items {
		name := item.Name
		if name == "" {
			if en := item.NameI18n["en"]; en != "" {
				name = en
			} else if ru := item.NameI18n["ru"]; ru != "" {
				name = ru
			}
		}
		if name == "" {
			return errors.New("column requires name")
		}
		id := item.ID
		if id == uuid.Nil {
			id = uuid.New()
		}
		keep[id] = struct{}{}
		order := item.OrderIndex
		if order == 0 && i > 0 {
			order = i
		}
		nameI18n := item.NameI18n
		if nameI18n == nil {
			nameI18n = map[string]string{}
		}
		if nameI18n["en"] == "" {
			nameI18n["en"] = name
		}
		if nameI18n["ru"] == "" {
			nameI18n["ru"] = name
		}
		fields := column.NormalizeOutputFields(item.OutputFields)
		promptDefault := item.SystemPromptDefault
		promptTemplate := item.SystemPromptTemplate
		if promptTemplate == "" {
			promptTemplate = promptDefault
		}
		if promptDefault == "" {
			promptDefault = promptTemplate
		}

		cur, getErr := u.Repo.Columns.Get(ctx, id)
		if getErr != nil {
			if !errors.Is(getErr, pgx.ErrNoRows) {
				return getErr
			}
			c := &column.Column{
				ID: id, Name: name, NameI18n: nameI18n,
				SystemPromptDefault: promptDefault, SystemPromptTemplate: promptTemplate,
				UserCustomPrompt: item.UserCustomPrompt, OutputFields: fields, Budget: item.Budget,
				RequiresGitDiff: item.RequiresGitDiff, OrderIndex: order,
				CreatedAt: now, UpdatedAt: now,
			}
			if err := u.Repo.Columns.Create(ctx, c); err != nil {
				return errors.Wrap(err, "create")
			}
			continue
		}
		cur.Name = name
		cur.NameI18n = nameI18n
		cur.SystemPromptDefault = promptDefault
		cur.SystemPromptTemplate = promptTemplate
		cur.UserCustomPrompt = item.UserCustomPrompt
		cur.OutputFields = fields
		cur.Budget = item.Budget
		cur.RequiresGitDiff = item.RequiresGitDiff
		cur.OrderIndex = order
		cur.UpdatedAt = now
		if err := u.Repo.Columns.Update(ctx, cur); err != nil {
			return errors.Wrap(err, "update")
		}
	}

	for _, cur := range existing {
		if _, ok := keep[cur.ID]; ok {
			continue
		}
		n, err := u.Repo.Columns.CountTasks(ctx, cur.ID)
		if err != nil {
			return errors.Wrap(err, "count tasks")
		}
		if n > 0 {
			return fmt.Errorf("cannot remove column %q (%s): it still has tasks", cur.Name, cur.ID)
		}
		if err := u.Repo.Columns.Delete(ctx, cur.ID); err != nil {
			return fmt.Errorf("cannot remove column %q (%s): %w", cur.Name, cur.ID, err)
		}
	}
	return nil
}

func scrubSecretMap(in map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range in {
		if strings.Contains(v, "*") {
			continue
		}
		out[k] = v
	}
	return out
}
