DO $$
DECLARE
  duplicate_count bigint;
BEGIN
  SELECT count(*)
  INTO duplicate_count
  FROM (
    SELECT command_id, project_id, case_number
    FROM analyzes.cl_verification_cases_master_proj
    WHERE project_id IS NOT NULL
    GROUP BY command_id, project_id, case_number
    HAVING count(*) > 1
  ) duplicates;

  IF duplicate_count > 0 THEN
    RAISE EXCEPTION
      'project-scoped verification-case duplicates exist: % key(s)',
      duplicate_count;
  END IF;
END $$;

DO $$
DECLARE
  constraint_name text;
BEGIN
  -- Older deployments used this fixed global key name. Remove it explicitly
  -- before the generic catalog-based cleanup below.
  ALTER TABLE analyzes.cl_verification_cases_master_proj
    DROP CONSTRAINT IF EXISTS idx_verification_cases_master_command_case_number_1;

  FOR constraint_name IN
    SELECT c.conname
    FROM pg_constraint c
    JOIN pg_class t ON t.oid = c.conrelid
    JOIN pg_namespace n ON n.oid = t.relnamespace
    WHERE n.nspname = 'analyzes'
      AND t.relname = 'cl_verification_cases_master_proj'
      AND c.contype = 'u'
      AND (
        SELECT array_agg(a.attname::text ORDER BY key_column.ordinality)
        FROM unnest(c.conkey) WITH ORDINALITY AS key_column(attnum, ordinality)
        JOIN pg_attribute a
          ON a.attrelid = c.conrelid
         AND a.attnum = key_column.attnum
      ) = ARRAY['command_id', 'case_number']::text[]
  LOOP
    EXECUTE format(
      'ALTER TABLE analyzes.cl_verification_cases_master_proj DROP CONSTRAINT %I',
      constraint_name
    );
  END LOOP;
END $$;

DROP INDEX IF EXISTS analyzes.idx_verification_cases_master_command_case_number_1;

DO $$
DECLARE
  index_record record;
BEGIN
  FOR index_record IN
    SELECT n.nspname AS schemaname, i.relname AS indexname
    FROM pg_index x
    JOIN pg_class i ON i.oid = x.indexrelid
    JOIN pg_class t ON t.oid = x.indrelid
    JOIN pg_namespace n ON n.oid = t.relnamespace
    WHERE n.nspname = 'analyzes'
      AND t.relname = 'cl_verification_cases_master_proj'
      AND x.indisunique
      AND x.indnkeyatts = 2
      AND NOT EXISTS (
        SELECT 1 FROM pg_constraint c WHERE c.conindid = x.indexrelid
      )
      AND (
        SELECT array_agg(a.attname::text ORDER BY key_column.ordinality)
        FROM unnest(x.indkey::smallint[]) WITH ORDINALITY AS key_column(attnum, ordinality)
        JOIN pg_attribute a
          ON a.attrelid = x.indrelid
         AND a.attnum = key_column.attnum
        WHERE key_column.ordinality <= x.indnkeyatts
      ) = ARRAY['command_id', 'case_number']::text[]
  LOOP
    EXECUTE format(
      'DROP INDEX IF EXISTS %I.%I',
      index_record.schemaname,
      index_record.indexname
    );
  END LOOP;
END $$;

CREATE UNIQUE INDEX IF NOT EXISTS
  uq_cl_verification_cases_master_proj_project_case
ON analyzes.cl_verification_cases_master_proj
  (command_id, project_id, case_number);
