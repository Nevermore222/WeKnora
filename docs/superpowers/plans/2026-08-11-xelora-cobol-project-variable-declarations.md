# Xelora COBOL Project-Scoped Variable Declarations Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Isolate TOIN and Fukuyama COBOL technical points by `project_id`, use each technical point's real `variable_declaration` in verification-case and code generation, and preserve existing TOIN output.

**Architecture:** Add a verified backup gate before online agent updates, migrate verification-case uniqueness to `command_id + project_id + case_number`, and generate new `1.1.0` n8n workflows from the existing source workflows. Both agent prompts receive project-scoped declaration contracts; production n8n continues to parse and persist output without rewriting COBOL semantics.

**Tech Stack:** Node.js ESM scripts, `node:test`, PostgreSQL SQL migration, n8n workflow JSON, Xelora custom-agent JSON config, Docker-hosted PostgreSQL, SSE API smoke tests.

---

## File Map

- Create `scripts/n8n/backup-xelora-cobol-project-scope.mjs`: export and verify both online agent records plus both current `1.0.0` workflow files.
- Create `scripts/n8n/backup-xelora-cobol-project-scope.test.mjs`: unit-test backup validation, manifest hashes, and failure gates without touching the online database.
- Create `scripts/n8n/migrations/2026-08-11-project-scope-verification-case-key.sql`: migrate the verification-case uniqueness key.
- Create `scripts/n8n/validate-xelora-cobol-project-scope-migration.mjs`: statically validate the migration and print target-database preflight SQL.
- Modify `scripts/n8n/upsert-xelora-cobol-tech-point-verification-agent.mjs`: add the per-tech-point variable-declaration authority contract.
- Modify `scripts/n8n/build-xelora-cobol-tech-point-verification-workflow.mjs`: require projects, transport all technical-point fields, and generate the `1.1.0` workflow.
- Modify `scripts/n8n/validate-xelora-cobol-tech-point-verification-workflow.mjs`: validate project isolation, declaration transport, fallback, and the Fukuyama fixtures.
- Modify `scripts/n8n/smoke-test-xelora-cobol-japanese-output.mjs`: exercise multiple technical points with isolated declarations and one missing-declaration fallback.
- Modify `scripts/n8n/upsert-xelora-cobol-verification-code-agent.mjs`: require preservation of real declarations and source statements.
- Modify `scripts/n8n/build-xelora-cobol-verification-code-workflow.mjs`: require `project_id`, query only project rows, attach source context, and generate the `1.1.0` workflow.
- Modify `scripts/n8n/validate-xelora-cobol-verification-code-workflow.mjs`: validate project-only queries, source joins, Fukuyama project naming, and source-context transport.
- Modify `scripts/n8n/smoke-test-xelora-cobol-verification-code-agent.mjs`: verify declarations survive through generated COBOL while retaining line-width and OCCURS checks.
- Create `E:\Xelora\n8n\二维表\Xelora版COBOL技术点检证用例工作流（基于tech_point）_1.1.0.json`: generated verification-case workflow artifact.
- Create `E:\Xelora\n8n\检证用例代码\Xelora版COBOL检证用例代码生成工作流_1.1.0.json`: generated code workflow artifact.
- Create `E:\Xelora\n8n\backups\xelora-cobol-project-scope\<timestamp>\`: runtime backup set; never add its database exports to Git.

## Preconditions

- Work only in `E:\Xelora\WeKnora`; preserve unrelated dirty-worktree changes.
- Stage files by explicit path. Never use `git add -A` or `git add .`.
- Do not update either online agent until Task 1 has produced a verified backup set.
- Do not regenerate TOIN cases during this implementation.
- Do not import or publish either n8n workflow until the migration and all validators pass.
- The local `Xelora-postgres` container stores Xelora agent configuration, but the target `analyzes.cl_verification_cases_master_proj` table is not present there. Applying Task 8 therefore requires the real `analyzes` PostgreSQL connection used by n8n.

### Task 1: Add The Agent And Workflow Backup Gate

**Files:**
- Create: `scripts/n8n/backup-xelora-cobol-project-scope.mjs`
- Create: `scripts/n8n/backup-xelora-cobol-project-scope.test.mjs`
- Runtime output: `E:\Xelora\n8n\backups\xelora-cobol-project-scope\<timestamp>\`

- [ ] **Step 1: Write the failing backup utility tests**

Create tests using only temporary directories and in-memory agent rows:

```js
import assert from "node:assert/strict";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import test from "node:test";
import {
  AGENT_NAMES,
  buildManifest,
  validateAgentRecord,
  verifyBackupSet,
  writeBackupSet,
} from "./backup-xelora-cobol-project-scope.mjs";

function agent(name, id) {
  return {
    id,
    tenant_id: 1,
    name,
    description: `${name} description`,
    config: { system_prompt: `${name} prompt`, knowledge_bases: ["kb-1"] },
    created_at: "2026-08-01T00:00:00Z",
    updated_at: "2026-08-11T00:00:00Z",
  };
}

test("writes two agents, two workflows, and a verifiable manifest", () => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "xelora-backup-"));
  const workflows = [
    { name: "verification-case-1.0.0.json", content: "{\"name\":\"case\"}" },
    { name: "verification-code-1.0.0.json", content: "{\"name\":\"code\"}" },
  ];
  const records = AGENT_NAMES.map((name, index) => agent(name, `agent-${index + 1}`));

  writeBackupSet(root, records, workflows);

  assert.equal(verifyBackupSet(root).ok, true);
  assert.equal(buildManifest(root).files.length, 4);
});

test("rejects an empty agent config", () => {
  assert.throws(
    () => validateAgentRecord({ ...agent(AGENT_NAMES[0], "agent-1"), config: {} }),
    /system_prompt/,
  );
});

test("rejects a missing required agent", () => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "xelora-backup-"));
  assert.throws(
    () => writeBackupSet(root, [agent(AGENT_NAMES[0], "agent-1")], []),
    /required agent/,
  );
});
```

- [ ] **Step 2: Run the tests and verify they fail**

Run:

```powershell
node --test scripts/n8n/backup-xelora-cobol-project-scope.test.mjs
```

Expected: FAIL with `ERR_MODULE_NOT_FOUND` for `backup-xelora-cobol-project-scope.mjs`.

- [ ] **Step 3: Implement the backup utility core**

Implement these exported constants and functions before adding the database CLI:

```js
export const AGENT_NAMES = [
  "COBOL技术点检证用例生成智能体（基于tech_point）",
  "COBOL检证用例代码生成智能体-SR6",
];

export function validateAgentRecord(record) {
  if (!record || !AGENT_NAMES.includes(record.name)) {
    throw new Error("Backup contains an unknown or missing required agent.");
  }
  if (!record.id || !Number.isInteger(Number(record.tenant_id))) {
    throw new Error(`Agent ${record.name} is missing id or tenant_id.`);
  }
  if (!record.config || typeof record.config.system_prompt !== "string" || !record.config.system_prompt.trim()) {
    throw new Error(`Agent ${record.name} is missing config.system_prompt.`);
  }
  return record;
}

