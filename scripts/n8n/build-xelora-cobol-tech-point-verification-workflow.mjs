#!/usr/bin/env node
import { execFileSync } from "node:child_process";
import fs from "node:fs";
import path from "node:path";

const repoRoot = process.cwd();
const workflowDir = path.resolve(repoRoot, "..", "n8n", "二维表");
const sourceWorkflowName = "COBOL技术点检证用例工作流（基于tech_point）_1.2.5.json";
const outputWorkflowName = "Xelora版COBOL技术点检证用例工作流（基于tech_point）_1.1.0.json";
const outputPath = path.join(workflowDir, outputWorkflowName);

const xeloraApiBaseUrl = "http://192.168.2.61:18080/api/v1";
const xeloraApiKey = "sk-5NB4aUSnLqpsYKoit_FuqbjECutYS-szIJfvxEWgBkKZb5x5";
const xeloraAgentName = "COBOL技术点检证用例生成智能体（基于tech_point）";
const xeloraKnowledgeBaseName = "Manual_ASP";
const xeloraKnowledgeTagName = "ASP_COBOL";

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

function getRequiredXeloraIds() {
  const agentId = runPsql(`
SELECT id
FROM custom_agents
WHERE deleted_at IS NULL
  AND name = ${sqlLiteral(xeloraAgentName)}
ORDER BY updated_at DESC
LIMIT 1;
`);
  if (!agentId) {
    throw new Error(`Xelora agent not found: ${xeloraAgentName}. Run upsert script first.`);
  }

  const knowledgeBaseId = runPsql(`
SELECT id
FROM knowledge_bases
WHERE deleted_at IS NULL
  AND name = ${sqlLiteral(xeloraKnowledgeBaseName)}
ORDER BY updated_at DESC
LIMIT 1;
`);
  if (!knowledgeBaseId) {
    throw new Error(`Xelora knowledge base not found: ${xeloraKnowledgeBaseName}.`);
  }

  const knowledgeIdsText = runPsql(`
SELECT k.id
FROM knowledges k
JOIN knowledge_tags t ON t.id = k.tag_id
WHERE k.deleted_at IS NULL
  AND t.deleted_at IS NULL
  AND k.knowledge_base_id = ${sqlLiteral(knowledgeBaseId)}
  AND t.knowledge_base_id = ${sqlLiteral(knowledgeBaseId)}
  AND t.name = ${sqlLiteral(xeloraKnowledgeTagName)}
ORDER BY k.title;
`);
  const knowledgeIds = knowledgeIdsText.split(/\r?\n/).map((id) => id.trim()).filter(Boolean);
  if (knowledgeIds.length === 0) {
    throw new Error(`No Manual_ASP documents found for tag: ${xeloraKnowledgeTagName}.`);
  }

  return { agentId, knowledgeBaseId, knowledgeIds };
}

function renameNodeInConnections(connections, oldName, newName) {
  if (connections[oldName]) {
    connections[newName] = connections[oldName];
    delete connections[oldName];
  }
  for (const conn of Object.values(connections)) {
    for (const main of conn.main || []) {
      for (const edge of main || []) {
        if (edge.node === oldName) edge.node = newName;
      }
    }
  }
}

function requireNode(workflow, name) {
  const node = workflow.nodes.find((entry) => entry.name === name);
  if (!node) throw new Error(`Node not found: ${name}`);
  return node;
}

