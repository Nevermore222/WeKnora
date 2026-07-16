# Xelora n8n Parallel Parameter Parse Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Create a new parallel n8n workflow that calls Xelora and `Manual_ASP` for CL command parameter parsing without changing the existing Dify/RAGFlow workflow.

**Architecture:** Keep the existing n8n workflow as the baseline. Add a generated workflow export under the external n8n assets folder, plus repository-local scripts that generate and validate that export. The new workflow uses a dedicated webhook, Xelora API calls, strict JSON validation, and isolated staging tables for comparison.

**Tech Stack:** n8n workflow JSON, Node.js ESM scripts, PostgreSQL SQL nodes, Xelora HTTP API, SSE response parsing, Git.

---

## File Structure

- Create: `E:\Xelora\WeKnora\scripts\n8n\validate-xelora-parameter-parse-workflow.mjs`
  - Validates the generated workflow export before import into n8n.
- Create: `E:\Xelora\WeKnora\scripts\n8n\build-xelora-parameter-parse-workflow.mjs`
  - Generates the new workflow JSON deterministically.
- Create: `E:\Xelora\n8n\参数解析\Xelora-CL-Parameter-Parse-Parallel_1.0.0.json`
  - New n8n workflow export for manual import or API import.
- Modify: `E:\Xelora\WeKnora\docs\superpowers\plans\2026-07-16-xelora-n8n-parallel-parameter-parse.md`
  - Mark completed steps during implementation.

No existing n8n export is modified. The existing Dify/RAGFlow workflow remains the production baseline.

## Environment Contract

The generated n8n workflow reads these runtime environment variables:

- `XELORA_BASE_URL`: base URL reachable from n8n, for example `http://Xelora-app:8080`.
- `N8N_XELORA_API_KEY`: Xelora API key sent with `X-API-Key`.
- `XELORA_PARAMETER_AGENT_ID`: dedicated CL command parameter parsing agent ID.
- `XELORA_MANUAL_ASP_KB_ID`: `Manual_ASP` knowledge base ID.

These names are fixed. The implementation must not hard-code secret values into JSON.

---

### Task 1: Add Workflow Validation Script

**Files:**
- Create: `E:\Xelora\WeKnora\scripts\n8n\validate-xelora-parameter-parse-workflow.mjs`

- [ ] **Step 1: Create the validation script**

Use `apply_patch` to add:

