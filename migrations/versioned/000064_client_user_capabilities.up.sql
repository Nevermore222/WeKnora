-- Migration: 000064_client_user_capabilities
-- Per-user capability flags for Personal client users (dual-gate with tenant RBAC).
-- can_* default TRUE so existing non-client users keep current power; provision/
-- register paths overwrite client users to system defaults (false).

DO $$ BEGIN RAISE NOTICE '[Migration 000064] Adding client user capability columns...'; END $$;

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS is_client_provisioned BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS can_create_knowledge_base BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS can_create_agent BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS can_bind_model BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS capabilities_overridden_at TIMESTAMPTZ NULL;

CREATE INDEX IF NOT EXISTS idx_users_is_client_provisioned
    ON users (is_client_provisioned)
    WHERE deleted_at IS NULL;

COMMENT ON COLUMN users.is_client_provisioned IS 'Personal client register / ProvisionClientUser';
COMMENT ON COLUMN users.can_create_knowledge_base IS 'Dual-gate with tenant RBAC';
COMMENT ON COLUMN users.can_create_agent IS 'Dual-gate with tenant RBAC';
COMMENT ON COLUMN users.can_bind_model IS 'Dual-gate with tenant RBAC';
COMMENT ON COLUMN users.capabilities_overridden_at IS 'Set when admin explicitly overrides capabilities';

DO $$ BEGIN RAISE NOTICE '[Migration 000064] Done.'; END $$;