function updateNodePosition(workflow, name, x, y) {
  const node = requireNode(workflow, name);
  node.position = [x, y];
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

const { agentId, knowledgeBaseId, knowledgeIds } = getRequiredXeloraIds();
const sourcePath = path.join(workflowDir, sourceWorkflowName);
const workflow = JSON.parse(fs.readFileSync(sourcePath, "utf8"));

workflow.name = "Xelora版COBOL技术点检证用例工作流（基于tech_point）_1.1.0";
workflow.active = false;
workflow.id = undefined;
workflow.versionId = undefined;
workflow.meta = undefined;
workflow.tags = [{ name: "Xelora" }, { name: "COBOL" }, { name: "tech_point" }];

const webhookParamsNode = requireNode(workflow, "解析Webhook参数");
const projectAssignment = webhookParamsNode.parameters.assignments.assignments.find(
  (entry) => entry.name === "project_id",
);
if (!projectAssignment) throw new Error("Webhook project_id assignment was not found.");
projectAssignment.value = "={{ $json.body?.project_id ?? $json.project_id }}";
projectAssignment.type = "number";

const sourceQueryNode = requireNode(workflow, "按小分類查询tech_point代表行");
sourceQueryNode.name = "按项目查询tech_point行";
renameNodeInConnections(
  workflow.connections,
  "按小分類查询tech_point代表行",
  "按项目查询tech_point行",
);
sourceQueryNode.parameters.query = `SELECT
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
ORDER BY t.sub_category, t.tech_point_no, t.id;`;

for (const node of workflow.nodes.filter((entry) => entry.type === "n8n-nodes-base.code")) {
  node.parameters.jsCode = String(node.parameters?.jsCode || "").replaceAll(
    "按小分類查询tech_point代表行",
    "按项目查询tech_point行",
  );
}

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
delete validateProjectNode.onError;

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
delete assertProjectNode.onError;

const projectCaseKeyGuardNode = {
  id: "check-project-case-key-v110",
  name: "检查项目级用例唯一键",
  type: "n8n-nodes-base.postgres",
  typeVersion: 2.1,
  position: [1080, 280],
  credentials: sourceQueryNode.credentials,
  parameters: {
    operation: "executeQuery",
    query: `WITH key_check AS (
  SELECT EXISTS (
    SELECT 1
    FROM pg_indexes
    WHERE schemaname = 'analyzes'
      AND tablename = 'cl_verification_cases_master_proj'
      AND indexdef ILIKE 'CREATE UNIQUE%'
      AND replace(replace(indexdef, ' ', ''), '"', '') ILIKE '%(command_id,project_id,case_number)%'
  ) AS has_project_case_key,
  EXISTS (
    SELECT 1
    FROM pg_indexes
    WHERE schemaname = 'analyzes'
      AND tablename = 'cl_verification_cases_master_proj'
      AND indexdef ILIKE 'CREATE UNIQUE%'
      AND replace(replace(indexdef, ' ', ''), '"', '') ILIKE '%(command_id,case_number)%'
      AND replace(replace(indexdef, ' ', ''), '"', '') NOT ILIKE '%(command_id,project_id,case_number)%'
  ) AS has_legacy_global_case_key
)
SELECT {{ $json.project_id }}::int AS project_id, has_project_case_key
FROM key_check;`,
    options: {},
  },
  notes: "在删除旧用例之前确认目标库已执行项目级唯一键迁移。",
};

const assertProjectCaseKeyNode = makeCodeNode(
  "assert-project-case-key-v110",
  "确认项目级用例唯一键",
  [1260, 280],
  `const row = $input.first()?.json || {};
if (row.has_project_case_key !== true && row.has_project_case_key !== 't') {
  throw new Error('目标数据库缺少项目级唯一键，请先执行 2026-08-11-project-scope-verification-case-key.sql');
}
if (row.has_legacy_global_case_key === true || row.has_legacy_global_case_key === 't') {
  throw new Error('目标数据库仍存在旧的全局(command_id, case_number)唯一键，请重新执行 2026-08-11-project-scope-verification-case-key.sql');
}
return [{ json: { project_id: Number(row.project_id), has_project_case_key: true, has_legacy_global_case_key: false } }];`,
  "唯一键不存在时立即停止，避免删除旧数据后在 ON CONFLICT 处失败。",
);
delete assertProjectCaseKeyNode.onError;

workflow.nodes.push(
  validateProjectNode,
  lookupProjectNode,
  assertProjectNode,
  projectCaseKeyGuardNode,
  assertProjectCaseKeyNode,
);
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
  main: [[{ node: "检查项目级用例唯一键", type: "main", index: 0 }]],
};
workflow.connections["检查项目级用例唯一键"] = {
  main: [[{ node: "确认项目级用例唯一键", type: "main", index: 0 }]],
};
workflow.connections["确认项目级用例唯一键"] = {
  main: [[{ node: "按项目查询tech_point行", type: "main", index: 0 }]],
};

