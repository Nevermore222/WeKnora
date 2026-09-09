-- SQLite: system-wide classifications and public/private resources.

ALTER TABLE knowledge_bases
    ADD COLUMN visibility VARCHAR(20) NOT NULL DEFAULT 'workspace'
    CHECK(visibility IN ('workspace', 'public', 'private'));
ALTER TABLE knowledge_bases
    ADD COLUMN classification_tags TEXT NOT NULL DEFAULT '{}';
ALTER TABLE custom_agents
    ADD COLUMN visibility VARCHAR(20) NOT NULL DEFAULT 'workspace'
    CHECK(visibility IN ('workspace', 'public', 'private'));

CREATE INDEX IF NOT EXISTS idx_knowledge_bases_visibility
    ON knowledge_bases(visibility);
CREATE INDEX IF NOT EXISTS idx_custom_agents_visibility
    ON custom_agents(visibility);

CREATE TABLE IF NOT EXISTS classification_modules (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id INTEGER NOT NULL DEFAULT 0,
    resource_type VARCHAR(50) NOT NULL DEFAULT 'knowledge_base',
    name VARCHAR(100) NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0,
    enabled BOOLEAN NOT NULL DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (tenant_id, resource_type, name)
);
CREATE INDEX IF NOT EXISTS idx_classification_modules_resource_type
    ON classification_modules(tenant_id, resource_type);

CREATE TABLE IF NOT EXISTS classification_options (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id INTEGER NOT NULL DEFAULT 0,
    module_id VARCHAR(36) NOT NULL REFERENCES classification_modules(id) ON DELETE CASCADE,
    resource_type VARCHAR(50) NOT NULL DEFAULT 'knowledge_base',
    parent_id VARCHAR(36) REFERENCES classification_options(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    sort_order INTEGER NOT NULL DEFAULT 0,
    enabled BOOLEAN NOT NULL DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_classification_options_tenant_module
    ON classification_options(tenant_id, module_id, resource_type);
CREATE UNIQUE INDEX IF NOT EXISTS idx_classification_options_sibling_name
    ON classification_options(module_id, IFNULL(parent_id, ''), name);

UPDATE knowledge_bases
   SET visibility = 'public'
 WHERE visibility = 'workspace'
   AND creator_id IN (SELECT id FROM users WHERE is_system_admin = 1);
UPDATE custom_agents
   SET visibility = 'public'
 WHERE visibility = 'workspace'
   AND created_by IN (SELECT id FROM users WHERE is_system_admin = 1);
