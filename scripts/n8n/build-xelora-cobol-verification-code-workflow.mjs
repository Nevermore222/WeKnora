#!/usr/bin/env node
import { execFileSync } from "node:child_process";
import fs from "node:fs";
import path from "node:path";

const repoRoot = process.cwd();
const workflowDir = path.resolve(repoRoot, "..", "n8n", "检证用例代码");
const sourcePath = path.join(workflowDir, "COBOL检证用例代码生成工作流_1.2.4.json");
const outputPath = path.join(workflowDir, "Xelora版COBOL检证用例代码生成工作流_1.1.0.json");

const xeloraApiBaseUrl = "http://192.168.2.61:18080/api/v1";
const xeloraApiKey = "sk-5NB4aUSnLqpsYKoit_FuqbjECutYS-szIJfvxEWgBkKZb5x5";
const agentName = "COBOL检证用例代码生成智能体-SR6";
const knowledgeBaseName = "Manual_ASP";
const knowledgeTagName = "ASP_COBOL";

function sqlLiteral(value) {
  return `'${String(value).replaceAll("'", "''")}'`;
}

function runPsql(sql) {
  return execFileSync(
    "docker",
    ["exec", "-i", "Xelora-postgres", "psql", "-U", "postgres", "-d", "Xelora", "-t", "-A"],
    { input: sql, encoding: "utf8" },
  ).trim();
}

function getXeloraIds() {
  const agentId = runPsql(`SELECT id FROM custom_agents WHERE deleted_at IS NULL AND name = ${sqlLiteral(agentName)} ORDER BY updated_at DESC LIMIT 1;`);
  if (!agentId) throw new Error(`Xelora agent not found: ${agentName}. Run the upsert script first.`);
  const knowledgeBaseId = runPsql(`SELECT id FROM knowledge_bases WHERE deleted_at IS NULL AND name = ${sqlLiteral(knowledgeBaseName)} ORDER BY updated_at DESC LIMIT 1;`);
  if (!knowledgeBaseId) throw new Error(`Knowledge base not found: ${knowledgeBaseName}`);
  const idsText = runPsql(`
SELECT k.id
FROM knowledges k
JOIN knowledge_tags t ON t.id = k.tag_id
WHERE k.deleted_at IS NULL AND t.deleted_at IS NULL
  AND k.knowledge_base_id = ${sqlLiteral(knowledgeBaseId)}
  AND t.knowledge_base_id = ${sqlLiteral(knowledgeBaseId)}
  AND t.name = ${sqlLiteral(knowledgeTagName)}
ORDER BY k.title;`);
  const knowledgeIds = idsText.split(/\r?\n/).map((id) => id.trim()).filter(Boolean);
  if (!knowledgeIds.length) throw new Error(`No documents found under tag ${knowledgeTagName}.`);
  return { agentId, knowledgeBaseId, knowledgeIds };
}

function requireNode(workflow, name) {
  const node = workflow.nodes.find((entry) => entry.name === name);
  if (!node) throw new Error(`Node not found: ${name}`);
  return node;
}

function renameNode(workflow, oldName, newName) {
  const node = requireNode(workflow, oldName);
  node.name = newName;
  if (workflow.connections[oldName]) {
    workflow.connections[newName] = workflow.connections[oldName];
    delete workflow.connections[oldName];
  }
  for (const connection of Object.values(workflow.connections)) {
    for (const branch of connection.main || []) {
      for (const edge of branch || []) if (edge.node === oldName) edge.node = newName;
    }
  }
  return node;
}

function removeNode(workflow, name) {
  workflow.nodes = workflow.nodes.filter((entry) => entry.name !== name);
  delete workflow.connections[name];
  for (const connection of Object.values(workflow.connections)) {
    for (const branch of connection.main || []) {
      for (let index = branch.length - 1; index >= 0; index--) {
        if (branch[index].node === name) branch.splice(index, 1);
      }
    }
  }
}

function makeCodeNode(id, name, position, jsCode, notes) {
  return {
    id,
    name,
    type: "n8n-nodes-base.code",
    typeVersion: 2,
    position,
    parameters: { jsCode },
    onError: "continueErrorOutput",
    notes,
  };
}

function makeHttpNode(id, name, position, parameters, notes) {
  return {
    id,
    name,
    type: "n8n-nodes-base.httpRequest",
    typeVersion: 4.2,
    position,
    parameters,
    onError: "continueErrorOutput",
    notes,
  };
}