const prepareRequestNode = requireNode(workflow, "汇总并准备用例生成请求");
prepareRequestNode.parameters.jsCode = `// 多批请求：将每个 tech_point 小分類 batch 转为一条 Xelora 智能体请求。
const batchInputs = $input.all();
if (!batchInputs || batchInputs.length === 0) throw new Error('没有批次数据');

const XELORA_API_BASE_URL = ${JSON.stringify(xeloraApiBaseUrl)};
const XELORA_API_KEY = ${JSON.stringify(xeloraApiKey)};
const XELORA_AGENT_ID = ${JSON.stringify(agentId)};
const XELORA_KB_ID = ${JSON.stringify(knowledgeBaseId)};
const XELORA_KNOWLEDGE_IDS = ${JSON.stringify(knowledgeIds)};

const webhookParams = $('解析Webhook参数').first();
const projectId = webhookParams?.json?.project_id ?? null;
const language = webhookParams?.json?.language ?? null;

const results = [];
for (const wrap of batchInputs) {
  const batchItem = wrap.json;
  const allRows = batchItem.batch_rows;
  if (!allRows || !Array.isArray(allRows) || allRows.length === 0) {
    throw new Error('批次缺少 batch_rows');
  }

  const commandId = batchItem.command_master_id || null;
  const commandName = (batchItem.command || '').trim();
  const batchIndex = batchItem.batch_index || 1;
  const batchTotal = batchItem.batch_total || 1;
  if (!commandName) throw new Error('无法获取command_name');

  const sourceParts = [];
  for (const row of allRows) {
    const techPointNo = String(row.tech_point_no || '').trim();
    const subCategory = String(row.sub_category || '').trim();
    const sourceContent = String(row.source_content || '').trim();
    const variableDeclaration = String(row.variable_declaration || '').trim();
    if (!techPointNo) throw new Error('批次包含缺少 tech_point_no 的技术点');
    const lines = [
      '【技術点NO】' + techPointNo,
      '【小分類】' + subCategory,
    ];
    if (variableDeclaration) lines.push('【変数宣言】' + variableDeclaration);
    lines.push('【SOURCE_CONTENT】' + sourceContent);
    sourceParts.push(lines.join(String.fromCharCode(10)));
  }

  const sourceContentFormatted = sourceParts.join('\\n\\n');
  if (!sourceContentFormatted) throw new Error('第 ' + batchIndex + ' 批所有小分類的source_content均为空');

  const retrievalQuery = [
    commandName + ' エラー',
    commandName + ' 異常',
    commandName + ' テスト',
    commandName + ' FACOM',
    'COBOL-85 ' + commandName,
    commandName + ' ROUNDED',
    commandName + ' ON SIZE ERROR'
  ].join(' ');

  const query = [
    '以下の SOURCE_CONTENT（' + batchIndex + '/' + batchTotal + ' バッチ、このバッチは ' + allRows.length + ' 小分類）に基づき、網羅的な COBOL 検証ケース表を作成してください。',
    'COMMAND_NAME: ' + commandName,
    'LANGUAGE_TYPE: COBOL',
    'RETRIEVAL_QUERY: ' + retrievalQuery,
    'JAPANESE_ONLY_OUTPUT: 最終回答の見出し、説明文、表頭、全セルをすべて日本語で記述してください。中国語の文章および中国語の簡体字を出力してはいけません。識別子、COBOL構文、数値、技術点NOのみ原文を維持してください。',
    'このバッチの SOURCE_CONTENT に現れるすべての【小分類】を漏れなく対象にしてください。',
    '各【技術点NO】ブロックは独立した根拠です。【変数宣言】と【SOURCE_CONTENT】を別の技術点へ流用してはいけません。',
    '主表は必ず次の10列にしてください：| 技術点NO | 用例番号 | 小分類 | テスト分類 | テスト目的 | パラメータ設定 | 入力データ | 期待結果 | 優先度 | 備考 |',
    '各小分類について、正常系、境界値、異常系を最低1件ずつ作成してください。ROUNDED、除算、ON SIZE ERROR、括弧演算、中間結果、符号、小数型が含まれる場合は、それぞれの専用ケースも追加してください。',
    '技術点NOは SOURCE_CONTENT の【技術点NO】または【技术点NO】と完全に一致させてください。',
    'SUB_CATEGORY_TYPE_CONTRACT: 小分類を各行の型および構文の正規情報として扱い、受取項目型、全オペランド型、演算子、Round有/無、SIZE有/無、ON SIZE ERRORを抽出してパラメータ設定に反映してください。9型、S9型、9V9型、S9V9型、COMP-3型を別の型へ置き換えてはいけません。',
    'CODEGEN_READY_PARAMETER_SETTING: パラメータ設定には、受取項目と全入力項目の具体的な COBOL PIC/USAGE 宣言、および完全な検証対象文を含めてください。同一セル内はセミコロンまたは <br> で区切り、PIC 9(n)、PIC ?、説明文だけの型指定は禁止します。',
    'CANONICAL_COBOL_DECLARATION_ORDER: すべての宣言は正しい COBOL 順序「01 データ名 PIC 型 [COMP-3].」で記述してください。例：01 RD1130-JYURYO PIC 9(5).、01 MSHIN-R4020 PIC 9(3)V9(2) COMP-3.。「PIC 9(5) RD1130-JYURYO」のような逆順は禁止します。',
    'CONCRETE_INPUT_ASSIGNMENTS: 入力データには、検証対象文の右辺で参照する各変数について NAME=value 形式の具体値を設定し、符号、整数桁、小数桁、USAGEに適合させてください。MAX_VAL、MIN_VAL、N/A、任意値などのプレースホルダーは禁止します。',
    'VERIFICATION_DESIGN_ASSUMPTION: 原文に PIC 桁数がない場合、実行可能な最小限の具体的 PIC を選択し、備考に「PIC桁数は検証設計上の仮定」と採用宣言を明記してください。Manual の記載事項として扱ってはいけません。',
    'Round有では検証対象文に ROUNDED を含め、Round無では含めないでください。SIZE有または ON SIZE ERROR では実行可能な異常処理句を含め、SIZE無では追加しないでください。',
    '各行の PIC 宣言、入力データ、検証対象文、期待結果を相互に整合させ、期待結果をその行だけから算出できるようにしてください。',
    '除算では被除数0と除数0を区別し、ゼロ除算として扱うのは除数0だけです。',
    'CRITICAL TABLE RULES: main Markdown table rows must have exactly 10 columns matching header order; never omit repeated tech point or sub-category cells.',
    'Do not use rowspan, ditto marks, same as above, blank repeated cells, MAX_VAL, MIN_VAL, N/A, TBD, or unknown in any table row.',
    'Every data row must repeat the full tech point number and full sub-category from SOURCE_CONTENT.',
    '最終出力前に全体を見直し、中国語の語句があれば自然な日本語へ書き換えてください。この確認作業自体は回答に記載しないでください。',
    'SOURCE_CONTENT BEGIN',
    sourceContentFormatted,
    'SOURCE_CONTENT END'
  ].join('\\n\\n');

  results.push({
    json: {
      command_master_id: commandId,
      command: commandName,
      project_id: projectId,
      language,
      batch_index: batchIndex,
      batch_total: batchTotal,
      sub_category_count: allRows.length,
      batch_rows: allRows,
      xelora_api_base_url: XELORA_API_BASE_URL,
      xelora_api_key: XELORA_API_KEY,
      xelora_agent_id: XELORA_AGENT_ID,
      xelora_knowledge_base_id: XELORA_KB_ID,
      xelora_knowledge_ids: XELORA_KNOWLEDGE_IDS,
      xelora_session_url: XELORA_API_BASE_URL + '/sessions',
      xelora_session_request_body: {
        title: 'COBOL tech_point cases ' + commandName + ' batch ' + batchIndex + '/' + batchTotal
      },
      xelora_agent_request_body: {
        query,
        agent_enabled: true,
        agent_id: XELORA_AGENT_ID,
        knowledge_base_ids: [XELORA_KB_ID],
        knowledge_ids: XELORA_KNOWLEDGE_IDS,
        mentioned_items: [{
          id: XELORA_KB_ID,
          name: 'Manual_ASP',
          type: 'kb',
          kb_type: 'document'
        }],
        web_search_enabled: false,
        channel: 'api'
      },
      xelora_query: query,
      request_time: new Date().toISOString()
    }
  });
}

return results;`;
prepareRequestNode.notes =
  "将每个 tech_point 小分類批次转换为 Xelora agent-chat 请求；工作流内部直接携带 API base、API key、agent_id、knowledge_base_id。";

