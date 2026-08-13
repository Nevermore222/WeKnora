import { readFileSync, writeFileSync } from "node:fs";

const [input, output] = process.argv.slice(2);
if (!input || !output) throw new Error("usage: node transform-toin-cl-workflow.mjs <input> <output>");
const workflow = JSON.parse(readFileSync(input, "utf8"));
const node = (name) => workflow.nodes.find((item) => item.name === name);
const xeloraSource = JSON.parse(readFileSync("E:/Xelora/n8n/二维表/Xelora版COBOL技术点检证用例工作流（基于tech_point）_1.1.0.json", "utf8"));

const queryNode = node("按小分類查询tech_point代表行");
queryNode.parameters.query = `SELECT tp.id, tp.command_id, tp.pgmid, tp.language, tp.sub_category, tp.tech_point_no,
       tp.source_content, tp.command_code, tp.command_name, tp.location_count, tp.variable_declaration
FROM analyzes.tech_point tp
WHERE tp.project_id = {{ $json.project_id }}
  AND UPPER(TRIM(tp.language)) = 'CL'
  AND UPPER(TRIM(tp.command_name)) NOT IN ('MOVE', 'IF')
  AND tp.sub_category IS NOT NULL AND TRIM(tp.sub_category) <> ''
  AND tp.command_id = {{ $json.command_id }}
  AND NOT EXISTS (
    SELECT 1 FROM analyzes.cl_tech_point_case_references ref
    WHERE ref.project_id = tp.project_id
      AND ref.language = tp.language
      AND ref.command_name = tp.command_name
      AND ref.tech_point_no = tp.tech_point_no
      AND ref.sub_category = tp.sub_category
      AND ref.source_tech_point_id = tp.id
  )
ORDER BY tp.sub_category, NULLIF(TRIM(tp.pgmid), ''), tp.id;`;

const groupNode = node("按批次分组小分類");
groupNode.parameters.jsCode = `// 每个技术点最多抽取五条：先覆盖 PGM，再按 PGM 桶轮询补齐
const allRows = $('按小分類查询tech_point代表行').all().map((item) => item.json);
if (!allRows.length) throw new Error('没有查询到未抽取的 CL tech_point 数据');
const params = $('解析Webhook参数').first()?.json || {};
const projectId = params.project_id ?? 1;
const groups = new Map();
const keyOf = (r) => [r.project_id, r.language, r.command_name, r.tech_point_no, r.sub_category].map((v) => String(v ?? '').trim()).join('\\u0000');
for (const row of allRows) { const key = keyOf(row); if (!groups.has(key)) groups.set(key, []); groups.get(key).push(row); }
const output = [];
for (const [key, rows] of groups) {
  rows.sort((a, b) => { const ap = String(a.pgmid ?? '').trim() || '\\uffff'; const bp = String(b.pgmid ?? '').trim() || '\\uffff'; return ap.localeCompare(bp) || Number(a.id) - Number(b.id); });
  const buckets = new Map();
  for (const row of rows) { const pgm = String(row.pgmid ?? '').trim(); if (!buckets.has(pgm)) buckets.set(pgm, []); buckets.get(pgm).push(row); }
  const refs = [];
  for (let round = 0; refs.length < 5; round++) { let added = false; for (const bucket of buckets.values()) { if (bucket[round]) { refs.push(bucket[round]); added = true; if (refs.length === 5) break; } } if (!added) break; }
  output.push({ json: { project_id: projectId, command_master_id: rows[0].command_id, command: rows[0].command_name, language: rows[0].language, tech_point_no: rows[0].tech_point_no, sub_category: rows[0].sub_category, references: refs, reference_ids: refs.map((r) => Number(r.id)), group_key: key } });
}
return output;`;

