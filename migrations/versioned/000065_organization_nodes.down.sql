-- Rollback: 000065_organization_nodes
DROP INDEX IF EXISTS idx_custom_agents_organization_node;
DROP INDEX IF EXISTS idx_knowledge_bases_organization_node;
ALTER TABLE custom_agents DROP COLUMN IF EXISTS organization_node_id;
ALTER TABLE knowledge_bases DROP COLUMN IF EXISTS organization_node_id;
DROP TABLE IF EXISTS organization_nodes;