```js
#!/usr/bin/env node
import fs from "node:fs";
import path from "node:path";

const repoRoot = process.cwd();
const workflowPath =
  process.argv[2] ||
  path.resolve(repoRoot, "..", "n8n", "参数解析", "Xelora-CL-Parameter-Parse-Parallel_1.0.0.json");

function fail(message) {
  console.error(`FAIL: ${message}`);
  process.exitCode = 1;
}

function assert(condition, message) {
  if (!condition) fail(message);
}

const raw = fs.readFileSync(workflowPath, "utf8");
let workflow;
try {
  workflow = JSON.parse(raw);
} catch (error) {
  fail(`workflow is not valid JSON: ${error.message}`);
  process.exit();
}

const serialized = JSON.stringify(workflow);
const nodeNames = new Set((workflow.nodes || []).map((node) => node.name));
const nodeTypes = new Set((workflow.nodes || []).map((node) => node.type));

assert(workflow.name === "Xelora - CL Parameter Parse - Parallel", "workflow name must identify the Xelora parallel flow");
assert(workflow.active === false, "workflow must be inactive before manual validation");
assert(Array.isArray(workflow.nodes), "workflow.nodes must be an array");
assert(workflow.nodes.length >= 10, "workflow must contain the core parallel parsing nodes");
assert(nodeTypes.has("n8n-nodes-base.webhook"), "workflow must contain a webhook node");
assert(nodeTypes.has("n8n-nodes-base.httpRequest"), "workflow must contain HTTP request nodes");
assert(nodeTypes.has("n8n-nodes-base.postgres"), "workflow must contain PostgreSQL nodes");

assert(nodeNames.has("Webhook - Xelora Parameter Parse"), "missing webhook node");
assert(nodeNames.has("Prepare Xelora Session Request"), "missing session request preparation node");
assert(nodeNames.has("Create Xelora Session"), "missing session creation node");
assert(nodeNames.has("Prepare Xelora Agent Request"), "missing agent request preparation node");
assert(nodeNames.has("Call Xelora Agent"), "missing agent call node");
assert(nodeNames.has("Parse Xelora Stream"), "missing stream parser node");
assert(nodeNames.has("Validate Parameter JSON"), "missing JSON validation node");
assert(nodeNames.has("Ensure Xelora Staging Tables"), "missing staging table node");
assert(nodeNames.has("Build Parameter Insert SQL"), "missing insert SQL builder node");
assert(nodeNames.has("Insert Xelora Parameter Rows"), "missing parameter insert node");
assert(nodeNames.has("Build Failure Insert SQL"), "missing failure SQL builder node");
assert(nodeNames.has("Insert Xelora Failure Row"), "missing failure insert node");

assert(!serialized.includes("192.168.8.247"), "workflow must not call the old Dify host");
assert(!serialized.includes("Authorization"), "workflow must not use Dify Authorization header");
assert(!serialized.includes("Bearer app-"), "workflow must not embed Dify app tokens");
assert(!serialized.includes("source_content"), "workflow must not read or pass source_content");
assert(!serialized.includes("SOURCE"), "workflow must not pass a SOURCE input");
assert(!serialized.includes("f786036e-bce5-4fe2-ad96-76a83ab2f78e"), "workflow must not reuse the old webhook path");
assert(serialized.includes("N8N_XELORA_API_KEY"), "workflow must use N8N_XELORA_API_KEY");
assert(serialized.includes("XELORA_PARAMETER_AGENT_ID"), "workflow must use XELORA_PARAMETER_AGENT_ID");
assert(serialized.includes("XELORA_MANUAL_ASP_KB_ID"), "workflow must use XELORA_MANUAL_ASP_KB_ID");
assert(serialized.includes("xelora_command_parameters_staging"), "workflow must write to staging parameter table");
assert(serialized.includes("xelora_parameter_parse_failures"), "workflow must write failures to failure table");

if (process.exitCode) process.exit();
console.log(`PASS: ${workflowPath}`);
```

- [ ] **Step 2: Run the validator before the workflow exists**

Run:

```powershell
node .\scripts\n8n\validate-xelora-parameter-parse-workflow.mjs
```

Expected: FAIL because `E:\Xelora\n8n\参数解析\Xelora-CL-Parameter-Parse-Parallel_1.0.0.json` does not exist yet.

- [ ] **Step 3: Commit the failing validator**

Run:

```powershell
git add -- scripts/n8n/validate-xelora-parameter-parse-workflow.mjs
git commit -m "test: add xelora n8n workflow validator"
```

Expected: commit succeeds and does not include Docker or entrypoint files.

---

### Task 2: Add Deterministic Workflow Builder

**Files:**
- Create: `E:\Xelora\WeKnora\scripts\n8n\build-xelora-parameter-parse-workflow.mjs`

- [ ] **Step 1: Create the builder script**

Use `apply_patch` to add:

```js
#!/usr/bin/env node
import fs from "node:fs";
import path from "node:path";

const repoRoot = process.cwd();
const outputPath = path.resolve(repoRoot, "..", "n8n", "参数解析", "Xelora-CL-Parameter-Parse-Parallel_1.0.0.json");

function code(jsCode) {
  return {
    type: "n8n-nodes-base.code",
    typeVersion: 2,
    parameters: { jsCode },
  };
}

function node(id, name, type, position, parameters, extra = {}) {
  return {
    id,
    name,
    type,
    typeVersion: extra.typeVersion || 1,
    position,
    parameters,
    ...Object.fromEntries(Object.entries(extra).filter(([key]) => key !== "typeVersion")),
  };
}

function connect(connections, from, to, outputIndex = 0) {
  connections[from] ||= { main: [] };
  connections[from].main[outputIndex] ||= [];
  connections[from].main[outputIndex].push({ node: to, type: "main", index: 0 });
}

const workflow = {
  name: "Xelora - CL Parameter Parse - Parallel",
  nodes: [],
  connections: {},
  active: false,
  settings: { executionOrder: "v1" },
  pinData: {},
  tags: [{ name: "Xelora Parallel" }, { name: "Parameter Parse" }],
};

workflow.nodes.push(
  node(
    "xelora-webhook-parameter-parse",
    "Webhook - Xelora Parameter Parse",
    "n8n-nodes-base.webhook",
    [0, 240],
    {
      httpMethod: "POST",
      path: "xelora-cl-parameter-parse-parallel",
      responseMode: "lastNode",
      options: { rawBody: true },
    },
    { typeVersion: 2.1, webhookId: "xelora-cl-parameter-parse-parallel" },
  ),
  code(`const input = $json.body || $json;