const { agentId, knowledgeBaseId, knowledgeIds } = getXeloraIds();
const workflow = JSON.parse(fs.readFileSync(sourcePath, "utf8"));
workflow.name = "Xelora版COBOL检证用例代码生成工作流_1.1.0";
workflow.active = false;
delete workflow.id;
delete workflow.versionId;
delete workflow.meta;
workflow.tags = [{ name: "Xelora" }, { name: "COBOL" }, { name: "代码生成" }];

const webhookNode = requireNode(workflow, "Webhook1");
webhookNode.parameters.path = "COBOL_TEST";

const parseWebhookNode = requireNode(workflow, "解析Webhook参数");
const projectIdAssignment = parseWebhookNode.parameters.assignments.assignments.find(
  (entry) => entry.name === "project_id",
);
if (projectIdAssignment) {
  projectIdAssignment.value = "={{ $json.body?.project_id || $json.body?.data?.project_id || $json.project_id || null }}";
  projectIdAssignment.type = "number";
}

removeNode(workflow, "判断project_id是否为空");
removeNode(workflow, "查询项目名称");
removeNode(workflow, "查询cl_verification_cases_master表");

const projectCasesNode = renameNode(
  workflow,
  "查询cl_verification_cases_master_proj表",
  "查询项目COBOL检证用例",
);
projectCasesNode.parameters.query = `SELECT
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
  t.id AS source_tech_point_id,
  t.source_content,
  t.variable_declaration
FROM analyzes.cl_verification_cases_master_proj v
LEFT JOIN analyzes.tech_point t
  ON t.project_id = v.project_id
 AND t.command_id = v.command_id
 AND t.tech_point_no = v.tech_point_no
WHERE v.command_id = {{ $json.command_id }}
  AND v.project_id = {{ $json.project_id }}
  AND TRIM(COALESCE(v.test_category, '')) IN ('正常系','境界値')
ORDER BY v.case_number ASC;`;

const validateProjectNode = makeCodeNode(
  "validate-project-id",
  "校验project_id",
  [-3248, 160],
  `const item = $input.first();
const raw = item?.json?.project_id;
const projectId = Number(raw);
if (!Number.isInteger(projectId) || projectId < 1) {
  throw new Error('project_id is required and must be a positive integer');
}
const commandId = Number(item?.json?.command_id);
if (!Number.isInteger(commandId) || commandId < 1) {
  throw new Error('command_id is required and must be a positive integer');
}
return [{ json: { ...item.json, project_id: projectId, command_id: commandId } }];`,
  "project_id 与 command_id 必须是正整数，入口不允许省略 project_id。",
);

const projectLookupNode = {
  id: "lookup-xelora-project",
  name: "查询并确认项目",
  type: "n8n-nodes-base.postgres",
  typeVersion: 2.5,
  position: [-3056, 160],
  parameters: {
    operation: "executeQuery",
    query: `SELECT
  {{ $json.project_id }}::integer AS project_id,
  MAX(p.project_name) AS project_name,
  EXISTS (
    SELECT 1 FROM analyzes.projects p2 WHERE p2.project_id = {{ $json.project_id }}
  ) AS project_exists
FROM analyzes.projects p
WHERE p.project_id = {{ $json.project_id }};`,
    options: {},
  },
  alwaysOutputData: true,
  onError: "continueErrorOutput",
  notes: "只确认调用方明确传入的项目，不为缺失或未知项目提供默认回退。",
};

const confirmProjectNode = makeCodeNode(
  "confirm-xelora-project",
  "确认项目存在",
  [-2864, 160],
  `const source = $input.first()?.json || {};
if (source.project_exists !== true && source.project_exists !== 'true') {
  throw new Error('project_id does not exist in analyzes.projects: ' + String(source.project_id || ''));
}
const request = $('校验project_id').first()?.json || {};
return [{ json: {
  ...request,
  project_id: Number(request.project_id),
  command_id: Number(request.command_id),
  project_name: source.project_name || null,
} }];`,
  "确认项目存在后，后续所有读取和写入均使用同一个 project_id。",
);

