-- Diff/PR gate + project context pack

ALTER TABLE columns
    ADD COLUMN IF NOT EXISTS requires_git_diff BOOLEAN NOT NULL DEFAULT false;

ALTER TABLE tasks
    ADD COLUMN IF NOT EXISTS git_pr_url TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS git_push_status TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS git_pr_status TEXT NOT NULL DEFAULT '';

ALTER TABLE app_settings
    ADD COLUMN IF NOT EXISTS context_pack JSONB NOT NULL DEFAULT '[]'::jsonb;

ALTER TABLE agent_runs
    ADD COLUMN IF NOT EXISTS context_pack_hash TEXT NOT NULL DEFAULT '';