const commandId = Number(input.command_id);
const command = String(input.command || "").trim();
const language = String(input.language || "CL").trim() || "CL";

if (!Number.isInteger(commandId) || commandId <= 0) {
  throw new Error("command_id must be a positive integer");
}
if (!command) {
  throw new Error("command must be non-empty");
}

return [{
  json: {
    command_id: commandId,
    command,
    language,
    workflow_source: "xelora_parallel",
    request_time: new Date().toISOString(),
    attempt_count: 1
  }
}];`),
  code(`const baseUrl = String($env.XELORA_BASE_URL || "http://Xelora-app:8080").replace(/\\/$/, "");
return [{
  json: {
    ...$json,
    xelora_base_url: baseUrl,
    session_url: baseUrl + "/api/v1/sessions",
    session_request_body: {
      title: "CL parameter parse " + $json.command
    }
  }
}];`),
  node(
    "xelora-create-session",
    "Create Xelora Session",
    "n8n-nodes-base.httpRequest",
    [760, 240],
    {
      method: "POST",
      url: "={{ $json.session_url }}",
      sendHeaders: true,
      headerParameters: {
        parameters: [
          { name: "X-API-Key", value: "={{ $env.N8N_XELORA_API_KEY }}" },
          { name: "Content-Type", value: "application/json" },
        ],
      },
      sendBody: true,
      specifyBody: "json",
      jsonBody: "={{ $json.session_request_body }}",
      options: { response: { response: { neverError: true } } },
    },
    { typeVersion: 4.2, onError: "continueErrorOutput" },
  ),
  code(`const prior = $("Prepare Xelora Session Request").first().json;
const response = $input.first().json;
const sessionId =
  response.session_id ||
  response.id ||
  response.data?.session_id ||
  response.data?.id ||
  response.result?.session_id ||
  response.result?.id;

if (!sessionId) {
  return [{
    json: {
      ...prior,
      valid: false,
      error_type: "session_create_failed",
      raw_response: JSON.stringify(response).slice(0, 20000)
    }
  }];
}

return [{
  json: {
    ...prior,
    session_id: String(sessionId),
    agent_url: prior.xelora_base_url + "/api/v1/agent-chat/" + String(sessionId)
  }
}];`),
  code(`const query = [
  "Parse all parameters of CL command " + $json.command + ".",
  "Use only the Manual_ASP knowledge base.",
  "Return exactly one JSON object and no Markdown.",
  "Schema: {command, language, parameters:[{parameter_name, parameter_type, data_type, enum_value, value_range, default_value, description, is_required, relationship_notes}]}."
].join(" ");

return [{
  json: {
    ...$json,
    agent_request_body: {
      query,
      agent_enabled: true,
      agent_id: $env.XELORA_PARAMETER_AGENT_ID,
      knowledge_base_ids: [$env.XELORA_MANUAL_ASP_KB_ID],
      mentioned_items: [{
        id: $env.XELORA_MANUAL_ASP_KB_ID,
        name: "Manual_ASP",
        type: "kb",
        kb_type: "document"
      }],
      web_search_enabled: false,
      channel: "api"
    }
  }
}];`),
  node(
    "xelora-call-agent",
    "Call Xelora Agent",
    "n8n-nodes-base.httpRequest",
    [1340, 240],
    {
      method: "POST",
      url: "={{ $json.agent_url }}",
      sendHeaders: true,
      headerParameters: {
        parameters: [
          { name: "X-API-Key", value: "={{ $env.N8N_XELORA_API_KEY }}" },
          { name: "Content-Type", value: "application/json" },
        ],
      },
      sendBody: true,
      specifyBody: "json",
      jsonBody: "={{ $json.agent_request_body }}",
      options: { response: { response: { neverError: true } } },
    },
    { typeVersion: 4.2, onError: "continueErrorOutput" },
  ),
  code(`const requestData = $("Prepare Xelora Agent Request").first().json;
