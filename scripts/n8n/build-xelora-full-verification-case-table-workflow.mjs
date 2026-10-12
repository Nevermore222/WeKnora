#!/usr/bin/env node
import fs from "node:fs";
import path from "node:path";

const repoRoot = process.cwd();
const workflowDir = path.resolve(repoRoot, "..", "n8n", "二维表");
const sourcePath = path.join(workflowDir, "CL命令检证用例二维表工作流（评估）_1.5.0_1.json");
const outputPath = path.join(workflowDir, "Xelora通用全量检证用例二维表工作流_1.0.0.json");

if (!fs.existsSync(sourcePath)) throw new Error(`Source workflow does not exist: ${sourcePath}`);
const workflow = JSON.parse(fs.readFileSync(sourcePath, "utf8"));
workflow.id = "XeloraFullVerTableA8B247V1";
workflow.name = "Xelora通用全量检证用例二维表工作流_1.0.0";
workflow.active = false;
workflow.tags = [{ name: "Xelora" }, { name: "通用" }, { name: "全量检证用例二维表" }];

const nodeByName = (name) => {
  const node = workflow.nodes.find((item) => item.name === name);
  if (!node) throw new Error(`Missing source node: ${name}`);
  return node;
};

const nodeByPrefix = (prefix) => {
  const node = workflow.nodes.find((item) => item.name.startsWith(prefix));
  if (!node) throw new Error(`Missing source node prefix: ${prefix}`);
  return node;
};

const codeNode = (id, name, position, jsCode, notes = "") => ({
  id,
  name,
  type: "n8n-nodes-base.code",
  typeVersion: 2,
  position,
  parameters: { jsCode },
  notes,
});

const postgresNode = (id, name, position, query, notes = "") => ({
  id,
  name,
  type: "n8n-nodes-base.postgres",
  typeVersion: 2.1,
  position,
  parameters: { operation: "executeQuery", query, options: {} },
  credentials: { postgres: { id: "tWcUIJhVg6bDlGDc", name: "TargetDB" } },
  onError: "stopWorkflow",
  notes,
});

const httpNode = (id, name, position, url, bodyExpression, notes = "") => ({
  id,
  name,
  type: "n8n-nodes-base.httpRequest",
  typeVersion: 4.2,
  position,
  parameters: {
    method: "POST",
    url,
    sendHeaders: true,
    headerParameters: { parameters: [
      { name: "X-API-Key", value: "={{ $json.xelora_api_key }}" },
      { name: "Content-Type", value: "application/json" },
    ] },
    sendBody: true,
    specifyBody: "json",
    jsonBody: bodyExpression,
    options: { timeout: 600000, response: { response: { neverError: true } } },
  },
  onError: "stopWorkflow",
  notes,
});

const webhook = nodeByName("Webhook接收参数");
webhook.parameters.path = "xelora-full-verification-case-table";
webhook.webhookId = "xelora-full-verification-case-table";
webhook.notes = "全量入口：只要求 command_id；命令、机型、语言、二维表智能体和知识库由数据库绑定解析";

const input = nodeByName("解析Webhook参数");
input.type = "n8n-nodes-base.code";
input.typeVersion = 2;
input.parameters = { jsCode: String.raw`const b = $json.body || $json;
const commandId = Number(b.command_id ?? b.commandId ?? b.command_master_id);
if (!Number.isInteger(commandId) || commandId <= 0) throw new Error("command_id must be a positive integer");
return [{ json: {
  command_id: commandId,
  command: String(b.command || "").trim(),
  language: String(b.language || "").trim(),
  workflow_source: "xelora_full_verification_case_table",
  request_time: new Date().toISOString()
} }];` };
input.notes = "仅强制 command_id；其余上下文由 command_master、mac_list、mac_language 和智能体绑定表查询";