workflow.nodes.push(validateProjectNode, projectLookupNode, confirmProjectNode);
workflow.connections["解析Webhook参数"] = {
  main: [[{ node: "校验project_id", type: "main", index: 0 }]],
};
workflow.connections["校验project_id"] = {
  main: [[{ node: "查询并确认项目", type: "main", index: 0 }]],
};
workflow.connections["查询并确认项目"] = {
  main: [[{ node: "确认项目存在", type: "main", index: 0 }]],
};
workflow.connections["确认项目存在"] = {
  main: [[{ node: "查询项目COBOL检证用例", type: "main", index: 0 }]],
};
workflow.connections["查询项目COBOL检证用例"] = {
  main: [[{ node: "合并查询结果", type: "main", index: 0 }]],
};

const mergeQueryNode = requireNode(workflow, "合并查询结果");
mergeQueryNode.parameters.jsCode = `const cases = $input.all();
const project = $('确认项目存在').first()?.json || {};
return cases.map((item) => ({
  json: {
    ...item.json,
    project_id: Number(project.project_id),
    project_name: project.project_name || null,
  },
}));`;
mergeQueryNode.notes = "仅合并已确认项目的名称和项目内检证用例，不读取无项目主表。";

const aggregateNode = requireNode(workflow, "AI批次汇总并校验");
const semanticRewriteStart = aggregateNode.parameters.jsCode.indexOf(
  "function detectResultFieldByDefinition",
);
const sharedBatchMergeStart = aggregateNode.parameters.jsCode.indexOf(
  "function buildUnifiedClCode",
);
if (semanticRewriteStart < 0 || sharedBatchMergeStart <= semanticRewriteStart) {
  throw new Error("The 1.2.4 semantic rewrite block was not found.");
}
aggregateNode.parameters.jsCode =
  aggregateNode.parameters.jsCode.slice(0, semanticRewriteStart) +
  aggregateNode.parameters.jsCode.slice(sharedBatchMergeStart);
const semanticRewriteCall =
  "const normalizedCobol = normalizeResultMoves(String(p.cobol_code || '').trimEnd());";
if (!aggregateNode.parameters.jsCode.includes(semanticRewriteCall)) {
  throw new Error("The 1.2.4 semantic rewrite call was not found.");
}
aggregateNode.parameters.jsCode = aggregateNode.parameters.jsCode.replace(
  semanticRewriteCall,
  "const normalizedCobol = String(p.cobol_code || '').trimEnd();",
);

const normalizeFilesNode = requireNode(workflow, "调用文件转码服务");
normalizeFilesNode.parameters.url = "=http://192.168.8.247:8765/normalize";
normalizeFilesNode.notes = "file_normalize_service；固定调用 http://192.168.8.247:8765/normalize，不读取 n8n 环境变量";

const collectNormalizePathNode = requireNode(workflow, "收集转码路径");
const collectNormalizePathCode = collectNormalizePathNode.parameters.jsCode;
const normalizeConfigStart = collectNormalizePathCode.indexOf("// 转码服务路径：");
const normalizeLogicStart = collectNormalizePathCode.indexOf("const orgItems");
if (normalizeConfigStart < 0 || normalizeLogicStart <= normalizeConfigStart) {
  throw new Error("File normalize path configuration block was not found.");
}
collectNormalizePathNode.parameters.jsCode =
  collectNormalizePathCode.slice(0, normalizeConfigStart) +
  `// 转码服务固定读取宿主机映射路径，不依赖 n8n 环境变量
const CONTAINER_ROOT = '/home/n8n/.n8n/verification_cases_code';
const HOST_ROOT = '/data/n8n/data/verification_cases_code';
const USE_CONTAINER_PATH = false;

` +
  collectNormalizePathCode
    .slice(normalizeLogicStart)
    .replace(
      "（可设置环境变量 FILE_NORMALIZE_CONTAINER_ROOT 与实际 base_dir 一致，或 FILE_NORMALIZE_USE_CONTAINER_PATH=1）",
      "（固定容器根目录为 /home/n8n/.n8n/verification_cases_code）",
    );

const summarizeNormalizeNode = requireNode(workflow, "更新保存状态");
summarizeNormalizeNode.parameters.jsCode = summarizeNormalizeNode.parameters.jsCode.replace(
  "FILE_NORMALIZE_HOST_ROOT 或 FILE_NORMALIZE_USE_CONTAINER_PATH=1",
  "固定映射 /home/n8n/.n8n/verification_cases_code -> /data/n8n/data/verification_cases_code",
);

