-- Rollback classification resource_type separation

DROP INDEX IF EXISTS idx_classification_options_resource_type;
DROP INDEX IF EXISTS idx_classification_modules_resource_type;

ALTER TABLE classification_modules DROP CONSTRAINT IF EXISTS classification_modules_tenant_id_resource_type_name_key;
ALTER TABLE classification_modules ADD CONSTRAINT classification_modules_tenant_id_name_key
    UNIQUE (tenant_id, name);

ALTER TABLE classification_options DROP COLUMN IF EXISTS resource_type;
ALTER TABLE classification_modules DROP COLUMN IF EXISTS resource_type;
