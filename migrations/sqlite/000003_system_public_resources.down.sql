DROP INDEX IF EXISTS idx_classification_options_sibling_name;
DROP INDEX IF EXISTS idx_classification_options_tenant_module;
DROP TABLE IF EXISTS classification_options;

DROP INDEX IF EXISTS idx_classification_modules_resource_type;
DROP TABLE IF EXISTS classification_modules;

DROP INDEX IF EXISTS idx_custom_agents_visibility;
DROP INDEX IF EXISTS idx_knowledge_bases_visibility;

ALTER TABLE custom_agents DROP COLUMN visibility;
ALTER TABLE knowledge_bases DROP COLUMN classification_tags;
ALTER TABLE knowledge_bases DROP COLUMN visibility;