const response = $input.first().json;
let answer = "";
let sawComplete = false;

function absorbEvent(event) {
  if (!event || typeof event !== "object") return;
  if (typeof event.answer === "string") answer += event.answer;
  if (typeof event.content === "string") answer += event.content;
  if (event.event === "complete" || event.type === "complete" || event.done === true) sawComplete = true;
}

if (typeof response === "string") {
  for (const line of response.split("\\n")) {
    if (!line.startsWith("data: ")) continue;
    const payload = line.slice(6).trim();
    if (!payload || payload === "[DONE]") {
      sawComplete = true;
      continue;
    }
    try {
      absorbEvent(JSON.parse(payload));
    } catch {
      continue;
    }
  }
} else if (typeof response.data === "string") {
  for (const line of response.data.split("\\n")) {
    if (!line.startsWith("data: ")) continue;
    const payload = line.slice(6).trim();
    if (!payload || payload === "[DONE]") {
      sawComplete = true;
      continue;
    }
    try {
      absorbEvent(JSON.parse(payload));
    } catch {
      continue;
    }
  }
} else {
  absorbEvent(response);
  absorbEvent(response.data);
  absorbEvent(response.result);
}

return [{
  json: {
    ...requestData,
    raw_response: answer.trim() || JSON.stringify(response),
    stream_complete: sawComplete,
    response_time: new Date().toISOString()
  }
}];`),
  code(`const requiredParameterFields = [
  "parameter_name",
  "parameter_type",
  "data_type",
  "enum_value",
  "value_range",
  "default_value",
  "description",
  "is_required",
  "relationship_notes"
];

function extractJsonObject(text) {
  const trimmed = String(text || "").trim();
  if (trimmed.startsWith("{") && trimmed.endsWith("}")) return trimmed;
  const start = trimmed.indexOf("{");
  const end = trimmed.lastIndexOf("}");
  if (start >= 0 && end > start) return trimmed.slice(start, end + 1);
  return trimmed;
}

function normalizeString(value) {
  if (value === null || value === undefined) return "";
  return String(value).trim();
}

let parsed;
try {
  parsed = JSON.parse(extractJsonObject($json.raw_response));
} catch (error) {
  return [{
    json: {
      ...$json,
      valid: false,
      error_type: "invalid_json",
      error_message: error.message
    }
  }];
}

if (!parsed || typeof parsed !== "object" || !Array.isArray(parsed.parameters)) {
  return [{
    json: {
      ...$json,
      valid: false,
      error_type: "schema_validation_failed",
      error_message: "root object must contain parameters array"
    }
  }];
}

const dedup = new Map();
for (const original of parsed.parameters) {
  if (!original || typeof original !== "object") continue;
  const normalized = {};
  for (const field of requiredParameterFields) {
    if (field === "is_required") {
      normalized[field] = original[field] === true || String(original[field]).toLowerCase() === "true";
    } else {
      normalized[field] = normalizeString(original[field]);
    }
  }
  if (!normalized.parameter_name) continue;
  const key = normalized.parameter_name.toUpperCase();
  const score = Object.entries(normalized).filter(([name, value]) => name !== "is_required" && String(value).trim()).length;
  const existing = dedup.get(key);
  if (!existing || score > existing.score) dedup.set(key, { score, value: normalized });
}

const parameters = Array.from(dedup.values()).map((entry, index) => ({
  ...entry.value,
  display_order: index + 1
}));