renameNodeInConnections(workflow.connections, "调用用例生成智能体", "创建Xelora会话");
const createSessionNode = requireNode(workflow, "调用用例生成智能体");
createSessionNode.name = "创建Xelora会话";
createSessionNode.parameters = {
  method: "POST",
  url: "={{ $json.xelora_session_url }}",
  sendHeaders: true,
  headerParameters: {
    parameters: [
      { name: "X-API-Key", value: "={{ $json.xelora_api_key }}" },
      { name: "Content-Type", value: "application/json" },
    ],
  },
  sendBody: true,
  specifyBody: "json",
  jsonBody: "={{ $json.xelora_session_request_body }}",
  options: { response: { response: { neverError: true } } },
};
createSessionNode.notes = "调用 Xelora /api/v1/sessions，为当前小分類批次创建独立会话。";

const extractSessionNode = makeCodeNode(
  "xelora-extract-cobol-tech-point-session",
  "提取Xelora会话ID",
  [496, 80],
  `let cur = null;
try {
  cur = $('小分類逐批处理').item;
} catch (e) {}
if (!cur || !cur.json) cur = $('小分類逐批处理').last();
if (!cur || !cur.json) throw new Error('无法获取当前小分類批次项');

const meta = cur.json;
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
      ...meta,
      status: 'failed',
      error_message: 'Xelora会话创建失败: ' + JSON.stringify(response).slice(0, 5000),
      full_verification_cases_response: '',
      response_time: new Date().toISOString()
    }
  }];
}

return [{
  json: {
    ...meta,
    xelora_session_id: String(sessionId),
    xelora_agent_url: meta.xelora_api_base_url + '/agent-chat/' + String(sessionId)
  }
}];`,
  "从 /sessions 响应中提取 session_id，并拼接 /agent-chat/{session_id}。",
);

