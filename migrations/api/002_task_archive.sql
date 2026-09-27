ALTER TABLE tasks ADD COLUMN IF NOT EXISTS archived_at TIMESTAMPTZ NULL;

CREATE INDEX IF NOT EXISTS tasks_archived_at_idx ON tasks (archived_at DESC) WHERE archived_at IS NOT NULL;
