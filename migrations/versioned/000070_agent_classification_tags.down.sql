-- Remove system-wide classification tags from custom agents.
ALTER TABLE custom_agents
    DROP COLUMN IF EXISTS classification_tags;
