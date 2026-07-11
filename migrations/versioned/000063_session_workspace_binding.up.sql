-- Session workspace binding: durable per-conversation default file-output workspace.
-- Stored as JSONB in the sessions table; consumed by the executor gateway to
-- route generated artifacts to the bound workspace root.
DO $$ BEGIN RAISE NOTICE '[Migration 000063] Adding workspace_binding column to sessions...'; END $$;

ALTER TABLE sessions ADD COLUMN IF NOT EXISTS workspace_binding JSONB;

DO $$ BEGIN RAISE NOTICE '[Migration 000063] workspace_binding column ready'; END $$;
