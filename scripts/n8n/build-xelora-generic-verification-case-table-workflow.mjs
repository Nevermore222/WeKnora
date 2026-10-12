#!/usr/bin/env node
import fs from "node:fs";
import path from "node:path";

const repoRoot = process.cwd();
const workflowDir = path.resolve(repoRoot, "..", "n8n", "二维表");
const sourcePath = path.join(workflowDir, "CL_命令检证用例二维表工作流（tech_point同款输入保存+SOURCE用例智能体）_xelora_1.4.0.json");
const outputPath = path.join(workflowDir, "Xelora通用检证用例二维表工作流_1.0.0.json");
if (!fs.existsSync(sourcePath)) throw new Error(`Source workflow does not exist: ${sourcePath}`);
const workflow = JSON.parse(fs.readFileSync(sourcePath, "utf8"));
workflow.id = "XeloraGenVerTableA8B247V1";
workflow.name = "Xelora通用检证用例二维表工作流_1.0.0";
workflow.active = false;
workflow.tags = [{ name: "Xelora" }, { name: "通用" }, { name: "检证用例二维表" }];

const node = (name) => {
  const found = workflow.nodes.find((entry) => entry.name === name);
  if (!found) throw new Error(`Missing node: ${name}`);
  return found;
};
const code = (id, name, position, jsCode, notes) => ({ id, name, type: "n8n-nodes-base.code", typeVersion: 2, position, parameters: { jsCode }, notes });
const postgres = (id, name, position, query, notes) => ({ id, name, type: "n8n-nodes-base.postgres", typeVersion: 2.1, position, parameters: { operation: "executeQuery", query, options: {} }, credentials: { postgres: { id: "tWcUIJhVg6bDlGDc", name: "TargetDB" } }, onError: "continueErrorOutput", notes });

const webhook = node("Webhook接收参数");
webhook.parameters.path = "xelora-verification-case-table-generic";
webhook.webhookId = "xelora-verification-case-table-generic";
webhook.notes = "通用入口：command_id 为必填；project_id、command、machine_type、language 可选，缺失时由 command_master、mac_list、mac_language、projects/tech_point 查询补齐；检证用例智能体和知识库由绑定配置解析";

const input = node("解析Webhook参数");
input.type = "n8n-nodes-base.code";
input.typeVersion = 2;
input.parameters = { jsCode: `const b = $json.body || $json;\nconst rawProjectId = b.project_id ?? b.projectId;\nconst project_id = rawProjectId == null || String(rawProjectId).trim() === '' ? null : Number(rawProjectId);\nconst command_id = Number(b.command_id ?? b.commandId ?? b.command_master_id ?? b.commandMasterId);\nconst command = String(b.command ?? b.command_name ?? '').trim();\nconst machine_type = String(b.machine_type ?? b.machineType ?? '').trim();\nconst language = String(b.language ?? '').trim().toUpperCase();\nif (project_id !== null && (!Number.isInteger(project_id) || project_id <= 0)) throw new Error('project_id must be a positive integer when provided');\nif (!Number.isInteger(command_id) || command_id <= 0) throw new Error('command_id must be a positive integer');\nconst batch_size = Number(b.batch_size ?? 500);\nconst sub_category_batch_size = Number(b.sub_category_batch_size ?? 10);\nreturn [{ json: { project_id, command_id, command, machine_type, language, batch_size: Number.isFinite(batch_size) && batch_size > 0 ? batch_size : 500, sub_category_batch_size: Number.isFinite(sub_category_batch_size) && sub_category_batch_size > 0 ? sub_category_batch_size : 10, request_time: new Date().toISOString() } }];` };
input.notes = "仅强制 command_id；命令、机型、语言和项目上下文在后续数据库节点解析，兼容前台 command_analysis 按钮";

