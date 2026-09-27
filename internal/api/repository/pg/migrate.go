package pg

import (
	"context"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"kaiban/internal/api/domain/settings"
	"kaiban/internal/api/env"
	"kaiban/migrations"
)

func (s *Store) ApplySchema(ctx context.Context) error {
	if _, err := s.db.Pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			filename TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`); err != nil {
		return err
	}
	entries, err := fs.Glob(migrations.FS, "api/*.sql")
	if err != nil {
		return err
	}
	sort.Strings(entries)

	var ledgerCount int
	if err := s.db.Pool.QueryRow(ctx, `SELECT count(*) FROM schema_migrations`).Scan(&ledgerCount); err != nil {
		return err
	}
	if ledgerCount == 0 {
		var hasColumns bool
		if err := s.db.Pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM information_schema.tables
				WHERE table_schema='public' AND table_name='columns'
			)`).Scan(&hasColumns); err != nil {
			return err
		}
		if hasColumns {
			// Pre-ledger DB: stamp only baseline schema files, then apply later
			// migrations (they use IF NOT EXISTS). Avoids skipping 002+.
			for _, name := range entries {
				base := migrationBase(name)
				if !isBaselineMigration(base) {
					continue
				}
				if _, err := s.db.Pool.Exec(ctx, `INSERT INTO schema_migrations (filename) VALUES ($1) ON CONFLICT DO NOTHING`, base); err != nil {
					return err
				}
			}
		}
	}

	for _, name := range entries {
		base := migrationBase(name)
		var exists bool
		if err := s.db.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE filename=$1)`, base).Scan(&exists); err != nil {
			return err
		}
		if exists {
			continue
		}
		sql, err := migrations.FS.ReadFile(name)
		if err != nil {
			return err
		}
		tx, err := s.db.Pool.Begin(ctx)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, string(sql)); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("%s: %w", base, err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (filename) VALUES ($1)`, base); err != nil {
			_ = tx.Rollback(ctx)
			return err
		}
		if err := tx.Commit(ctx); err != nil {
			return err
		}
	}
	return nil
}

func migrationBase(name string) string {
	if i := strings.LastIndex(name, "/"); i >= 0 {
		return name[i+1:]
	}
	return name
}

func isBaselineMigration(base string) bool {
	return strings.HasPrefix(base, "000_") || strings.HasPrefix(base, "001_")
}

func (s *Store) SeedLLM(ctx context.Context, base, key, model string) error {
	if key == "" {
		return nil
	}
	provider := settings.InferProviderFromURL(base)
	_, err := s.db.Pool.Exec(ctx, `
		UPDATE app_settings
		SET llm_provider = $1, llm_base_url = $2, llm_api_key = $3, llm_model = $4, updated_at = now()
		WHERE llm_api_key = '' OR llm_api_key IS NULL
	`, provider, base, key, model)
	return err
}

func (s *Store) SeedIntegrations(ctx context.Context, e *env.Env) error {
	if e == nil {
		return nil
	}
	type seed struct {
		id, typ, name, base, email, token string
	}
	items := []seed{
		{"00000000-0000-4000-8000-000000000201", "jira", "Jira", e.JiraURL, e.JiraEmail, e.JiraToken},
		{"00000000-0000-4000-8000-000000000202", "confluence", "Confluence", e.ConfluenceURL, e.ConfluenceEmail, e.ConfluenceToken},
		{"00000000-0000-4000-8000-000000000203", "gitlab", "GitLab", e.GitLabURL, "", e.GitLabToken},
		{"00000000-0000-4000-8000-000000000204", "github", "GitHub", e.GitHubURL, "", e.GitHubToken},
	}
	for _, it := range items {
		if it.base == "" || it.token == "" {
			continue
		}
		cred, err := marshalJSON(map[string]string{
			"email":     it.email,
			"api_token": it.token,
			"token":     it.token,
		}, "integration credentials")
		if err != nil {
			return err
		}
		_, err = s.db.Pool.Exec(ctx, `
			INSERT INTO integrations (id, type, name, base_url, credentials, status, last_error, created_at, updated_at)
			SELECT $1::uuid, $2, $3, $4, $5::jsonb, 'enabled', '', now(), now()
			WHERE NOT EXISTS (SELECT 1 FROM integrations WHERE type = $2)
		`, it.id, it.typ, it.name, it.base, cred)
		if err != nil {
			return err
		}
	}
	return nil
}