const callAgentNode = makeHttpNode(
  "xelora-call-cobol-tech-point-agent",
  "调用Xelora用例生成智能体",
  [720, 80],
  {
    method: "POST",
    url: "={{ $json.xelora_agent_url }}",
    sendHeaders: true,
    headerParameters: {
      parameters: [
        { name: "X-API-Key", value: "={{ $json.xelora_api_key }}" },
        { name: "Content-Type", value: "application/json" },
      ],
    },
    sendBody: true,
    specifyBody: "json",
    jsonBody: "={{ $json.xelora_agent_request_body }}",
    options: { response: { response: { neverError: true } } },
  },
  "调用 Xelora /api/v1/agent-chat/{session_id}，返回 SSE 流式响应。",
);

const processResponseNode = requireNode(workflow, "处理本批智能体响应");
processResponseNode.position = [944, 80];
processResponseNode.parameters.jsCode = `let cur = null;
try {
  cur = $('提取Xelora会话ID').item;
} catch (e) {}
if (!cur || !cur.json) cur = $('提取Xelora会话ID').last();
if (!cur || !cur.json) throw new Error('无法获取当前Xelora会话批次项');

const meta = cur.json;
let fullAnswer = '';
let status = 'failed';
let errorMessage = null;
let sawComplete = false;
const response = $input.first().json;

function absorbEvent(event) {
  if (!event || typeof event !== 'object') return;
  if (typeof event.answer === 'string') fullAnswer += event.answer;
  const responseType = String(event.response_type || event.event || event.type || '').toLowerCase();
  const isAnswerChunk =
    responseType === '' ||
    responseType === 'answer' ||
    responseType === 'message' ||
    responseType === 'agent_message' ||
    responseType === 'final_answer' ||
    responseType === 'final_answer_chunk';
  const isNonAnswerEvent =
    responseType === 'agent_query' ||
    responseType === 'tool_call' ||
    responseType === 'tool_result' ||
    responseType === 'thinking' ||
    responseType === 'complete' ||
    responseType === 'error';
  if (typeof event.content === 'string' && isAnswerChunk && !isNonAnswerEvent) fullAnswer += event.content;
	if (responseType === 'complete') sawComplete = true;
	if (responseType === 'error' && !fullAnswer) errorMessage = String(event.content || event.error || event.data?.error || 'Xelora error');
}

function parseSseText(text) {
  for (const line of String(text || '').split('\\n')) {
    const trimmed = line.trimStart();
    if (!trimmed.startsWith('data:')) continue;
    const payload = trimmed.slice(5).trim();
    if (!payload || payload === '[DONE]') {
      sawComplete = true;
      continue;
    }
    try {
      absorbEvent(JSON.parse(payload));
    } catch (e) {}
  }
}

try {
  if (typeof response === 'string') {
    parseSseText(response);
  } else if (typeof response?.data === 'string') {
    parseSseText(response.data);
  } else {
    absorbEvent(response);
    absorbEvent(response?.data);
    absorbEvent(response?.result);
  }
	if (sawComplete && !errorMessage && fullAnswer.trim()) status = 'success';
	else if (!sawComplete) errorMessage = errorMessage || 'Xelora SSE 未收到成功 complete 事件';
	else errorMessage = errorMessage || 'Xelora 成功结束但正文为空';
} catch (error) {
  errorMessage = 'Xelora响应解析失败: ' + error.message;
}

return [{
  json: {
    command_master_id: meta.command_master_id,
    command: meta.command,
    language: meta.language,
    project_id: meta.project_id ?? null,
    batch_index: meta.batch_index,
    batch_total: meta.batch_total,
    sub_category_count: meta.sub_category_count,
    xelora_session_id: meta.xelora_session_id || '',
    full_verification_cases_response: fullAnswer.trim(),
    status,
    error_message: errorMessage,
    stream_complete: sawComplete,
    request_time: meta.request_time || new Date().toISOString(),
    response_time: new Date().toISOString()
  }
}];`;
processResponseNode.notes = "解析 Xelora SSE 流，提取 answer/final_answer 内容，保留批次元数据供后续表格解析与落库。";