const requestNode = node("汇总并准备用例生成请求");
requestNode.parameters.jsCode = `// 将每个技术点的最多五条引用构造成智能体请求
const XELORA_API_BASE_URL = "http://192.168.2.61:18080/api/v1";
const XELORA_API_KEY = "sk-5NB4aUSnLqpsYKoit_FuqbjECutYS-szIJfvxEWgBkKZb5x5";
const XELORA_AGENT_ID = "7286db41-b3e9-4289-ac4d-e6382d6be6e2";
const XELORA_KB_ID = "51c574e9-0445-4c41-b0c7-6eecda9c4638";
const XELORA_KNOWLEDGE_IDS = ["1fb94dcb-7dfb-4aad-aaa8-d9117dbc1244", "0029c569-941c-497b-9063-58d5b221ba4f"];
const results = [];
for (const item of $input.all()) {
  const meta = item.json;
  const refs = Array.isArray(meta.references) ? meta.references : [];
  if (!refs.length) continue;
  const sourceContent = refs.map((r) => [
    '---BEGIN_REFERENCE---',
    'source_tech_point_id: ' + r.id,
    'PGMID: ' + (r.pgmid ?? ''),
    'TECH_POINT_NO: ' + (r.tech_point_no ?? ''),
    'SUB_CATEGORY: ' + (r.sub_category ?? ''),
    'VARIABLE_DECLARATION: ' + (r.variable_declaration ?? ''),
    '---BEGIN_SOURCE_CONTENT---', String(r.source_content ?? ''), '---END_SOURCE_CONTENT---', '---END_REFERENCE---'
  ].join('\\n')).join('\\n\\n');
  const query = [
    'COMMAND_NAME: ' + meta.command,
    'LANGUAGE_TYPE: CL',
    'TECH_POINT_NO: ' + meta.tech_point_no,
    'SUB_CATEGORY: ' + meta.sub_category,
    '以下 references 是同一技术点的证据集合，不要求每条引用分别生成 case。',
    '每条 SOURCE_CONTENT 必须保持原始片段结构；variable_declaration 非空时按原变量名、类型、长度、精度和定义生成参数，空值不得臆造。',
    '生成可直接支撑未来真实 CL 检证代码的二维表，必须保留技术点NO、PGMID、source_tech_point_id，并确保参数配置、输入数据、预期结果相互可执行。',
    'REFERENCES BEGIN', JSON.stringify(refs), 'REFERENCES END', 'SOURCE_CONTENT BEGIN', sourceContent, 'SOURCE_CONTENT END'
  ].join('\\n\\n');
  results.push({ json: { ...meta, references: refs, xelora_api_base_url: XELORA_API_BASE_URL, xelora_api_key: XELORA_API_KEY, xelora_agent_id: XELORA_AGENT_ID, xelora_knowledge_base_id: XELORA_KB_ID, xelora_knowledge_ids: XELORA_KNOWLEDGE_IDS, xelora_session_url: XELORA_API_BASE_URL + '/sessions', xelora_session_request_body: { title: 'CL tech_point cases ' + meta.command + ' ' + meta.tech_point_no }, xelora_agent_request_body: { query, agent_enabled: true, agent_id: XELORA_AGENT_ID, knowledge_base_ids: [XELORA_KB_ID], knowledge_ids: XELORA_KNOWLEDGE_IDS, mentioned_items: [{ id: XELORA_KB_ID, name: 'Manual_ASP', type: 'kb', kb_type: 'document' }], web_search_enabled: false, channel: 'api' }, xelora_query: query } });
}
return results;`;

