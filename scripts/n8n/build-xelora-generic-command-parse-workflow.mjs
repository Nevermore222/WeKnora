#!/usr/bin/env node
import fs from "node:fs";
import path from "node:path";

const repoRoot = process.cwd();
const workflowDir = path.resolve(repoRoot, "..", "n8n", "参数解析");
const sourcePath = path.join(workflowDir, "Xelora版CL命令参数解析工作流（并行）_1.0.0.json");
const outputPath = path.join(workflowDir, "Xelora通用命令参数与概述解析工作流_1.0.3.json");

if (!fs.existsSync(sourcePath)) throw new Error(`Source workflow does not exist: ${sourcePath}`);
const workflow = JSON.parse(fs.readFileSync(sourcePath, "utf8"));
workflow.id = "XeloraGenericA8B247V3";
workflow.name = "Xelora通用命令参数与概述解析工作流_1.0.3";
workflow.tags = [];

const nodeByName = (name) => {
  const found = workflow.nodes.find((node) => node.name === name);
  if (!found) throw new Error(`Source workflow is missing node: ${name}`);
  return found;
};

const code = (id, name, position, jsCode, notes) => ({
  id,
  name,
  type: "n8n-nodes-base.code",
  typeVersion: 2,
  position,
  parameters: { jsCode },
  notes,
});

const postgres = (id, name, position, query, notes, credential = { id: "tWcUIJhVg6bDlGDc", name: "TargetDB" }) => ({
  id,
  name,
  type: "n8n-nodes-base.postgres",
  typeVersion: 2.1,
  position,
  parameters: { operation: "executeQuery", query, options: {} },
  credentials: { postgres: credential },
  onError: "continueErrorOutput",
  notes,
});

const webhook = nodeByName("Webhook接收参数");
webhook.parameters.path = "xelora-command-parse-generic";
webhook.webhookId = "xelora-command-parse-generic";
webhook.notes = "通用入口：command_id（可兼容接收 command）；命令、机型、语言由 command_master 关联数据解析，智能体和知识库由绑定配置解析";

const input = nodeByName("解析Webhook参数");
input.parameters.jsCode = `const input = $json.body || $json;
const commandId = Number(input.command_id);
const command = String(input.command || "").trim();

if (!Number.isInteger(commandId) || commandId <= 0) throw new Error("command_id must be a positive integer");

return [{ json: {
  command_id: commandId,
  command,
  workflow_source: "xelora_generic_command_parse",
  request_time: new Date().toISOString(),
  attempt_count: 1
} }];`;
input.notes = "要求 command_id；command、machine_type、language 从 analyzes.command_master 及关联表解析，前台无需重复传参";

const prepareBinding = code(
  "xelora-prepare-binding-query",
  "准备查询通用智能体绑定",
  [500, 240],
  `const commandId = Number($json.command_id);
const bindingQuery = \`WITH command_context AS (
  SELECT cm.command,
         cm.mac_id,
         ml.mac_name,
         COALESCE(cm_lang.language, cm.language) AS language
  FROM analyzes.command_master cm
  LEFT JOIN analyzes.mac_list ml ON ml.id = cm.mac_id
  LEFT JOIN analyzes.mac_language cm_lang ON cm_lang.id = cm.language_id
  WHERE cm.id = \${commandId}
), candidates AS (
  SELECT mla.workflow_role,
         ac.base_url,
         ac.agent_id,
         ac.api_key,
         mla.knowledge_base_ids,
         mla.knowledge_base_names,
         mla.updated_at
  FROM analyzes.mac_language_agent mla
  JOIN analyzes.agent_config ac ON ac.id = mla.agent_config_id
  JOIN command_context cc
    ON cc.mac_id = mla.mac_id
   AND lower(trim(mla.language)) = lower(trim(cc.language))
  WHERE mla.is_active = true
    AND ac.is_active = true
    AND mla.workflow_role IN ('command_parameter_parse', 'command_overview_parse')
), parameter AS (
  SELECT * FROM candidates WHERE workflow_role = 'command_parameter_parse'
  ORDER BY updated_at DESC NULLS LAST LIMIT 1
), overview AS (
  SELECT * FROM candidates WHERE workflow_role = 'command_overview_parse'
  ORDER BY updated_at DESC NULLS LAST LIMIT 1
)
SELECT
  cc.command AS command_name,
  cc.mac_name AS command_machine_type,
  cc.language AS command_language,
  p.base_url AS parameter_base_url,
  p.agent_id AS parameter_agent_id,
  p.api_key AS parameter_api_key,
  p.knowledge_base_ids AS parameter_kb_ids,
  p.knowledge_base_names->>0 AS parameter_kb_name,
  o.base_url AS overview_base_url,
  o.agent_id AS overview_agent_id,
  o.api_key AS overview_api_key,
  o.knowledge_base_ids AS overview_kb_ids,
  o.knowledge_base_names->>0 AS overview_kb_name
FROM (SELECT 1) seed
LEFT JOIN command_context cc ON TRUE
LEFT JOIN parameter p ON TRUE
LEFT JOIN overview o ON TRUE;\`;
return [{ json: { ...$json, binding_query: bindingQuery } }];`,
  "按机型、语言和功能角色从 srflow 配置表读取智能体、API Key 与知识库",
);

