#!/usr/bin/env node
import fs from "node:fs";
import path from "node:path";

const repoRoot = process.cwd();
const outputPath = path.resolve(
  repoRoot,
  "..",
  "n8n",
  "参数解析",
  "Xelora-CL-Parameter-Parse-Parallel_1.0.0.json",
);

function codeNode(id, name, position, jsCode, extra = {}) {
  return {
    id,
    name,
    type: "n8n-nodes-base.code",
    typeVersion: 2,
    position,
    parameters: { jsCode },
    ...extra,
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

function postgresNode(id, name, position, query) {
  return node(
    id,
    name,
    "n8n-nodes-base.postgres",
    position,
    {
      operation: "executeQuery",
      query,
      options: {},
    },
    {
      typeVersion: 2.1,
      credentials: { postgres: { id: "tWcUIJhVg6bDlGDc", name: "TargetDB" } },
      onError: "continueErrorOutput",
    },
  );
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
  codeNode(
    "xelora-parse-input",
    "Parse Webhook Input",
    [240, 240],
    `const input = $json.body || $json;
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
}];`,
  ),
  codeNode(
    "xelora-prepare-session",
    "Prepare Xelora Session Request",
    [500, 240],
    `const baseUrl = String($env.XELORA_BASE_URL || "http://Xelora-app:8080").replace(/\\/$/, "");
return [{
  json: {
    ...$json,
    xelora_base_url: baseUrl,
    session_url: baseUrl + "/api/v1/sessions",
    session_request_body: {
      title: "CL parameter parse " + $json.command
    }
  }
}];`,
  ),
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
  codeNode(
    "xelora-extract-session",
    "Extract Xelora Session",
    [1040, 240],
    `const prior = $("Prepare Xelora Session Request").first().json;
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
}];`,
  ),
  codeNode(
    "xelora-prepare-agent",
    "Prepare Xelora Agent Request",
    [1200, 240],
    `const query = [
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
}];`,
  ),
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
  codeNode(
    "xelora-parse-stream",
    "Parse Xelora Stream",
    [1540, 240],
    `const requestData = $("Prepare Xelora Agent Request").first().json;
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
}];`,
  ),
  codeNode(
    "xelora-validate-json",
    "Validate Parameter JSON",
    [1740, 240],
    `const requiredParameterFields = [
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
}];`,
  ),
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
  postgresNode(
    "xelora-ensure-tables",
    "Ensure Xelora Staging Tables",
    [2160, 120],
    `CREATE TABLE IF NOT EXISTS analyzes.xelora_command_parameters_staging (
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
  ),
  codeNode(
    "xelora-build-insert-sql",
    "Build Parameter Insert SQL",
    [2400, 120],
    `function sqlString(value) {
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
}];`,
  ),
  postgresNode(
    "xelora-insert-rows",
    "Insert Xelora Parameter Rows",
    [2640, 120],
    "={{ $json.sql_query }}",
  ),
  codeNode(
    "xelora-build-failure-sql",
    "Build Failure Insert SQL",
    [2160, 400],
    `function sqlString(value) {
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
}];`,
  ),
  postgresNode(
    "xelora-insert-failure",
    "Insert Xelora Failure Row",
    [2400, 400],
    "={{ $json.sql_query }}",
  ),
);

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
fs.writeFileSync(outputPath, JSON.stringify(workflow, null, 2) + "\n", "utf8");
console.log(`Wrote ${outputPath}`);
