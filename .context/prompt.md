# Kaiban — спецификация для реализации

Роль исполнителя: Senior Full-Stack Architect и Lead Developer. Нужно спроектировать и реализовать систему управления агентами на Kanban-доске. Система модульная, понятная, пригодная к локальному запуску через Docker Compose. Этот документ — единственный источник правды по продукту и архитектуре v1.

---

## 1. Продукт

**Kaiban** — локальная single-user система: один разработчик на своей машине ведёт задачи по доске из ролей. Каждая колонка — роль (Продакт, аналитики, разработчик, QA). Агент колонки вызывается **вручную**, работает внутренним LLM-loop с tools (нативные интеграции + MCP), пишет markdown-отчёт. Вперёд карточку двигает только человек (Approve). Назад можно перетащить или вернуть через Retry, обязательно дополнив контекст.

Не SaaS, не мультитенант, не командный self-host в v1. В схеме сразу есть `users` (seed-пользователь), чтобы аудит был именным и позже можно было вырастить команду.

**Success criteria v1**

- Поднимается `docker compose up` (Postgres + Go API + Next.js).
- Есть одна доска с 6 seed-колонками, CRUD колонок и overlay-промптов.
- Создаётся задача (title + markdown + variables), агент стартует по кнопке, стримит прогресс по SSE, пишет отчёт.
- Approve двигает вперёд только после успешного прогона; Retry/DnD назад — в любую предыдущую колонку с обязательным комментарием.
- Подключенные Jira / Confluence / GitLab / MCP HTTP-SSE доступны **всем** агентам сразу.
- Git: один глобальный репозиторий, на задачу — ветка.
- CI: `go test ./...`, линтер Go, `next build`.

---

## 2. Зафиксированные решения (не переспрашивать)