const resolveBinding = postgres(
  "xelora-resolve-binding",
  "查询通用智能体与知识库绑定",
  [760, 240],
  "={{ $json.binding_query }}",
  "从 analyzes.mac_language_agent 和 analyzes.agent_config 读取两类功能绑定",
  { id: "tWcUIJhVg6bDlGDc", name: "TargetDB" },
);

const normalizeBinding = code(
  "xelora-normalize-binding",
  "校验并整理通用绑定",
  [1000, 240],
  `const base = $("解析Webhook参数").first().json;
const row = $input.first().json || {};
function ids(value) {
  if (Array.isArray(value)) return value.map(String).filter(Boolean);
  if (value && Array.isArray(value.array)) return value.array.map(String).filter(Boolean);
  if (typeof value === "string") {
    try { return ids(JSON.parse(value)); } catch { return value ? [value] : []; }
  }
  return [];
}
const parameterKbIds = ids(row.parameter_kb_ids);
const overviewKbIds = ids(row.overview_kb_ids);
const missing = [];
if (!row.command_name || !row.command_machine_type || !row.command_language) missing.push("command_context");
if (!row.parameter_base_url || !row.parameter_agent_id || !row.parameter_api_key || !parameterKbIds.length) missing.push("command_parameter_parse");
if (!row.overview_base_url || !row.overview_agent_id || !row.overview_api_key || !overviewKbIds.length) missing.push("command_overview_parse");
if (missing.length) return [{ json: {
  ...base,
  command: String(row.command_name || base.command || ""),
  machine_type: String(row.command_machine_type || ""),
  language: String(row.command_language || ""),
  valid: false,
  attempt_count: 2,
  error_type: "workflow_binding_not_found",
  error_message: "No command context or agent/knowledge-base binding found for command_id=" + base.command_id + ": " + missing.join(", ")
} }];
return [{ json: {
  ...base,
  command: String(row.command_name),
  machine_type: String(row.command_machine_type),
  language: String(row.command_language),
  parameter_base_url: String(row.parameter_base_url).replace(/\\\/$/, ""),
  parameter_agent_id: String(row.parameter_agent_id),
  parameter_api_key: String(row.parameter_api_key),
  parameter_kb_ids: parameterKbIds,
  parameter_kb_name: String(row.parameter_kb_name || parameterKbIds[0]),
  overview_base_url: String(row.overview_base_url).replace(/\\\/$/, ""),
  overview_agent_id: String(row.overview_agent_id),
  overview_api_key: String(row.overview_api_key),
  overview_kb_ids: overviewKbIds,
  overview_kb_name: String(row.overview_kb_name || overviewKbIds[0])
} }];`,
  "缺少任一功能角色绑定时进入统一失败记录，不静默回退到旧 ASP/CL 智能体",
);

workflow.nodes.push(prepareBinding, resolveBinding, normalizeBinding);
const bindingValid = {
  id: "xelora-binding-valid-if",
  name: "判断通用绑定是否完整",
  type: "n8n-nodes-base.if",
  typeVersion: 1,
  position: [1200, 240],
  parameters: { conditions: { boolean: [{ value1: "={{ $json.valid !== false }}", value2: true }] } },
  notes: "绑定完整才创建 Xelora 会话；缺失时直接保存失败记录",
};
workflow.nodes.push(bindingValid);
const connections = workflow.connections;
connections["解析Webhook参数"] = { main: [[{ node: prepareBinding.name, type: "main", index: 0 }]] };
connections[prepareBinding.name] = { main: [[{ node: resolveBinding.name, type: "main", index: 0 }]] };
connections[resolveBinding.name] = { main: [[{ node: normalizeBinding.name, type: "main", index: 0 }]] };
connections[normalizeBinding.name] = { main: [[{ node: bindingValid.name, type: "main", index: 0 }]] };
connections[bindingValid.name] = { main: [
  [{ node: "准备Xelora会话请求", type: "main", index: 0 }],
  [{ node: "准备失败记录SQL", type: "main", index: 0 }],
] };

const session = nodeByName("准备Xelora会话请求");
session.parameters.jsCode = `const apiBase = String($json.parameter_base_url || "").replace(/\\\/$/, "");
return [{ json: { ...$json, xelora_api_base_url: apiBase, session_url: apiBase + "/sessions", session_request_body: {
  title: $json.machine_type + " " + $json.language + " command parameter parse " + $json.command
} } }];`;

