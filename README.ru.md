# Kaiban

<p align="center">
  <img src="docs/assets/logo.svg" alt="Kaiban" width="64" height="64"/>
</p>

**Локальная Kanban-доска, где каждая колонка — LLM-агент по роли.**

Полная документация (quick start, архитектура, конфигурация) — в [README.md](README.md) на английском.

## Быстрый старт

```bash
cp .env.example .env   # укажите OPENAI_API_KEY
docker compose up --build
```

- UI: http://localhost:3000  
- API: http://localhost:8080/health  

В настройках: язык (RU/EN), LLM, Git, колонки и промпты, Jira / Confluence / GitLab / GitHub, MCP. На доске — **Автопрогон** для автоматического прохождения этапов.

## Идея

Создаёте задачу → агент колонки пишет отчёт → **Approve** двигает вперёд → при необходимости возврат назад с комментарием. Интеграции и MCP-tools доступны всем ролям. Git: GitLab или GitHub на карточке (или общий URL в настройках), ветка на задачу.

Лицензия: [Apache 2.0](LICENSE).