return [{
  json: {
    ...$json,
    valid: true,
    parsed_command: normalizeString(parsed.command) || $json.command,
    parsed_language: normalizeString(parsed.language) || $json.language,
    parameters,
    no_parameters_found: parameters.length === 0
  }
}];`),
  node(
    "xelora-valid-if",
    "Is Valid Parameter JSON",
    "n8n-nodes-base.if",
    [1920, 240],
    {
      conditions: {
        boolean: [{ value1: "={{ $json.valid }}", value2: true }],
      },
    },
    { typeVersion: 1 },
  ),
  node(
    "xelora-ensure-tables",
    "Ensure Xelora Staging Tables",
    "n8n-nodes-base.postgres",
    [2160, 120],
    {
      operation: "executeQuery",
      query: `CREATE TABLE IF NOT EXISTS analyzes.xelora_command_parameters_staging (
  id SERIAL PRIMARY KEY,
  command_id INTEGER NOT NULL,
  command VARCHAR(100) NOT NULL,
  language VARCHAR(50) NOT NULL,
  parameter_name VARCHAR(100) NOT NULL,
  parameter_type VARCHAR(50),
  data_type VARCHAR(50),
  enum_value VARCHAR(500),
  value_range VARCHAR(500),
  default_value VARCHAR(500),
  display_order INTEGER DEFAULT 0,
  description TEXT,
  is_required BOOLEAN DEFAULT false,
  relationship_notes TEXT,
  raw_response TEXT,
  session_id VARCHAR(200),
  workflow_source VARCHAR(50) DEFAULT 'xelora_parallel',
  created_at TIMESTAMP DEFAULT NOW(),
  updated_at TIMESTAMP DEFAULT NOW(),
  CONSTRAINT xelora_command_parameters_staging_unique UNIQUE (command_id, parameter_name)
);