const organizeNode = requireNode(workflow, "组织检证用例二维表");
organizeNode.parameters.jsCode = `// 将数据库二维表按批组织为 Xelora SR6 代码生成请求。
const items = $input.all();
const webhookParams = $('解析Webhook参数').first();
if (items.length === 0) return [];

const XELORA_API_BASE_URL = ${JSON.stringify(xeloraApiBaseUrl)};
const XELORA_API_KEY = ${JSON.stringify(xeloraApiKey)};
const XELORA_AGENT_ID = ${JSON.stringify(agentId)};
const XELORA_KB_ID = ${JSON.stringify(knowledgeBaseId)};
const XELORA_KNOWLEDGE_IDS = ${JSON.stringify(knowledgeIds)};

const commandMasterId = webhookParams?.json?.command_id || null;
const command = webhookParams?.json?.command || '';
const version = webhookParams?.json?.version || 1;
const language = webhookParams?.json?.language || 'COBOL';
const projectId = Number(webhookParams?.json?.project_id);
let batchSize = parseInt(webhookParams?.json?.code_gen_batch_size ?? 14, 10);
if (!Number.isFinite(batchSize) || batchSize < 1) batchSize = 14;
if (!commandMasterId) throw new Error('缺少必需参数 command_id');
if (!Number.isInteger(projectId) || projectId < 1) throw new Error('缺少有效的 project_id');

const projectName = items[0]?.json?.project_name || null;
const projCode = projectId === 1 ? 'T' : projectId === 3 ? 'F' : 'P' + projectId;
const pgmBaseCode = 'B' + projCode + String(commandMasterId).padStart(3, '0');
const now = new Date();
const timestamp = now.getFullYear().toString() +
  String(now.getMonth() + 1).padStart(2, '0') +
  String(now.getDate()).padStart(2, '0') +
  String(now.getHours()).padStart(2, '0') +
  String(now.getMinutes()).padStart(2, '0') +
  String(now.getSeconds()).padStart(2, '0');
const subFolderName = commandMasterId + '_' + command;
const timeFolderName = command + '_' + timestamp;
let baseDir = '/home/n8n/.n8n/verification_cases_code';
if (projectId && projectName) {
  const safeProjectName = String(projectName).replace(/[^a-zA-Z0-9_-]/g, '_');
  baseDir += '/' + safeProjectName + '/ASP/COBOL';
} else {
  baseDir += '/ASP/COBOL';
}

const rows = items.map((item) => item.json);
for (const row of rows) {
  const techPointNo = String(row.tech_point_no || '').trim();
  if (!techPointNo) {
    throw new Error('检证用例缺少 tech_point_no: case_number=' + String(row.case_number || ''));
  }
  if (row.source_tech_point_id === null || row.source_tech_point_id === undefined) {
    throw new Error('项目技术点上下文不存在: project_id=' + projectId + ', tech_point_no=' + techPointNo);
  }
}
const batches = [];
for (let i = 0; i < rows.length; i += batchSize) batches.push(rows.slice(i, i + batchSize));

return batches.map((batchRows, index) => {
  const seq = String(index + 1).padStart(2, '0');
  const contextByTechPoint = new Map();
  for (const row of batchRows) {
    const techPointNo = String(row.tech_point_no || '').trim();
    const context = {
      techPointNo,
      subCategory: String(row.sub_category || '').trim(),
      sourceId: String(row.source_tech_point_id),
      sourceContent: String(row.source_content || '').trim(),
      declaration: String(row.variable_declaration || '').trim(),
    };
    const previous = contextByTechPoint.get(techPointNo);
    if (previous && JSON.stringify(previous) !== JSON.stringify(context)) {
      throw new Error('同一技术点的 source context 不一致: tech_point_no=' + techPointNo);
    }
    contextByTechPoint.set(techPointNo, context);
  }
  const techPointContext = [...contextByTechPoint.values()].map((context) => {
    return [
      'TECH_POINT_CONTEXT BEGIN',
      'TECH_POINT_NO: ' + context.techPointNo,
      'SUB_CATEGORY: ' + context.subCategory,
      'SOURCE_TECH_POINT_ID: ' + context.sourceId,
      'SOURCE_CONTENT:',
      context.sourceContent,
      'VARIABLE_DECLARATION:',
      context.declaration,
      'VARIABLE_DECLARATION_MODE: ' + (context.declaration ? 'AUTHORITATIVE' : 'CURRENT_TECH_POINT_FALLBACK'),
      'TECH_POINT_CONTEXT END'
    ].join('\\n');
  }).join('\\n\\n');
  let table = '| 技術点NO | 小分類 | 用例番号 | テスト分類 | テスト目的 | パラメータ設定 | 入力データ | 期待結果 | 優先度 | 備考 |\\n';
  table += '|---|---|---|---|---|---|---|---|---|---|\\n';
  for (const row of batchRows) {
    const cells = [row.tech_point_no, row.sub_category, row.case_number, row.test_category,
      row.test_purpose, row.parameter_config, row.input_data, row.expected_result, row.priority, row.notes];
    table += '| ' + cells.map((cell) => String(cell || '').replace(/\\|/g, '\\\\|').replace(/\\r?\\n/g, '<br>')).join(' | ') + ' |\\n';
  }

  const query = [
    'COMMAND_NAME: ' + command,
    'PGM_BASE_CODE: ' + pgmBaseCode,
    'PGM_FILE_SEQ: ' + seq,
    'BATCH_INDEX: ' + (index + 1) + '/' + batches.length,
    'LANGUAGE_TYPE: COBOL',
    '每个 TECH_POINT_CONTEXT 块只属于块内的 TECH_POINT_NO。禁止跨技术点复用 SOURCE_CONTENT 或 VARIABLE_DECLARATION。',
    'VARIABLE_DECLARATION 非空时必须作为该技术点变量类型的权威定义；为空时只允许对当前技术点按旧逻辑推断。',
    techPointContext,
    '以下の TEST_CASES_TABLE の各行を実行可能な検証コードへ変換してください。',
	'TP_NO は追跡用メタデータにすぎません。技術点の網羅性は、小分類に記載された受取型、全オペランド型、演算順序、括弧、ROUNDED、SIZE ERROR を実行可能な COBOL で実装したかどうかで判定されます。',
	'各行について、コメントと TP_NO を除いても実行コードだけで小分類の全要件を識別できるようにしてください。',
    'システムプロンプトの SR6 規約を最優先し、完全な CL、COBOL、PF の三ファイルを指定された Markdown 見出しで返してください。',
    'コード、注釈、説明は日本語規約に従い、プレースホルダーや省略を使用しないでください。',
    'TEST_CASES_TABLE BEGIN',
    table,
    'TEST_CASES_TABLE END'
  ].join('\\n\\n');

  return { json: {
    command_master_id: commandMasterId,
    command, version, language, project_id: projectId, project_name: projectName,
    batch_index: index + 1, batch_total: batches.length, cases_in_batch: batchRows.length,
    total_cases_all_rows: rows.length, pgm_base_code: pgmBaseCode, pgm_file_seq: seq,
    sub_folder_name: subFolderName, time_folder_name: timeFolderName,
    base_dir_container: baseDir, test_cases_table: table, tech_point_context: techPointContext,
    xelora_api_base_url: XELORA_API_BASE_URL,
    xelora_api_key: XELORA_API_KEY,
    xelora_agent_id: XELORA_AGENT_ID,
    xelora_knowledge_base_id: XELORA_KB_ID,
    xelora_knowledge_ids: XELORA_KNOWLEDGE_IDS,
    xelora_session_url: XELORA_API_BASE_URL + '/sessions',
    xelora_session_request_body: { title: 'COBOL SR6 code ' + command + ' batch ' + (index + 1) + '/' + batches.length },
    xelora_agent_request_body: {
      query,
      agent_enabled: true,
      agent_id: XELORA_AGENT_ID,
      knowledge_base_ids: [XELORA_KB_ID],
      knowledge_ids: XELORA_KNOWLEDGE_IDS,
      mentioned_items: [{ id: XELORA_KB_ID, name: 'Manual_ASP', type: 'kb', kb_type: 'document' }],
      web_search_enabled: false,
      channel: 'api'
    },
    request_time: new Date().toISOString(),
    original_case_ids: batchRows.map((row) => row.id)
  }};
});`;
organizeNode.notes = "按批组织二维表，并直接携带 Xelora API、智能体、Manual_ASP 与 ASP_COBOL 文档信息。";