const prepareBinding = codeNode(
  "xelora-full-prepare-binding",
  "准备查询全量二维表绑定",
  [420, 260],
  String.raw`const commandId = Number($json.command_id);
const bindingQuery = "WITH command_context AS (" +
  " SELECT cm.id AS command_id, cm.command, cm.mac_id, ml.mac_name," +
  " COALESCE(mlang.language, cm.language) AS language" +
  " FROM analyzes.command_master cm" +
  " LEFT JOIN analyzes.mac_list ml ON ml.id = cm.mac_id" +
  " LEFT JOIN analyzes.mac_language mlang ON mlang.id = cm.language_id" +
  " WHERE cm.id = " + commandId +
  "), binding AS (" +
  " SELECT mla.workflow_role, ac.base_url, ac.agent_id, ac.api_key," +
  " mla.knowledge_base_ids, mla.knowledge_base_names, mla.updated_at" +
  " FROM analyzes.mac_language_agent mla" +
  " JOIN analyzes.agent_config ac ON ac.id = mla.agent_config_id" +
  " JOIN command_context cc ON cc.mac_id = mla.mac_id" +
  " AND lower(trim(mla.language)) = lower(trim(cc.language))" +
  " WHERE mla.workflow_role = 'verification_case_table_parse'" +
  " AND mla.is_active = true AND ac.is_active = true" +
  " ORDER BY mla.updated_at DESC NULLS LAST LIMIT 1" +
  ") SELECT cc.command_id, cc.command, cc.mac_name AS machine_type, cc.language," +
  " b.base_url, b.agent_id, b.api_key, b.knowledge_base_ids, b.knowledge_base_names" +
  " FROM command_context cc LEFT JOIN binding b ON TRUE;";
return [{ json: { ...$json, binding_query: bindingQuery } }];`,
  "按 command_id 解析机型、语言和 verification_case_table_parse 绑定，不回退旧 ASP/CL 智能体",
);

const resolveBinding = postgresNode(
  "xelora-full-resolve-binding",
  "查询全量二维表绑定",
  [700, 260],
  "={{ $json.binding_query }}",
  "从 analyzes.mac_language_agent 和 analyzes.agent_config 读取全量二维表生成智能体与知识库",
);

const validateBinding = codeNode(
  "xelora-full-validate-binding",
  "校验全量二维表绑定",
  [960, 260],
  String.raw`const base = $("解析Webhook参数").first().json;
const row = $input.first()?.json || {};
function ids(value) {
  if (Array.isArray(value)) return value.map(String).filter(Boolean);
  if (value && Array.isArray(value.array)) return value.array.map(String).filter(Boolean);
  if (typeof value === "string") { try { return ids(JSON.parse(value)); } catch { return value ? [value] : []; } }
  return [];
}
const kbIds = ids(row.knowledge_base_ids);
const missing = [];
if (!row.command || !row.machine_type || !row.language) missing.push("command_context");
if (!row.base_url || !row.agent_id || !row.api_key || !kbIds.length) missing.push("verification_case_table_parse");
if (missing.length) throw new Error("全量二维表绑定缺失 command_id=" + base.command_id + ": " + missing.join(", "));
return [{ json: {
  ...base,
  command: String(row.command), machine_type: String(row.machine_type), language: String(row.language),
  xelora_base_url: String(row.base_url).replace(/\/$/, ""), xelora_agent_id: String(row.agent_id),
  xelora_api_key: String(row.api_key), knowledge_base_ids: kbIds,
  knowledge_base_name: String((Array.isArray(row.knowledge_base_names) ? row.knowledge_base_names[0] : row.knowledge_base_names) || kbIds[0])
} }];`,
  "绑定不完整时立即失败，禁止伪造成功或进入第二阶段流程",
);

const queryParameters = nodeByName("查询参数解析表");
queryParameters.parameters.query = "SELECT command_id, parameter_name, parameter_type, data_type, enum_value, value_range, default_value, display_order, description, is_required, relationship_notes FROM analyzes.command_parameters WHERE command_id = {{ $('解析Webhook参数').first().json.command_id }} ORDER BY display_order ASC, parameter_name ASC;";
queryParameters.onError = "stopWorkflow";

const hasParameters = nodeByName("判断是否有参数数据");
hasParameters.parameters.conditions = { number: [{ value1: '={{ $items("查询参数解析表").length }}', operation: "larger" }] };

