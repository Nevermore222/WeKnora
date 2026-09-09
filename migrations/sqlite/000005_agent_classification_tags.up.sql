-- Add system-wide classification tags to custom agents.
ALTER TABLE custom_agents
    ADD COLUMN classification_tags TEXT NOT NULL DEFAULT '{}';