workflow.nodes.push(extractSessionNode, callAgentNode);

workflow.connections["创建Xelora会话"] = {
  main: [[{ node: "提取Xelora会话ID", type: "main", index: 0 }]],
};
workflow.connections["提取Xelora会话ID"] = {
  main: [[{ node: "调用Xelora用例生成智能体", type: "main", index: 0 }]],
};
workflow.connections["调用Xelora用例生成智能体"] = {
  main: [[{ node: "处理本批智能体响应", type: "main", index: 0 }]],
};

const parseCasesNode = requireNode(workflow, "解析本批检证用例");
const normalizerAnchor = "function isValidTestCaseCore(tc, tp, pc, ind, er) {";
const cellNormalizer = `function normalizeCellText(value) {
  return String(value == null ? '' : value)
    .replace(/&lt;br\\s*\\/?\\s*&gt;/gi, '\\n')
    .replace(/<br\\s*\\/?\\s*>/gi, '\\n')
    .replace(/\\\\r\\\\n|\\\\n|\\\\r/g, '\\n')
    .replace(/\\r\\n?/g, '\\n')
    .replace(/[ \\t]*\\n[ \\t]*/g, '\\n')
    .replace(/\\n{3,}/g, '\\n\\n')
    .trim();
}
`;
if (!parseCasesNode.parameters.jsCode.includes(normalizerAnchor)) {
  throw new Error(`Parser normalizer anchor not found: ${normalizerAnchor}`);
}
parseCasesNode.parameters.jsCode = parseCasesNode.parameters.jsCode.replace(
  normalizerAnchor,
  cellNormalizer + "\n" + normalizerAnchor,
);

const parsedRowAnchor = `          sub_category: subCat.substring(0, 256),
          test_category: tcS,
          test_purpose: tpS,
          parameter_config: pcS,
          input_data: idS,
          expected_result: erS,
          priority,
          notes: notes || '',`;