workflow.nodes = workflow.nodes.filter((item) => !["创建Xelora会话", "提取Xelora会话ID", "调用Xelora用例生成智能体", "处理本批智能体响应", "调用用例生成智能体"].includes(item.name));
for (const name of ["创建Xelora会话", "提取Xelora会话ID", "调用Xelora用例生成智能体"]) {
  const sourceNode = xeloraSource.nodes.find((item) => item.name === name);
  if (!sourceNode) throw new Error(`missing Xelora source node: ${name}`);
  const copy = JSON.parse(JSON.stringify(sourceNode));
  copy.position = [copy.position[0] + 900, copy.position[1]];
  workflow.nodes.push(copy);
}
const responseNode = xeloraSource.nodes.find((item) => item.name === "处理本批智能体响应");
const responseCopy = JSON.parse(JSON.stringify(responseNode));
responseCopy.parameters.jsCode = responseCopy.parameters.jsCode.replace(
  "stream_complete: sawComplete,",
  "stream_complete: sawComplete,\n    references: meta.references || [],",
);
responseCopy.position = [responseCopy.position[0] + 900, responseCopy.position[1]];
workflow.nodes.push(responseCopy);
workflow.connections["小分類逐批处理"] = { main: [[{ node: "创建Xelora会话", type: "main", index: 0 }], [{ node: "最后尾收", type: "main", index: 0 }]] };
workflow.connections["创建Xelora会话"] = { main: [[{ node: "提取Xelora会话ID", type: "main", index: 0 }]] };
workflow.connections["提取Xelora会话ID"] = { main: [[{ node: "调用Xelora用例生成智能体", type: "main", index: 0 }]] };
workflow.connections["调用Xelora用例生成智能体"] = { main: [[{ node: "处理本批智能体响应", type: "main", index: 0 }]] };
workflow.connections["处理本批智能体响应"] = { main: [[{ node: "准备查询MAX序号SQL", type: "main", index: 0 }]] };
workflow.nodes.find((item) => item.name === "汇总并准备用例生成请求").notes = "每个技术点最多五条 PGM 覆盖引用，直接构造 Xelora agent-chat 请求。";
const numberingNode = node("准备本批检证用例接续编号");
numberingNode.parameters.jsCode = numberingNode.parameters.jsCode.replace(
  "return [{ json: { _verification_cases_for_insert: [], _case_count: 0 } }];",
  "return [{ json: { _verification_cases_for_insert: [], _reference_rows: $('处理本批智能体响应').last()?.json?.references || [], _case_count: 0 } }];",
).replace(
  "return [{ json: { _verification_cases_for_insert: rows, _case_count: rows.length } }];",
  "return [{ json: { _verification_cases_for_insert: rows, _reference_rows: $('处理本批智能体响应').last()?.json?.references || [], _case_count: rows.length } }];",
);

const saveNode = node("保存最终检证用例");
saveNode.parameters.query = `BEGIN;
INSERT INTO analyzes.cl_verification_cases_master_proj (command_id, project_id, case_number, sub_category, tech_point_no, test_category, test_purpose, parameter_config, input_data, expected_result, priority, notes)
VALUES
{{ ($json._verification_cases_for_insert || []).filter(tc => tc && tc.command_id != null && tc.command_id !== '').map(tc => {
  function q(v) { if (v == null || v === '') return 'NULL'; return "'" + String(v).replace(/'/g, "''") + "'"; }
  return '(' + tc.command_id + ',' + (tc.project_id ?? 'NULL') + ',' + q(tc.case_number) + ',' + q(tc.sub_category) + ',' + q(tc.tech_point_no) + ',' + q(tc.test_category) + ',' + q(tc.test_purpose) + ',' + q(tc.parameter_config) + ',' + q(tc.input_data) + ',' + q(tc.expected_result) + ',' + q(tc.priority || 'P1') + ',' + q(tc.notes) + ')';
}).join(',\\n') }}
ON CONFLICT (command_id, case_number) DO UPDATE SET project_id = EXCLUDED.project_id, sub_category = EXCLUDED.sub_category, tech_point_no = EXCLUDED.tech_point_no, test_category = EXCLUDED.test_category, test_purpose = EXCLUDED.test_purpose, parameter_config = EXCLUDED.parameter_config, input_data = EXCLUDED.input_data, expected_result = EXCLUDED.expected_result, priority = EXCLUDED.priority, notes = EXCLUDED.notes, updated_at = CURRENT_TIMESTAMP;

INSERT INTO analyzes.cl_tech_point_case_references (project_id, language, command_id, command_name, tech_point_no, sub_category, source_tech_point_id, pgmid, source_content, variable_declaration, extraction_batch_id)
VALUES
{{ ($json._reference_rows || []).map(ref => { function rQ(v) { if (v == null || v === '') return 'NULL'; return "'" + String(v).replace(/'/g, "''") + "'"; } return '(' + ($json.project_id ?? '1') + ',' + rQ($json.language || 'CL') + ',' + ($json.command_master_id ?? 'NULL') + ',' + rQ($json.command || '') + ',' + rQ($json.tech_point_no || '') + ',' + rQ($json.sub_category || '') + ',' + ref.id + ',' + rQ(ref.pgmid) + ',' + rQ(ref.source_content) + ',' + rQ(ref.variable_declaration) + ', gen_random_uuid())'; }).join(',\\n') }}
ON CONFLICT DO NOTHING;
COMMIT;`;

workflow.name = "Xelora TOIN CL 技术点引用抽样与二维表工作流 1.4.0";
workflow.active = false;
writeFileSync(output, JSON.stringify(workflow, null, 2) + "\n", "utf8");