CREATE TABLE IF NOT EXISTS analyzes.xelora_parameter_parse_failures (
  id SERIAL PRIMARY KEY,
  command_id INTEGER,
  command VARCHAR(100),
  language VARCHAR(50),
  stage VARCHAR(100) NOT NULL,
  status VARCHAR(50) NOT NULL,
  error_type VARCHAR(100),
  error_message TEXT,
  raw_response TEXT,
  session_id VARCHAR(200),
  attempt_count INTEGER DEFAULT 1,
  created_at TIMESTAMP DEFAULT NOW()
);`,
      options: {},
    },
    {
      typeVersion: 2.1,
      credentials: { postgres: { id: "tWcUIJhVg6bDlGDc", name: "TargetDB" } },
      onError: "continueErrorOutput",
    },
  ),
  code(`function sqlString(value) {
  if (value === null || value === undefined || value === "") return "NULL";
  return "'" + String(value).replace(/'/g, "''").replace(/\\\\/g, "\\\\\\\\") + "'";
}

function sqlBoolean(value) {
  return value === true ? "TRUE" : "FALSE";
}

const rows = ($json.parameters || []).map((parameter) => \`(
  \${Number($json.command_id)},
  \${sqlString($json.command)},
  \${sqlString($json.language)},
  \${sqlString(parameter.parameter_name)},
  \${sqlString(parameter.parameter_type)},
  \${sqlString(parameter.data_type)},
  \${sqlString(parameter.enum_value)},
  \${sqlString(parameter.value_range)},
  \${sqlString(parameter.default_value)},
  \${Number(parameter.display_order || 0)},
  \${sqlString(parameter.description)},
  \${sqlBoolean(parameter.is_required)},
  \${sqlString(parameter.relationship_notes)},
  \${sqlString($json.raw_response)},
  \${sqlString($json.session_id)}
)\`);

if (!rows.length) {
  return [{ json: { ...$json, sql_query: "SELECT 1 AS no_parameters_found;" } }];
}

return [{
  json: {
    ...$json,
    sql_query: \`INSERT INTO analyzes.xelora_command_parameters_staging (
  command_id, command, language, parameter_name, parameter_type, data_type,
  enum_value, value_range, default_value, display_order, description,
  is_required, relationship_notes, raw_response, session_id
) VALUES \${rows.join(",\\n")}
ON CONFLICT (command_id, parameter_name) DO UPDATE SET
  command = EXCLUDED.command,
  language = EXCLUDED.language,
  parameter_type = EXCLUDED.parameter_type,
  data_type = EXCLUDED.data_type,
  enum_value = EXCLUDED.enum_value,
  value_range = EXCLUDED.value_range,
  default_value = EXCLUDED.default_value,
  display_order = EXCLUDED.display_order,
  description = EXCLUDED.description,
  is_required = EXCLUDED.is_required,
  relationship_notes = EXCLUDED.relationship_notes,
  raw_response = EXCLUDED.raw_response,
  session_id = EXCLUDED.session_id,
  updated_at = NOW();\`
  }
}];`),
  node(
    "xelora-insert-rows",
    "Insert Xelora Parameter Rows",
    "n8n-nodes-base.postgres",
    [2640, 120],
    {
      operation: "executeQuery",
      query: "={{ $json.sql_query }}",
      options: {},
    },
    {
      typeVersion: 2.1,
      credentials: { postgres: { id: "tWcUIJhVg6bDlGDc", name: "TargetDB" } },
      onError: "continueErrorOutput",
    },
  ),
  code(`function sqlString(value) {
  if (value === null || value === undefined || value === "") return "NULL";
  return "'" + String(value).replace(/'/g, "''").replace(/\\\\/g, "\\\\\\\\") + "'";
}

return [{
  json: {
    ...$json,
    sql_query: \`INSERT INTO analyzes.xelora_parameter_parse_failures (
  command_id, command, language, stage, status, error_type, error_message,
  raw_response, session_id, attempt_count
) VALUES (
  \${Number($json.command_id || 0) || "NULL"},
  \${sqlString($json.command)},
  \${sqlString($json.language)},
  'xelora_parameter_parse',
  'failed',
  \${sqlString($json.error_type || "unknown_error")},
  \${sqlString($json.error_message || "")},
  \${sqlString($json.raw_response || "")},
  \${sqlString($json.session_id || "")},
  \${Number($json.attempt_count || 1)}
);\`
  }
}];`),
  node(
    "xelora-insert-failure",
    "Insert Xelora Failure Row",
    "n8n-nodes-base.postgres",
    [2160, 400],
    {
      operation: "executeQuery",
      query: "={{ $json.sql_query }}",
      options: {},
    },
    {
      typeVersion: 2.1,
      credentials: { postgres: { id: "tWcUIJhVg6bDlGDc", name: "TargetDB" } },
      onError: "continueErrorOutput",
    },
  ),
);

workflow.nodes[1].id = "xelora-parse-input";
workflow.nodes[1].name = "Parse Webhook Input";
workflow.nodes[1].position = [240, 240];

workflow.nodes[2].id = "xelora-prepare-session";
workflow.nodes[2].name = "Prepare Xelora Session Request";
workflow.nodes[2].position = [500, 240];

workflow.nodes[4].id = "xelora-extract-session";
workflow.nodes[4].name = "Extract Xelora Session";
workflow.nodes[4].position = [1040, 240];

workflow.nodes[5].id = "xelora-prepare-agent";
workflow.nodes[5].name = "Prepare Xelora Agent Request";
workflow.nodes[5].position = [1200, 240];

workflow.nodes[7].id = "xelora-parse-stream";
workflow.nodes[7].name = "Parse Xelora Stream";
workflow.nodes[7].position = [1540, 240];

workflow.nodes[8].id = "xelora-validate-json";
workflow.nodes[8].name = "Validate Parameter JSON";
workflow.nodes[8].position = [1740, 240];

workflow.nodes[11].id = "xelora-build-insert-sql";
workflow.nodes[11].name = "Build Parameter Insert SQL";
workflow.nodes[11].position = [2400, 120];

workflow.nodes[13].id = "xelora-build-failure-sql";
workflow.nodes[13].name = "Build Failure Insert SQL";
workflow.nodes[13].position = [2400, 400];

