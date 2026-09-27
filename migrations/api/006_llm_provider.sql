-- LLM provider preset (openai | opencode | custom)

ALTER TABLE app_settings
    ADD COLUMN IF NOT EXISTS llm_provider TEXT NOT NULL DEFAULT 'openai';

UPDATE app_settings
SET llm_provider = 'opencode'
WHERE llm_base_url ILIKE '%opencode.ai/zen%'
  AND llm_provider = 'openai';
