-- Rollback: 000064_client_user_capabilities

DROP INDEX IF EXISTS idx_users_is_client_provisioned;

ALTER TABLE users
    DROP COLUMN IF EXISTS capabilities_overridden_at,
    DROP COLUMN IF EXISTS can_bind_model,
    DROP COLUMN IF EXISTS can_create_agent,
    DROP COLUMN IF EXISTS can_create_knowledge_base,
    DROP COLUMN IF EXISTS is_client_provisioned;
