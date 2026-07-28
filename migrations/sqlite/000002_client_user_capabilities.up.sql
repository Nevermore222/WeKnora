-- SQLite: client user capability columns (mirrors 000064 postgres)

ALTER TABLE users ADD COLUMN is_client_provisioned BOOLEAN NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN can_create_knowledge_base BOOLEAN NOT NULL DEFAULT 1;
ALTER TABLE users ADD COLUMN can_create_agent BOOLEAN NOT NULL DEFAULT 1;
ALTER TABLE users ADD COLUMN can_bind_model BOOLEAN NOT NULL DEFAULT 1;
ALTER TABLE users ADD COLUMN capabilities_overridden_at DATETIME NULL;

CREATE INDEX IF NOT EXISTS idx_users_is_client_provisioned ON users (is_client_provisioned);