connect(workflow.connections, "Webhook - Xelora Parameter Parse", "Parse Webhook Input");
connect(workflow.connections, "Parse Webhook Input", "Prepare Xelora Session Request");
connect(workflow.connections, "Prepare Xelora Session Request", "Create Xelora Session");
connect(workflow.connections, "Create Xelora Session", "Extract Xelora Session");
connect(workflow.connections, "Extract Xelora Session", "Prepare Xelora Agent Request");
connect(workflow.connections, "Prepare Xelora Agent Request", "Call Xelora Agent");
connect(workflow.connections, "Call Xelora Agent", "Parse Xelora Stream");
connect(workflow.connections, "Parse Xelora Stream", "Validate Parameter JSON");
connect(workflow.connections, "Validate Parameter JSON", "Is Valid Parameter JSON");
connect(workflow.connections, "Is Valid Parameter JSON", "Ensure Xelora Staging Tables", 0);
connect(workflow.connections, "Ensure Xelora Staging Tables", "Build Parameter Insert SQL");
connect(workflow.connections, "Build Parameter Insert SQL", "Insert Xelora Parameter Rows");
connect(workflow.connections, "Is Valid Parameter JSON", "Build Failure Insert SQL", 1);
connect(workflow.connections, "Build Failure Insert SQL", "Insert Xelora Failure Row");

fs.mkdirSync(path.dirname(outputPath), { recursive: true });
fs.writeFileSync(outputPath, JSON.stringify(workflow, null, 2) + "\\n", "utf8");
console.log(`Wrote ${outputPath}`);
```

- [ ] **Step 2: Run the builder**

Run:

```powershell
node .\scripts\n8n\build-xelora-parameter-parse-workflow.mjs
```

Expected:

```text
Wrote E:\Xelora\n8n\参数解析\Xelora-CL-Parameter-Parse-Parallel_1.0.0.json
```

- [ ] **Step 3: Run the validator**

Run:

```powershell
node .\scripts\n8n\validate-xelora-parameter-parse-workflow.mjs
```

Expected:

```text
PASS: E:\Xelora\n8n\参数解析\Xelora-CL-Parameter-Parse-Parallel_1.0.0.json
```

- [ ] **Step 4: Commit builder and workflow export**

Run:

```powershell
git add -- scripts/n8n/build-xelora-parameter-parse-workflow.mjs scripts/n8n/validate-xelora-parameter-parse-workflow.mjs ..\n8n\参数解析\Xelora-CL-Parameter-Parse-Parallel_1.0.0.json
git commit -m "feat: add parallel xelora parameter parse workflow"
```

Expected: commit includes only the two scripts and the new Xelora workflow export.

---

### Task 3: Add Retry Behavior

**Files:**
- Modify: `E:\Xelora\WeKnora\scripts\n8n\build-xelora-parameter-parse-workflow.mjs`
- Modify: `E:\Xelora\n8n\参数解析\Xelora-CL-Parameter-Parse-Parallel_1.0.0.json`

- [ ] **Step 1: Extend the builder with one retry branch**

In the builder script, add these nodes after `Validate Parameter JSON` and before failure insert:

```js
code(`if ($json.valid === true) return [{ json: $json }];
if (Number($json.attempt_count || 1) >= 2) return [{ json: $json }];
return [{
  json: {
    ...$json,
    attempt_count: Number($json.attempt_count || 1) + 1,
    retry_reason: $json.error_type || "invalid_response"
  }
}];`)
```

Name the node `Prepare Retry Attempt`.

Add an IF node named `Should Retry Xelora Parse` with:

```json
{
  "conditions": {
    "boolean": [
      {
        "value1": "={{ $json.valid !== true && Number($json.attempt_count || 1) < 2 }}",
        "value2": true
      }
    ]
  }
}
```

Connect invalid first-attempt results back to `Prepare Xelora Session Request`. Connect invalid second-attempt results to `Build Failure Insert SQL`.

- [ ] **Step 2: Regenerate workflow**

Run:

```powershell
node .\scripts\n8n\build-xelora-parameter-parse-workflow.mjs
```

Expected: workflow export is rewritten.

- [ ] **Step 3: Validate workflow**

Run:

```powershell
node .\scripts\n8n\validate-xelora-parameter-parse-workflow.mjs
```

Expected:

```text
PASS: E:\Xelora\n8n\参数解析\Xelora-CL-Parameter-Parse-Parallel_1.0.0.json
```

- [ ] **Step 4: Commit retry behavior**

Run:

```powershell
git add -- scripts/n8n/build-xelora-parameter-parse-workflow.mjs ..\n8n\参数解析\Xelora-CL-Parameter-Parse-Parallel_1.0.0.json
git commit -m "feat: add retry path to xelora parameter parse workflow"
```

Expected: commit includes only the builder and regenerated workflow.

---

### Task 4: Import and Local Smoke Test

**Files:**
- Read: `E:\Xelora\n8n\参数解析\Xelora-CL-Parameter-Parse-Parallel_1.0.0.json`
- No repository code change required unless n8n import exposes a schema issue.

- [ ] **Step 1: Import the generated workflow into n8n**

Use the n8n UI import action and select:

```text
E:\Xelora\n8n\参数解析\Xelora-CL-Parameter-Parse-Parallel_1.0.0.json
```

Expected: n8n shows a new inactive workflow named:

```text
Xelora - CL Parameter Parse - Parallel
```

- [ ] **Step 2: Configure runtime environment**

Set these variables for the n8n container or n8n process:

```text
XELORA_BASE_URL=http://Xelora-app:8080
N8N_XELORA_API_KEY=inject-from-secure-n8n-runtime
XELORA_PARAMETER_AGENT_ID=inject-from-secure-n8n-runtime
XELORA_MANUAL_ASP_KB_ID=inject-from-secure-n8n-runtime
```

Expected: variables are visible to n8n expressions as `$env.XELORA_BASE_URL`, `$env.N8N_XELORA_API_KEY`, `$env.XELORA_PARAMETER_AGENT_ID`, and `$env.XELORA_MANUAL_ASP_KB_ID`.

- [ ] **Step 3: Run a manual test execution**

Trigger the imported workflow manually with:

```json
{
  "command_id": 123,
  "command": "SORTD",
  "language": "CL"
}
```

Expected:

- `Create Xelora Session` returns a session identifier.
- `Call Xelora Agent` returns streamed or direct answer content.
- `Validate Parameter JSON` sets `valid` to `true` when Xelora returns schema-compliant JSON.
- `Insert Xelora Parameter Rows` writes only to `analyzes.xelora_command_parameters_staging`.
- The original Dify/RAGFlow workflow is unchanged and not triggered by this manual execution.

- [ ] **Step 4: Check database isolation**

Run this SQL against TargetDB:

```sql
SELECT command_id, command, language, parameter_name, workflow_source, updated_at
FROM analyzes.xelora_command_parameters_staging
WHERE command = 'SORTD'
ORDER BY display_order, parameter_name;

