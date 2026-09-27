CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    login         TEXT NOT NULL UNIQUE,
    display_name  TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS app_settings (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    llm_base_url        TEXT NOT NULL DEFAULT 'https://api.openai.com/v1',
    llm_api_key         TEXT NOT NULL DEFAULT '',
    llm_model           TEXT NOT NULL DEFAULT 'gpt-4.1',
    git_repo_url        TEXT NOT NULL DEFAULT '',
    git_default_branch  TEXT NOT NULL DEFAULT 'main',
    locale              TEXT NOT NULL DEFAULT 'ru' CHECK (locale IN ('ru', 'en')),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS columns (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name                    TEXT NOT NULL,
    name_i18n               JSONB NOT NULL DEFAULT '{}'::jsonb,
    system_prompt_default   TEXT NOT NULL,
    system_prompt_template  TEXT NOT NULL,
    user_custom_prompt      TEXT NULL,
    order_index             INT NOT NULL UNIQUE,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS tasks (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title                   TEXT NOT NULL,
    description             TEXT NOT NULL DEFAULT '',
    variables               JSONB NOT NULL DEFAULT '{}'::jsonb,
    column_id               UUID NOT NULL REFERENCES columns(id),
    execution_status        TEXT NOT NULL DEFAULT 'idle',
    git_branch              TEXT NULL,
    current_report          TEXT NOT NULL DEFAULT '',
    context_data            JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_by              UUID NOT NULL REFERENCES users(id),
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS task_column_reports (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id      UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    column_id    UUID NOT NULL REFERENCES columns(id),
    report_md    TEXT NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS audit_events (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id     UUID NULL REFERENCES tasks(id) ON DELETE SET NULL,
    actor_type  TEXT NOT NULL,
    actor_id    TEXT NOT NULL,
    action      TEXT NOT NULL,
    payload     JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS audit_events_task_created_idx ON audit_events (task_id, created_at);

CREATE TABLE IF NOT EXISTS jobs (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id       UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    column_id     UUID NOT NULL REFERENCES columns(id),
    status        TEXT NOT NULL DEFAULT 'queued',
    attempt       INT NOT NULL DEFAULT 0,
    last_error    TEXT NULL,
    leased_until  TIMESTAMPTZ NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS jobs_dequeue_idx ON jobs (status, created_at) WHERE status = 'queued';

CREATE TABLE IF NOT EXISTS integrations (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type         TEXT NOT NULL,
    name         TEXT NOT NULL,
    base_url     TEXT NOT NULL,
    credentials  JSONB NOT NULL DEFAULT '{}'::jsonb,
    status       TEXT NOT NULL DEFAULT 'disabled',
    last_error   TEXT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS mcp_servers (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name          TEXT NOT NULL,
    endpoint      TEXT NOT NULL,
    headers       JSONB NOT NULL DEFAULT '{}'::jsonb,
    capabilities  JSONB NOT NULL DEFAULT '{}'::jsonb,
    status        TEXT NOT NULL DEFAULT 'disabled',
    last_error    TEXT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO users (id, login, display_name)
VALUES ('00000000-0000-4000-8000-000000000001', 'local', 'Local User')
ON CONFLICT (login) DO NOTHING;

INSERT INTO app_settings (id, locale)
SELECT '00000000-0000-4000-8000-000000000010', 'ru'
WHERE NOT EXISTS (SELECT 1 FROM app_settings);

INSERT INTO columns (id, name, name_i18n, system_prompt_default, system_prompt_template, order_index)
VALUES
(
  '00000000-0000-4000-8000-000000000101',
  'Product',
  '{"ru":"Продакт","en":"Product"}'::jsonb,
  $p$You are Product on this board. Write in the user locale.

Deliver a short product brief a human can approve in two minutes.
Cover: problem, who it is for, value, success metric, in/out of scope, main risks, open questions.
Use tools only when they change the answer. End with markdown: clear headings, bullets, no fluff.$p$,
  $p$You are Product on this board. Write in the user locale.

Deliver a short product brief a human can approve in two minutes.
Cover: problem, who it is for, value, success metric, in/out of scope, main risks, open questions.
Use tools only when they change the answer. End with markdown: clear headings, bullets, no fluff.$p$,
  0
),
(
  '00000000-0000-4000-8000-000000000102',
  'Business Analyst',
  '{"ru":"Бизнес-аналитик","en":"Business Analyst"}'::jsonb,
  $p$You are a Business Analyst. Write in the user locale.

From the product brief and task context, produce: business requirements, user scenarios, constraints, acceptance criteria (testable).
Keep it short and concrete. Use tools when useful. End with markdown a human can approve in two minutes: headings, bullets, no fluff.$p$,
  $p$You are a Business Analyst. Write in the user locale.

From the product brief and task context, produce: business requirements, user scenarios, constraints, acceptance criteria (testable).
Keep it short and concrete. Use tools when useful. End with markdown a human can approve in two minutes: headings, bullets, no fluff.$p$,
  1
),
(
  '00000000-0000-4000-8000-000000000103',
  'System Analyst',
  '{"ru":"Системный аналитик","en":"System Analyst"}'::jsonb,
  $p$You are a System Analyst. Write in the user locale.

Produce system requirements: data, APIs, integrations, edge cases that affect design.
Be specific (names, contracts, failure modes). Use tools when useful. End with short markdown: headings, bullets, no fluff.$p$,
  $p$You are a System Analyst. Write in the user locale.

Produce system requirements: data, APIs, integrations, edge cases that affect design.
Be specific (names, contracts, failure modes). Use tools when useful. End with short markdown: headings, bullets, no fluff.$p$,
  2
),
(
  '00000000-0000-4000-8000-000000000104',
  'Developer',
  '{"ru":"Разработчик","en":"Developer"}'::jsonb,
  $p$You are a Developer. Write in the user locale.

Implement the task in this task's git branch. Prefer small working changes. Commit/push with tools when available.
End with a short markdown report: what changed, how to verify, risks. No fluff.$p$,
  $p$You are a Developer. Write in the user locale.

Implement the task in this task's git branch. Prefer small working changes. Commit/push with tools when available.
End with a short markdown report: what changed, how to verify, risks. No fluff.$p$,
  3
),
(
  '00000000-0000-4000-8000-000000000105',
  'QA',
  '{"ru":"QA","en":"QA"}'::jsonb,
  $p$You are QA. Write in the user locale.

Produce a test plan, edge cases, and a pass/fail checklist tied to acceptance criteria when present.
End with short markdown a human can act on. No fluff.$p$,
  $p$You are QA. Write in the user locale.

Produce a test plan, edge cases, and a pass/fail checklist tied to acceptance criteria when present.
End with short markdown a human can act on. No fluff.$p$,
  4
),
(
  '00000000-0000-4000-8000-000000000106',
  'Auto QA',
  '{"ru":"Auto QA","en":"Auto QA"}'::jsonb,
  $p$You are Auto QA. Write in the user locale.

Propose or run practical automated checks for this change (commands, scripts, critical paths).
Report what you ran or would run, results, and blockers. Short markdown, no fluff.$p$,
  $p$You are Auto QA. Write in the user locale.

Propose or run practical automated checks for this change (commands, scripts, critical paths).
Report what you ran or would run, results, and blockers. Short markdown, no fluff.$p$,
  5
)
ON CONFLICT (id) DO NOTHING;
