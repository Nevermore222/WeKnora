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
  "请基于Manual_ASP知识库深度解析CL命令" + $json.command + "的全部参数。",
  "必须输出可直接落入前台参数详情表的高质量数据：参数说明要完整，取值范围、默认值、必填性、枚举值和参数间关系必须尽量从知识库抽取。",
  "description字段需要说明用途、指定规则、格式/长度/类型约束、典型值或注意事项；relationship_notes字段需要说明依赖、互斥、联动、前置条件、错误条件和与其他参数/系统变量的关系。",
  "最终JSON面向中文前台系统：除JSON字段名、命令名、参数名、枚举关键字、data_type原文类型名外，所有说明性内容都必须使用中文。",
  "description、relationship_notes、value_range、default_value必须使用中文句子，禁止输出日文说明、日文句子、平假名或片假名。",
  "Manual中的日文概念必须翻译成中文，例如ワンタッチ記述名翻译为一键描述名、ライブラリ名翻译为库名、省略時翻译为省略时。",
  "不要把Manual_ASP中的日文说明句直接复制到description、relationship_notes、value_range或default_value；必须先翻译并整理为中文。",
  "严禁输出OCR/编码乱码片段，例如縺、繧、譁、蜿、蛹、譛、隸、螟、莉、荳、逧、窶、ï、þ、�。如果检索到的Manual_ASP内容存在乱码，必须理解后改写为中文；无法可靠理解时对应字段留空，不能复制乱码。",
  $json.retry_reason ? ("上一次输出未通过校验，失败原因：" + $json.retry_reason + "。本次必须修正后再输出。") : "",
  "data_type字段必须尽量使用Manual_ASP原文中的日语类型/分类名称，例如名前型、文字ストリング型、整数型、論理型等；不要把原文类型泛化成STRING、NUMBER、BOOLEAN。",
  "不要只输出简单英文短句或日文短句；除参数名、关键字、枚举值和data_type原文外，其他说明性内容必须使用中文。",
  "如果知识库存在证据，不得把description、value_range、default_value、relationship_notes留空。",
  "Return exactly one JSON object and no Markdown.",
  "输出第一个字符必须是{，最后一个字符必须是}；禁止输出Markdown代码块围栏或json代码块包装。",
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
  const responseType = String(event.response_type || event.event || event.type || "").toLowerCase();
  const isAnswerChunk =
    responseType === "" ||
    responseType === "answer" ||
    responseType === "message" ||
    responseType === "agent_message" ||
    responseType === "final_answer" ||
    responseType === "final_answer_chunk";
  const isNonAnswerEvent =
    responseType === "agent_query" ||
    responseType === "tool_call" ||
    responseType === "tool_result" ||
    responseType === "thinking" ||
    responseType === "complete" ||
    responseType === "error";
  if (typeof event.content === "string" && isAnswerChunk && !isNonAnswerEvent) answer += event.content;
  if (responseType === "complete" || event.done === true) sawComplete = true;
  if (responseType === "error" && event.content && !answer) answer = JSON.stringify(event);
}

