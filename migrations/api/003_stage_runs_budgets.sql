-- Stage contracts, agent runs, budget limits

ALTER TABLE columns
    ADD COLUMN IF NOT EXISTS output_fields JSONB NOT NULL DEFAULT '[]'::jsonb,
    ADD COLUMN IF NOT EXISTS max_tokens INT NULL,
    ADD COLUMN IF NOT EXISTS max_cost_usd DOUBLE PRECISION NULL,
    ADD COLUMN IF NOT EXISTS max_wall_sec INT NULL,
    ADD COLUMN IF NOT EXISTS max_tool_calls INT NULL,
    ADD COLUMN IF NOT EXISTS max_llm_steps INT NULL;

ALTER TABLE app_settings
    ADD COLUMN IF NOT EXISTS max_tokens INT NOT NULL DEFAULT 200000,
    ADD COLUMN IF NOT EXISTS max_cost_usd DOUBLE PRECISION NOT NULL DEFAULT 5,
    ADD COLUMN IF NOT EXISTS max_wall_sec INT NOT NULL DEFAULT 1800,
    ADD COLUMN IF NOT EXISTS max_tool_calls INT NOT NULL DEFAULT 80,
    ADD COLUMN IF NOT EXISTS max_llm_steps INT NOT NULL DEFAULT 40,
    ADD COLUMN IF NOT EXISTS price_input_per_1k DOUBLE PRECISION NOT NULL DEFAULT 0.002,
    ADD COLUMN IF NOT EXISTS price_output_per_1k DOUBLE PRECISION NOT NULL DEFAULT 0.008;

CREATE TABLE IF NOT EXISTS agent_runs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id         UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    column_id       UUID NOT NULL REFERENCES columns(id),
    status          TEXT NOT NULL DEFAULT 'running',
    model           TEXT NOT NULL DEFAULT '',
    prompt_hash     TEXT NOT NULL DEFAULT '',
    stop_reason     TEXT NOT NULL DEFAULT '',
    tokens_in       INT NOT NULL DEFAULT 0,
    tokens_out      INT NOT NULL DEFAULT 0,
    cost_usd        DOUBLE PRECISION NOT NULL DEFAULT 0,
    tool_calls      INT NOT NULL DEFAULT 0,
    llm_steps       INT NOT NULL DEFAULT 0,
    started_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    finished_at     TIMESTAMPTZ NULL
);

CREATE INDEX IF NOT EXISTS agent_runs_task_started_idx ON agent_runs (task_id, started_at DESC);

CREATE TABLE IF NOT EXISTS agent_run_events (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    run_id       UUID NOT NULL REFERENCES agent_runs(id) ON DELETE CASCADE,
    seq          INT NOT NULL,
    kind         TEXT NOT NULL,
    message      TEXT NOT NULL DEFAULT '',
    payload      JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS agent_run_events_run_seq_idx ON agent_run_events (run_id, seq);
