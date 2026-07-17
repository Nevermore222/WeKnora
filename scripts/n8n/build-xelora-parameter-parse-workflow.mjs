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

function postgresNode(id, name, position, query, extra = {}) {
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
      ...extra,
    },
  );
}

const workflow = {
  name: "Xelora版CL命令参数解析工作流（并行）_1.0.0",
  nodes: [],
  connections: {},
  active: false,
  settings: { executionOrder: "v1" },
  pinData: {},
  tags: [{ name: "Xelora并行" }, { name: "参数解析" }],
};

workflow.nodes.push(
  node(
    "xelora-webhook-parameter-parse",
    "Webhook接收参数",
    "n8n-nodes-base.webhook",
    [0, 240],
    {
      httpMethod: "POST",
      path: "xelora-cl-parameter-parse-parallel",
      responseMode: "lastNode",
      options: { rawBody: true },
    },
    {
      typeVersion: 2.1,
      webhookId: "xelora-cl-parameter-parse-parallel",
      notes: "接收参数：command_id(主表ID)、command、language；用于并行验证Xelora智能体参数解析流程",
    },
  ),
  codeNode(
    "xelora-parse-input",
    "解析Webhook参数",
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
    { notes: "从Webhook接收参数：command_id(主表ID)、command、language" },
  ),
  codeNode(
    "xelora-prepare-session",
    "准备Xelora会话请求",
    [500, 240],
    `const configuredApiBase = String($env.XELORA_API_BASE_URL || "").trim();
const hostBase = String($env.XELORA_BASE_URL || "http://Xelora-app:8080").replace(/\\/$/, "");
const apiBase = (configuredApiBase || hostBase + "/api/v1").replace(/\\/$/, "");
return [{
  json: {
    ...$json,
    xelora_api_base_url: apiBase,
    session_url: apiBase + "/sessions",
    session_request_body: {
      title: "CL parameter parse " + $json.command
    }
  }
}];`,
    { notes: "准备创建Xelora会话；使用XELORA_API_BASE_URL，局域网调用时配置为对外开放URL" },
  ),
  node(
    "xelora-create-session",
    "创建Xelora会话",
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
    {
      typeVersion: 4.2,
      onError: "continueErrorOutput",
      notes: "调用Xelora /api/v1/sessions 创建会话，后续智能体流式调用复用该session_id",
    },
  ),
  codeNode(
    "xelora-extract-session",
    "提取Xelora会话ID",
    [1040, 240],
    `const prior = $("准备Xelora会话请求").first().json;
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
    agent_url: prior.xelora_api_base_url + "/agent-chat/" + String(sessionId)
  }
}];`,
    { notes: "从创建会话响应中提取session_id，并拼接Xelora智能体流式调用地址 /agent-chat/{session_id}" },
  ),
  codeNode(
    "xelora-prepare-agent",
    "准备Xelora智能体请求",
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
    { notes: "准备调用Xelora内CL命令参数解析智能体；不再传入数据库源码字段，以Manual_ASP知识库为准" },
  ),
  node(
    "xelora-call-agent",
    "调用Xelora参数解析智能体",
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
    {
      typeVersion: 4.2,
      onError: "continueErrorOutput",
      notes: "替代原“调用参数解析智能体”节点；通过Xelora API流式调用绑定Manual_ASP知识库的智能体",
    },
  ),
  codeNode(
    "xelora-parse-stream",
    "处理AI响应",
    [1540, 240],
    `const requestData = $("准备Xelora智能体请求").first().json;
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
    { notes: "解析Xelora SSE流式响应，提取完整AI参数分析结果；对应原工作流“处理AI响应”节点" },
  ),
  codeNode(
    "xelora-validate-json",
    "解析参数二维表",
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
    { notes: "从AI响应中解析参数JSON，规范化为参数二维表数据；输出字段保持后续入库所需结构" },
  ),
  node(
    "xelora-valid-if",
    "判断参数解析是否成功",
    "n8n-nodes-base.if",
    [1920, 240],
    {
      conditions: {
        boolean: [{ value1: "={{ $json.valid }}", value2: true }],
      },
    },
    { typeVersion: 1, notes: "判断AI响应是否为可入库的参数JSON：true=保存参数，false=进入重试/失败记录" },
  ),
  node(
    "xelora-should-retry",
    "判断是否重新调用",
    "n8n-nodes-base.if",
    [2160, 320],
    {
      conditions: {
        boolean: [
          {
            value1: "={{ $json.valid !== true && Number($json.attempt_count || 1) < 2 }}",
            value2: true,
          },
        ],
      },
    },
    { typeVersion: 1, notes: "解析失败时最多重新调用一次Xelora智能体，避免无限循环" },
  ),
  codeNode(
    "xelora-prepare-retry",
    "准备重新调用请求",
    [2400, 320],
    `if ($json.valid === true) return [{ json: $json }];
if (Number($json.attempt_count || 1) >= 2) return [{ json: $json }];
return [{
  json: {
    ...$json,
    attempt_count: Number($json.attempt_count || 1) + 1,
    retry_reason: $json.error_type || "invalid_response"
  }
}];`,
    { notes: "准备重新调用参数解析智能体的请求，保留失败原因并增加迭代次数" },
  ),
  postgresNode(
    "xelora-ensure-tables",
    "初始化Xelora参数表",
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
    { notes: "首次运行时创建Xelora并行参数解析暂存表和失败记录表（仅执行一次即可）" },
  ),
  codeNode(
    "xelora-build-insert-sql",
    "准备保存参数SQL",
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
    { notes: "将参数数组转换为批量写入SQL，准备保存到Xelora并行参数解析暂存表" },
  ),
  postgresNode(
    "xelora-insert-rows",
    "保存参数到表",
    [2640, 120],
    "={{ $json.sql_query }}",
    { notes: "将解析后的参数逐行保存到Xelora参数暂存表，如果已存在则更新" },
  ),
  codeNode(
    "xelora-build-failure-sql",
    "准备失败记录SQL",
    [2640, 440],
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
    { notes: "智能体调用或JSON解析失败时，生成失败记录SQL，便于并行流程排查" },
  ),
  postgresNode(
    "xelora-insert-failure",
    "保存失败记录",
    [2880, 440],
    "={{ $json.sql_query }}",
    { notes: "保存Xelora参数解析失败信息，不影响原有参数解析工作流并行运行" },
  ),
);

connect(workflow.connections, "Webhook接收参数", "解析Webhook参数");
connect(workflow.connections, "解析Webhook参数", "准备Xelora会话请求");
connect(workflow.connections, "准备Xelora会话请求", "创建Xelora会话");
connect(workflow.connections, "创建Xelora会话", "提取Xelora会话ID");
connect(workflow.connections, "提取Xelora会话ID", "准备Xelora智能体请求");
connect(workflow.connections, "准备Xelora智能体请求", "调用Xelora参数解析智能体");
connect(workflow.connections, "调用Xelora参数解析智能体", "处理AI响应");
connect(workflow.connections, "处理AI响应", "解析参数二维表");
connect(workflow.connections, "解析参数二维表", "判断参数解析是否成功");
connect(workflow.connections, "判断参数解析是否成功", "初始化Xelora参数表", 0);
connect(workflow.connections, "初始化Xelora参数表", "准备保存参数SQL");
connect(workflow.connections, "准备保存参数SQL", "保存参数到表");
connect(workflow.connections, "判断参数解析是否成功", "判断是否重新调用", 1);
connect(workflow.connections, "判断是否重新调用", "准备重新调用请求", 0);
connect(workflow.connections, "准备重新调用请求", "准备Xelora会话请求");
connect(workflow.connections, "判断是否重新调用", "准备失败记录SQL", 1);
connect(workflow.connections, "准备失败记录SQL", "保存失败记录");

fs.mkdirSync(path.dirname(outputPath), { recursive: true });
fs.writeFileSync(outputPath, JSON.stringify(workflow, null, 2) + "\n", "utf8");
console.log(`Wrote ${outputPath}`);