const queryReturnCodes = nodeByName("查询CL終了コード");
queryReturnCodes.parameters.query = String.raw`WITH cmd AS (
  SELECT command_name, return_codes FROM analyzes.asp_cl_commands
  WHERE UPPER(TRIM(command_name)) = UPPER(TRIM('{{ $('校验全量二维表绑定').first().json.command.replace(/'/g, "''") }}')) LIMIT 1
), cmd_codes AS (
  SELECT DISTINCT (m)[1] AS return_code FROM cmd,
  LATERAL regexp_matches(COALESCE(return_codes, ''), '\\d{4}', 'g') AS m
), priority_codes AS (
  SELECT prc.return_code FROM analyzes.priority_return_codes prc
  JOIN cmd_codes cc ON cc.return_code = prc.return_code WHERE prc.is_active = true
)
SELECT cmd.command_name, cmd.return_codes,
  COALESCE((SELECT string_agg(return_code::text, ',' ORDER BY return_code) FROM priority_codes), '') AS priority_return_codes
FROM cmd;`;
queryReturnCodes.onError = "continueErrorOutput";

const assembleCodes = nodeByName("组装終了コード查询");
assembleCodes.parameters.jsCode = String.raw`const row = $input.first()?.json || {};
const raw = String(row.priority_return_codes || "");
const codes = [...new Set(raw.match(/\d{4}/g) || [])].sort();
return [{ json: { command: $("校验全量二维表绑定").first().json.command, return_code_list: codes.join(","), return_codes: codes, return_code_source: codes.length ? "priority" : "none", message_codes_in_clause: codes.map(c => "'S" + c + "'").join(","), has_return_codes: codes.length > 0 } }];`;
const queryMessages = nodeByName("查询系统消息原文");
queryMessages.parameters.query = "={{ $json.has_return_codes ? \"SELECT code, raw_content FROM analyzes.asp_system_messages WHERE code IN (\" + $json.message_codes_in_clause + \") ORDER BY code;\" : 'SELECT NULL::text AS code, NULL::text AS raw_content WHERE false;' }}";
const mergeCodes = nodeByName("合并終了コード原文");
mergeCodes.parameters.jsCode = String.raw`const base = $("组装終了コード查询").first().json;
const rows = $input.all().map(i => i.json).filter(r => r && r.code);
const map = new Map(rows.map(r => [String(r.code).toUpperCase(), r.raw_content || ""]));
let raw = "";
for (const code of base.return_codes || []) raw += "### [" + code + "] S" + code + "\n" + (map.get("S" + code) || "[メッセージ未登録]") + "\n\n";
return [{ json: { ...base, return_code_messages_raw: raw.trim() } }];`;

const merge = workflow.nodes.find((item) => item.type === "n8n-nodes-base.merge" && item.name.includes("参数与"));
if (!merge) throw new Error("Missing parameter/return-code merge node");

const format = nodeByName("格式化参数二维表");
format.parameters.jsCode = String.raw`const bind = $("校验全量二维表绑定").first().json;
const rows = $input.all().map(i => i.json || {});
const parameters = rows.filter(r => r.parameter_name);
const codeData = rows.find(r => r.return_code_source !== undefined) || {};
const cell = (v) => String(v ?? "-").replace(/\|/g, "\\|").replace(/\r?\n/g, "<br>") || "-";
let table = "| 参数名称 | 参数类型 | 数据类型 | 枚举值 | 取值范围 | 默认值 | 参数描述 | 是否必需 | 参数关系备注 |\n|---|---|---|---|---|---|---|---|---|\n";
for (const p of parameters) table += "| " + [p.parameter_name, p.parameter_type, p.data_type, p.enum_value, p.value_range, p.default_value, p.description, p.is_required === true || p.is_required === "true" ? "是" : "否", p.relationship_notes].map(cell).join(" | ") + " |\n";
const query = [
  "请基于绑定的知识库，一次性生成" + bind.machine_type + "机型、" + bind.language + "语言命令" + bind.command + "的全量检证用例二维表。",
  "覆盖正常系、异常系、边界值、默认值、省略、枚举、依赖、互斥、组合、权限和顺序场景；生成后直接返回表格。",
  "只输出一个 Markdown 表格，表头必须为：用例番号 | テスト分類 | テスト目的 | パラメータ設定 | 入力データ | 期待結果 | 優先度 | 備考；用例番号必须以 TC 开头。",
  "参数二维表：\n" + table,
  "RETURN_CODE_LIST：" + String(codeData.return_code_list || "") + "\nRETURN_CODE_MESSAGES_RAW：\n" + String(codeData.return_code_messages_raw || "")
].join("\n\n");
return [{ json: { ...bind, parameter_table: table, return_code_list: codeData.return_code_list || "", return_code_messages_raw: codeData.return_code_messages_raw || "", return_code_source: codeData.return_code_source || "none", xelora_query: query, session_url: bind.xelora_base_url + "/sessions", session_request_body: { title: bind.machine_type + " " + bind.language + " full verification table " + bind.command } } }];`;