const createSessionNode = renameNode(workflow, "调用Dify智能体", "创建Xelora会话");
createSessionNode.parameters = {
  method: "POST",
  url: "={{ $json.xelora_session_url }}",
  sendHeaders: true,
  headerParameters: { parameters: [
    { name: "X-API-Key", value: "={{ $json.xelora_api_key }}" },
    { name: "Content-Type", value: "application/json" },
  ] },
  sendBody: true,
  specifyBody: "json",
  jsonBody: "={{ $json.xelora_session_request_body }}",
  options: {
    batching: { batch: { batchSize: 5, batchInterval: 300 } },
    response: { response: { neverError: true } },
    timeout: 9990000,
  },
};
createSessionNode.notes = "为每个代码生成批次创建独立 Xelora 会话。";

const extractSessionNode = makeCodeNode(
  "xelora-extract-cobol-code-session",
  "提取Xelora会话ID",
  [-1840, 128],
  `const responses = $input.all();
const requests = $('组织检证用例二维表').all();

function pairedItemIndex(item) {
  const paired = item?.pairedItem;
  if (Array.isArray(paired)) return Number(paired[0]?.item);
  if (paired && typeof paired === 'object') return Number(paired.item);
  return Number.NaN;
}

const requestByBatch = new Map(requests.map((item) => [Number(item.json.batch_index), item.json]));
const results = [];
for (let index = 0; index < responses.length; index++) {
  const response = responses[index];
  const body = response.json || {};
  const title = String(body.title || body.data?.title || body.result?.title || '');
  const titleMatch = title.match(/batch\\s+(\\d+)\\s*\\//i);
  const pairedIndex = pairedItemIndex(response);
  const batchIndex = titleMatch
    ? Number(titleMatch[1])
    : Number.isInteger(pairedIndex) && requests[pairedIndex]
      ? Number(requests[pairedIndex].json.batch_index)
      : Number.NaN;
  const meta = requestByBatch.get(batchIndex);
  if (!meta) continue;
  const sessionId = body.session_id || body.id || body.data?.session_id || body.data?.id || body.result?.session_id || body.result?.id;
  if (!sessionId) continue;
  results.push({
    json: {
      ...meta,
      xelora_session_id: String(sessionId),
      xelora_agent_url: meta.xelora_api_base_url + '/agent-chat/' + String(sessionId)
    },
    pairedItem: { item: requests.findIndex((item) => Number(item.json.batch_index) === batchIndex) }
  });
}
return results.sort((a, b) => Number(a.json.batch_index) - Number(b.json.batch_index));`,
  "按 pairedItem 或会话标题中的批次号关联 session_id；缺失会话由响应汇总节点补为失败批次。",
);

