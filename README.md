<p align="center">
  <img src="docs/assets/banner.svg" alt="Kaiban" width="100%"/>
</p>

<p align="center">
  <strong>Open-source Kanban where every column is an AI agent</strong><br/>
  Local-first board for product → analytics → development → QA, with human approve gates and live agent logs.
</p>

<p align="center">
  <a href="https://github.com/gonnafaraway/kaiban/stargazers"><img src="https://img.shields.io/github/stars/gonnafaraway/kaiban?style=flat&color=0F6CBD" alt="Stars"/></a>
  <a href="LICENSE"><img src="https://img.shields.io/badge/license-Apache%202.0-blue.svg" alt="License"/></a>
  <a href="go.mod"><img src="https://img.shields.io/badge/Go-1.27-00ADD8?logo=go&logoColor=white" alt="Go"/></a>
  <a href="frontend/package.json"><img src="https://img.shields.io/badge/Next.js-15-black?logo=nextdotjs&logoColor=white" alt="Next.js"/></a>
  <a href="docker-compose.yml"><img src="https://img.shields.io/badge/Docker-Compose-2496ED?logo=docker&logoColor=white" alt="Docker"/></a>
  <img src="https://img.shields.io/badge/UI-RU%20%7C%20EN-479EF5" alt="i18n"/>
</p>

<p align="center">
  <a href="#-quick-start">Quick Start</a> ·
  <a href="#-why-kaiban">Why Kaiban</a> ·
  <a href="#-how-it-works">How it works</a> ·
  <a href="#-features">Features</a> ·
  <a href="#-architecture">Architecture</a> ·
  <a href="#-configuration">Configuration</a> ·
  <a href="#-contributing">Contributing</a> ·
  <a href="README.ru.md">Русский</a>
</p>

<p align="center">
  <img src="docs/assets/hero.png" alt="Kaiban board preview" width="920"/>
</p>

---

## 🌟 What is Kaiban?

**Kaiban** is a local Kanban system for shipping software with LLM agents. Each column is a **role** (Product, Business Analyst, System Analyst, Developer, QA, Auto QA). You create a task, run the agent in the current column, review the markdown report, then **approve** to the next stage — or send it back with context.

It is built for a single developer on their machine: one `docker compose up`, your LLM of choice (any OpenAI-compatible API), optional Jira / Confluence / GitLab / GitHub / MCP tools, and a git branch per task.

> Not a SaaS. Not multi-tenant. Your board, your keys, your repo.

---

## 📖 Table of contents