const normalizedParsedRow = `          sub_category: normalizeCellText(subCat).substring(0, 256),
          test_category: normalizeCellText(tcS),
          test_purpose: normalizeCellText(tpS),
          parameter_config: normalizeCellText(pcS),
          input_data: normalizeCellText(idS),
          expected_result: normalizeCellText(erS),
          priority,
          notes: normalizeCellText(notes),`;
if (!parseCasesNode.parameters.jsCode.includes(parsedRowAnchor)) {
  throw new Error("Parser output-row anchor not found for cell normalization.");
}
parseCasesNode.parameters.jsCode = parseCasesNode.parameters.jsCode.replace(
  parsedRowAnchor,
  normalizedParsedRow,
);

const prepareDeleteSqlNode = requireNode(workflow, "准备删除SQL");
prepareDeleteSqlNode.parameters.jsCode = `const deleteData = $input.first();
if (!deleteData?.json) throw new Error('无法获取删除数据');
const commandId = Number(deleteData.json.command_id);
const projectId = Number(deleteData.json.project_id);
if (!Number.isInteger(commandId) || commandId <= 0) throw new Error('command_id 无效');
if (!Number.isInteger(projectId) || projectId <= 0) throw new Error('project_id 无效');
const deleteSql = 'DELETE FROM analyzes.cl_verification_cases_master_proj WHERE command_id = ' +
  commandId + ' AND project_id = ' + projectId + ';';
return [{ json: { sql: deleteSql, command_id: commandId, project_id: projectId } }];`;

const prepareMaxSqlNode = requireNode(workflow, "准备查询MAX序号SQL");
prepareMaxSqlNode.parameters.jsCode = `const item = $input.first()?.json || {};
const commandId = Number(item.command_master_id);
const projectId = Number(item.project_id);
if (!Number.isInteger(commandId) || commandId <= 0) throw new Error('command_master_id 无效');
if (!Number.isInteger(projectId) || projectId <= 0) throw new Error('project_id 无效');
const sql = "SELECT COALESCE(MAX(CAST(REGEXP_REPLACE(TRIM(UPPER(case_number)), '^TC0*', '', 'i') AS INTEGER)), 0) AS max_seq " +
  'FROM analyzes.cl_verification_cases_master_proj WHERE command_id = ' + commandId +
  ' AND project_id = ' + projectId + ';';
return [{ json: { sql, command_master_id: commandId, project_id: projectId } }];`;