const parameterRequest = nodeByName("准备Xelora智能体请求");
parameterRequest.parameters.jsCode = `const kbName = $json.parameter_kb_name || $json.parameter_kb_ids[0];
const query = [
  "请基于" + kbName + "知识库深度解析" + $json.machine_type + "机型的" + $json.language + "命令" + $json.command + "的全部参数。",
  "必须输出可直接落入前台参数详情表的高质量数据：参数说明、取值范围、默认值、必填性、枚举值和参数间关系尽量从知识库抽取。",
  "description和relationship_notes使用中文说明；可保留必要的原文术语、参数名、关键字和示例。",
  $json.retry_reason ? ("上一次输出未通过校验，失败原因：" + $json.retry_reason + "。本次必须修正。") : "",
  "Return exactly one JSON object and no Markdown.",
  "Schema: {command, language, parameters:[{parameter_name, parameter_type, data_type, enum_value, value_range, default_value, description, is_required, relationship_notes}]}"
].join(" ");
return [{ json: { ...$json, agent_request_body: {
  query,
  agent_enabled: true,
  agent_id: $json.parameter_agent_id,
  knowledge_base_ids: $json.parameter_kb_ids,
  mentioned_items: [{ id: $json.parameter_kb_ids[0], name: kbName, type: "kb", kb_type: "document" }],
  web_search_enabled: false,
  channel: "api"
} } }];`;
parameterRequest.notes = "使用数据库解析出的参数解析智能体和知识库，不依赖固定环境变量";

const parameterCall = nodeByName("调用Xelora参数解析智能体");
parameterCall.notes = "调用当前机型/语言绑定的参数解析智能体";
const markdown = nodeByName("生成参数二维表Markdown");
markdown.parameters.jsCode = `function cell(value) {
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

const machineType = String(source.machine_type || "");
const language = String(source.language || "");
return [{
  json: {
    ...source,
    parameter_table_markdown: markdown,
    overview_session_url: source.overview_base_url + "/sessions",
    overview_session_request_body: {
      title: machineType + " " + language + " command overview " + source.command
    }
  }
}];`;

const overviewRequest = nodeByName("准备概述智能体请求");
overviewRequest.parameters.jsCode = `const prior = $("生成参数二维表Markdown").first().json;
const response = $input.first().json;
const sessionId = response.session_id || response.id || response.data?.session_id || response.data?.id || response.result?.session_id || response.result?.id;
if (!sessionId) return [{ json: { ...prior, overview_status: "failed", overview_error_message: "overview session create failed: " + JSON.stringify(response).slice(0, 2000) } }];
const kbName = prior.overview_kb_name || prior.overview_kb_ids[0];
const overviewQuery = [
  "请基于" + kbName + "知识库和以下参数二维表，生成" + prior.machine_type + "机型" + prior.language + "命令" + prior.command + "的完整Markdown概述文档。",
  "输出命令介绍、命令格式、系统要求、参数含义、注意事项、使用示例、終了代码、相关命令和最佳实践。",
  "必须理解知识库原文后重写，不要复制版式字符或乱码；没有明确依据的内容请明确说明未检索到。",
  "参数二维表如下：\\n" + prior.parameter_table_markdown
].join("\\n\\n");
return [{ json: { ...prior, overview_session_id: String(sessionId), overview_agent_url: prior.overview_base_url + "/agent-chat/" + String(sessionId), overview_agent_request_body: {
  query: overviewQuery,
  agent_enabled: true,
  agent_id: prior.overview_agent_id,
  knowledge_base_ids: prior.overview_kb_ids,
  mentioned_items: [{ id: prior.overview_kb_ids[0], name: kbName, type: "kb", kb_type: "document" }],
  web_search_enabled: false,
  channel: "api"
} } }];`;
overviewRequest.notes = "使用数据库解析出的概述智能体和知识库，不依赖固定 CL/ASP 环境变量";
nodeByName("调用概述智能体").notes = "调用当前机型/语言绑定的概述解析智能体";

for (const httpNode of workflow.nodes.filter((node) => node.type === "n8n-nodes-base.httpRequest")) {
  const headers = httpNode.parameters?.headerParameters?.parameters || [];
  const isOverview = ["创建概述会话", "调用概述智能体"].includes(httpNode.name);
  for (const header of headers) {
    if (String(header.name).toLowerCase() === "x-api-key") {
      header.value = isOverview ? "={{ $json.overview_api_key }}" : "={{ $json.parameter_api_key }}";
    }
  }
  httpNode.parameters.options = {
    ...(httpNode.parameters.options || {}),
    timeout: 600000,
  };
}

fs.writeFileSync(outputPath, JSON.stringify(workflow, null, 2) + "\n", "utf8");
console.log(`Wrote ${outputPath}`);
