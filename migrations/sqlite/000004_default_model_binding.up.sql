-- SQLite: enforce one system-wide default per model type.
WITH ranked_defaults AS (
    SELECT id,
           ROW_NUMBER() OVER (
               PARTITION BY type
               ORDER BY updated_at DESC, id
           ) AS row_number
      FROM models
     WHERE is_default = 1
       AND deleted_at IS NULL
)
UPDATE models
   SET is_default = 0
 WHERE id IN (
    SELECT id FROM ranked_defaults WHERE row_number > 1
 );

CREATE UNIQUE INDEX IF NOT EXISTS uq_models_global_type_default
    ON models(type)
 WHERE is_default = 1
   AND deleted_at IS NULL;