| Тема | Решение |
| --- | --- |
| Режим | Local single-user |
| Исполнение агентов | Встроенный runtime: kaiban вызывает OpenAI-compatible LLM + tools в agent loop |
| LLM | Только OpenAI-compatible (один адаптер: base URL, API key, model) |
| Апрув | Всегда gate: вперёд только после ручного Approve |
| Старт агента | Только кнопка «Запустить агента» в текущей колонке, не автостарт при входе |
| Колонки | Seed при первом запуске, дальше полный CRUD и порядок |
| Seed-колонки | Продакт → Бизнес-аналитик → Системный аналитик → Разработчик → QA → Auto QA |
| Workspace | Задача привязана к git: глобальный repo в settings, ветка на задачу |
| Интеграции | Нативные адаптеры Jira, Confluence, GitLab **и** динамические MCP |
| MCP транспорт | Только remote HTTP/SSE (не stdio в v1) |
| Tools | Все включённые интеграции и MCP tools доступны всем агентам всех колонок |
| Фронт | Next.js App Router + TypeScript + shadcn/ui + Tailwind + @dnd-kit; **обязательно Feature-Sliced Design (FSD)** |
| Бэкенд | Go, Fiber, REST JSON + OpenAPI; **обязательно слои и каталоги [go-arch-template](https://github.com/gonnafaraway/go-arch-template)** |
| Realtime | SSE (статусы, логи агента, события доски) |
| Очередь | In-process worker pool + очередь в Postgres (`FOR UPDATE SKIP LOCKED`) |
| Промпты | Overlay: `system_prompt_template` + append `user_custom_prompt`; Reset обнуляет custom |
| Retry | Возврат в **любую предыдущую** колонку + новый контекст, процесс с этой колонки |
| DnD | Вперёд drop запрещён; назад можно, но модал с обязательным новым контекстом до применения |
| Секреты | Отдельная таблица `integrations`, credentials **просто в БД** (без шифрования в v1; local-only) |
| Auth | Таблица `users`, один seed-user; все действия от него |
| Карточка | title + markdown description + variables + артефакты: Jira issue (статусы агентов), Confluence URL (требования), GitLab repo (MR). Любую карточку можно архивировать (скрывается с доски). Файлов в v1 нет |
| Архив | Отдельный экран `/archive`: полный контекст (описание, артефакты, variables, retry notes, отчёты), возврат на доску |
| Аудит | Отдельная таблица событий; UI-история задачи строится из неё |
| Ошибка агента | Колонка не меняется, `execution_status=failed`; правка контекста + Retry текущего |
| Комментарии | Approve — опционально; Retry / возврат назад — обязательно |
| Результат агента | Markdown-отчёт по прогону колонки; git diff смотрят в репозитории/ветке, не в UI v1 |
| Доски | Одна доска на инстанс, колонки глобальные |
| i18n | ru + en с первого дня, дефолт `ru` |
| Деплой | docker-compose: `postgres`, `api`, `web` |
| Миграции | Atlas |
| Тесты | Минимум: `go test ./...` + Go lint + `next build` в CI |
| Язык этого промпта | Русский; идентификаторы кода, API, файлов — английские |

**Явно не в v1**

- OAuth интеграций (только API token / PAT / base URL).
- MCP stdio.
- Несколько досок, мультитенант, SSO, биллинг.
- Вложения файлов, object storage.
- gRPC.
- Внешние брокеры (Redis/NATS).
- Шифрование credentials at rest.
- Автостарт агента при смене колонки.
- Показ git diff в UI.
- Playwright e2e, TDD как обязательный процесс.

---

## 3. Стек

**Backend**

- Go (актуальный стабильный). Модуль и дерево каталогов как в [gonnafaraway/go-arch-template](https://github.com/gonnafaraway/go-arch-template): Clean Architecture / ports & adapters + DDD.
- Fiber только в `internal/api/transport/http` (как HTTP-адаптер шаблона).
- `pgx` через `internal/api/storage/postgres`.
- Логи: пакет `internal/api/infrastructure/local/log` (как в шаблоне; Zap или эквивалент structured JSON). Не логировать из domain.
- Atlas: `migrations/api` (как `migrations/api` в шаблоне).
- OpenAPI: `docs/api/openapi` — source of truth; фронт генерирует клиента. Go реализует API строго по спеке.
- Composition root: `internal/api/app` (`Prepare*` / явная DI, без service locator).
- Долгоживущие процессы: `internal/api/service` (`api` + `jobs` + `controller`). CDC/gRPC/Kafka из шаблона **не копировать**.

**Frontend**

- Next.js (App Router), React, TypeScript.
- Каталог `frontend/src` **строго FSD**: слои `app` → `pages` → `widgets` → `features` → `entities` → `shared`. Слой `processes` не использовать.
- Tailwind + shadcn/ui живут в `shared/ui`.
- @dnd-kit — в widget/feature канбана, не в `shared`.
- next-intl в `shared/i18n` + провайдер в `app`.
- SSE в `shared/api` или `entities/*/api`, виджеты только подписываются.
- Линтер архитектуры: [Steiger](https://github.com/feature-sliced/steiger) в CI фронта.

**Data / runtime**

- PostgreSQL 16.
- Git CLI на машине API-контейнера (образ с `git`).
- LLM: любой OpenAI-compatible Chat Completions (+ tool calling).

**Интеграции**

- Jira REST, Confluence REST, GitLab REST — нативные Go-клиенты.
- MCP: клиент к remote MCP over HTTP/SSE, реестр tools в рантайме.

---

## 4. Пользовательские сценарии

### 4.1 Первый запуск

1. `docker compose up`.
2. Применяются Atlas-миграции, seed: user, board, 6 колонок с базовыми system prompt (ru и en в шаблонах или i18n-ключах + дефолтный язык инстанса).
3. В Settings: LLM (base URL, key, model), путь/URL git-репозитория, интеграции.

### 4.2 Создание задачи

1. Кнопка «Создать задачу».
2. Поля: title, description (markdown), variables (список key/value), артефакты:
   - `jira_issue` — ключ или URL тикета, куда агенты пишут статусы (комментарии/переходы);
   - `confluence_url` — страница требований;
   - `gitlab_repo` — проект/URL, куда открывать MR с ветки задачи.
   Артефакты хранятся в `context_data.artifacts` (JSONB).
3. Задача появляется в первой колонке (`order_index` min), `execution_status=idle`.
4. Создаётся git-ветка `kaiban/task-<short-id>` от default branch: приоритет у `gitlab_repo` задачи, иначе глобальный git из Settings. Если репо не задано — задача живёт, git-tools недоступны.

### 4.3 Прогон колонки

1. Пользователь открывает карточку, при необходимости правит описание/variables.
2. Жмёт «Запустить агента».
3. Job в очереди Postgres, worker поднимает agent loop:
   - system = overlay промпта колонки;
   - user = описание + variables + linked artifacts (Jira/Confluence/GitLab) + сжатая история + отчёты предыдущих колонок + инструкции retry;
   - tools = все enabled native + все tools с registered MCP.
4. SSE: `task.updated`, `agent.log`, `agent.tool_call`, `agent.finished`.
5. Успех: markdown-отчёт сохраняется, `execution_status=succeeded`. Approve доступен.
6. Провал: `execution_status=failed`, отчёт/ошибка сохранены, Approve недоступен, доступны правка контекста и Retry текущего.

### 4.4 Approve

1. Только если текущая колонка успешно завершена агентом (`succeeded`).
2. Комментарий опционален.
3. Карточка переходит в следующую колонку по `order_index`, `execution_status=idle`.
4. Агент следующей колонки **не** стартует сам.
5. Если колонка последняя — Approve завершает задачу (`status=done` / колонка остаётся последней, `execution_status=done`). На последней колонке после success: Approve = Done.

### 4.5 Retry / возврат назад

1. Retry текущего: обязательный комментарий (новые инструкции), колонка та же, `idle` или сразу можно снова Run (после Retry статус `idle`, запуск ручной).
2. Возврат в предыдущую колонку (кнопка или DnD назад): выбор колонки с `order_index < current`, обязательный комментарий. `column_id` меняется, `idle`. Данные и история не удаляются.
3. Вперёд DnD и Approve «через колонки» запрещены.

### 4.6 Настройка колонки

- Имя, порядок, system prompt (базовый), user overlay, Reset overlay.
- Редактирование базового шаблона **разрешено как правка seed/template в БД**, но Reset **не** затирает `system_prompt_template`: Reset чистит только `user_custom_prompt`.
- Поставка обновляет factory-default шаблоны отдельно (поле `system_prompt_default` + возможность «восстановить базовый из поставки»).

Уточнение overlay и reset:

- `system_prompt_default` — неизменяемый заводской текст (можно обновить миграцией).
- `system_prompt_template` — рабочая копия базового, пользователь может править.
- `user_custom_prompt` — overlay, nullable.
- Runtime system prompt = `system_prompt_template` + `\n\n` + `user_custom_prompt` (если не null).
- «Сбросить overlay» → `user_custom_prompt = NULL`.
- «Восстановить заводской базовый» → `system_prompt_template = system_prompt_default`.

### 4.7 Интеграции и MCP

- UI: CRUD учёток type=jira|confluence|gitlab, base URL, token, опциональный email/username, status=enabled/disabled/error (проверка кнопкой Test connection).
- UI: CRUD MCP servers: name, endpoint URL, optional headers/token, status, список capabilities/tools после handshake.
- После enabled — tools сразу в глобальном каталоге агентов, без перезапуска процесса (hot reload registry).

---

## 5. Архитектура

Эталон бэкенда: **https://github.com/gonnafaraway/go-arch-template** (ветка `main`). Копировать **слои, имена пакетов, направление зависимостей, паттерны данных и bootstrap**, а не доменные сущности шаблона (Company/Order) и не лишние стораджи.

Эталон фронтенда: **Feature-Sliced Design** (https://feature-sliced.github.io/documentation/) — слои, слайсы, сегменты, правило импорта только вниз.

```
┌─────────────────────┐     REST+SSE      ┌──────────────────────────────────┐
│ Next.js + FSD       │ ◄──────────────► │ Go API (go-arch-template layers) │
│ app/pages/widgets/  │                   │ transport → handlers → usecase   │
│ features/entities/  │                   │ domain ← repository              │
│ shared              │                   │ service: api + jobs              │
└─────────────────────┘                   └──────────┬───────────────────────┘
                                                     │
                         ┌───────────────────────────┼───────────────────────────┐
                         ▼                           ▼                           ▼
                   PostgreSQL                   Git worktree                LLM + MCP
                   (storage/postgres)           (integration/git)           (integration)
```

### 5.1 Backend: слои go-arch-template

Правило зависимостей (внутрь):  
`cmd` → `app` → `service` / `handlers` / `transport` → `usecase` → `domain`.  
`repository` (интерфейсы рядом с реализациями, usecase зависит от интерфейсов).  
`infrastructure` + `storage` + `integration` — внешний мир; **domain их не импортирует**.

| Слой шаблона | Путь | В Kaiban |
| --- | --- | --- |
| Domain | `internal/api/domain/` | Богатые сущности: `task`, `column`, `user`, `job`, `integration`, `mcpserver`, `auditevent`, `settings`. Инварианты state machine — методы/функции domain (`Approve`, `ReturnTo`, `EnqueueRun`), domain errors. Без Fiber, SQL, HTTP-клиентов. |
| Use case | `internal/api/usecase/` | Один сценарий = один пакет/метод: create/update task, run, approve, retry, return, column CRUD/reorder/reset-overlay, settings, integrations CRUD+test, mcp CRUD+test, list events. Usecase-DTO на границе. Оркестрация: repos + integrations + enqueue job. |
| Repository | `internal/api/repository/` | Интерфейсы + `postgres_*.go` + `mock.go` на агрегат. Data mapper: domain ↔ row. Очередь jobs — `repository/job` (`Enqueue`, `Lease`, `Complete`). |
| Storage | `internal/api/storage/` | Только `postgres` (+ `storage.go` фабрика). Mongo/Redis/Kafka/S3/Nexus из шаблона **не добавлять**. |
| Infrastructure | `internal/api/infrastructure/` | `local/log`, `local/trace` (no-op, если нет OTLP). `external/` — сырые HTTP-клиенты Jira/Confluence/GitLab/OpenAI-compatible/MCP, **без** доменных правил. |
| Integration | `internal/api/integration/` | Anti-corruption: интерфейсы `LLM`, `ToolRegistry`, `GitWorkspace`, `Jira`, `Confluence`, `GitLab`, `MCP`. Реализации собирают tools для агента. Domain/usecase говорят на этих интерфейсах. |
| Transport | `internal/api/transport/http/` | Fiber app, middleware (log, request_id), SSE, `response.go`, validation входных DTO. gRPC/RPC из шаблона **не поднимать**. |
| Handlers | `internal/api/handlers/` | Тонкие контроллеры по ресурсам: `task`, `column`, `settings`, `integration`, `mcp`, `me`, `health`. Bind routes в `handlers/router.go`. Никакого agent loop здесь. |
| Validator | `internal/api/validator/` | Валидация входных DTO (не подмена domain invariants). |
| Env | `internal/api/env/` | Конфиг из env. |
| App | `internal/api/app/` | Composition root: `PrepareStorage`, `PrepareRepos`, `PrepareIntegrations`, `PrepareUsecases`, `PrepareHTTP`. Явный DI. |
| Service | `internal/api/service/` | `api.go` — HTTP сервер; `jobs.go` — worker pool (lease Postgres jobs, вызов usecase `ExecuteAgentJob`); `controller.go` — параллельный start + graceful shutdown. Без CDC. |

Паттерны шаблона, обязательные в коде:

- Repository, Factory (`Prepare*`), Strategy (postgres vs mock), Adapter, DI, Middleware, Use Case как команда, DTO на границах, ACL для внешних API.
- Entities в domain; Value objects где уместно (`ExecutionStatus`, `Email` не обязательно).
- DTO только в transport/handlers/usecase, не в domain.
- Data mapper в repository/postgres, не в handler.

Поток HTTP как в шаблоне:

```
HTTP → transport middleware → handler → usecase → domain
                         ↘ repository → storage/postgres
                         ↘ integration → infrastructure/external
```

Поток агента:

```
handler Run → usecase RunTask → domain.Enqueue + job repo
service/jobs lease → usecase ExecuteAgentJob → integration LLM/tools/git
                 → domain success/fail → repos + audit → SSE hub (transport)
```

Запрещено: LLM из handler; смена колонки в обход domain/usecase; импорт `infrastructure` из `domain`; god-object `App` со всей логикой (app только wiring); копировать billing/oauth/cdc/mongodb из шаблона.

### 5.2 Frontend: Feature-Sliced Design

Слои сверху вниз (импорт **только с слоя ниже**, не из соседа на том же слое):

1. `app` — провайдеры, Next.js routing, глобальные стили, SSE-подписка уровня приложения.
2. `pages` — экраны: board, task-details, settings-general, settings-columns, settings-integrations, settings-mcp.
3. `widgets` — крупные блоки: `kanban-board`, `task-panel`, `audit-timeline` (без гигантского god-widget).
4. `features` — действия пользователя: `create-task`, `run-agent`, `approve-task`, `retry-task`, `return-task`, `edit-column-prompt`, `reorder-columns`, `upsert-integration`, `upsert-mcp-server`.
5. `entities` — `task`, `column`, `integration`, `mcp-server`, `settings`, `user`, `audit-event`.
6. `shared` — `ui` (shadcn), `api` (сгенерированный OpenAPI-клиент), `config`, `lib`, `i18n`.

Сегменты слайса (классика FSD): `ui/`, `model/`, `api/`, при необходимости `lib/`. Публичный доступ только через `index.ts` слайса. Кросс-импорты слайсов одного слоя запрещены (если нужно общее — опустить в entity/shared или завести `@x` public API по гайду FSD, лучше избегать в v1).

Next.js App Router: файлы маршрутов в `src/app/**/page.tsx` — **тонкие**, только рендер `pages/*`. Бизнес-UI не писать внутри route files.

Steiger: правила `fsd/no-public-api-sidestep`, `fsd/forbidden-imports`. CI падает при нарушении.

---

## 6. Модель данных (Postgres)

Требования исходного ТЗ сохранить и расширить. Имена таблиц — snake_case, PK UUID.

### 6.1 DDL (целевая схема)

Реализовать Atlas-схему, эквивалентную:

```sql
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    login         TEXT NOT NULL UNIQUE,
    display_name  TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE app_settings (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    llm_base_url        TEXT NOT NULL DEFAULT 'https://api.openai.com/v1',
    llm_api_key         TEXT NOT NULL DEFAULT '',
    llm_model           TEXT NOT NULL DEFAULT 'gpt-4.1',
    git_repo_url        TEXT NOT NULL DEFAULT '',
    git_default_branch  TEXT NOT NULL DEFAULT 'main',
    locale              TEXT NOT NULL DEFAULT 'ru' CHECK (locale IN ('ru', 'en')),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE columns (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name                    TEXT NOT NULL,
    name_i18n               JSONB NOT NULL DEFAULT '{}'::jsonb,
    system_prompt_default   TEXT NOT NULL,
    system_prompt_template  TEXT NOT NULL,
    user_custom_prompt      TEXT NULL,
    order_index             INT NOT NULL,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (order_index)
);

CREATE TABLE tasks (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title                   TEXT NOT NULL,
    description             TEXT NOT NULL DEFAULT '',
    variables               JSONB NOT NULL DEFAULT '{}'::jsonb,
    column_id               UUID NOT NULL REFERENCES columns(id),
    execution_status        TEXT NOT NULL DEFAULT 'idle'
        CHECK (execution_status IN (
            'idle', 'queued', 'running', 'succeeded', 'failed', 'done'
        )),
    git_branch              TEXT NULL,
    current_report          TEXT NOT NULL DEFAULT '',
    context_data            JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_by              UUID NOT NULL REFERENCES users(id),
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE task_column_reports (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id      UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    column_id    UUID NOT NULL REFERENCES columns(id),
    report_md    TEXT NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE audit_events (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id     UUID NULL REFERENCES tasks(id) ON DELETE SET NULL,
    actor_type  TEXT NOT NULL CHECK (actor_type IN ('user', 'agent', 'system')),
    actor_id    TEXT NOT NULL,
    action      TEXT NOT NULL,
    payload     JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX audit_events_task_created_idx ON audit_events (task_id, created_at);

CREATE TABLE jobs (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id       UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    column_id     UUID NOT NULL REFERENCES columns(id),
    status        TEXT NOT NULL DEFAULT 'queued'
        CHECK (status IN ('queued', 'running', 'succeeded', 'failed', 'canceled')),
    attempt       INT NOT NULL DEFAULT 0,
    last_error    TEXT NULL,
    leased_until  TIMESTAMPTZ NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX jobs_dequeue_idx ON jobs (status, created_at) WHERE status = 'queued';

CREATE TABLE integrations (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type         TEXT NOT NULL CHECK (type IN ('jira', 'confluence', 'gitlab')),
    name         TEXT NOT NULL,
    base_url     TEXT NOT NULL,
    credentials  JSONB NOT NULL DEFAULT '{}'::jsonb,
    status       TEXT NOT NULL DEFAULT 'disabled'
        CHECK (status IN ('disabled', 'enabled', 'error')),
    last_error   TEXT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE mcp_servers (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name          TEXT NOT NULL,
    endpoint      TEXT NOT NULL,
    headers       JSONB NOT NULL DEFAULT '{}'::jsonb,
    capabilities  JSONB NOT NULL DEFAULT '{}'::jsonb,
    status        TEXT NOT NULL DEFAULT 'disabled'
        CHECK (status IN ('disabled', 'enabled', 'error')),
    last_error    TEXT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

`credentials` JSONB (примеры, не логировать):

- jira: `{"email":"...","api_token":"..."}`
- confluence: `{"email":"...","api_token":"..."}`
- gitlab: `{"token":"..."}`

На GET интеграций маскировать секреты (`api_token`: последние 4 символа). Полный секрет только внутри API-процесса.

`context_data` JSONB минимум:

```json
{
  "extra_instructions": "",
  "retry_notes": []
}
```

История **не** дублируется в `tasks.history`: источник правды — `audit_events`. Поле history из исходного наброска не использовать.

### 6.2 Seed колонок

`order_index` 0..5:

| order | name ru | name en | смысл агента |
| --- | --- | --- | --- |
| 0 | Продакт | Product | проблема, ценность, критерии успеха, scope |
| 1 | Бизнес-аналитик | Business Analyst | бизнес-требования, сценарии, ограничения |
| 2 | Системный аналитик | System Analyst | системные требования, контракты, данные |
| 3 | Разработчик | Developer | реализация в git-ветке задачи |
| 4 | QA | QA | тест-план, ручные проверки, баги |
| 5 | Auto QA | Auto QA | автопроверки, команды тестов в репо |

Заводские system prompts — подробные, на языке `app_settings.locale` **или** двуязычный шаблон с инструкцией «отвечай на языке locale». Предпочтительнее: хранить промпт на английском для качества LLM + «Respond in {locale}».

---

## 7. State machine

Допустимые `execution_status`: `idle` → `queued` → `running` → `succeeded` | `failed`. Из `failed`: Retry текущего → `idle`. Из `succeeded`: Approve → следующая колонка `idle` либо `done` если последняя. Из любого не-`running`: ReturnTo(column) если target.order < current.order → target + `idle`.

Запрещено:

- Approve если не `succeeded`.
- Run если `queued`/`running`.
- Переход вперёд без Approve.
- Удаление audit_events при retry.
- Смена колонки worker'ом (worker только меняет execution_status и отчёт).

Псевдологика:

```
Run(task):
  require status in {idle, failed, succeeded}  // повторный run с succeeded = новый прогон текущей колонки
  enqueue job
  status = queued

Worker:
  status = running
  run agent
  on ok: save report, status = succeeded, audit agent.completed
  on err: status = failed, audit agent.failed

Approve(task, comment?):
  require succeeded
  if last column: status = done
  else: column = next, status = idle
  audit user.approved

ReturnTo(task, targetColumn, comment):
  require comment nonempty
  require target.order < current.order
  require status != running
  cancel queued job if any
  column = target, status = idle
  append retry note to context_data
  audit user.returned
```

Повторный Run после `succeeded` в той же колонке разрешён (перезапуск агента до Approve): старый отчёт остаётся в `task_column_reports`, текущий перезаписывается.

---

## 8. Agent loop

- Модель и ключ из `app_settings`.
- Tool calling в цикле, лимит: 40 шагов или 10 минут (job lease + timeout), что раньше.
- Tools: git (status, diff internal for model, commit, push если remote), filesystem только внутри worktree ветки задачи, jira_*, confluence_*, gitlab_*, mcp_* (префикс server name).
- Запись в git только в ветку задачи, не в default branch.
- Финальный ассистентский ответ без tool call = markdown-отчёт (обязательно непустой).
- Ошибки tool возвращаются в модель текстом, не роняют loop, кроме фатальных (нет LLM ключа, panic).
- Прогресс: каждый tool call и текстовый чанк — SSE + audit по завершении tool (без секретов).

---

## 9. Prompt overlay

Функция живёт в domain (колонка) или usecase колонки, **не** в handler.

```go
func BuildSystemPrompt(col Column) string {
    base := col.SystemPromptTemplate
    if strings.TrimSpace(col.UserCustomPrompt) == "" {
        return base
    }
    return base + "\n\n# User overlay\n" + col.UserCustomPrompt
}
```

Пользовательский слой не заменяет базовый. Заводской текст живёт в `system_prompt_default`.

---

## 10. MCP

Таблица `mcp_servers`. При enable:

1. Handshake MCP initialize по HTTP/SSE endpoint.
2. `tools/list` → `capabilities`.
3. Зарегистрировать tools в глобальном dispatcher.

При disable — снять tools из dispatcher.

Конфиг MCP: name, endpoint, optional headers (Authorization). CLI не обязателен в v1, но API должен позволить создать сервер (UI форма достаточна).

Агенту tools видны как `mcp_<slug>_<toolname>` с JSON schema из MCP.

---

## 11. Нативные интеграции

Каждая: Test connection, enable/disable, CRUD.

Минимальный набор tools v1 (реальные HTTP-вызовы, не заглушки):

**Jira:** search issues (JQL), get issue, create issue, add comment.  
**Confluence:** search, get page by id, get page body storage/markdown если доступно.  
**GitLab:** get project, list merge requests, create MR from task branch into default branch, get file raw (опционально).

После сохранения интеграции реестр tools обновляется сразу.

---

## 12. API

Префикс `/api/v1`. JSON. Ошибки: `{ "error": { "code": "...", "message": "..." } }`.

OpenAPI файл: `docs/api/openapi/openapi.yaml` (как `docs/api/openapi` в go-arch-template).

**Health:** `GET /health`

**Settings:**  
`GET /api/v1/settings`  
`PUT /api/v1/settings`

**Users:** `GET /api/v1/me`

**Columns:**  
`GET /api/v1/columns`  
`POST /api/v1/columns`  
`PATCH /api/v1/columns/{id}`  
`DELETE /api/v1/columns/{id}` (запрет если есть задачи; или запрет удалять последнюю колонку)  
`POST /api/v1/columns/{id}/reset-overlay`  
`POST /api/v1/columns/{id}/restore-default-prompt`  
`PUT /api/v1/columns/reorder` body: `[{id, order_index}]`

**Tasks:**  
`GET /api/v1/tasks`  
`POST /api/v1/tasks`  
`GET /api/v1/tasks/{id}`  
`PATCH /api/v1/tasks/{id}`  // title, description, variables, context extra_instructions  
`GET /api/v1/tasks/{id}/events`  
`POST /api/v1/tasks/{id}/run`  
`POST /api/v1/tasks/{id}/approve` `{ "comment": "" }`  
`POST /api/v1/tasks/{id}/return` `{ "column_id": "...", "comment": "..." }`  // comment required  
`POST /api/v1/tasks/{id}/retry` `{ "comment": "..." }` // current column, comment required

**Integrations:** CRUD + `POST /api/v1/integrations/{id}/test`

**MCP:** CRUD + `POST /api/v1/mcp-servers/{id}/test` (initialize + tools/list)

**SSE:** `GET /api/v1/events` (глобальный поток инстанса)  
События: `task.created`, `task.updated`, `column.updated`, `agent.log`, `agent.tool_call`, `job.updated`.

Realtime для фронта — этот SSE, не polling.

---

## 13. Фронтенд (FSD)

Страницы (слой `pages`, маршруты Next — обёртки):

- `/` — `pages/board` + widget `kanban-board`.
- `/tasks/[id]` — `pages/task-details`: описание, variables, отчёт, `widgets/audit-timeline`, features Run / Approve / Retry / Return.
- `/settings` — `pages/settings-general` (LLM, git, locale).
- `/settings/integrations` — `pages/settings-integrations`.
- `/settings/mcp` — `pages/settings-mcp`.
- `/settings/columns` — `pages/settings-columns` (промпты и overlay).

Доска:

- Колонки по `order_index`, карточки с title + badge execution_status (`entities/task`).
- DnD в `features` канбана или widget: drop вправо / вперёд — revert + toast «только Approve».
- Drop влево — модал комментария; OK вызывает feature `return-task`.
- Модал создания — `features/create-task`.
- i18n всех строк UI (`shared/i18n`).

Тема: нейтральный рабочий UI, не маркетинговый лендинг.

Не класть OpenAPI-клиент и shadcn в features. Entity не импортирует pages/widgets/features.

---

## 14. Файловая структура

Go — **корень репозитория как в go-arch-template** (не `backend/internal/http`). Frontend — соседний пакет `frontend/`.

```
kaiban/
  .context/prompt.md
  docker-compose.yml
  Makefile                 # цели как в шаблоне + frontend
  README.md
  go.mod
  go.sum
  atlas.hcl
  .github/workflows/ci.yml
  cmd/api/main.go          # только start controller
  docs/api/openapi/openapi.yaml
  migrations/api/          # Atlas
  build/                   # Dockerfile API, как build/ в шаблоне
  internal/api/
    app/
    domain/
      task/
      column/
      user/
      job/
      integration/
      mcpserver/
      auditevent/
      settings/
    env/
    handlers/
      router.go
      task/
      column/
      settings/
      integration/
      mcp/
      health/
    infrastructure/
      external/
        jira/
        confluence/
        gitlab/
        openai/
        mcp/
      local/
        log/
        trace/
    integration/           # ACL + tool registry + git workspace + agent runner port
    repository/
      task/
      column/
      user/
      job/
      integration/
      mcpserver/
      auditevent/
      settings/
      repository.go
    service/
      api.go
      jobs.go
      controller.go
    storage/
      postgres/
      storage.go
    transport/
      http/
        middleware/
        response.go
        transport.go
        validation.go
        sse.go
    usecase/
      task/
      column/
      settings/
      integration/
      mcp/
      agentjob/
    validator/
  frontend/
    Dockerfile
    package.json
    next.config.ts
    steiger.config.js
    src/
      app/                      # FSD app + Next routes
        layout.tsx
        providers.tsx
        page.tsx                # → pages/board
        tasks/[id]/page.tsx
        settings/...
      pages/
        board/
        task-details/
        settings-general/
        settings-columns/
        settings-integrations/
        settings-mcp/
      widgets/
        kanban-board/
        task-panel/
        audit-timeline/
      features/
        create-task/
        run-agent/
        approve-task/
        retry-task/
        return-task/
        edit-column-prompt/
        reorder-columns/
        upsert-integration/
        upsert-mcp-server/
      entities/
        task/
        column/
        integration/
        mcp-server/
        settings/
        user/
        audit-event/
      shared/
        ui/
        api/
        config/
        lib/
        i18n/
```

---

## 15. Безопасность и надёжность

- Local-only: сервисы слушают внутри compose network; в README предупреждение не публиковать порты в интернет.
- Credentials в `integrations.credentials` и `app_settings.llm_api_key` в БД plaintext (осознанно, v1). Не писать их в audit, SSE, логи.
- GET маскирует секреты.
- Path traversal: git и file tools ограничены worktree задачи.
- Worker lease: если процесс умер, job снова `queued` по истечении `leased_until`.
- Идемпотентность Approve: повторный Approve в `idle` следующей колонки — 409.
- Потеря LLM не удаляет задачу и отчёты.

---

## 16. Наблюдаемость

- JSON logs: request_id, task_id, job_id, column_id.
- `audit_events.action` примеры: `task.created`, `agent.started`, `agent.tool_call`, `agent.completed`, `agent.failed`, `user.approved`, `user.returned`, `user.retried`, `column.prompt_updated`, `integration.enabled`.
- Лента карточки = SELECT events WHERE task_id ORDER BY created_at.

---

## 17. Запуск и CI

`docker-compose.yml`:

- `postgres:16`
- `api` build из `build/` (как в шаблоне), wait for postgres, atlas migrate `migrations/api`, run `:8080`
- `web` build frontend, `:3000`, `API_URL=http://api:8080`

Makefile: `make up`, `make lint`, `make test`, `make migrate`.

CI: golangci-lint, `go test ./...`, `go mod tidy` check, atlas, frontend: Steiger (FSD) + lint + `next build`.

После изменений Go: `make lint`, `go mod tidy`, `go mod vendor` если в репо принят vendor; иначе без vendor, но tidy обязателен.

---

## 18. Ожидаемый результат работы по этому промпту

Сделать рабочий вертикальный продукт по всему контуру выше, не слайды.

Обязательные артефакты в репозитории:

1. Atlas-схема / миграции = DDL из раздела 6.
2. Структура каталогов из раздела 14 (слои go-arch-template + FSD, без плоского `internal/http` и без `src/components` вне FSD).
3. Код:
   - domain entities task/column + тесты инвариантов (`go test` на approve, illegal forward, return, fail stays) **в domain/usecase, не в handler**;
   - mock-репозитории как в шаблоне;
   - prompt overlay в domain/usecase;
   - MCP/native tools в `integration` + raw clients в `infrastructure/external`;
   - handlers + `docs/api/openapi/openapi.yaml`;
   - `service/jobs` worker;
   - Next Kanban по FSD + Steiger в CI + SSE.
4. README: как запустить, какие env, как подключить LLM и git repo.
5. Seed промптов колонок — осмысленные, не «you are a helpful assistant».

---

## 19. Инварианты (acceptance)

- Нельзя переместить карточку вперёд без `execution_status=succeeded` + Approve.
- Нельзя начать Run без явной кнопки.
- Overlay не уничтожает `system_prompt_default`.
- Retry назад не трёт `audit_events` и старые `task_column_reports`.
- Disabled integration не попадает в tools.
- Смена MCP/integration без рестарта процесса обновляет tool list.
- SSE обновляет доску без reload.
- UI ru/en переключается в settings.
- Секреты не видны в полном виде в API list/get после сохранения.
- Дерево Go совпадает с go-arch-template (`internal/api/{domain,usecase,repository,storage,infrastructure,integration,transport,handlers,app,service,env,validator}`).
- Дерево фронта — FSD-слои; Steiger без forbidden-imports.

---

## 20. Стиль кода

- Бэкенд: пакеты и DI как в go-arch-template. Domain без импорта Fiber/`pgx`/HTTP-клиентов. Usecase зависит от интерфейсов repository/integration. `internal/api/app` — только wiring.
- Интерфейсы портов: `LLMClient`, `Tool`, `JobQueue`/job repository, Git, MCP registry — в integration/repository, не в handlers.
- Фронт: FSD public API слайса, Steiger clean. Не складывать всё в `shared`. Не импортировать `pages` из `features`.
- Никакого god-object с бизнес-логикой.
- Комментарии только там, где неочевидный инвариант.
- Не коммитить `.env` с живыми ключами.
- Не добавлять markdown-файлы «для галочки» сверх README и `.context`.
- Не тащить из шаблона Mongo, Kafka, S3, Nexus, billing, OAuth, CDC, gRPC «на вырост».
---
