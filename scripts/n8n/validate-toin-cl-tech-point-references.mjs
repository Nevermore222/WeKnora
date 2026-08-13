import { execFileSync } from "node:child_process";

const sql = `
SELECT json_build_object(
  'table_exists', to_regclass('analyzes.cl_tech_point_case_references') IS NOT NULL,
  'unique_exists', EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conname = 'cl_tech_point_case_references_unique'
  ),
  'foreign_key_exists', EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conname LIKE 'cl_tech_point_case_references_source_tech_point_id_fkey%'
  )
);`;
const output = execFileSync("docker", ["exec", "-e", "PGPASSWORD=postgres", "Xelora-app", "psql", "-v", "ON_ERROR_STOP=1", "-h", "192.168.8.247", "-p", "5432", "-U", "postgres", "-d", "n8n", "-At", "-c", sql], { encoding: "utf8" }).trim();
const result = JSON.parse(output);
for (const [key, value] of Object.entries(result)) {
  if (!value) throw new Error(`validation failed: ${key}`);
}
console.log(JSON.stringify(result));
