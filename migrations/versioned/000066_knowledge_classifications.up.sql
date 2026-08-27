CREATE TABLE IF NOT EXISTS classification_modules (
 id VARCHAR(36) PRIMARY KEY, tenant_id BIGINT NOT NULL, name VARCHAR(100) NOT NULL,
 sort_order INT NOT NULL DEFAULT 0, enabled BOOLEAN NOT NULL DEFAULT TRUE,
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 UNIQUE (tenant_id, name)
);
CREATE TABLE IF NOT EXISTS classification_options (
 id VARCHAR(36) PRIMARY KEY, tenant_id BIGINT NOT NULL, module_id VARCHAR(36) NOT NULL REFERENCES classification_modules(id) ON DELETE CASCADE,
 parent_id VARCHAR(36) REFERENCES classification_options(id) ON DELETE CASCADE, name VARCHAR(100) NOT NULL,
 sort_order INT NOT NULL DEFAULT 0, enabled BOOLEAN NOT NULL DEFAULT TRUE,
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 UNIQUE (module_id, parent_id, name)
);
CREATE INDEX IF NOT EXISTS idx_classification_options_tenant_module ON classification_options(tenant_id,module_id);
ALTER TABLE knowledge_bases ADD COLUMN IF NOT EXISTS classification_tags JSONB NOT NULL DEFAULT '{}'::jsonb;