function parseSseText(text) {
  for (const line of String(text || "").split("\\n")) {
    const trimmed = line.trimStart();
    if (!trimmed.startsWith("data:")) continue;
    const payload = trimmed.slice(5).trim();
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
}

if (typeof response === "string") {
  parseSseText(response);
} else if (typeof response.data === "string") {
  parseSseText(response.data);
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

function countMatches(text, pattern) {
  const matches = String(text || "").match(pattern);
  return matches ? matches.length : 0;
}

function hasChineseExplanation(text) {
  return /用于|如果|需要|必须|可以|不能|表示|说明|关系|依赖|互斥|影响|默认|取值|参数|文件|命令|删除|条件|存在|省略|否则|因此|前置|错误|当.+时/.test(String(text || ""));
}

function hasJapaneseSentence(text) {
  const value = String(text || "");
  const kanaCount = countMatches(value, /[\u3040-\u30ff]/g);
  return kanaCount >= 10 || /します|ください|場合|対象|省略時|指定した|指定します|異なります|できません|必要です/.test(value);
}

function hasGarbledText(text) {
  return /�|ï|þ|ü|蜻|譁|縺|繧|莠|螂|窶|讎|閭|菴|荳|隸|譛|蜿|蠑|逧|莉|蛹|螟/.test(String(text || ""));
}

function needsChineseRewrite(text) {
  const value = String(text || "").trim();
  if (!value) return false;
  return hasGarbledText(value) || (hasJapaneseSentence(value) && !hasChineseExplanation(value));
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

const nonChineseFields = [];
for (const parameter of parameters) {
  for (const field of ["description", "relationship_notes", "value_range", "default_value"]) {
    if (needsChineseRewrite(parameter[field])) {
      nonChineseFields.push((parameter.parameter_name || "<unknown>") + "." + field);
    }
  }
}

if (nonChineseFields.length) {
  return [{
    json: {
      ...$json,
      valid: false,
      error_type: "non_chinese_explanation",
      error_message: "description/relationship_notes/value_range/default_value must be Chinese and must not contain Japanese explanatory sentences or garbled text: " + nonChineseFields.slice(0, 20).join(", "),
      parsed_command: normalizeString(parsed.command) || $json.command,
      parsed_language: normalizeString(parsed.language) || $json.language,
      parameters,
      no_parameters_found: parameters.length === 0
    }
  }];
}

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
  \${sqlString(parameter.parameter_name)},
  \${sqlString(parameter.parameter_type)},
  \${sqlString(parameter.data_type)},
  \${sqlString(parameter.enum_value)},
  \${sqlString(parameter.value_range)},
  \${sqlString(parameter.default_value)},
  \${Number(parameter.display_order || 0)},
  \${sqlString(parameter.description)},
  \${sqlBoolean(parameter.is_required)},
  \${sqlString(parameter.relationship_notes)}
)\`);

if (!rows.length) {
  return [{
    json: {
      ...$json,
      sql_query: \`DELETE FROM analyzes.command_parameters
WHERE command_id = \${Number($json.command_id)};\`
    }
  }];
}

return [{
  json: {
    ...$json,
    sql_query: \`DELETE FROM analyzes.command_parameters
WHERE command_id = \${Number($json.command_id)};

INSERT INTO analyzes.command_parameters (
  command_id, parameter_name, parameter_type, data_type,
  enum_value, value_range, default_value, display_order, description,
  is_required, relationship_notes
) VALUES \${rows.join(",\\n")}
ON CONFLICT (command_id, parameter_name) DO UPDATE SET
  parameter_type = EXCLUDED.parameter_type,
  data_type = EXCLUDED.data_type,
  enum_value = EXCLUDED.enum_value,
  value_range = EXCLUDED.value_range,
  default_value = EXCLUDED.default_value,
  display_order = EXCLUDED.display_order,
  description = EXCLUDED.description,
  is_required = EXCLUDED.is_required,
  relationship_notes = EXCLUDED.relationship_notes,
  updated_at = NOW();\`
  }
}];`,
    { notes: "将参数数组转换为批量写入SQL，准备保存到正式command_parameters表" },
  ),
  postgresNode(
    "xelora-insert-rows",
    "保存参数到表",
    [2640, 120],
    "={{ $json.sql_query }}",
    { notes: "将解析后的参数逐行保存到command_parameters表，如果已存在则更新" },
  ),
  codeNode(
    "xelora-build-parameter-markdown",
    "生成参数二维表Markdown",
    [2880, 120],
    `function cell(value) {
  const text = value === null || value === undefined || value === "" ? "-" : String(value);
  return text.replace(/\\|/g, "\\\\|").replace(/\\r?\\n/g, "<br>");
}

const source = $("解析参数二维表").first().json;
const parameters = Array.isArray(source.parameters) ? source.parameters : [];
let markdown = "| 参数名称 | 参数类型 | 数据类型 | 枚举值 | 取值范围 | 默认值 | 参数描述 | 是否必需 | 参数关系备注 |\\n";
markdown += "|---|---|---|---|---|---|---|---|---|\\n";
for (const parameter of parameters) {
  markdown += "| " + [
    cell(parameter.parameter_name),
    cell(parameter.parameter_type),
    cell(parameter.data_type),
    cell(parameter.enum_value),
    cell(parameter.value_range),
    cell(parameter.default_value),
    cell(parameter.description),
    parameter.is_required ? "是" : "否",
    cell(parameter.relationship_notes)
  ].join(" | ") + " |\\n";
}

return [{
  json: {
    ...source,
    parameter_table_markdown: markdown,
    overview_session_url: source.xelora_api_base_url + "/sessions",
    overview_session_request_body: {
      title: "CL command overview " + source.command
    }
  }
}];`,
    { notes: "参数保存后，将解析结果转换为Markdown参数二维表，供Xelora概述智能体生成detail_info" },
  ),
  node(
    "xelora-create-overview-session",
    "创建概述会话",
    "n8n-nodes-base.httpRequest",
    [3120, 120],
    {
      method: "POST",
      url: "={{ $json.overview_session_url }}",
      sendHeaders: true,
      headerParameters: {
        parameters: [
          { name: "X-API-Key", value: "={{ $env.N8N_XELORA_API_KEY }}" },
          { name: "Content-Type", value: "application/json" },
        ],
      },
      sendBody: true,
      specifyBody: "json",
      jsonBody: "={{ $json.overview_session_request_body }}",
      options: { response: { response: { neverError: true } } },
    },
    {
      typeVersion: 4.2,
      onError: "continueErrorOutput",
      notes: "为CL命令概述智能体创建独立Xelora会话",
    },
  ),
  codeNode(
    "xelora-prepare-overview-agent",
    "准备概述智能体请求",
    [3360, 120],
    `const prior = $("生成参数二维表Markdown").first().json;
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
      overview_status: "failed",
      overview_error_message: "overview session create failed: " + JSON.stringify(response).slice(0, 2000)
    }
  }];
}

const query = [
  "请基于Manual_ASP知识库和以下参数二维表，生成CL命令" + prior.command + "的完整Markdown概述文档。",
  "输出格式必须与CL命令概述智能体要求一致：命令介绍、命令格式、系统要求、参数含义介绍、注意事项、使用示例、終了代码、相关命令、最佳实践。",
  "終了代码模块按照Manual_ASP原文中的集合形式输出即可，例如直接列出0000、0137等代码集合；不要生成包含含义或处理建议的二维表。没有明确依据时说明未检索到，不要套用通用代码表。",
  "参数二维表如下：\\n" + prior.parameter_table_markdown
].join("\\n\\n");

const overviewQuery = [
  "请基于Manual_ASP知识库和以下参数二维表，生成CL命令" + prior.command + "的完整Markdown概述文档。",
  "必须理解Manual原文后重写，不要复制PDF/扫描版中的框图、错位空格、竖排花括号、表格边框或OCR残留字符。",
  "命令格式模块必须输出重写后的规范化语法：命令名和参数名使用正常连续写法；可选参数用[]；多选值用{ A | B | C }；不要输出R S T M B R这类分隔字母。",
  "如果Manual原文的命令格式区域出现乱码或版式字符，请根据参数含义、可选项、互斥关系和示例重构干净语法，不要尝试复刻原图。",
  "禁止输出明显乱码或抽取残留字符，例如�、ï、þ、ü、蜻、譁、縺、繧、莠、螂、窶等。",
  "输出格式必须与CL命令概述智能体要求一致：命令介绍、命令格式、系统要求、参数含义介绍、注意事项、使用示例、終了代码、相关命令、最佳实践。",
  "終了代码模块按照Manual_ASP原文中的集合形式输出即可，例如直接列出0000、0137等代码集合；不要生成包含含义或处理建议的二维表。没有明确依据时说明未检索到，不要套用通用代码表。",
  "参数二维表如下：\\n" + prior.parameter_table_markdown
].join("\\n\\n");

return [{
  json: {
    ...prior,
    overview_session_id: String(sessionId),
    overview_agent_url: prior.xelora_api_base_url + "/agent-chat/" + String(sessionId),
    overview_agent_request_body: {
      query: overviewQuery,
      agent_enabled: true,
      agent_id: $env.XELORA_CL_OVERVIEW_AGENT_ID,
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
    { notes: "准备调用Xelora CL命令概述智能体；传入参数二维表并以Manual_ASP知识库补全概述" },
  ),
  node(
    "xelora-call-overview-agent",
    "调用概述智能体",
    "n8n-nodes-base.httpRequest",
    [3600, 120],
    {
      method: "POST",
      url: "={{ $json.overview_agent_url }}",
      sendHeaders: true,
      headerParameters: {
        parameters: [
          { name: "X-API-Key", value: "={{ $env.N8N_XELORA_API_KEY }}" },
          { name: "Content-Type", value: "application/json" },
        ],
      },
      sendBody: true,
      specifyBody: "json",
      jsonBody: "={{ $json.overview_agent_request_body }}",
      options: { response: { response: { neverError: true } } },
    },
    {
      typeVersion: 4.2,
      onError: "continueErrorOutput",
      notes: "调用Xelora CL命令概述智能体，获取Markdown命令概述",
    },
  ),
  codeNode(
    "xelora-parse-overview-stream",
    "处理概述响应",
    [3840, 120],
    `const requestData = $("准备概述智能体请求").first().json;
const response = $input.first().json;
let answer = "";
let sawComplete = false;

function absorbEvent(event) {
  if (!event || typeof event !== "object") return;
  if (typeof event.answer === "string") answer += event.answer;
  const responseType = String(event.response_type || event.event || event.type || "").toLowerCase();
  const isAnswerChunk =
    responseType === "" ||
    responseType === "answer" ||
    responseType === "message" ||
    responseType === "agent_message" ||
    responseType === "final_answer" ||
    responseType === "final_answer_chunk";
  const isNonAnswerEvent =
    responseType === "agent_query" ||
    responseType === "tool_call" ||
    responseType === "tool_result" ||
    responseType === "thinking" ||
    responseType === "complete" ||
    responseType === "error";
  if (typeof event.content === "string" && isAnswerChunk && !isNonAnswerEvent) answer += event.content;
  if (responseType === "complete" || event.done === true) sawComplete = true;
  if (responseType === "error" && event.content && !answer) answer = JSON.stringify(event);
}

function parseSseText(text) {
  for (const line of String(text || "").split("\\n")) {
    const trimmed = line.trimStart();
    if (!trimmed.startsWith("data:")) continue;
    const payload = trimmed.slice(5).trim();
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
}

if (typeof response === "string") {
  parseSseText(response);
} else if (typeof response.data === "string") {
  parseSseText(response.data);
} else {
  absorbEvent(response);
  absorbEvent(response.data);
  absorbEvent(response.result);
}

const overview = answer.trim();
function hasGarbledText(text) {
  const value = String(text || "");
  return /�|ï|þ|ü|蜻|譁|縺|繧|莠|螂|窶|讎|閭|菴|荳|隸|譛|蜿|蠑|逧|莉/.test(value);
}
const overview_has_garbled_text = hasGarbledText(overview);
const cleanOverview = overview_has_garbled_text ? "" : overview;
return [{
  json: {
    ...requestData,
    overview_response: cleanOverview,
    overview_status: cleanOverview ? "success" : "failed",
    overview_error_message: cleanOverview
      ? ""
      : overview_has_garbled_text
        ? "overview response contains garbled text; detail_info was not updated"
        : "overview response is empty: " + JSON.stringify(response).slice(0, 2000),
    overview_has_garbled_text,
    overview_stream_complete: sawComplete,
    overview_response_time: new Date().toISOString()
  }
}];`,
    { notes: "解析Xelora概述智能体SSE响应，提取Markdown概述正文" },
  ),
  codeNode(
    "xelora-build-overview-update-sql",
    "准备更新命令主表SQL",
    [4080, 120],
    `function sqlString(value) {
  if (value === null || value === undefined || value === "") return "NULL";
  return "'" + String(value).replace(/'/g, "''").replace(/\\\\/g, "\\\\\\\\") + "'";
}

const detailInfo = $json.overview_response || "";
if (!detailInfo.trim()) {
  return [{
    json: {
      ...$json,
      sql_query: "SELECT 1 AS overview_empty;"
    }
  }];
}

return [{
  json: {
    ...$json,
    sql_query: \`UPDATE analyzes.command_master
SET detail_info = \${sqlString(detailInfo)}::text,
    updated_at = CURRENT_TIMESTAMP
WHERE id = \${Number($json.command_id)};\`
  }
}];`,
    { notes: "将概述Markdown转换为更新command_master.detail_info的SQL；不建表不改结构" },
  ),
  postgresNode(
    "xelora-update-command-master",
    "更新命令主表",
    [4320, 120],
    "={{ $json.sql_query }}",
    { notes: "将概述智能体生成的Markdown写入analyzes.command_master.detail_info" },
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
connect(workflow.connections, "判断参数解析是否成功", "准备保存参数SQL", 0);
connect(workflow.connections, "准备保存参数SQL", "保存参数到表");
connect(workflow.connections, "保存参数到表", "生成参数二维表Markdown");
connect(workflow.connections, "生成参数二维表Markdown", "创建概述会话");
connect(workflow.connections, "创建概述会话", "准备概述智能体请求");
connect(workflow.connections, "准备概述智能体请求", "调用概述智能体");
connect(workflow.connections, "调用概述智能体", "处理概述响应");
connect(workflow.connections, "处理概述响应", "准备更新命令主表SQL");
connect(workflow.connections, "准备更新命令主表SQL", "更新命令主表");
connect(workflow.connections, "判断参数解析是否成功", "判断是否重新调用", 1);
connect(workflow.connections, "判断是否重新调用", "准备重新调用请求", 0);
connect(workflow.connections, "准备重新调用请求", "准备Xelora会话请求");
connect(workflow.connections, "判断是否重新调用", "准备失败记录SQL", 1);
connect(workflow.connections, "准备失败记录SQL", "保存失败记录");

fs.mkdirSync(path.dirname(outputPath), { recursive: true });
fs.writeFileSync(outputPath, JSON.stringify(workflow, null, 2) + "\n", "utf8");
console.log(`Wrote ${outputPath}`);