const prepare = code("xelora-table-prepare-binding", "准备查询检证用例二维表绑定", [420, -40], `function q(v) { return "'" + String(v).replace(/'/g, "''") + "'"; }\nconst commandId = Number($json.command_id);\nconst suppliedProjectId = Number.isInteger($json.project_id) && $json.project_id > 0 ? String($json.project_id) : 'NULL';\nconst binding_query = \`WITH command_context AS (\n  SELECT cm.command, cm.mac_id, ml.mac_name, COALESCE(cm_lang.language, cm.language) AS language\n  FROM analyzes.command_master cm\n  LEFT JOIN analyzes.mac_list ml ON ml.id = cm.mac_id\n  LEFT JOIN analyzes.mac_language cm_lang ON cm_lang.id = cm.language_id\n  WHERE cm.id = \${commandId}\n), tech_projects AS (\n  SELECT DISTINCT tp.project_id\n  FROM analyzes.tech_point tp\n  WHERE tp.command_id = \${commandId}\n), machine_projects AS (\n  SELECT DISTINCT p.project_id\n  FROM analyzes.projects p\n  JOIN command_context cc ON cc.mac_id = p.mac_id\n), project_choice AS (\n  SELECT CASE\n    WHEN \${suppliedProjectId} IS NOT NULL THEN \${suppliedProjectId}\n    WHEN (SELECT COUNT(*) FROM tech_projects) = 1 THEN (SELECT MAX(project_id) FROM tech_projects)\n    WHEN (SELECT COUNT(*) FROM machine_projects) = 1 THEN (SELECT MAX(project_id) FROM machine_projects)\n    ELSE NULL\n  END AS project_id\n), candidates AS (\n  SELECT mla.workflow_role, ac.base_url, ac.agent_id, ac.api_key, mla.knowledge_base_ids, mla.knowledge_base_names, mla.updated_at\n  FROM analyzes.mac_language_agent mla\n  JOIN analyzes.mac_list ml ON ml.id = mla.mac_id\n  JOIN analyzes.agent_config ac ON ac.id = mla.agent_config_id\n  JOIN command_context cc ON cc.mac_id = mla.mac_id AND lower(trim(mla.language)) = lower(trim(cc.language))\n  WHERE mla.is_active = true AND ac.is_active = true\n    AND mla.workflow_role = 'verification_case_table_parse'\n), binding AS (\n  SELECT * FROM candidates ORDER BY updated_at DESC NULLS LAST LIMIT 1\n)\nSELECT cc.command AS command_name, cc.mac_name AS command_machine_type, cc.language AS command_language, pc.project_id,\n       b.base_url, b.agent_id, b.api_key, b.knowledge_base_ids, b.knowledge_base_names\nFROM command_context cc\nCROSS JOIN project_choice pc\nLEFT JOIN binding b ON TRUE;\`;\nreturn [{ json: { ...$json, binding_query } }];`, "按 command_id 从 command_master 解析命令/机型/语言，并自动解析唯一项目，再读取二维表智能体与知识库绑定");
const resolve = postgres("xelora-table-resolve-binding", "查询检证用例二维表绑定", [650, -40], "={{ $json.binding_query }}", "使用 TargetDB 查询通用二维表绑定");
const normalize = code("xelora-table-normalize-binding", "校验二维表绑定", [880, -40], `const base = $("解析Webhook参数").first().json;\nconst row = $input.first()?.json || {};\nfunction ids(v) { if (Array.isArray(v)) return v.map(String).filter(Boolean); if (v && Array.isArray(v.array)) return v.array.map(String).filter(Boolean); if (typeof v === 'string') { try { return ids(JSON.parse(v)); } catch { return v ? [v] : []; } } return []; }\nconst kbIds = ids(row.knowledge_base_ids);\nconst projectId = Number(row.project_id ?? base.project_id);\nconst command = String(row.command_name || base.command || '').trim();\nconst machine = String(row.command_machine_type || base.machine_type || '').trim();\nconst language = String(row.command_language || base.language || '').trim().toUpperCase();\nconst missing = [];\nif (!command) missing.push('command_context.command');\nif (!machine) missing.push('command_context.machine_type');\nif (!language) missing.push('command_context.language');\nif (!Number.isInteger(projectId) || projectId <= 0) missing.push('project_context.project_id (ambiguous or not found; pass project_id only when command has multiple projects)');\nif (!row.base_url || !row.agent_id || !row.api_key || !kbIds.length) missing.push('verification_case_table_parse');\nif (missing.length) return [{ json: { ...base, command, machine_type: machine, language, project_id: Number.isInteger(projectId) && projectId > 0 ? projectId : null, valid: false, error_type: 'workflow_context_or_binding_not_found', error_message: 'Unable to resolve command context/binding for command_id=' + base.command_id + ': ' + missing.join(', ') } }];\nreturn [{ json: { ...base, command, machine_type: machine, language, project_id: projectId, valid: true, xelora_api_base_url: String(row.base_url).replace(/\\\/$/, ''), xelora_agent_id: String(row.agent_id), xelora_api_key: String(row.api_key), xelora_kb_ids: kbIds, xelora_kb_name: String(row.knowledge_base_names?.[0] || kbIds[0]) } }];`, "从数据库补齐命令上下文和项目；绑定缺失或项目无法唯一解析时明确失败，不回退到旧智能体");
const valid = { id: "xelora-table-binding-if", name: "判断二维表绑定是否完整", type: "n8n-nodes-base.if", typeVersion: 1, position: [1090, -40], parameters: { conditions: { boolean: [{ value1: "={{ $json.valid === true }}", value2: true }] } }, notes: "绑定完整才继续查询技术点" };
workflow.nodes.push(prepare, resolve, normalize, valid);