export function sha256(content) {
  return crypto.createHash("sha256").update(content).digest("hex");
}

export function buildManifest(directory) {
  const files = fs.readdirSync(directory)
    .filter((name) => name !== "manifest.json")
    .sort()
    .map((name) => {
      const content = fs.readFileSync(path.join(directory, name));
      return { name, bytes: content.length, sha256: sha256(content) };
    });
  return { version: 1, files };
}

export function writeBackupSet(directory, records, workflows) {
  fs.mkdirSync(directory, { recursive: true });
  const byName = new Map(records.map((record) => [record.name, validateAgentRecord(record)]));
  for (const name of AGENT_NAMES) {
    const record = byName.get(name);
    if (!record) throw new Error(`Missing required agent: ${name}`);
    const safeName = name.includes("代码生成") ? "verification-code-agent.json" : "verification-case-agent.json";
    fs.writeFileSync(path.join(directory, safeName), `${JSON.stringify(record, null, 2)}\n`, "utf8");
  }
  for (const workflow of workflows) {
    JSON.parse(workflow.content);
    fs.writeFileSync(path.join(directory, workflow.name), workflow.content, "utf8");
  }
  const manifest = buildManifest(directory);
  fs.writeFileSync(path.join(directory, "manifest.json"), `${JSON.stringify(manifest, null, 2)}\n`, "utf8");
  return manifest;
}

