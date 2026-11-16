CREATE TABLE IF NOT EXISTS mcp_endpoints (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    name VARCHAR(255) NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    enabled BOOLEAN NOT NULL DEFAULT true,
    token_hash VARCHAR(64) NOT NULL DEFAULT '',
    token_hint VARCHAR(16) NOT NULL DEFAULT '',
    knowledge_base_ids JSONB NOT NULL DEFAULT '[]',
    tools JSONB NOT NULL DEFAULT '[]',
    default_agent_id VARCHAR(36) NOT NULL DEFAULT '',
    rate_limit_per_minute INTEGER NOT NULL DEFAULT 60,
    last_used_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);
CREATE INDEX IF NOT EXISTS idx_mcp_endpoints_tenant ON mcp_endpoints (tenant_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_mcp_endpoints_token_hash ON mcp_endpoints (token_hash)
    WHERE token_hash <> '' AND deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_mcp_endpoints_deleted ON mcp_endpoints (deleted_at)
    WHERE deleted_at IS NOT NULL;
