-- Enforce one system-wide administrator-selected default per model type.
-- Keep the most recently updated row if older deployments contain duplicates.
WITH ranked_defaults AS (
    SELECT id,
           ROW_NUMBER() OVER (
               PARTITION BY type
               ORDER BY updated_at DESC, id
           ) AS row_number
      FROM models
     WHERE is_default = TRUE
       AND deleted_at IS NULL
)
UPDATE models AS model
   SET is_default = FALSE
  FROM ranked_defaults AS ranked
 WHERE model.id = ranked.id
   AND ranked.row_number > 1;

CREATE UNIQUE INDEX IF NOT EXISTS uq_models_global_type_default
    ON models (type)
 WHERE is_default = TRUE
   AND deleted_at IS NULL;
