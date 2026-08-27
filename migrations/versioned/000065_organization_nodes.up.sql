-- Migration: 000065_organization_nodes
-- Tenant-local company / department / group hierarchy. Kept separate from
-- organizations, which remains the cross-tenant collaboration-space model.

CREATE TABLE IF NOT EXISTS organization_nodes (
    id         VARCHAR(36) PRIMARY KEY,
    tenant_id  BIGINT NOT NULL,
    parent_id  VARCHAR(36) NULL REFERENCES organization_nodes(id) ON DELETE RESTRICT,
    name       VARCHAR(255) NOT NULL,
    type       VARCHAR(32) NOT NULL,
    depth      INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT organization_nodes_type_check CHECK (type IN ('company', 'department', 'group')),
    CONSTRAINT organization_nodes_depth_check CHECK (depth BETWEEN 0 AND 2)
);

CREATE INDEX IF NOT EXISTS idx_organization_nodes_tenant
    ON organization_nodes (tenant_id);
CREATE INDEX IF NOT EXISTS idx_organization_nodes_parent
    ON organization_nodes (tenant_id, parent_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_organization_nodes_root_company
    ON organization_nodes (tenant_id)
    WHERE type = 'company' AND parent_id IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_organization_nodes_name_parent
    ON organization_nodes (tenant_id, COALESCE(parent_id, ''), lower(name));

ALTER TABLE knowledge_bases
    ADD COLUMN IF NOT EXISTS organization_node_id VARCHAR(36) NULL REFERENCES organization_nodes(id) ON DELETE SET NULL;
ALTER TABLE custom_agents
    ADD COLUMN IF NOT EXISTS organization_node_id VARCHAR(36) NULL REFERENCES organization_nodes(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_knowledge_bases_organization_node
    ON knowledge_bases (tenant_id, organization_node_id);
CREATE INDEX IF NOT EXISTS idx_custom_agents_organization_node
    ON custom_agents (tenant_id, organization_node_id);