const createSession = httpNode("xelora-full-create-session", "创建Xelora全量二维表会话", [1800, 260], "={{ $json.session_url }}", "={{ $json.session_request_body }}", "创建 Xelora 会话，不创建评估会话");
const extractSession = codeNode("xelora-full-extract-session", "提取Xelora会话ID", [2040, 260], String.raw`const prior = $("格式化参数二维表").first().json;
const r = $input.first()?.json || {};
const id = r.session_id || r.id || r.data?.session_id || r.data?.id || r.result?.session_id || r.result?.id;
if (!id) throw new Error("Xelora session create failed: " + JSON.stringify(r).slice(0, 1000));
return [{ json: { ...prior, session_id: String(id), chat_url: prior.xelora_base_url + "/agent-chat/" + String(id), chat_request_body: { query: prior.xelora_query, agent_enabled: true, agent_id: prior.xelora_agent_id, knowledge_base_ids: prior.knowledge_base_ids, mentioned_items: [{ id: prior.knowledge_base_ids[0], name: prior.knowledge_base_name, type: "kb", kb_type: "document" }], web_search_enabled: false, channel: "api" } } }];`, "提取会话 ID 并准备一次性二维表生成请求");
const callAgent = httpNode("xelora-full-call-agent", "调用Xelora全量检证用例智能体", [2280, 260], "={{ $json.chat_url }}", "={{ $json.chat_request_body }}", "调用数据库绑定的唯一全量二维表生成智能体，直接进入解析和落表");
const processResponse = codeNode("xelora-full-process-response", "处理Xelora全量二维表响应", [2520, 260], String.raw`const prior = $("提取Xelora会话ID").first().json;
const response = $input.first()?.json || {};
let answer = ""; let completed = false; let error = "";
const append = (value) => { if (typeof value === "string" && value.trim()) answer += value; };
const consume = (event) => {
  if (!event || typeof event !== "object") return;
  const type = String(event.response_type || event.event || event.type || "").toLowerCase();
  if (["error", "failed", "failure"].includes(type)) { error = event.message || event.error || event.content || "Xelora agent error"; return; }
  if (event.done === true || event.complete === true || ["complete", "completed", "done", "final"].includes(type)) completed = true;
  const isNonAnswer = ["thinking", "tool_call", "tool", "status", "progress"].includes(type);
  if (!isNonAnswer) {
    append(event.answer);
    append(event.data?.answer);
    append(event.data?.content);
    if (["message", "answer", "content", "assistant", "final", "complete", "completed", "done"].includes(type) || event.done === true || event.complete === true) append(event.content);
  }
};
const payload = typeof response.data === "string" ? response.data : response;
if (typeof payload === "string") {
  for (const line of payload.split(/\r?\n/)) {
    if (!line.startsWith("data:")) continue;
    const value = line.slice(5).trim();
    if (value === "[DONE]") { completed = true; continue; }
    try { consume(JSON.parse(value)); } catch {}
  }
} else {
  consume(payload);
  if (payload.data && typeof payload.data === "object") consume(payload.data);
  error = error || (payload.error || (payload.message && !answer ? payload.message : ""));
}
if (error || !completed || !String(answer).trim()) throw new Error(error || "Xelora agent returned no completed non-empty Markdown table");
return [{ json: { ...prior, command_master_id: prior.command_id, full_analysis_response: String(answer).trim(), status: "success", response_time: new Date().toISOString() } }];`, "要求 complete/[DONE] 且正文非空，禁止伪成功");