SELECT command_id, command, status, error_type, created_at
FROM analyzes.xelora_parameter_parse_failures
WHERE command = 'SORTD'
ORDER BY created_at DESC;
```

Expected: successful rows appear in `xelora_command_parameters_staging`, or failure rows appear in `xelora_parameter_parse_failures`. No rows are overwritten in `analyzes.command_parameters`.

---

## Self-Review

Spec coverage:

- Parallel workflow creation is covered by Task 2.
- Original Dify/RAGFlow workflow preservation is covered by validator checks and the new output file path.
- No DB `SOURCE` or `source_content` is covered by validator checks.
- Xelora session creation and agent call are covered by Task 2.
- `Manual_ASP` knowledge base use is covered through `XELORA_MANUAL_ASP_KB_ID`.
- Strict JSON validation is covered by `Validate Parameter JSON`.
- Isolated writes are covered by staging and failure tables.
- One retry is covered by Task 3.
- Local smoke test and DB isolation check are covered by Task 4.

Placeholder scan:

- Secret values are not stored in git. Runtime values are environment variables.
- Runtime secret examples use non-secret sentinel values and are not copied into source code or generated workflow JSON.

Type consistency:

- `command_id` is normalized as a positive integer.
- `command` and `language` are strings.
- `parameters` is always an array after validation.
- `is_required` is normalized to boolean.
- SQL insert values are escaped before query construction.