const query = node("按小分類查询tech_point代表行");
query.parameters.query = query.parameters.query.replace("AND UPPER(TRIM(tp.language)) = 'CL'", "AND UPPER(TRIM(tp.language)) = UPPER(TRIM('{{ $json.language }}'))");
query.notes = "按请求语言查询技术点并保留原 SOURCE/变量声明";

const group = node("按批次分组小分類");
group.parameters.jsCode = group.parameters.jsCode.replace("const params = $('解析Webhook参数').first()?.json || {};", "const params = $('解析Webhook参数').first()?.json || {};\nconst binding = $('校验二维表绑定').first()?.json || {};" ).replace("output.push({ json: { project_id: projectId,", "output.push({ json: { ...binding, project_id: projectId,");

const summarize = node("汇总并准备用例生成请求");
let summaryCode = summarize.parameters.jsCode;
summaryCode = summaryCode.replace(/const XELORA_API_BASE_URL = [^;]+;/, "const XELORA_API_BASE_URL = $json.xelora_api_base_url;")
  .replace(/const XELORA_API_KEY = [^;]+;/, "const XELORA_API_KEY = $json.xelora_api_key;")
  .replace(/const XELORA_AGENT_ID = [^;]+;/, "const XELORA_AGENT_ID = $json.xelora_agent_id;")
  .replace(/const XELORA_KB_ID = [^;]+;/, "const XELORA_KB_ID = $json.xelora_kb_ids[0];")
  .replace(/const XELORA_KNOWLEDGE_IDS = [^;]+;/, "const XELORA_KNOWLEDGE_IDS = $json.xelora_kb_ids;")
  .replace(/'LANGUAGE_TYPE: CL'/g, "'LANGUAGE_TYPE: ' + meta.language")
  .replace(/language: 'CL'/g, "language: meta.language")
  .replace(/name: 'Manual_ASP'/g, "name: meta.xelora_kb_name || 'knowledge-base'")
  .replace(/name: 'CL tech_point cases '/g, "name: meta.machine_type + ' ' + meta.language + ' tech_point cases '");
summarize.parameters.jsCode = summaryCode;
summarize.notes = "按数据库绑定调用对应机型/语言的检证用例二维表智能体";

