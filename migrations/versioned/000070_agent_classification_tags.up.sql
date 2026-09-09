-- Add system-wide classification tags to custom agents.
ALTER TABLE custom_agents
    ADD COLUMN IF NOT EXISTS classification_tags JSONB NOT NULL DEFAULT '{}'::jsonb;
