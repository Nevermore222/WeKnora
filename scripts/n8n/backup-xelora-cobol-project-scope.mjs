#!/usr/bin/env node
import assert from "node:assert/strict";
import { execFileSync } from "node:child_process";
import crypto from "node:crypto";
import fs from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

export const AGENT_NAMES = [
  "COBOL技术点检证用例生成智能体（基于tech_point）",
  "COBOL检证用例代码生成智能体-SR6",
];

const modulePath = fileURLToPath(import.meta.url);
const repoRoot = path.resolve(path.dirname(modulePath), "..", "..");
const n8nRoot = path.resolve(repoRoot, "..", "n8n");

const workflowPaths = [
  path.join(
    n8nRoot,
    "二维表",
    "Xelora版COBOL技术点检证用例工作流（基于tech_point）_1.0.0.json",
  ),
  path.join(
    n8nRoot,
    "检证用例代码",
    "Xelora版COBOL检证用例代码生成工作流_1.0.0.json",
  ),
];

export function validateAgentRecord(record) {
  if (!record || !AGENT_NAMES.includes(record.name)) {
    throw new Error("Backup contains an unknown or missing required agent.");
  }
  const tenantId = Number(record.tenant_id);
  if (!record.id || !Number.isInteger(tenantId) || tenantId <= 0) {
    throw new Error(`Agent ${record.name} is missing id or tenant_id.`);
  }
  if (
    !record.config ||
    typeof record.config.system_prompt !== "string" ||
    !record.config.system_prompt.trim()
  ) {
    throw new Error(`Agent ${record.name} is missing config.system_prompt.`);
  }
  return record;
}

export function sha256(content) {
  return crypto.createHash("sha256").update(content).digest("hex");
}

export function buildManifest(directory) {
  const files = fs
    .readdirSync(directory, { withFileTypes: true })
    .filter((entry) => entry.isFile() && entry.name !== "manifest.json")
    .map((entry) => entry.name)
    .sort()
    .map((name) => {
      const content = fs.readFileSync(path.join(directory, name));
      return { name, bytes: content.length, sha256: sha256(content) };
    });
  return { version: 1, files };
}

export function writeBackupSet(directory, records, workflows) {
  const validatedRecords = records.map(validateAgentRecord);
  const byName = new Map(validatedRecords.map((record) => [record.name, record]));
  if (validatedRecords.length !== AGENT_NAMES.length || byName.size !== AGENT_NAMES.length) {
    throw new Error("Backup must contain each required agent exactly once.");
  }
  if (workflows.length !== 2) {
    throw new Error("Backup must contain exactly two workflow files.");
  }

  fs.mkdirSync(directory, { recursive: true });
  for (const name of AGENT_NAMES) {
    const record = byName.get(name);
    if (!record) throw new Error(`Missing required agent: ${name}`);
    const safeName = name.includes("代码生成")
      ? "verification-code-agent.json"
      : "verification-case-agent.json";
    const outputPath = path.join(directory, safeName);
    fs.writeFileSync(outputPath, `${JSON.stringify(record, null, 2)}\n`, "utf8");
    assert.deepEqual(JSON.parse(fs.readFileSync(outputPath, "utf8")), record);
  }

  const workflowNames = new Set();
  for (const workflow of workflows) {
    if (!workflow?.name || !workflow.name.endsWith(".json")) {
      throw new Error("Backup workflow must be a named JSON artifact.");
    }
    if (workflowNames.has(workflow.name)) {
      throw new Error(`Duplicate backup workflow name: ${workflow.name}`);
    }
    workflowNames.add(workflow.name);
    const parsed = JSON.parse(workflow.content);
    const outputPath = path.join(directory, workflow.name);
    fs.writeFileSync(outputPath, workflow.content, "utf8");
    assert.deepEqual(JSON.parse(fs.readFileSync(outputPath, "utf8")), parsed);
  }

  const manifest = buildManifest(directory);
  fs.writeFileSync(
    path.join(directory, "manifest.json"),
    `${JSON.stringify(manifest, null, 2)}\n`,
    "utf8",
  );
  return manifest;
}

export function verifyBackupSet(directory) {
  const expected = JSON.parse(
    fs.readFileSync(path.join(directory, "manifest.json"), "utf8"),
  );
  const actual = buildManifest(directory);
  assert.deepEqual(actual, expected);
  if (actual.files.length !== 4) {
    throw new Error(`Backup must contain four payload files; found ${actual.files.length}.`);
  }

  for (const name of ["verification-case-agent.json", "verification-code-agent.json"]) {
    validateAgentRecord(JSON.parse(fs.readFileSync(path.join(directory, name), "utf8")));
  }
  return { ok: true, files: actual.files.length };
}

function runPsql(sql) {
  return execFileSync(
    "docker",
    [
      "exec",
      "-i",
      "Xelora-postgres",
      "psql",
      "-U",
      "postgres",
      "-d",
      "Xelora",
      "-t",
      "-A",
    ],
    { input: sql, encoding: "utf8" },
  ).trim();
}

function sqlLiteral(value) {
  return `'${String(value).replaceAll("'", "''")}'`;
}

function loadAgent(name) {
  const json = runPsql(`
SELECT jsonb_build_object(
  'id', id,
  'tenant_id', tenant_id,
  'name', name,
  'description', description,
  'config', config,
  'created_at', created_at,
  'updated_at', updated_at
)::text
FROM custom_agents
WHERE deleted_at IS NULL
  AND name = ${sqlLiteral(name)}
ORDER BY updated_at DESC
LIMIT 1;
`);
  if (!json) throw new Error(`Online agent not found: ${name}`);
  return validateAgentRecord(JSON.parse(json));
}

function main() {
  const timestamp = new Date()
    .toISOString()
    .replace(/[-:]/g, "")
    .replace(/\.\d{3}Z$/, "Z");
  const backupDir = path.join(
    n8nRoot,
    "backups",
    "xelora-cobol-project-scope",
    timestamp,
  );
  const workflows = workflowPaths.map((file) => ({
    name: path.basename(file),
    content: fs.readFileSync(file, "utf8"),
  }));

  writeBackupSet(backupDir, AGENT_NAMES.map(loadAgent), workflows);
  const result = verifyBackupSet(backupDir);
  console.log(`VERIFIED_BACKUP=${backupDir}`);
  console.log(`BACKUP_FILES=${result.files}`);
}

if (process.argv[1] && path.resolve(process.argv[1]) === modulePath) {
  main();
}
