#!/usr/bin/env node
import fs from "node:fs";
import path from "node:path";

const migrationPath = path.join(
  process.cwd(),
  "scripts/n8n/migrations/2026-08-11-project-scope-verification-case-key.sql",
);
const sql = fs.readFileSync(migrationPath, "utf8");

for (const marker of [
  "command_id, project_id, case_number",
  "RAISE EXCEPTION",
  "cl_verification_cases_master_proj",
  "uq_cl_verification_cases_master_proj_project_case",
  "idx_verification_cases_master_command_case_number_1",
]) {
  if (!sql.includes(marker)) throw new Error(`Migration is missing: ${marker}`);
}

if (/DELETE\s+FROM\s+analyzes\.cl_verification_cases_master_proj/i.test(sql)) {
  throw new Error("Migration must not delete verification-case data.");
}
if (!/WHERE\s+project_id\s+IS\s+NOT\s+NULL/i.test(sql)) {
  throw new Error("Duplicate preflight must preserve legacy null-project rows.");
}
if (!/CREATE\s+UNIQUE\s+INDEX\s+IF\s+NOT\s+EXISTS/i.test(sql)) {
  throw new Error("Migration must create an idempotent unique index.");
}
if (/^\s*(BEGIN|COMMIT)\s*;/im.test(sql)) {
  throw new Error("Migration must be compatible with auto-commit SQL clients; do not wrap it in BEGIN/COMMIT.");
}
if (!/DROP\s+CONSTRAINT\s+IF\s+EXISTS\s+idx_verification_cases_master_command_case_number_1/i.test(sql)) {
  throw new Error("Migration must remove the legacy global command/case constraint.");
}

console.log("COBOL project-scope migration validation passed.");
console.log(`MIGRATION=${migrationPath}`);
console.log(
  "PREFLIGHT=SELECT command_id, project_id, case_number, count(*) FROM analyzes.cl_verification_cases_master_proj WHERE project_id IS NOT NULL GROUP BY command_id, project_id, case_number HAVING count(*) > 1;",
);