const callAgentNode = makeHttpNode(
  "xelora-call-cobol-verification-code-agent",
  "调用Xelora代码生成智能体",
  [-1680, 128],
  {
    method: "POST",
    url: "={{ $json.xelora_agent_url }}",
    sendHeaders: true,
    headerParameters: { parameters: [
      { name: "X-API-Key", value: "={{ $json.xelora_api_key }}" },
      { name: "Content-Type", value: "application/json" },
    ] },
    sendBody: true,
    specifyBody: "json",
    jsonBody: "={{ $json.xelora_agent_request_body }}",
    options: {
      batching: { batch: { batchSize: 2, batchInterval: 1000 } },
      response: { response: { neverError: true } },
      timeout: 9990000,
    },
  },
  "调用 /agent-chat/{session_id} 获取 SR6 三文件 SSE 响应。",
);

const processNode = renameNode(workflow, "处理AI响应", "处理Xelora智能体响应");
processNode.position = [-1488, 128];
processNode.parameters.jsCode = `// 仅解析 Xelora SSE 并提取 CL/COBOL/PF 三个文件段；不校验代码语义。
const responses = $input.all();
const sessionRequests = $('提取Xelora会话ID').all();
const originalRequests = $('组织检证用例二维表').all();

function pairedItemIndex(item) {
  const paired = item?.pairedItem;
  if (Array.isArray(paired)) return Number(paired[0]?.item);
  if (paired && typeof paired === 'object') return Number(paired.item);
  return Number.NaN;
}

function collectAnswer(response) {
  let answer = '';
  let complete = false;
  let error = null;
  let sessionId = '';
  function absorb(event) {
    if (!event || typeof event !== 'object') return;
    if (!sessionId && (event.session_id || event.data?.session_id)) {
      sessionId = String(event.session_id || event.data.session_id);
    }
    const responseType = String(event.response_type || event.event || event.type || '').toLowerCase();
    if (typeof event.answer === 'string') answer += event.answer;
    const answerType = !responseType || ['answer','message','agent_message','final_answer','final_answer_chunk'].includes(responseType);
    if (answerType && typeof event.content === 'string') answer += event.content;
	if (responseType === 'complete') complete = true;
	if (responseType === 'error') error = String(event.content || event.error || event.data?.error || 'Xelora error');
  }
  function parseSse(text) {
    for (const line of String(text || '').split('\\n')) {
      const trimmed = line.trimStart();
      if (!trimmed.startsWith('data:')) continue;
      const payload = trimmed.slice(5).trim();
      if (!payload || payload === '[DONE]') { complete = true; continue; }
      try { absorb(JSON.parse(payload)); } catch (e) {}
    }
  }
  if (typeof response === 'string') parseSse(response);
  else if (typeof response?.data === 'string') parseSse(response.data);
  else { absorb(response); absorb(response?.data); absorb(response?.result); }
  return { answer: answer.trimEnd(), complete, error, sessionId };
}

function trimOuterBlankLines(value) {
  return String(value || '')
    .replace(/^(?:[ \\t]*\\r?\\n)+/, '')
    .replace(/(?:\\r?\\n[ \\t]*)+$/, '');
}

function sliceFromPhysicalLine(value, matchIndex) {
  const text = String(value || '');
  const lineStart = text.lastIndexOf('\\n', Math.max(0, matchIndex - 1)) + 1;
  return text.slice(lineStart);
}

function extractSection(text, titlePatterns, codeStartPattern) {
  for (const titlePattern of titlePatterns) {
    const match = text.match(titlePattern);
    if (!match) continue;
    const rest = text.slice((match.index || 0) + match[0].length);
    const fenced = rest.match(/\\x60\\x60\\x60(?:[\\w-]*)?[ \\t]*\\r?\\n([\\s\\S]*?)\\x60\\x60\\x60/);
    if (fenced) return trimOuterBlankLines(fenced[1]);
    const start = rest.search(codeStartPattern);
    if (start >= 0) {
      const afterStart = sliceFromPhysicalLine(rest, start);
      const nextTitle = afterStart.search(/\\n##\\s+TOINSRC\\//i);
      return trimOuterBlankLines(nextTitle >= 0 ? afterStart.slice(0, nextTitle) : afterStart);
    }
  }
  const fences = text.matchAll(/\\x60\\x60\\x60(?:[\\w-]*)?[ \\t]*\\r?\\n([\\s\\S]*?)\\x60\\x60\\x60/g);
  for (const match of fences) if (codeStartPattern.test(match[1])) return trimOuterBlankLines(match[1]);
  const start = text.search(codeStartPattern);
  if (start >= 0) return trimOuterBlankLines(sliceFromPhysicalLine(text, start).split(/\\n##\\s+TOINSRC\\//i)[0]);
  return '';
}

const results = [];
const seenBatches = new Set();
const requestBySession = new Map(
  sessionRequests
    .filter((item) => item.json?.xelora_session_id)
    .map((item) => [String(item.json.xelora_session_id), item.json])
);
for (let index = 0; index < responses.length; index++) {
  const wrap = responses[index];
  const stream = collectAnswer(wrap.json);
  const pairedIndex = pairedItemIndex(wrap);
  const requestIndex = Number.isInteger(pairedIndex) && sessionRequests[pairedIndex] ? pairedIndex : index;
  const meta = requestBySession.get(stream.sessionId) || sessionRequests[requestIndex]?.json;
  if (!meta) continue;
  seenBatches.add(Number(meta.batch_index));
  const text = stream.answer;
  const base = meta.pgm_base_code;
  const seq = meta.pgm_file_seq;
  const clCode = extractSection(text, [
    new RegExp('##\\\\s+TOINSRC/' + base + 'C(?:' + seq + '|01)\\\\.CL', 'i'),
    new RegExp('##\\\\s+' + base + 'C(?:' + seq + '|01)\\\\.CL', 'i')
  ], /^\\s*(?:PGM|VAR\\s+\\?)/im);
  const cobolCode = extractSection(text, [
    new RegExp('##\\\\s+TOINSRC/' + base + 'B' + seq + '\\\\.CBL', 'i'),
    new RegExp('##\\\\s+' + base + 'B' + seq + '\\\\.CBL', 'i')
  ], /IDENTIFICATION\\s+DIVISION/i);
  let pfCode = extractSection(text, [
    new RegExp('##\\\\s+TOINSRC/' + base + 'F(?:' + seq + '|01)\\\\.PF', 'i'),
    new RegExp('##\\\\s+' + base + 'F(?:' + seq + '|01)\\\\.PF', 'i')
  ], /^\\s*\\/FILE/im);
  if (pfCode && !/\\/END\\s*$/i.test(pfCode)) pfCode += '\\n/END';
  const missing = [!clCode && 'CL', !cobolCode && 'COBOL', !pfCode && 'PF'].filter(Boolean);
  results.push({ json: {
    command_master_id: meta.command_master_id, command: meta.command, version: meta.version,
    project_id: meta.project_id || null, project_name: meta.project_name || null,
    batch_index: meta.batch_index, batch_total: meta.batch_total,
    cases_in_batch: meta.cases_in_batch, total_cases: meta.total_cases_all_rows,
    test_cases_table: meta.test_cases_table, tech_point_context: meta.tech_point_context,
    pgm_base_code: base, pgm_file_seq: seq,
    sub_folder_name: meta.sub_folder_name, time_folder_name: meta.time_folder_name,
    base_dir_container: meta.base_dir_container,
    cl_code: clCode.trimEnd(), cobol_code: cobolCode.trimEnd(), pf_code: pfCode.trimEnd(),
    full_response: text,
	status: stream.complete && !stream.error && text && missing.length === 0 ? 'success' : 'failed',
	error_message: stream.error || (!stream.complete ? 'Xelora SSE 未收到成功 complete 事件' : !text ? 'Xelora 成功结束但正文为空' : missing.length ? '智能体响应缺少文件段: ' + missing.join(', ') : null),
    stream_complete: stream.complete,
    xelora_session_id: meta.xelora_session_id,
    response_session_id: stream.sessionId || '',
    request_time: meta.request_time, response_time: new Date().toISOString()
  }});
}

for (const original of originalRequests) {
  const meta = original.json;
  const batchIndex = Number(meta.batch_index);
  if (seenBatches.has(batchIndex)) continue;
  results.push({ json: {
    command_master_id: meta.command_master_id,
    command: meta.command,
    version: meta.version,
    project_id: meta.project_id || null,
    project_name: meta.project_name || null,
    batch_index: meta.batch_index,
    batch_total: meta.batch_total,
    cases_in_batch: meta.cases_in_batch,
    total_cases: meta.total_cases_all_rows,
    test_cases_table: meta.test_cases_table,
    tech_point_context: meta.tech_point_context,
    pgm_base_code: meta.pgm_base_code,
    pgm_file_seq: meta.pgm_file_seq,
    sub_folder_name: meta.sub_folder_name,
    time_folder_name: meta.time_folder_name,
    base_dir_container: meta.base_dir_container,
    cl_code: '', cobol_code: '', pf_code: '', full_response: '',
    status: 'failed',
    error_message: '缺少第 ' + meta.batch_index + '/' + meta.batch_total + ' 批的 Xelora 会话或智能体响应',
    stream_complete: false,
    request_time: meta.request_time,
    response_time: new Date().toISOString()
  }});
}

return results.sort((a, b) => Number(a.json.batch_index) - Number(b.json.batch_index));`;
processNode.notes = "按 pairedItem 关联并解析 Xelora SSE；缺失响应补为失败批次供原汇总节点处理，不执行代码语义校验。";

aggregateNode.notes = "全部 Xelora 批次完成后合并为 _batch_payloads；仅汇总调用状态和批次完整性，不校验代码语义。";

workflow.nodes.push(extractSessionNode, callAgentNode);
workflow.connections["创建Xelora会话"] = { main: [[{ node: "提取Xelora会话ID", type: "main", index: 0 }]] };
workflow.connections["提取Xelora会话ID"] = { main: [[{ node: "调用Xelora代码生成智能体", type: "main", index: 0 }]] };
workflow.connections["调用Xelora代码生成智能体"] = { main: [[{ node: "处理Xelora智能体响应", type: "main", index: 0 }]] };

fs.writeFileSync(outputPath, JSON.stringify(workflow, null, 2), "utf8");
console.log(`Wrote ${outputPath}`);
console.log(`XELORA_COBOL_VERIFICATION_CODE_AGENT_ID=${agentId}`);
console.log(`XELORA_MANUAL_ASP_KB_ID=${knowledgeBaseId}`);
console.log(`XELORA_MANUAL_ASP_TAG=${knowledgeTagName}`);
console.log(`XELORA_MANUAL_ASP_KNOWLEDGE_IDS=${knowledgeIds.join(",")}`);
