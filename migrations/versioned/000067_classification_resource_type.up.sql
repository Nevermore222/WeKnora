-- Add resource_type to classification_modules and classification_options
-- to allow separate classification trees for knowledge bases and agents

ALTER TABLE classification_modules ADD COLUMN resource_type VARCHAR(50) DEFAULT 'knowledge_base' NOT NULL;
ALTER TABLE classification_options ADD COLUMN resource_type VARCHAR(50) DEFAULT 'knowledge_base' NOT NULL;

-- Update unique constraints to include resource_type
ALTER TABLE classification_modules DROP CONSTRAINT IF EXISTS classification_modules_tenant_id_name_key;
ALTER TABLE classification_modules ADD CONSTRAINT classification_modules_tenant_id_resource_type_name_key
    UNIQUE (tenant_id, resource_type, name);

-- classification_options already has module_id_parent_id_name unique constraint
-- which naturally separates by resource_type through module_id
-- But we add an index on resource_type for query performance
CREATE INDEX idx_classification_modules_resource_type ON classification_modules(resource_type);
CREATE INDEX idx_classification_options_resource_type ON classification_options(resource_type);

-- Migrate existing data: current classifications belong to knowledge_base
-- (resource_type is already set to 'knowledge_base' by default column value)

-- Optional: duplicate existing tree for agents (comment out if not needed)
-- INSERT INTO classification_modules (id, tenant_id, resource_type, name, sort_order, enabled, created_at, updated_at)
-- SELECT gen_random_uuid(), tenant_id, 'agent', name, sort_order, enabled, created_at, updated_at
-- FROM classification_modules WHERE resource_type = 'knowledge_base';
