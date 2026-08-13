CREATE TABLE IF NOT EXISTS analyzes.cl_tech_point_case_references (
  id bigserial PRIMARY KEY,
  project_id integer NOT NULL,
  language varchar(20) NOT NULL,
  command_id bigint NULL,
  command_name varchar(256) NOT NULL,
  tech_point_no varchar(256) NOT NULL,
  sub_category varchar(256) NOT NULL,
  source_tech_point_id bigint NOT NULL REFERENCES analyzes.tech_point(id),
  pgmid varchar(256) NULL,
  source_content text NULL,
  variable_declaration text NULL,
  extraction_batch_id uuid NOT NULL,
  created_at timestamptz NOT NULL DEFAULT current_timestamp,
  CONSTRAINT cl_tech_point_case_references_unique
    UNIQUE (project_id, language, command_name, tech_point_no, sub_category, source_tech_point_id)
);

CREATE INDEX IF NOT EXISTS cl_tech_point_case_references_group_idx
  ON analyzes.cl_tech_point_case_references (project_id, language, command_name, tech_point_no, sub_category);

CREATE INDEX IF NOT EXISTS cl_tech_point_case_references_source_idx
  ON analyzes.cl_tech_point_case_references (source_tech_point_id);