const latest = nodeByName("查询最新版本号");
latest.parameters.query = "SELECT COALESCE(MAX(version), 0) AS max_version FROM analyzes.cl_verification_cases WHERE command_id = {{ $json.command_id }};";
const version = nodeByName("计算新版本号");
version.parameters.jsCode = String.raw`const latest = $input.first()?.json || {};
const source = $("处理Xelora全量二维表响应").first().json;
return [{ json: { ...source, version: Number(latest.max_version || 0) + 1 } }];`;
const parse = nodeByName("解析检证用例");
parse.onError = "stopWorkflow";
parse.parameters.jsCode = parse.parameters.jsCode
  .replaceAll("full_analysis_response", "full_analysis_response")
  .replace("return [{ json: { _empty_parse: true, _reason: 'no_command_or_body' } }];", "throw new Error('Xelora response missing command_id or Markdown body');")
  .replace("return [{ json: { _empty_parse: true, _reason: 'no_table' } }];", "throw new Error('Xelora response did not contain a Markdown table');")
  .replace("return [{ json: { _empty_parse: true, _reason: 'no_data_rows' } }];", "throw new Error('Markdown table contained no valid verification cases');");
const save = nodeByName("保存检证用例");
save.onError = "stopWorkflow";

const removeNames = new Set([
  "判断是否为内置函数", "调用内置函数检证用例生成智能体", "处理内置函数AI响应", "处理AI响应", "检查迭代次数",
  "调用检证用例生成智能体",
  "判断是否跳过评估", "准备评估请求", "判断是否为内置函数（评估）", "准备内置函数评估请求", "调用评估智能体",
  "调用内置函数评估智能体", "处理评估结果", "处理内置函数评估结果", "合并评估结果", "判断是否合格", "准备重新调用请求",
  "初始化cl_verification_cases表",
]);
workflow.nodes = workflow.nodes.filter((node) => !removeNames.has(node.name));
workflow.nodes.push(prepareBinding, resolveBinding, validateBinding, createSession, extractSession, callAgent, processResponse);

const fail = codeNode("xelora-full-binding-failure", "全量二维表绑定失败", [1240, 420], String.raw`throw new Error("全量二维表绑定失败");`);
workflow.nodes.push(fail);

const c = workflow.connections;
for (const key of Object.keys(c)) if (removeNames.has(key)) delete c[key];
c["Webhook接收参数"] = { main: [[{ node: input.name, type: "main", index: 0 }]] };
c[input.name] = { main: [[{ node: prepareBinding.name, type: "main", index: 0 }]] };
c[prepareBinding.name] = { main: [[{ node: resolveBinding.name, type: "main", index: 0 }]] };
c[resolveBinding.name] = { main: [[{ node: validateBinding.name, type: "main", index: 0 }]] };
c[validateBinding.name] = { main: [[{ node: queryParameters.name, type: "main", index: 0 }, { node: queryReturnCodes.name, type: "main", index: 0 }], [{ node: fail.name, type: "main", index: 0 }]] };
c[queryParameters.name] = { main: [[{ node: hasParameters.name, type: "main", index: 0 }]] };
c[hasParameters.name] = { main: [[{ node: merge.name, type: "main", index: 0 }], [{ node: merge.name, type: "main", index: 0 }]] };
c[queryReturnCodes.name] = { main: [[{ node: nodeByName("组装終了コード查询").name, type: "main", index: 0 }]] };
c[nodeByName("组装終了コード查询").name] = { main: [[{ node: queryMessages.name, type: "main", index: 0 }]] };
c[queryMessages.name] = { main: [[{ node: mergeCodes.name, type: "main", index: 0 }]] };
c[mergeCodes.name] = { main: [[{ node: merge.name, type: "main", index: 1 }]] };
c[merge.name] = { main: [[{ node: format.name, type: "main", index: 0 }]] };
c[format.name] = { main: [[{ node: createSession.name, type: "main", index: 0 }]] };
c[createSession.name] = { main: [[{ node: extractSession.name, type: "main", index: 0 }]] };
c[extractSession.name] = { main: [[{ node: callAgent.name, type: "main", index: 0 }]] };
c[callAgent.name] = { main: [[{ node: processResponse.name, type: "main", index: 0 }]] };
c[processResponse.name] = { main: [[{ node: latest.name, type: "main", index: 0 }]] };
c[latest.name] = { main: [[{ node: version.name, type: "main", index: 0 }]] };
c[version.name] = { main: [[{ node: parse.name, type: "main", index: 0 }]] };

fs.writeFileSync(outputPath, JSON.stringify(workflow, null, 2) + "\n", "utf8");
console.log(`Wrote ${outputPath}`);