const parse = node("解析本批检证用例");
parse.notes = "保留通用检证用例字段并兼容中日文表头";

// 业务错误必须阻断执行，避免 SplitInBatches 继续到尾节点返回伪成功。
for (const name of ["按小分類查询tech_point代表行", "按批次分组小分類", "汇总并准备用例生成请求", "删除该命令存量数据", "创建Xelora会话", "调用Xelora用例生成智能体", "处理本批智能体响应", "解析本批检证用例", "准备本批检证用例接续编号", "保存最终检证用例"]) node(name).onError = "stopWorkflow";

const response = node("处理本批智能体响应");
response.parameters.jsCode = response.parameters.jsCode.replace(
  "return [{\n  json: {\n    command_master_id: meta.command_master_id,",
  "if (status !== 'success') throw new Error(errorMessage || 'Xelora agent returned no usable verification-case response');\n\nreturn [{\n  json: {\n    command_master_id: meta.command_master_id,"
);
parse.parameters.jsCode = parse.parameters.jsCode
  .replace("return [{ json: { _empty_parse: true, _reason: 'no_command_or_body' } }];", "throw new Error('本批检证用例响应为空或缺少 command_id');")
  .replace("return [{ json: { _empty_parse: true, _reason: 'no_table' } }];", "throw new Error('本批检证用例响应未找到可解析的二维表');")
  .replace("return [{ json: { _empty_parse: true, _reason: 'no_data_rows' } }];", "throw new Error('本批检证用例二维表未解析出有效数据行');");

const tail = node("最后尾收");
tail.onError = "stopWorkflow";
tail.parameters.jsCode = tail.parameters.jsCode.replace(
  "return [{ json: { completed: true, message:",
  "const expected = $('汇总并准备用例生成请求').all().length;\nlet inserted = 0;\ntry { inserted = $('准备本批检证用例接续编号').all().reduce((n, item) => n + Number(item.json?._case_count || 0), 0); } catch {}\nif (!expected) return [{ json: { completed: false, status: 'no_pending_data', message: '当前命令没有待处理的检证用例批次' } }];\nif (!inserted) throw new Error('检证用例未实际解析或插入，禁止返回成功');\nreturn [{ json: { completed: true, inserted_case_count: inserted, message:"
);

for (const name of ["创建Xelora会话", "调用Xelora用例生成智能体"]) {
  const http = node(name);
  http.parameters.options = { ...(http.parameters.options || {}), timeout: 120000 };
}

const c = workflow.connections;
if (c["调用用例生成智能体"] && !c["调用Xelora用例生成智能体"]) { c["调用Xelora用例生成智能体"] = c["调用用例生成智能体"]; delete c["调用用例生成智能体"]; }
if (c["调用用例生成智能体"]) { c["调用Xelora用例生成智能体"] = c["调用用例生成智能体"]; delete c["调用用例生成智能体"]; }
c["解析Webhook参数"] = { main: [[{ node: prepare.name, type: "main", index: 0 }]] };
c[prepare.name] = { main: [[{ node: resolve.name, type: "main", index: 0 }]] };
c[resolve.name] = { main: [[{ node: normalize.name, type: "main", index: 0 }]] };
c[normalize.name] = { main: [[{ node: valid.name, type: "main", index: 0 }]] };
c[valid.name] = { main: [[{ node: query.name, type: "main", index: 0 }], [{ node: "最后尾收", type: "main", index: 0 }]] };
// 当前 n8n 版本的 SplitInBatches: output 0 为 done，output 1 为 loop。
// 原副本将这两个出口反接，导致每次请求直接尾收而从不进入智能体/INSERT 链。
c["小分類逐批处理"] = { main: [[{ node: "最后尾收", type: "main", index: 0 }], [{ node: "创建Xelora会话", type: "main", index: 0 }]] };

fs.writeFileSync(outputPath, JSON.stringify(workflow, null, 2) + "\n", "utf8");
console.log(`Wrote ${outputPath}`);