- [✨ Why Kaiban](#-why-kaiban)
- [🚀 Quick Start](#-quick-start)
- [🧩 How it works](#-how-it-works)
- [🔥 Features](#-features)
- [🏗 Architecture](#-architecture)
- [⚙️ Configuration](#-configuration)
- [🗺 Roadmap ideas](#-roadmap-ideas)
- [🤝 Contributing](#-contributing)
- [📄 License](#-license)

---

## ✨ Why Kaiban

| Pain | What Kaiban does |
| --- | --- |
| Agents scattered in chat threads | One board, one task card, full audit trail |
| No human gate between roles | **Approve** is mandatory to move forward |
| Opaque agent runs | Live SSE log + markdown report per column |
| Hard to wire tools | Native Jira / Confluence / GitLab / GitHub + remote MCP |
| Cloud lock-in | Runs locally via Docker Compose |

**Key ideas:**

- **Columns = roles** with editable system prompts and overlays
- **Human-in-the-loop** forward motion (or **Autoplay** when you want hands-off)
- **Git-aware** tasks (GitLab or GitHub repo per card, or a global URL in settings; branch per card)
- **RU / EN** UI from day one

---

## 🚀 Quick Start

### Prerequisites

- [Docker](https://docs.docker.com/get-docker/) + Docker Compose
- An OpenAI-compatible API key (OpenAI, local gateway, Azure OpenAI proxy, etc.)

### 1. Clone and configure

```bash
git clone https://github.com/gonnafaraway/kaiban.git
cd kaiban
cp .env.example .env
```

Edit `.env` and set at least:

```env
OPENAI_API_KEY=sk-...
OPENAI_API_BASE=https://api.openai.com/v1
OPENAI_MODEL=gpt-4.1
```

### 2. Launch

```bash
docker compose up --build
```

| Service | URL |
| --- | --- |
| **UI** | http://localhost:3000 |
| **API health** | http://localhost:8080/health |
| **Postgres** | `localhost:5433` |

### 3. First minutes

1. Open **Settings → LLM** and confirm the model endpoint.
2. Optionally set **Git** repo URL and connect **Integrations** / **MCP**.
3. On the board, create a task → **Run agent** (or toggle **Autoplay**).
4. When status is `succeeded`, approve to the next column.

<details>
<summary><strong>Local development without Docker (API + frontend)</strong></summary>

```bash
# Postgres available (compose postgres maps :5433)
# DATABASE_URL=postgres://kaiban:kaiban@localhost:5433/kaiban?sslmode=disable

go run ./cmd/api

cd frontend
npm install
npm run dev
```

</details>

---

## 🧩 How it works

```mermaid
flowchart LR
  A[Create task] --> B[Product agent]
  B -->|succeeded + Approve| C[Business Analyst]
  C -->|Approve| D[System Analyst]
  D -->|Approve| E[Developer]
  E -->|Approve| F[QA]
  F -->|Approve| G[Auto QA]
  G -->|Approve| H[Done]
  C -.->|Return + comment| B
  E -.->|Return + comment| D
```

1. **Seed columns** appear on first boot (Product → … → Auto QA) with default prompts.
2. Agent runs an LLM tool-loop (integrations + MCP), streams progress over **SSE**, writes a **markdown report**.
3. **Forward** only after `succeeded` (button, drag to next column, or Autoplay).
4. **Backward** anytime to a previous column — comment required.
5. Archive hides a card from the board without deleting history.

---

## 🔥 Features

### Board & agents

- Drag-and-drop Kanban with role columns
- Per-column system prompts + overlay instructions
- Live agent log and run history
- Autoplay: idle → run → succeeded → approve → next column

### Integrations

- **Jira** / **Confluence** / **GitLab** / **GitHub** (token-based)
- Per-task artifact: link a **GitLab** project (MR) or **GitHub** repo (PR)
- After a successful run: push the task branch and open MR/PR when the matching integration is enabled
- Remote **MCP** servers (HTTP/SSE)
- Tools available to every column agent

### Product polish

- Settings hub: Language · LLM · Git · Columns · Integrations · MCP
- Instant RU ↔ EN locale switch
- Archive with full context restore

---

## 🏗 Architecture

| Layer | Stack |
| --- | --- |
| API | Go 1.27, Fiber, Clean Architecture ([go-arch-template](https://github.com/gonnafaraway/go-arch-template)) |
| UI | Next.js 15 App Router, TypeScript, Feature-Sliced Design |
| Data | PostgreSQL 16, in-process job workers (`SKIP LOCKED`) |
| Realtime | Server-Sent Events |
| Spec | OpenAPI under `docs/api/openapi` |

```text
kaiban/
├── cmd/api                 # API entrypoint
├── internal/api            # domain · usecase · repository · handlers
├── frontend/               # Next.js (FSD in src/)
├── migrations/api          # SQL bootstrap + seeds
├── docs/                   # assets + OpenAPI
└── docker-compose.yml      # postgres · api · web
```

---

## ⚙️ Configuration

| Variable | Purpose |
| --- | --- |
| `OPENAI_API_KEY` | LLM credentials |
| `OPENAI_API_BASE` | OpenAI-compatible base URL |
| `OPENAI_MODEL` | Default model id |
| `LLM_HTTP_TIMEOUT` | Long-run agent HTTP timeout (default `120s`) |
| `DATABASE_URL` | Postgres DSN (compose overrides for the `api` service) |
| `GIT_WORK_DIR` | Workspace for git operations inside the API |
| `JIRA_*` / `CONFLUENCE_*` / `GITLAB_*` / `GITHUB_*` | Optional seed for integrations (`GITHUB_URL` defaults to `https://api.github.com`) |

Secrets belong in `.env` — never commit them. See `.env.example`.

---

## 🗺 Roadmap ideas

Not promised, but useful directions:

- Richer git UX (diff preview in the card)
- Credential encryption at rest
- Multi-board / team mode
- MCP stdio transports
- Packaged demo fixtures for first-run wow

Open an issue if you want to champion one of these.

---

## 🤝 Contributing

Contributions of all kinds are welcome — bug reports, docs, UI polish, integrations.

See [CONTRIBUTING.md](CONTRIBUTING.md) for local setup and PR checks.

```bash
make lint
go test ./...
cd frontend && npm run build
```

---

## 📄 License

Kaiban is licensed under the [Apache License 2.0](LICENSE).

---

## ⭐ Stay in the loop

If Kaiban helps your workflow, a star helps others discover it — and keeps the project moving.

<p align="center">
  <a href="https://github.com/gonnafaraway/kaiban">
    <img src="docs/assets/logo.svg" alt="Kaiban logo" width="48" height="48"/>
  </a>
</p>

<p align="center">
  <sub>Built for developers who want agents on a board — not lost in chat history.</sub>
</p>