const continueNumberingNode = requireNode(workflow, "准备本批检证用例接续编号");
continueNumberingNode.parameters.jsCode = `const parsedCases = $input.all();
if (parsedCases.length === 1 && parsedCases[0].json && parsedCases[0].json._empty_parse) {
  return [{ json: { _verification_cases_for_insert: [], _case_count: 0 } }];
}

const projectId = Number($('解析Webhook参数').first()?.json?.project_id);
if (!Number.isInteger(projectId) || projectId <= 0) throw new Error('project_id 无效');

let maxSeq = 0;
const assembled = $('组装本批解析输入').last()?.json;
if (assembled && (assembled._db_max_seq != null || assembled.max_seq != null)) {
  maxSeq = parseInt(assembled._db_max_seq ?? assembled.max_seq ?? 0, 10) || 0;
} else {
  maxSeq = parseInt($('查询当前最大case序号').first()?.json?.max_seq ?? 0, 10) || 0;
}

if (!parsedCases.length) {
  return [{ json: { _verification_cases_for_insert: [], _case_count: 0 } }];
}
const commandMasterId = Number(parsedCases[0].json.command_id);
if (!Number.isInteger(commandMasterId) || commandMasterId <= 0) {
  throw new Error('无法获取command_master_id');
}

let currentBatch = null;
try {
  currentBatch = $('小分類逐批处理').item;
} catch (error) {}
if (!currentBatch?.json) currentBatch = $('小分類逐批处理').last();
const sourceRows = currentBatch?.json?.batch_rows;
if (!Array.isArray(sourceRows) || !sourceRows.length) throw new Error('当前批次缺少技术点原始行');

const sourceByTechPoint = new Map();
for (const sourceRow of sourceRows) {
  const techPointNo = String(sourceRow.tech_point_no || '').trim();
  if (!techPointNo) throw new Error('项目技术点缺少 tech_point_no');
  if (sourceByTechPoint.has(techPointNo)) {
    throw new Error('项目批次内 tech_point_no 重复: ' + techPointNo);
  }
  sourceByTechPoint.set(techPointNo, sourceRow);
}

function padCaseNum(number) {
  return 'TC' + String(number).padStart(4, '0');
}

const sorted = [...parsedCases].sort((left, right) => {
  const categoryOrder = String(left.json.sub_category || '').localeCompare(
    String(right.json.sub_category || ''),
    'ja',
  );
  if (categoryOrder !== 0) return categoryOrder;
  return String(left.json.case_number || '').localeCompare(
    String(right.json.case_number || ''),
    undefined,
    { numeric: true },
  );
});

const rows = sorted.map((testCase, index) => {
  const techPointNo = String(testCase.json.tech_point_no || '').trim().substring(0, 128);
  if (!techPointNo) throw new Error('智能体输出行缺少 tech_point_no');
  const sourceRow = sourceByTechPoint.get(techPointNo);
  if (!sourceRow) throw new Error('智能体输出了当前批次以外的 tech_point_no: ' + techPointNo);

  const outputSubCategory = String(testCase.json.sub_category || '').trim();
  const sourceSubCategory = String(sourceRow.sub_category || '').trim();
  if (outputSubCategory !== sourceSubCategory) {
    throw new Error('tech_point_no 与 sub_category 不匹配: ' + techPointNo);
  }

  return {
    command_id: commandMasterId,
    project_id: projectId,
    case_number: padCaseNum(maxSeq + index + 1),
    sub_category: outputSubCategory.substring(0, 256),
    tech_point_no: techPointNo,
    test_category: String(testCase.json.test_category || '').substring(0, 100),
    test_purpose: testCase.json.test_purpose || '',
    parameter_config: testCase.json.parameter_config || '',
    input_data: testCase.json.input_data || '',
    expected_result: testCase.json.expected_result || '',
    priority: String(testCase.json.priority || 'P1').substring(0, 50),
    notes: testCase.json.notes || '',
  };
});

return [{ json: { _verification_cases_for_insert: rows, _case_count: rows.length } }];`;

const saveCasesNode = requireNode(workflow, "保存最终检证用例");
const oldConflictClause =
  "ON CONFLICT (command_id, case_number) DO UPDATE SET project_id = EXCLUDED.project_id, ";
if (!saveCasesNode.parameters.query.includes(oldConflictClause)) {
  throw new Error("Legacy verification-case conflict clause was not found.");
}
saveCasesNode.parameters.query = saveCasesNode.parameters.query.replace(
  oldConflictClause,
  "ON CONFLICT (command_id, project_id, case_number) DO UPDATE SET ",
);

updateNodePosition(workflow, "准备查询MAX序号SQL", 1088, 80);
updateNodePosition(workflow, "查询当前最大case序号", 1280, 80);
updateNodePosition(workflow, "组装本批解析输入", 1432, 80);
updateNodePosition(workflow, "解析本批检证用例", 1584, 80);
updateNodePosition(workflow, "准备本批检证用例接续编号", 1888, 80);
updateNodePosition(workflow, "本批是否有行可插入", 2192, 80);
updateNodePosition(workflow, "保存最终检证用例", 2368, 80);

fs.writeFileSync(outputPath, JSON.stringify(workflow, null, 2), "utf8");

console.log(`Wrote ${outputPath}`);
console.log(`XELORA_COBOL_TECH_POINT_VERIFICATION_AGENT_ID=${agentId}`);
console.log(`XELORA_MANUAL_ASP_KB_ID=${knowledgeBaseId}`);
console.log(`XELORA_MANUAL_ASP_TAG=${xeloraKnowledgeTagName}`);
console.log(`XELORA_MANUAL_ASP_KNOWLEDGE_IDS=${knowledgeIds.join(",")}`);
