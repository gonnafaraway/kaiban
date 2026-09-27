# Contributing to Kaiban

Thanks for helping make Kaiban better.

## Development setup

```bash
# API + Postgres + UI
docker compose up --build

# or split locally
# Postgres on :5433 (see docker-compose), then:
go run ./cmd/api
cd frontend && npm install && npm run dev
```

Copy `.env.example` → `.env` and set at least `OPENAI_API_KEY` (any OpenAI-compatible endpoint works via `OPENAI_API_BASE` / `OPENAI_MODEL`).

## Checks before a PR

```bash
make lint
go test ./...
cd frontend && npm run build
```

## Guidelines

- **Backend** follows [go-arch-template](https://github.com/gonnafaraway/go-arch-template) layers under `internal/api`.
- **Frontend** follows Feature-Sliced Design under `frontend/src`.
- Prefer small, focused PRs with a clear “why”.
- Commit messages: [Conventional Commits](https://www.conventionalcommits.org/), lowercase subject
  (`feat(api): add …`, `fix: …`, `refactor: …`). No Title Case.
- UI strings go through `frontend/src/shared/i18n` (ru + en).
- Do not commit secrets (`.env`, tokens, API keys).

## Issues

Bug reports and feature ideas are welcome — open an issue with steps to reproduce or a short problem statement.