export function verifyBackupSet(directory) {
  const expected = JSON.parse(fs.readFileSync(path.join(directory, "manifest.json"), "utf8"));
  const actual = buildManifest(directory);
  assert.deepEqual(actual, expected);
  return { ok: true, files: actual.files.length };
}
```

- [ ] **Step 4: Add the database-backed CLI entrypoint**

Use `docker exec Xelora-postgres psql` to export full JSON rows, copy the two exact `1.0.0` workflow files, write the manifest, and verify it before printing success:

```js
function runPsql(sql) {
  return execFileSync(
    "docker",
    ["exec", "-i", "Xelora-postgres", "psql", "-U", "postgres", "-d", "Xelora", "-t", "-A"],
    { input: sql, encoding: "utf8" },
  ).trim();
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
WHERE deleted_at IS NULL AND name = '${name.replaceAll("'", "''")}'
ORDER BY updated_at DESC
LIMIT 1;
`);
  if (!json) throw new Error(`Online agent not found: ${name}`);
  return validateAgentRecord(JSON.parse(json));
}

function main() {
  const timestamp = new Date().toISOString().replace(/[-:]/g, "").replace(/\.\d{3}Z$/, "Z");
  const backupDir = path.resolve(process.cwd(), "..", "n8n", "backups", "xelora-cobol-project-scope", timestamp);
  const workflows = [
    path.resolve(process.cwd(), "..", "n8n", "二维表", "Xelora版COBOL技术点检证用例工作流（基于tech_point）_1.0.0.json"),
    path.resolve(process.cwd(), "..", "n8n", "检证用例代码", "Xelora版COBOL检证用例代码生成工作流_1.0.0.json"),
  ].map((file) => ({ name: path.basename(file), content: fs.readFileSync(file, "utf8") }));
  writeBackupSet(backupDir, AGENT_NAMES.map(loadAgent), workflows);
  const result = verifyBackupSet(backupDir);
  console.log(`VERIFIED_BACKUP=${backupDir}`);
  console.log(`BACKUP_FILES=${result.files}`);
}
```

Guard `main()` with an ESM main-module check so unit tests can import the file without querying Docker.

- [ ] **Step 5: Run the backup tests**

Run:

```powershell
node --test scripts/n8n/backup-xelora-cobol-project-scope.test.mjs
```

Expected: 3 tests pass and no Docker command runs.

- [ ] **Step 6: Commit the backup utility**

```powershell
git add -- scripts/n8n/backup-xelora-cobol-project-scope.mjs scripts/n8n/backup-xelora-cobol-project-scope.test.mjs
git commit -m "feat: add COBOL agent backup gate"
```

### Task 2: Add The Project-Scoped Database Migration

**Files:**
- Create: `scripts/n8n/migrations/2026-08-11-project-scope-verification-case-key.sql`
- Create: `scripts/n8n/validate-xelora-cobol-project-scope-migration.mjs`

- [ ] **Step 1: Write the migration validator first**

The validator must fail unless the SQL contains all required safety and identity clauses:

```js
import fs from "node:fs";
import path from "node:path";

const migrationPath = path.join(
  process.cwd(),
  "scripts/n8n/migrations/2026-08-11-project-scope-verification-case-key.sql",
);
const sql = fs.readFileSync(migrationPath, "utf8");

for (const marker of [
  "BEGIN;",
  "command_id, project_id, case_number",
  "RAISE EXCEPTION",
  "cl_verification_cases_master_proj",
  "COMMIT;",
]) {
  if (!sql.includes(marker)) throw new Error(`Migration is missing: ${marker}`);
}
if (/DELETE\s+FROM\s+analyzes\.cl_verification_cases_master_proj/i.test(sql)) {
  throw new Error("Migration must not delete verification-case data.");
}

console.log("COBOL project-scope migration validation passed.");
```

- [ ] **Step 2: Run the validator and verify it fails**

Run:

```powershell
node scripts/n8n/validate-xelora-cobol-project-scope-migration.mjs
```

Expected: FAIL because the migration file does not exist.

- [ ] **Step 3: Write the idempotent migration**

Create a transaction that detects duplicates, removes only the legacy two-column unique rules, and creates the three-column unique index:

```sql
BEGIN;

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
    EXECUTE format('DROP INDEX IF EXISTS %I.%I', index_record.schemaname, index_record.indexname);
  END LOOP;
END $$;

CREATE UNIQUE INDEX IF NOT EXISTS
  uq_cl_verification_cases_master_proj_project_case
ON analyzes.cl_verification_cases_master_proj
  (command_id, project_id, case_number);

COMMIT;
```

- [ ] **Step 4: Run the migration validator**

Run:

```powershell
node scripts/n8n/validate-xelora-cobol-project-scope-migration.mjs
```

Expected: `COBOL project-scope migration validation passed.`

- [ ] **Step 5: Commit the migration artifacts**

```powershell
git add -- scripts/n8n/migrations/2026-08-11-project-scope-verification-case-key.sql scripts/n8n/validate-xelora-cobol-project-scope-migration.mjs
git commit -m "feat: scope COBOL verification cases by project"
```

### Task 3: Strengthen The Verification-Case Agent Contract

**Files:**
- Modify: `scripts/n8n/upsert-xelora-cobol-tech-point-verification-agent.mjs`
- Modify: `scripts/n8n/validate-xelora-cobol-tech-point-verification-workflow.mjs`

- [ ] **Step 1: Add failing marker assertions to the validator**

Add these required agent markers:

```js
for (const marker of [
  "PROJECT_SCOPED_TECH_POINT_INPUT",
  "VARIABLE_DECLARATION_AUTHORITY",
  "NO_CROSS_TECH_POINT_DECLARATION_REUSE",
  "MISSING_VARIABLE_DECLARATION_FALLBACK",
]) {
  if (!agentScript.includes(marker)) {
    throw new Error(`Agent script is missing project-scope marker: ${marker}`);
  }
}
```

- [ ] **Step 2: Run the validator and verify the new assertions fail**

Run:

```powershell
node scripts/n8n/validate-xelora-cobol-tech-point-verification-workflow.mjs
```

Expected: FAIL with `missing project-scope marker`.

- [ ] **Step 3: Append the authoritative declaration contract**

Append this higher-priority block to `strictTableRules`:

```text
PROJECT_SCOPED_TECH_POINT_INPUT:
- Each input block belongs to exactly one project-scoped technical point.
- Keep 技術点NO, 小分類, SOURCE_CONTENT, and 変数宣言 associated with that block.

VARIABLE_DECLARATION_AUTHORITY:
- When 変数宣言 is non-empty, it is authoritative for data names, level numbers, PIC, USAGE, sign, scale, group hierarchy, and subordinate items.
- Copy the applicable declarations into パラメータ設定 with the tested SOURCE_CONTENT statement.
- Do not replace supplied declarations with inferred declarations.

NO_CROSS_TECH_POINT_DECLARATION_REUSE:
- Never borrow, merge, or copy a declaration from another 技術点NO in the same batch.
- Similar names or identical 小分類 values do not permit declaration reuse.
- Every output row must trace to exactly one input 技術点NO.

MISSING_VARIABLE_DECLARATION_FALLBACK:
- Only when the current block has no 変数宣言, infer executable declarations from that block's 小分類 and SOURCE_CONTENT.
- Mark inferred declarations in 備考 with PIC桁数は検証設計上の仮定.
```

- [ ] **Step 4: Run the validator again**

Run:

```powershell
node scripts/n8n/validate-xelora-cobol-tech-point-verification-workflow.mjs
```

Expected: the new agent-marker assertions pass; later `1.1.0` workflow assertions may still fail until Task 4.

- [ ] **Step 5: Commit the prompt-source change**

```powershell
git add -- scripts/n8n/upsert-xelora-cobol-tech-point-verification-agent.mjs scripts/n8n/validate-xelora-cobol-tech-point-verification-workflow.mjs
git commit -m "feat: bind COBOL declarations to technical points"
```

Do not run the upsert script in this task. The verified backup must exist first.

### Task 4: Generate The Project-Aware Verification-Case Workflow

**Files:**
- Modify: `scripts/n8n/build-xelora-cobol-tech-point-verification-workflow.mjs`
- Modify: `scripts/n8n/validate-xelora-cobol-tech-point-verification-workflow.mjs`
- Create: `E:\Xelora\n8n\二维表\Xelora版COBOL技术点检证用例工作流（基于tech_point）_1.1.0.json`

- [ ] **Step 1: Point the validator at `1.1.0` and add failing workflow assertions**

Change `workflowPath` to the `1.1.0` filename and assert:

```js
const queryNode = workflow.nodes.find((node) => node.name === "按项目查询tech_point行");
assert.ok(queryNode, "Missing project-scoped tech-point query node");
const query = queryNode.parameters?.query || "";
assert.match(query, /t\.project_id\s*=\s*\{\{\s*\$json\.project_id\s*\}\}/);
assert.match(query, /variable_declaration/);
assert.doesNotMatch(query, /DISTINCT\s+ON\s*\(sub_category\)/i);

const serialized = JSON.stringify(workflow);
for (const marker of ["【技術点NO】", "【小分類】", "【変数宣言】", "【SOURCE_CONTENT】"]) {
  assert.ok(serialized.includes(marker), `Missing per-tech-point marker: ${marker}`);
}
assert.ok(serialized.includes("ON CONFLICT (command_id, project_id, case_number)"));
assert.ok(!serialized.includes("ON CONFLICT (command_id, case_number)"));
```

Import `node:assert/strict` at the top of the validator.

- [ ] **Step 2: Run the validator and verify it fails because `1.1.0` does not exist**

Run:

```powershell
node scripts/n8n/validate-xelora-cobol-tech-point-verification-workflow.mjs
```

Expected: FAIL with `ENOENT` for the `1.1.0` workflow.

- [ ] **Step 3: Change the builder output and workflow name to `1.1.0`**

Replace both version literals:

```js
const outputWorkflowName = "Xelora版COBOL技术点检证用例工作流（基于tech_point）_1.1.0.json";
workflow.name = "Xelora版COBOL技术点检证用例工作流（基于tech_point）_1.1.0";
```

- [ ] **Step 4: Insert explicit project validation before the tech-point query**

Change the parsed `project_id` assignment to number type, then add a pure validation node and an always-one-row project lookup:

```js
const webhookParamsNode = requireNode(workflow, "解析Webhook参数");
const projectAssignment = webhookParamsNode.parameters.assignments.assignments.find(
  (entry) => entry.name === "project_id",
);
projectAssignment.value = "={{ $json.body?.project_id ?? $json.project_id }}";
projectAssignment.type = "number";

const validateProjectNode = makeCodeNode(
  "validate-project-id-v110",
  "校验project_id",
  [460, 280],
  `const item = $input.first()?.json || {};
const projectId = Number(item.project_id);
if (!Number.isInteger(projectId) || projectId <= 0) {
  throw new Error('project_id 为必填正整数');
}
return [{ json: { ...item, project_id: projectId } }];`,
  "缺失或非法 project_id 时，在任何查询或删除前失败",
);

const sourceQueryNode = requireNode(workflow, "按小分類查询tech_point代表行");
const lookupProjectNode = {
  id: "lookup-project-v110",
  name: "查询并确认项目",
  type: sourceQueryNode.type,
  typeVersion: sourceQueryNode.typeVersion,
  position: [680, 280],
  credentials: sourceQueryNode.credentials,
  parameters: {
    operation: "executeQuery",
    query: `SELECT requested.project_id AS requested_project_id, p.project_id, p.project_name
FROM (SELECT {{ $json.project_id }}::int AS project_id) requested
LEFT JOIN analyzes.projects p ON p.project_id = requested.project_id;`,
    options: {},
  },
};

const assertProjectNode = makeCodeNode(
  "assert-project-v110",
  "确认项目存在",
  [900, 280],
  `const row = $input.first()?.json || {};
if (!row.project_id) throw new Error('project_id 对应项目不存在: ' + row.requested_project_id);
return [{ json: { project_id: Number(row.project_id), project_name: row.project_name || '' } }];`,
  "项目不存在时停止，不进入 tech_point 查询",
);
```

Push the nodes and rewire:

```js
workflow.nodes.push(validateProjectNode, lookupProjectNode, assertProjectNode);
workflow.connections["解析Webhook参数"] = { main: [[{ node: "校验project_id", type: "main", index: 0 }]] };
workflow.connections["校验project_id"] = { main: [[{ node: "查询并确认项目", type: "main", index: 0 }]] };
workflow.connections["查询并确认项目"] = { main: [[{ node: "确认项目存在", type: "main", index: 0 }]] };
workflow.connections["确认项目存在"] = { main: [[{ node: "按项目查询tech_point行", type: "main", index: 0 }]] };
```

- [ ] **Step 5: Replace the representative-row SQL with the full project query**

Rename the node and set this query:

```sql
SELECT
  t.id,
  t.command_id,
  t.pgmid,
  t.language,
  t.sub_category,
  t.tech_point_no,
  t.source_content,
  t.variable_declaration,
  t.command_code,
  t.command_name,
  t.location_count,
  t.project_id
FROM analyzes.tech_point t
WHERE t.sub_category IS NOT NULL
  AND TRIM(t.sub_category) <> ''
  AND t.tech_point_no IS NOT NULL
  AND TRIM(t.tech_point_no) <> ''
  AND t.command_id = {{ $('解析Webhook参数').first().json.command_id }}
  AND t.project_id = {{ $json.project_id }}
ORDER BY t.sub_category, t.tech_point_no, t.id;
```

Update every node reference from `按小分類查询tech_point代表行` to `按项目查询tech_point行`.

- [ ] **Step 6: Serialize the complete per-tech-point input block**

Replace the source block builder with:

```js
const sourceParts = [];
for (const row of allRows) {
  const techPointNo = String(row.tech_point_no || "").trim();
  const subCategory = String(row.sub_category || "").trim();
  const sourceContent = String(row.source_content || "").trim();
  const variableDeclaration = String(row.variable_declaration || "").trim();
  if (!techPointNo) throw new Error("批次包含缺少 tech_point_no 的技术点");
  const lines = [
    `【技術点NO】${techPointNo}`,
    `【小分類】${subCategory}`,
  ];
  if (variableDeclaration) lines.push(`【変数宣言】${variableDeclaration}`);
  lines.push(`【SOURCE_CONTENT】${sourceContent}`);
  sourceParts.push(lines.join("\n"));
}
const sourceContentFormatted = sourceParts.join("\n\n");
```

Keep `tech_point_no`, `sub_category`, `source_content`, and `variable_declaration` inside `batch_rows` for later traceability.

- [ ] **Step 7: Make deletion, numbering, and upsert unconditionally project-scoped**

Require an integer project ID in the delete and max-number code nodes and emit SQL with both keys:

```js
const projectId = Number(deleteData.json.project_id);
if (!Number.isInteger(projectId) || projectId <= 0) throw new Error("project_id 无效");
const deleteSql = `DELETE FROM analyzes.cl_verification_cases_master_proj WHERE command_id = ${commandId} AND project_id = ${projectId};`;
```

Use the same validation in the maximum-number query and end it with:

```sql
WHERE command_id = ${commandId} AND project_id = ${projectId};
```

Change the insert clause to:

```sql
ON CONFLICT (command_id, project_id, case_number) DO UPDATE SET
  sub_category = EXCLUDED.sub_category,
  tech_point_no = EXCLUDED.tech_point_no,
  test_category = EXCLUDED.test_category,
  test_purpose = EXCLUDED.test_purpose,
  parameter_config = EXCLUDED.parameter_config,
  input_data = EXCLUDED.input_data,
  expected_result = EXCLUDED.expected_result,
  priority = EXCLUDED.priority,
  notes = EXCLUDED.notes,
  updated_at = CURRENT_TIMESTAMP;
```

Do not update `project_id` inside the conflict action.

Replace the current `sub_category -> tech_point_no` fallback in `准备本批检证用例接续编号`. The parsed row must carry its own technical-point number, and that exact number must exist in the current project query:

```js
const sourceRows = $("按项目查询tech_point行").all().map((item) => item.json || {});
const sourceByTechPoint = new Map();
for (const sourceRow of sourceRows) {
  const techPointNo = String(sourceRow.tech_point_no || "").trim();
  if (!techPointNo) throw new Error("项目技术点缺少 tech_point_no");
  if (sourceByTechPoint.has(techPointNo)) {
    throw new Error(`项目内 tech_point_no 重复: ${techPointNo}`);
  }
  sourceByTechPoint.set(techPointNo, sourceRow);
}

const rows = sorted.map((tc, idx) => {
  const techPointNo = String(tc.json.tech_point_no || "").trim().substring(0, 128);
  if (!techPointNo) throw new Error("智能体输出行缺少 tech_point_no");
  const sourceRow = sourceByTechPoint.get(techPointNo);
  if (!sourceRow) throw new Error(`智能体输出了当前项目批次以外的 tech_point_no: ${techPointNo}`);
  const outputSubCategory = String(tc.json.sub_category || "").trim();
  const sourceSubCategory = String(sourceRow.sub_category || "").trim();
  if (outputSubCategory !== sourceSubCategory) {
    throw new Error(`tech_point_no 与 sub_category 不匹配: ${techPointNo}`);
  }
  return {
    command_id: commandMasterId,
    project_id: projectId,
    case_number: padCaseNum(maxSeq + idx + 1),
    sub_category: outputSubCategory.substring(0, 256),
    tech_point_no: techPointNo,
    test_category: String(tc.json.test_category || "").substring(0, 100),
    test_purpose: tc.json.test_purpose || "",
    parameter_config: tc.json.parameter_config || "",
    input_data: tc.json.input_data || "",
    expected_result: tc.json.expected_result || "",
    priority: String(tc.json.priority || "P1").substring(0, 50),
    notes: tc.json.notes || "",
  };
});
```

This removes the remaining path that could bind two technical points sharing one `sub_category` to the last declaration seen for that category.

- [ ] **Step 8: Build and validate the new workflow**

Run:

```powershell
node scripts/n8n/build-xelora-cobol-tech-point-verification-workflow.mjs
node scripts/n8n/validate-xelora-cobol-tech-point-verification-workflow.mjs
```

Expected:

```text
Wrote E:\Xelora\n8n\二维表\Xelora版COBOL技术点检证用例工作流（基于tech_point）_1.1.0.json
COBOL tech-point code-generation contract validation passed.
```

- [ ] **Step 9: Commit the verification-workflow source changes**

```powershell
git add -- scripts/n8n/build-xelora-cobol-tech-point-verification-workflow.mjs scripts/n8n/validate-xelora-cobol-tech-point-verification-workflow.mjs
git commit -m "feat: isolate COBOL tech points by project"
```

The generated JSON is outside the Git repository; retain it as a workspace artifact and in the backup directory.

### Task 5: Strengthen The COBOL Code Agent Contract

**Files:**
- Modify: `scripts/n8n/upsert-xelora-cobol-verification-code-agent.mjs`
- Modify: `scripts/n8n/validate-xelora-cobol-verification-code-workflow.mjs`

- [ ] **Step 1: Add failing code-agent marker assertions**

Require these markers in the code-agent script:

```js
for (const marker of [
  "PROJECT_SCOPED_SOURCE_CONTEXT",
  "REAL_DECLARATION_PRESERVATION",
  "NO_CROSS_TECH_POINT_SOURCE_REUSE",
]) {
  if (!agentScript.includes(marker)) throw new Error(`Code agent is missing: ${marker}`);
}
```

- [ ] **Step 2: Run the validator and verify it fails**

Run:

```powershell
node scripts/n8n/validate-xelora-cobol-verification-code-workflow.mjs
```

Expected: FAIL with `Code agent is missing`.

- [ ] **Step 3: Append the source-preservation contract**

Append to `xeloraContract`:

```text
PROJECT_SCOPED_SOURCE_CONTEXT:
- TECH_POINT_CONTEXT contains project-scoped blocks keyed by 技術点NO.
- Match each verification-case row only to the block with the same 技術点NO.

REAL_DECLARATION_PRESERVATION:
- When a block contains 変数宣言, preserve its data names, levels, PIC clauses, USAGE, sign, scale, group hierarchy, and subordinate items in generated COBOL.
- Preserve the associated SOURCE_CONTENT statement semantics.
- Do not replace real declarations with generic working-storage declarations.

NO_CROSS_TECH_POINT_SOURCE_REUSE:
- Never use one block's 変数宣言 or SOURCE_CONTENT to implement another 技術点NO.
- parameter_config remains the immediate case specification; TECH_POINT_CONTEXT is the factual source used to confirm it.
```

- [ ] **Step 4: Run the validator again**

Run:

```powershell
node scripts/n8n/validate-xelora-cobol-verification-code-workflow.mjs
```

Expected: the new prompt-marker assertions pass; `1.1.0` workflow assertions may still fail until Task 6.

- [ ] **Step 5: Commit the code-agent prompt source**

```powershell
git add -- scripts/n8n/upsert-xelora-cobol-verification-code-agent.mjs scripts/n8n/validate-xelora-cobol-verification-code-workflow.mjs
git commit -m "feat: preserve real COBOL declarations in code generation"
```

Do not execute the upsert yet.

### Task 6: Generate The Project-Aware Code Workflow

**Files:**
- Modify: `scripts/n8n/build-xelora-cobol-verification-code-workflow.mjs`
- Modify: `scripts/n8n/validate-xelora-cobol-verification-code-workflow.mjs`
- Create: `E:\Xelora\n8n\检证用例代码\Xelora版COBOL检证用例代码生成工作流_1.1.0.json`

- [ ] **Step 1: Point the validator at `1.1.0` and add failing project assertions**

Require the new output file and assert:

```js
const workflowText = fs.readFileSync(workflowPath, "utf8");
assert.ok(!workflow.nodes.some((node) => node.name === "查询cl_verification_cases_master表"));
assert.ok(!workflow.nodes.some((node) => node.name === "判断project_id是否为空"));

const query = requireNode("查询项目COBOL检证用例").parameters.query;
for (const marker of [
  "v.project_id = {{ $json.project_id }}",
  "t.project_id = v.project_id",
  "t.command_id = v.command_id",
  "t.tech_point_no = v.tech_point_no",
  "t.id AS source_tech_point_id",
  "t.source_content",
  "t.variable_declaration",
]) {
  assert.ok(query.includes(marker), `Missing project query marker: ${marker}`);
}
assert.ok(workflowText.includes("TECH_POINT_CONTEXT"));
assert.ok(workflowText.includes("projectId === 3 ? 'F'"));
```

- [ ] **Step 2: Run the validator and verify `1.1.0` is missing**

Run:

```powershell
node scripts/n8n/validate-xelora-cobol-verification-code-workflow.mjs
```

Expected: FAIL with `ENOENT` for the code workflow `1.1.0` file.

- [ ] **Step 3: Change the builder output and workflow name to `1.1.0`**

```js
const outputPath = path.join(
  workflowDir,
  "Xelora版COBOL检证用例代码生成工作流_1.1.0.json",
);
workflow.name = "Xelora版COBOL检证用例代码生成工作流_1.1.0";
```

- [ ] **Step 4: Remove the unscoped branch and insert mandatory project validation**

Replace the optional branch, redundant project-name query, and old merge code with this complete project gate. Keep `判断是否有数据` and its existing downstream empty-data handling:

```js
const parsedParamsNode = requireNode(workflow, "解析Webhook参数");
const projectAssignment = parsedParamsNode.parameters.assignments.assignments.find(
  (entry) => entry.name === "project_id",
);
projectAssignment.value = "={{ $json.body?.project_id ?? $json.body?.data?.project_id ?? $json.project_id }}";
projectAssignment.type = "number";

const projectCasesNode = renameNode(
  workflow,
  "查询cl_verification_cases_master_proj表",
  "查询项目COBOL检证用例",
);

const removedNodeNames = new Set([
  "判断project_id是否为空",
  "查询项目名称",
  "查询cl_verification_cases_master表",
]);
workflow.nodes = workflow.nodes.filter((node) => !removedNodeNames.has(node.name));
for (const name of removedNodeNames) delete workflow.connections[name];

const validateProjectNode = makeCodeNode(
  "validate-code-project-id-v110",
  "校验project_id",
  [460, 280],
  `const item = $input.first()?.json || {};
const projectId = Number(item.project_id);
if (!Number.isInteger(projectId) || projectId <= 0) {
  throw new Error('project_id 为必填正整数');
}
return [{ json: { ...item, project_id: projectId } }];`,
  "缺失或非法 project_id 时，在任何检证用例查询或文件输出前失败",
);

const lookupProjectNode = {
  id: "lookup-code-project-v110",
  name: "查询并确认项目",
  type: projectCasesNode.type,
  typeVersion: projectCasesNode.typeVersion,
  position: [680, 280],
  credentials: projectCasesNode.credentials,
  parameters: {
    operation: "executeQuery",
    query: `SELECT requested.project_id AS requested_project_id, p.project_id, p.project_name
FROM (SELECT {{ $json.project_id }}::int AS project_id) requested
LEFT JOIN analyzes.projects p ON p.project_id = requested.project_id;`,
    options: {},
  },
};

const assertProjectNode = makeCodeNode(
  "assert-code-project-v110",
  "确认项目存在",
  [900, 280],
  `const row = $input.first()?.json || {};
if (!row.project_id) throw new Error('project_id 对应项目不存在: ' + row.requested_project_id);
return [{ json: { project_id: Number(row.project_id), project_name: row.project_name || '' } }];`,
  "项目不存在时停止，不进入检证用例查询",
);

workflow.nodes.push(validateProjectNode, lookupProjectNode, assertProjectNode);

const mergeNode = requireNode(workflow, "合并查询结果");
mergeNode.parameters.jsCode = `const project = $('确认项目存在').first()?.json || {};
const cases = $('查询项目COBOL检证用例').all();
return cases.map((item) => ({
  json: {
    ...(item.json || {}),
    project_id: Number(project.project_id),
    project_name: project.project_name || '',
  },
}));`;

workflow.connections["解析Webhook参数"] = { main: [[{ node: "校验project_id", type: "main", index: 0 }]] };
workflow.connections["校验project_id"] = { main: [[{ node: "查询并确认项目", type: "main", index: 0 }]] };
workflow.connections["查询并确认项目"] = { main: [[{ node: "确认项目存在", type: "main", index: 0 }]] };
workflow.connections["确认项目存在"] = { main: [[{ node: "查询项目COBOL检证用例", type: "main", index: 0 }]] };
workflow.connections["查询项目COBOL检证用例"] = { main: [[{ node: "合并查询结果", type: "main", index: 0 }]] };
```

The only path into case querying now passes both numeric validation and the project-existence check.

- [ ] **Step 5: Replace the project verification-case SQL**

Rename the project query node and use:

```sql
SELECT
  v.id,
  v.command_id,
  v.project_id,
  v.case_number,
  v.test_category,
  v.test_purpose,
  v.parameter_config,
  v.input_data,
  v.expected_result,
  v.priority,
  v.notes,
  v.created_at,
  v.sub_category,
  v.tech_point_no,
  t.source_tech_point_id,
  t.source_content,
  t.variable_declaration
FROM analyzes.cl_verification_cases_master_proj v
LEFT JOIN LATERAL (
  SELECT
    t.id AS source_tech_point_id,
    t.source_content,
    t.variable_declaration
  FROM analyzes.tech_point t
  WHERE t.project_id = v.project_id
    AND t.command_id = v.command_id
    AND t.tech_point_no = v.tech_point_no
  ORDER BY t.id
  LIMIT 1
) t ON true
WHERE v.command_id = {{ $('解析Webhook参数').first().json.command_id }}
  AND v.project_id = {{ $json.project_id }}
  AND TRIM(COALESCE(v.test_category, '')) IN ('正常系','境界値')
ORDER BY v.case_number ASC;
```

The `source_tech_point_id` value distinguishes an actual joined technical point from a row whose two context columns happen to be empty. Add `"t.id AS source_tech_point_id"` to the validator's required SQL markers.

- [ ] **Step 6: Add grouped `TECH_POINT_CONTEXT` to each agent request**

Build one context block per `tech_point_no` in the current batch:

```js
const contexts = new Map();
for (const row of batchRows) {
  const techPointNo = String(row.tech_point_no || "").trim();
  if (!techPointNo) throw new Error("检证用例缺少 tech_point_no");
  if (row.source_tech_point_id == null) {
    throw new Error(`当前项目找不到检证用例对应的技术点: ${techPointNo}`);
  }
  if (contexts.has(techPointNo)) continue;
  const lines = [
    `【技術点NO】${techPointNo}`,
    `【小分類】${String(row.sub_category || "").trim()}`,
  ];
  const declaration = String(row.variable_declaration || "").trim();
  if (declaration) lines.push(`【変数宣言】${declaration}`);
  lines.push(`【SOURCE_CONTENT】${String(row.source_content || "").trim()}`);
  contexts.set(techPointNo, lines.join("\n"));
}
const techPointContext = [...contexts.values()].join("\n\n");
```

Add it to the request query before the existing test-case table:

```js
const query = [
  `以下は project_id=${projectId} の技術点別原文コンテキストです。`,
  "【TECH_POINT_CONTEXT】",
  techPointContext,
  "【TEST_CASES_TABLE】",
  markdownTable,
  "各用例は同じ技術点NOの変数宣言とSOURCE_CONTENTだけを使用してください。",
].join("\n\n");
```

Store `tech_point_context: techPointContext` in the batch metadata for diagnostics.

- [ ] **Step 7: Correct Fukuyama project naming without changing TOIN**

Replace the old `project_id=2` mapping:

```js
const numericProjectId = Number(projectId);
const projCode = numericProjectId === 1 ? "T" : numericProjectId === 3 ? "F" : `P${numericProjectId}`;
const pgmBaseCode = "B" + projCode + String(commandMasterId).padStart(3, "0");
```

The validator must assert `project_id=1 -> BT...` and `project_id=3 -> BF...` using the organizer Code node through `new Function`.

- [ ] **Step 8: Build and validate the code workflow**

Run:

```powershell
node scripts/n8n/build-xelora-cobol-verification-code-workflow.mjs
node scripts/n8n/validate-xelora-cobol-verification-code-workflow.mjs
```

Expected:

```text
Wrote E:\Xelora\n8n\检证用例代码\Xelora版COBOL检证用例代码生成工作流_1.1.0.json
Xelora COBOL verification-code workflow validation passed.
```

- [ ] **Step 9: Commit the code-workflow source changes**

```powershell
git add -- scripts/n8n/build-xelora-cobol-verification-code-workflow.mjs scripts/n8n/validate-xelora-cobol-verification-code-workflow.mjs
git commit -m "feat: make COBOL code generation project-aware"
```

### Task 7: Add Fukuyama Regression And Live Smoke Coverage

**Files:**
- Modify: `scripts/n8n/validate-xelora-cobol-tech-point-verification-workflow.mjs`
- Modify: `scripts/n8n/validate-xelora-cobol-verification-code-workflow.mjs`
- Modify: `scripts/n8n/smoke-test-xelora-cobol-japanese-output.mjs`
- Modify: `scripts/n8n/smoke-test-xelora-cobol-verification-code-agent.mjs`

- [ ] **Step 1: Add the four supplied Fukuyama fixtures to the verification validator**

Use exact fixture rows:

```js
const fukuyamaFixtures = [
  {
    project_id: 3,
    tech_point_no: "ACEPTV0000214",
    sub_category: "ACCEPT【変数(GROUP型,TIME)】",
    source_content: "ACCEPT W-TIME FROM TIME",
    variable_declaration: "01 W-TIME.",
  },
  {
    project_id: 3,
    tech_point_no: "ACEPTV0000217",
    sub_category: "ACCEPT【変数(GROUP型,SPECIAL)】",
    source_content: "ACCEPT WK-TENCD FROM CSL",
    variable_declaration: "03 WK-TENCD.",
  },
  {
    project_id: 3,
    tech_point_no: "ADD-1L00000030V000003V0000030",
    sub_category: "ADD【定数(All無,数字)TO無変数(9型,変数(9型,ROUND無)】",
    source_content: "ADD 1 , I GIVING J",
    variable_declaration: "03 I PIC 9(03). 03 J PIC 9(03).",
  },
  {
    project_id: 3,
    tech_point_no: "ADD-1L00000030V000017V0000170",
    sub_category: "ADD【定数(All無,数字)TO無変数(COMP-3型,変数(COMP-3型,ROUND無)】",
    source_content: "ADD 1 I1 GIVING I2",
    variable_declaration: "01 I1 PIC 9(03) PACKED-DECIMAL. 01 I2 PIC 9(03) PACKED-DECIMAL.",
  },
];
```

Execute the organizer Code node with all four rows and assert each serialized block contains only its own declaration. Add a fifth row with `variable_declaration: ""` and assert the block contains its other three markers but no declaration copied from the preceding row.

- [ ] **Step 2: Run the verification validator**

Run:

```powershell
node scripts/n8n/validate-xelora-cobol-tech-point-verification-workflow.mjs
```

Expected: all five per-tech-point serialization cases pass.

- [ ] **Step 3: Add project-isolation fixtures to the code-workflow validator**

Execute the code organizer twice with the same `command_id` and `case_number`, once with project 1 and once with project 3. Assert:

```js
assert.match(toinResult.pgm_base_code, /^BT/);
assert.match(fukuyamaResult.pgm_base_code, /^BF/);
assert.notEqual(toinResult.project_id, fukuyamaResult.project_id);
assert.ok(fukuyamaResult.tech_point_context.includes("01 I1 PIC 9(03) PACKED-DECIMAL."));
assert.ok(!fukuyamaResult.tech_point_context.includes("03 I PIC 9(03)."));
```

- [ ] **Step 4: Replace the verification-agent live smoke query with mixed declaration blocks**

Use at least two supplied Fukuyama blocks plus one empty-declaration block in a single request. Parse the returned Markdown rows and fail when:

- `ACEPTV0000214` does not contain `W-TIME`;
- `ACEPTV0000217` contains `W-TIME`;
- `ADD-1L00000030V000017V0000170` does not contain `PACKED-DECIMAL`;
- any row uses a declaration from another fixture;
- the empty-declaration row fails to generate executable declarations;
- the answer is not Japanese or lacks the exact 10-column header.

- [ ] **Step 5: Extend the code-agent live smoke request with TECH_POINT_CONTEXT**

Add a Fukuyama `ADD` case whose `parameter_config` and context both contain `I1/I2 PIC 9(03) PACKED-DECIMAL`. Retain every existing line-width, comment-column, indentation, level-number, and OCCURS assertion, and add:

```js
checks.realPackedDeclarations =
  /\bI1\s+PIC\s+9\(03\)\s+PACKED-DECIMAL\b/i.test(cobolSource) &&
  /\bI2\s+PIC\s+9\(03\)\s+PACKED-DECIMAL\b/i.test(cobolSource);
checks.noForeignDeclaration = !/\bW-TIME\b/.test(cobolSource);
```

- [ ] **Step 6: Run all offline validators**

Run:

```powershell
node --test scripts/n8n/backup-xelora-cobol-project-scope.test.mjs
node scripts/n8n/validate-xelora-cobol-project-scope-migration.mjs
node scripts/n8n/validate-xelora-cobol-tech-point-verification-workflow.mjs
node scripts/n8n/validate-xelora-cobol-verification-code-workflow.mjs
```

Expected: every command exits 0 with its success message.

- [ ] **Step 7: Commit the regression coverage**

```powershell
git add -- scripts/n8n/validate-xelora-cobol-tech-point-verification-workflow.mjs scripts/n8n/validate-xelora-cobol-verification-code-workflow.mjs scripts/n8n/smoke-test-xelora-cobol-japanese-output.mjs scripts/n8n/smoke-test-xelora-cobol-verification-code-agent.mjs
git commit -m "test: cover Fukuyama COBOL declarations"
```

### Task 8: Back Up, Migrate, Update Agents, And Rebuild Artifacts

**Files:**
- Read: `scripts/n8n/migrations/2026-08-11-project-scope-verification-case-key.sql`
- Runtime output: `E:\Xelora\n8n\backups\xelora-cobol-project-scope\<timestamp>\`
- Runtime output: both `1.1.0` workflow JSON files

- [ ] **Step 1: Create and verify the production backup set**

Run before either upsert:

```powershell
node scripts/n8n/backup-xelora-cobol-project-scope.mjs
```

Expected output includes an absolute `VERIFIED_BACKUP=...` directory and `BACKUP_FILES=4`. Open `manifest.json`, verify all hashes are 64 hex characters, and verify both agent JSON files contain non-empty `config.system_prompt`.

- [ ] **Step 2: Run the target database duplicate preflight**

Execute this read-only query against the real PostgreSQL database used by the n8n `Postgres account` credential:

```sql
SELECT command_id, project_id, case_number, count(*) AS duplicate_count
FROM analyzes.cl_verification_cases_master_proj
WHERE project_id IS NOT NULL
GROUP BY command_id, project_id, case_number
HAVING count(*) > 1
ORDER BY command_id, project_id, case_number;
```

Expected: zero rows. If rows are returned, stop without applying the migration or updating workflows.

- [ ] **Step 3: Apply the migration using the real analyzes database connection**

Run from the operator's authenticated PostgreSQL session:

```powershell
psql $env:ANALYZES_DATABASE_URL -v ON_ERROR_STOP=1 -f scripts/n8n/migrations/2026-08-11-project-scope-verification-case-key.sql
```

Expected: `BEGIN`, three `DO` completions, `CREATE INDEX`, `COMMIT`. Do not substitute the local `Xelora-postgres` container; it does not contain the target table.

- [ ] **Step 4: Verify the migrated identity**

Run against the same database:

```sql
SELECT indexname, indexdef
FROM pg_indexes
WHERE schemaname = 'analyzes'
  AND tablename = 'cl_verification_cases_master_proj'
ORDER BY indexname;
```

Expected: `uq_cl_verification_cases_master_proj_project_case` covers `(command_id, project_id, case_number)`, and no unique index remains on only `(command_id, case_number)`.

- [ ] **Step 5: Update the two online agents after backup verification**

Run:

```powershell
node scripts/n8n/upsert-xelora-cobol-tech-point-verification-agent.mjs
node scripts/n8n/upsert-xelora-cobol-verification-code-agent.mjs
```

Expected: both commands print stable agent IDs, `Manual_ASP`, and `ASP_COBOL` metadata.

- [ ] **Step 6: Rebuild and revalidate both `1.1.0` artifacts**

Run:

```powershell
node scripts/n8n/build-xelora-cobol-tech-point-verification-workflow.mjs
node scripts/n8n/build-xelora-cobol-verification-code-workflow.mjs
node scripts/n8n/validate-xelora-cobol-tech-point-verification-workflow.mjs
node scripts/n8n/validate-xelora-cobol-verification-code-workflow.mjs
```

Expected: both workflow paths end in `_1.1.0.json` and both validators pass.

- [ ] **Step 7: Run the two live SSE smoke tests**

Run:

```powershell
node scripts/n8n/smoke-test-xelora-cobol-japanese-output.mjs
node scripts/n8n/smoke-test-xelora-cobol-verification-code-agent.mjs
```

Expected: verification output uses isolated declarations and Japanese 10-column output; generated COBOL preserves real declarations and passes all existing structural checks.

### Task 9: Verify Cross-Project Safety And Prepare n8n Publication

**Files:**
- Read: `E:\Xelora\n8n\二维表\Xelora版COBOL技术点检证用例工作流（基于tech_point）_1.1.0.json`
- Read: `E:\Xelora\n8n\检证用例代码\Xelora版COBOL检证用例代码生成工作流_1.1.0.json`

- [ ] **Step 1: Verify both generated artifacts are inactive and versioned**

Run:

```powershell
node -e "const fs=require('fs'); for(const p of process.argv.slice(1)){const w=JSON.parse(fs.readFileSync(p,'utf8')); if(w.active) throw new Error(p+' must be inactive'); if(!w.name.endsWith('_1.1.0')) throw new Error(p+' has wrong name'); console.log(w.name+'|nodes='+w.nodes.length+'|active='+w.active)}" "E:\Xelora\n8n\二维表\Xelora版COBOL技术点检证用例工作流（基于tech_point）_1.1.0.json" "E:\Xelora\n8n\检证用例代码\Xelora版COBOL检证用例代码生成工作流_1.1.0.json"
```

Expected: two lines ending in `active=false`.

- [ ] **Step 2: Verify the old `1.0.0` files still match the backup manifest**

Set the backup directory printed by Task 8 and compare both original files with its manifest:

```powershell
$backupDir = $env:VERIFIED_BACKUP
if (-not $backupDir) { throw 'VERIFIED_BACKUP is not set' }
$manifest = Get-Content -LiteralPath (Join-Path $backupDir 'manifest.json') -Raw | ConvertFrom-Json
$originals = @(
  'E:\Xelora\n8n\二维表\Xelora版COBOL技术点检证用例工作流（基于tech_point）_1.0.0.json',
  'E:\Xelora\n8n\检证用例代码\Xelora版COBOL检证用例代码生成工作流_1.0.0.json'
)
foreach ($file in $originals) {
  $name = Split-Path -Leaf $file
  $saved = $manifest.files | Where-Object name -eq $name
  if (-not $saved) { throw "Backup manifest is missing $name" }
  $actual = (Get-FileHash -LiteralPath $file -Algorithm SHA256).Hash.ToLowerInvariant()
  if ($actual -ne $saved.sha256) { throw "Original workflow changed: $name" }
  Write-Output "UNCHANGED=$name"
}
```

Expected: two `UNCHANGED=` lines; no `1.0.0` workflow was overwritten.

- [ ] **Step 3: Run a transactional database isolation proof**

Run the following against the real analyzes database. It clones one existing row into a transaction-only probe, inserts the same key for projects 1 and 3, proves that a second project-1 copy is rejected, and always rolls back:

```sql
BEGIN;

DO $$
BEGIN
  IF EXISTS (
    SELECT 1
    FROM analyzes.cl_verification_cases_master_proj
    WHERE case_number = 'TC99999999'
  ) THEN
    RAISE EXCEPTION 'probe case_number TC99999999 already exists';
  END IF;
END $$;

WITH template AS (
  SELECT *
  FROM analyzes.cl_verification_cases_master_proj
  ORDER BY id
  LIMIT 1
)
INSERT INTO analyzes.cl_verification_cases_master_proj (
  command_id, project_id, case_number, sub_category, tech_point_no,
  test_category, test_purpose, parameter_config, input_data,
  expected_result, priority, notes
)
SELECT
  template.command_id, probe.project_id, 'TC99999999', template.sub_category,
  template.tech_point_no, template.test_category, template.test_purpose,
  template.parameter_config, template.input_data, template.expected_result,
  template.priority, template.notes
FROM template
CROSS JOIN (VALUES (1), (3)) AS probe(project_id);

DO $$
BEGIN
  BEGIN
    INSERT INTO analyzes.cl_verification_cases_master_proj (
      command_id, project_id, case_number, sub_category, tech_point_no,
      test_category, test_purpose, parameter_config, input_data,
      expected_result, priority, notes
    )
    SELECT
      command_id, project_id, case_number, sub_category, tech_point_no,
      test_category, test_purpose, parameter_config, input_data,
      expected_result, priority, notes
    FROM analyzes.cl_verification_cases_master_proj
    WHERE project_id = 1 AND case_number = 'TC99999999';
    RAISE EXCEPTION 'same-project duplicate was unexpectedly accepted';
  EXCEPTION
    WHEN unique_violation THEN
      RAISE NOTICE 'same-project duplicate correctly rejected';
  END;
END $$;

SELECT command_id, project_id, case_number
FROM analyzes.cl_verification_cases_master_proj
WHERE case_number = 'TC99999999'
ORDER BY project_id;

ROLLBACK;
```

Expected query result inside the transaction:

```text
project_id | case_number
-----------+------------
1          | TC99999999
3          | TC99999999
```

- [ ] **Step 4: Import the `1.1.0` workflows without replacing `1.0.0`**

The n8n management API previously returned `401 Unauthorized`. Import both JSON files as new inactive workflows through an authenticated n8n session or provide a valid n8n API key before automation. Confirm the imported workflow names end in `_1.1.0`; do not publish or activate them yet.

- [ ] **Step 5: Run one Fukuyama verification workflow execution**

Invoke the imported verification workflow with an existing Fukuyama command and:

```json
{
  "command_id": 229,
  "command": "ADD",
  "language": "COBOL",
  "project_id": 3,
  "sub_category_batch_size": 10
}
```

Expected: only `tech_point.project_id=3` rows are read, `variable_declaration` appears in agent context, and only `cl_verification_cases_master_proj.project_id=3` rows are replaced.

- [ ] **Step 6: Verify TOIN remains unchanged**

Before and after the Fukuyama run, compare this query for project 1:

```sql
SELECT md5(string_agg(row_to_json(v)::text, '' ORDER BY v.id)) AS toin_digest
FROM analyzes.cl_verification_cases_master_proj v
WHERE v.command_id = 229 AND v.project_id = 1;
```

Expected: identical non-null digest values.

- [ ] **Step 7: Run one Fukuyama code workflow execution**

Invoke the code workflow with:

```json
{
  "command_id": 229,
  "command": "ADD",
  "language": "COBOL",
  "project_id": 3,
  "code_gen_batch_size": 14
}
```

Expected: output uses the Fukuyama `BF229...` base, preserves real declaration types, and passes the existing file normalization and batch aggregation steps.

- [ ] **Step 8: Activate only after evidence is recorded**

Record the backup directory, migration index definition, validator outputs, both live SSE results, Fukuyama execution IDs, and the unchanged TOIN digest. Activate the two `1.1.0` workflows only when all evidence is present.

## Final Verification

Run from `E:\Xelora\WeKnora`:

```powershell
node --test scripts/n8n/backup-xelora-cobol-project-scope.test.mjs
node scripts/n8n/validate-xelora-cobol-project-scope-migration.mjs
node scripts/n8n/validate-xelora-cobol-tech-point-verification-workflow.mjs
node scripts/n8n/validate-xelora-cobol-verification-code-workflow.mjs
node scripts/n8n/smoke-test-xelora-cobol-japanese-output.mjs
node scripts/n8n/smoke-test-xelora-cobol-verification-code-agent.mjs
git status --short
```

Success means:

- the verified pre-update backup exists;
- the database identity is project-scoped;
- both `1.1.0` workflow artifacts exist and `1.0.0` remains unchanged;
- missing or unknown `project_id` fails before destructive work;
- Fukuyama declarations remain bound to their own `tech_point_no`;
- empty declarations use only the existing local inference path;
- Fukuyama rows and files do not overwrite or masquerade as TOIN;
- all offline validators and live SSE checks pass;
- unrelated dirty-worktree files remain untouched.
