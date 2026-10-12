#!/usr/bin/env node
import fs from "node:fs";
import path from "node:path";

const repoRoot = process.cwd();
const workflowDir = path.resolve(repoRoot, "..", "n8n", "检证用例代码");
const sourcePath = path.join(workflowDir, "Xelora版CL技术点检证用例代码生成工作流_技术点三文件_1.0.44.json");
const outputPath = path.join(workflowDir, "Xelora通用检证用例代码生成工作流_1.0.0.json");
if (!fs.existsSync(sourcePath)) throw new Error(`Source workflow does not exist: ${sourcePath}`);
const DEFAULT_CODE_PROJECT_ID = 1;
const workflow = JSON.parse(fs.readFileSync(sourcePath, "utf8"));
workflow.id = "XeloraGenVerCodeA8B247V1";
workflow.name = "Xelora通用检证用例代码生成工作流_1.0.0";
workflow.active = false;
workflow.tags = [{ name: "Xelora" }, { name: "通用" }, { name: "检证用例代码" }];

const node = (name) => {
  const found = workflow.nodes.find((entry) => entry.name === name);
  if (!found) throw new Error(`Missing node: ${name}`);
  return found;
};
const code = (id, name, position, jsCode, notes) => ({ id, name, type: "n8n-nodes-base.code", typeVersion: 2, position, parameters: { jsCode }, notes });
const postgres = (id, name, position, query, notes) => ({ id, name, type: "n8n-nodes-base.postgres", typeVersion: 2.1, position, parameters: { operation: "executeQuery", query, options: {} }, credentials: { postgres: { id: "tWcUIJhVg6bDlGDc", name: "TargetDB" } }, onError: "continueErrorOutput", notes });

const input = node("解析参数");
input.parameters.jsCode = `const b = $json.body || $json;
const rawCommand = String(b.command ?? b.command_name ?? b.commandName ?? '').trim().toUpperCase();
const command_id = Number(b.command_id ?? b.commandId ?? b.command_master_id ?? b.commandMasterId);
const command = rawCommand.replace(new RegExp('^' + command_id + '_'), '');
const batchValue = b.case_table_batch_no ?? b.caseTableBatchNo;
const case_table_batch_no = batchValue == null || String(batchValue).trim() === '' ? null : Number(batchValue);
const timestampValue = b.command_run_timestamp ?? b.commandRunTimestamp;
const command_run_timestamp = timestampValue == null || String(timestampValue).trim() === '' ? null : String(timestampValue).trim();
const force_reprocess = b.force_reprocess === true || b.forceReprocess === true || String(b.force_reprocess ?? b.forceReprocess ?? '').trim().toLowerCase() === 'true';
if (!command) throw new Error('command is required');
if (!Number.isInteger(command_id) || command_id <= 0) throw new Error('command_id is required');
if (case_table_batch_no != null && (!Number.isInteger(case_table_batch_no) || case_table_batch_no <= 0)) throw new Error('case_table_batch_no must be a positive integer when supplied');
if (command_run_timestamp && !/^\\d{14}$/.test(command_run_timestamp)) throw new Error('command_run_timestamp must be YYYYMMDDHHMMSS');
const test_category = String(b.test_category ?? b.testCategory ?? '').trim();
const modeValue = String(b.generation_mode ?? b.generationMode ?? '').trim().toLowerCase();
const generation_mode = modeValue || (command_id === 227 ? 'move_master' : 'case_table');
if (!['case_table', 'move_master'].includes(generation_mode)) throw new Error('generation_mode must be case_table or move_master');
return [{ json: { project_id: ${DEFAULT_CODE_PROJECT_ID}, command, command_id, test_category, case_table_batch_no, command_run_timestamp, force_reprocess, generation_mode, request_id: String(Date.now()) } }];`;
input.notes = "二维表代码生成入口：前台只需传 command 和 command_id；project_id 固定由工作流使用默认代码项目 1；仅从最新 8 列检证用例二维表遍历生成。case_table_batch_no 可只回归一个五用例批次。";

const webhook = node("Webhook");
webhook.parameters.path = "xelora-verification-code-generic";
webhook.webhookId = "xelora-verification-code-generic";
webhook.notes = "通用检证用例代码入口，不限制 ASP/CL";

const query = node("查询待处理CL技术点");
const oldQueryName = query.name;
query.name = "查询最新检证用例二维表";
query.parameters.query = `WITH move_cases AS (
  SELECT c.command_id, cm.command AS command_name, 1::int AS project_id, cm.language,
         1::int AS case_table_version, c.case_number, c.test_category, c.test_purpose,
         c.parameter_config, c.input_data, c.expected_result, c.priority, c.notes
  FROM analyzes.cl_verification_cases_master c
  JOIN analyzes.command_master cm ON cm.id = c.command_id
  WHERE c.command_id = {{$json.command_id}}
    AND {{$json.command_id}} = 227
    AND c.case_number LIKE 'MOVE_TC%'
    AND COALESCE(c.parameter_config::jsonb->>'batch_id','') = 'MOVE_BATCH_001'
), legacy_cases AS (
  SELECT c.command_id, cm.command AS command_name,
         '{{$json.project_id}}'::int AS project_id, cm.language,
         c.version AS case_table_version, c.case_number, c.test_category,
         c.test_purpose, c.parameter_config, c.input_data, c.expected_result,
         c.priority, c.notes
  FROM analyzes.cl_verification_cases c
  JOIN analyzes.command_master cm ON cm.id=c.command_id
  WHERE c.command_id={{$json.command_id}}
    AND {{$json.command_id}} <> 227
    AND c.version = (SELECT max(version)::int FROM analyzes.cl_verification_cases WHERE command_id={{$json.command_id}})
)
SELECT * FROM move_cases
UNION ALL
SELECT * FROM legacy_cases
ORDER BY case_number;`;
query.notes = "唯一数据源：按 command_id 读取 cl_verification_cases 最新版本的完整八列检证用例；不读取或回退到 tech_point/SOURCE。";

const sampler = node("按PGM优先抽取最多五条SOURCE");
sampler.name = "按二维表遍历代码生成批次";
sampler.parameters.jsCode = `const request = $('解析参数').first().json;
const allRows = $input.all().map((item) => item.json || {}).filter((row) => String(row.case_number || '').trim());
const rows = request.test_category ? allRows.filter((row) => String(row.test_category || '').trim() === request.test_category) : allRows;
if (!rows.length) return [{ json: { status: 'failed', error_message: '当前命令没有最新版本的检证用例二维表数据' } }];
const required = ['case_number','test_category','test_purpose','expected_result','priority'];
const invalid = rows.find((row) => required.some((field) => String(row[field] ?? '').trim() === ''));
if (invalid) return [{ json: { status: 'failed', error_message: '最新检证用例二维表存在空字段: ' + String(invalid.case_number || '(unknown)') } }];
const caseNumbers = rows.map((row) => String(row.case_number).trim().toUpperCase());
const malformed = caseNumbers.find((caseNumber) => request.generation_mode === 'move_master' ? !/^MOVE_TC[0-9]+$/.test(caseNumber) : !/^TC[0-9]+$/.test(caseNumber));
if (malformed) return [{ json: { status: 'failed', error_message: '检证用例番号格式不正确: ' + malformed } }];
const duplicate = caseNumbers.find((caseNumber, index) => caseNumbers.indexOf(caseNumber) !== index);
if (duplicate) return [{ json: { status: 'failed', error_message: '最新检证用例二维表存在重复用例番号: ' + duplicate } }];
const first = rows[0];
if (request.command && String(first.command_name || '').trim().toUpperCase() !== request.command) return [{ json: { status: 'failed', error_message: 'command 与 command_id 不匹配: ' + request.command + ' / ' + first.command_name } }];
const chunkSize = 5;
const safeCategory = (value) => String(value || 'UNCLASSIFIED').trim().replace(/[<>:"/\\|?*]/g, '_').replace(/\\s+/g, '_').slice(0, 100) || 'UNCLASSIFIED';
const sortCases = (a, b) => String(a.case_number).trim().localeCompare(String(b.case_number).trim(), undefined, { numeric: true, sensitivity: 'base' });
const groups = new Map();
for (const row of rows) {
  const category = String(row.test_category || 'UNCLASSIFIED').trim() || 'UNCLASSIFIED';
  if (!groups.has(category)) groups.set(category, []);
  groups.get(category).push(row);
}
const categories = [...groups.keys()].sort((a, b) => a.localeCompare(b, undefined, { sensitivity: 'base' }));
const categoryPlans = categories.map((category, categoryIndex) => {
  const categoryRows = groups.get(category).slice().sort(sortCases);
  return { category, categoryIndex: categoryIndex + 1, rows: categoryRows, totalBatches: Math.ceil(categoryRows.length / chunkSize) };
});
const total = categoryPlans.reduce((sum, plan) => sum + plan.totalBatches, 0);
const batches = [];
let globalBatchNo = 0;
for (const plan of categoryPlans) {
  for (let offset = 0; offset < plan.rows.length; offset += chunkSize) {
    const cases = plan.rows.slice(offset, offset + chunkSize).map((row) => ({
      case_number: String(row.case_number || '').trim().toUpperCase(), test_category: plan.category,
      test_purpose: row.test_purpose, parameter_config: row.parameter_config,
      input_data: row.input_data, expected_result: row.expected_result,
      priority: row.priority, notes: row.notes,
    }));
    const categoryBatchNo = Math.floor(offset / chunkSize) + 1;
    globalBatchNo += 1;
    const generationKey = request.generation_mode === 'move_master'
      ? 'MOVE_V' + first.case_table_version + '_C' + String(plan.categoryIndex).padStart(2, '0') + '_' + safeCategory(plan.category) + '_B' + String(categoryBatchNo).padStart(2, '0')
      : 'CASE_TABLE_V' + first.case_table_version + '_C' + String(plan.categoryIndex).padStart(2, '0') + '_' + safeCategory(plan.category) + '_B' + String(categoryBatchNo).padStart(2, '0');
    batches.push({ json: {
      project_id: Number(first.project_id), language: String(first.language || '').trim().toUpperCase(),
      command_id: Number(first.command_id), command_name: first.command_name,
      generation_key: generationKey, sub_category: 'verification_case_table', file_group_no: categoryBatchNo,
      machine_type: $('校验代码生成绑定').first().json.machine_type,
      generation_mode: request.generation_mode, case_table_version: Number(first.case_table_version),
      test_category: plan.category, category_batch_no: categoryBatchNo, category_total_batches: plan.totalBatches,
      case_table_batch_no: globalBatchNo, case_table_total_batches: total, cases,
    }});
  }
}
const selected = request.case_table_batch_no == null ? batches : batches.filter((item) => item.json.case_table_batch_no === request.case_table_batch_no);
if (!selected.length) return [{ json: { status: 'failed', error_message: 'case_table_batch_no is outside latest table range: ' + request.case_table_batch_no + '/' + total } }];
return selected.map((item) => {
  const batch = item.json;
  if (request.generation_mode !== 'move_master' && (batch.cases.length > 5 || batch.cases.some((entry) => String(entry.test_category || '').trim() !== batch.test_category))) throw new Error('batch violates five-case or single-test-category contract: ' + batch.generation_key);
  return item;
});`;
sampler.notes = "遍历最新版本二维表：先校验 8 列非空、TC 编号合法且唯一；按 test_category 分组后各组按用例番号排序，每批固定最多五条，不跨类别合并。";

const allocation = node("分配或恢复程序基号");
allocation.parameters.query = `WITH locked AS (
  SELECT pg_advisory_xact_lock(918273, ({{$json.project_id}}::int * 1000000 + {{$json.command_id}}::int)::int) AS lock_token
), existing AS (
  SELECT base_no
  FROM analyzes.cl_tech_point_code_generations, locked
  WHERE project_id = {{$json.project_id}}::int
    AND language = CASE WHEN '{{$json.generation_mode}}' = 'move_master' THEN 'COBOL' ELSE 'CL' END
    AND command_id = {{$json.command_id}}::bigint
    AND tech_point_no = '{{$json.generation_key}}'
    AND base_no BETWEEN 0 AND 1295
  ORDER BY updated_at DESC NULLS LAST
  LIMIT 1
), next_no AS (
  SELECT MIN(candidate) AS base_no
  FROM generate_series(0, 1295) AS candidate, locked
  WHERE NOT EXISTS (
    SELECT 1
    FROM analyzes.cl_tech_point_code_generations used
    WHERE used.project_id = {{$json.project_id}}::int
      AND used.language = CASE WHEN '{{$json.generation_mode}}' = 'move_master' THEN 'COBOL' ELSE 'CL' END
      AND used.command_id = {{$json.command_id}}::bigint
      AND used.base_no = candidate
  )
), upserted AS (
  INSERT INTO analyzes.cl_tech_point_code_generations
    (generation_run_id, project_id, language, command_id, command_name, tech_point_no, sub_category, status, program_prefix, base_no, allocated_at, started_at, updated_at)
  VALUES
    (gen_random_uuid(), {{$json.project_id}}::int, CASE WHEN '{{$json.generation_mode}}' = 'move_master' THEN 'COBOL' ELSE 'CL' END, {{$json.command_id}}::bigint, '{{$json.command_name}}', '{{$json.generation_key}}', '{{$json.sub_category}}', 'generating', 'BO', COALESCE((SELECT base_no FROM existing), (SELECT base_no FROM next_no)), current_timestamp, current_timestamp, current_timestamp)
  ON CONFLICT (project_id, language, command_id, tech_point_no) DO UPDATE SET
    status = CASE WHEN analyzes.cl_tech_point_code_generations.status = 'completed' AND {{$json.force_reprocess}}::boolean = false THEN 'completed' ELSE 'generating' END,
    generation_run_id = CASE WHEN analyzes.cl_tech_point_code_generations.status = 'completed' AND {{$json.force_reprocess}}::boolean = false THEN analyzes.cl_tech_point_code_generations.generation_run_id ELSE EXCLUDED.generation_run_id END,
    base_no = CASE WHEN analyzes.cl_tech_point_code_generations.status = 'completed' AND {{$json.force_reprocess}}::boolean = false THEN analyzes.cl_tech_point_code_generations.base_no ELSE COALESCE(analyzes.cl_tech_point_code_generations.base_no, EXCLUDED.base_no) END,
    program_prefix = 'BO',
    updated_at = current_timestamp
  RETURNING *
)
SELECT upserted.*, upserted.tech_point_no AS generation_key FROM upserted;`;

const mergeAllocation = node("合并抽样SOURCE与程序基号");
mergeAllocation.name = "合并二维表批次与程序基号";
mergeAllocation.parameters.jsCode = `const allocations = $input.all().map((item) => item.json || {});
const batches = $('按二维表遍历代码生成批次').all().map((item) => item.json || {});
if (!allocations.length) throw new Error('program allocation result missing');
function key(item) { return [Number(item.project_id), Number(item.command_id), String(item.generation_key || '').trim()].join('\\u0000'); }
const byKey = new Map(batches.map((item) => [key(item), item]));
return allocations.map((allocation) => {
  const batch = byKey.get(key(allocation));
  if (!batch || !Array.isArray(batch.cases) || !batch.cases.length) throw new Error('verification-case-table batch is missing for allocated key ' + key(allocation));
  const { tech_point_no: legacyStorageKey, project_id: ignoredProjectId, command_id: ignoredCommandId, command_name: ignoredCommandName, language: ignoredLanguage, generation_mode: ignoredMode, ...allocationData } = allocation;
  return { json: { ...batch, ...allocationData, project_id: batch.project_id, command_id: batch.command_id, command_name: batch.command_name, language: batch.language, generation_mode: batch.generation_mode, cases: batch.cases } };
});`;
mergeAllocation.notes = "仅按二维表批次键合并程序基号；不包含 references、SOURCE 或技术点逻辑。";

const prep = code("xelora-code-prepare-binding", "准备查询检证用例代码绑定", [420, -40], `const commandId = Number($json.command_id);\nconst requestedProjectId = ${DEFAULT_CODE_PROJECT_ID};\nconst binding_query = \`WITH command_context AS (\n  SELECT cm.command AS command_name, cm.mac_id, ml.mac_name AS machine_type, COALESCE(mcl.language, cm.language) AS language\n  FROM analyzes.command_master cm\n  JOIN analyzes.mac_list ml ON ml.id = cm.mac_id\n  LEFT JOIN analyzes.mac_language mcl ON mcl.id = cm.language_id\n  WHERE cm.id = \${commandId}\n), binding AS (\n  SELECT ac.base_url, ac.agent_id, ac.api_key, mla.knowledge_base_ids, mla.knowledge_base_names\n  FROM analyzes.mac_language_agent mla\n  JOIN analyzes.agent_config ac ON ac.id = mla.agent_config_id\n  JOIN command_context cc ON cc.mac_id = mla.mac_id AND lower(trim(cc.language)) = lower(trim(mla.language))\n  WHERE mla.is_active = true AND ac.is_active = true AND mla.workflow_role = 'verification_case_code_generate'\n  ORDER BY mla.updated_at DESC NULLS LAST LIMIT 1\n)\nSELECT cc.command_name, cc.machine_type, cc.language, \${requestedProjectId}::int AS project_id, b.base_url, b.agent_id, b.api_key, b.knowledge_base_ids, b.knowledge_base_names\nFROM command_context cc LEFT JOIN binding b ON TRUE;\`;\nreturn [{ json: { ...$json, binding_query } }];`, "按 command_id 解析命令、机型、语言和智能体绑定；项目由工作流内部默认代码项目确定，前台无需传 project_id。");
const resolve = postgres("xelora-code-resolve-binding", "查询检证用例代码绑定", [650, -40], "={{ $json.binding_query }}", "使用 TargetDB 查询通用代码生成绑定");
const normalize = code("xelora-code-normalize-binding", "校验代码生成绑定", [880, -40], `const base = $("解析参数").first().json;\nconst row = $input.first()?.json || {};\nfunction ids(v) { if (Array.isArray(v)) return v.map(String).filter(Boolean); if (v && Array.isArray(v.array)) return v.array.map(String).filter(Boolean); if (typeof v === 'string') { try { return ids(JSON.parse(v)); } catch { return v ? [v] : []; } } return []; }\nconst kbIds = ids(row.knowledge_base_ids);\nconst context = { ...base, command_name: String(row.command_name || ''), machine_type: String(row.machine_type || ''), language: String(row.language || '').trim().toUpperCase(), project_id: Number(row.project_id || 0) };\nif (!context.command_name || !context.machine_type || !context.language) return [{ json: { ...context, valid: false, error_type: 'command_not_found', error_message: 'command_id was not found in command_master' } }];\nif (!Number.isInteger(context.project_id) || context.project_id <= 0) return [{ json: { ...context, valid: false, error_type: 'internal_project_configuration', error_message: '工作流默认代码项目配置无效' } }];\nif (!row.base_url || !row.agent_id || !row.api_key || !kbIds.length) return [{ json: { ...context, valid: false, error_type: 'workflow_binding_not_found', error_message: 'No verification_case_code_generate binding for ' + context.machine_type + '/' + context.language } }];\nreturn [{ json: { ...context, valid: true, xelora_api_base_url: String(row.base_url).replace(/\\\/$/, ''), xelora_agent_id: String(row.agent_id), xelora_api_key: String(row.api_key), xelora_kb_ids: kbIds, xelora_kb_name: String(row.knowledge_base_names?.[0] || kbIds[0]) } }];`, "命令、机型、语言和智能体绑定缺失时明确失败；项目由内部默认值提供，不读取前台 project_id。");
const valid = { id: "xelora-code-binding-if", name: "判断代码生成绑定是否完整", type: "n8n-nodes-base.if", typeVersion: 1, position: [1090, -40], parameters: { conditions: { boolean: [{ value1: "={{ $json.valid === true }}", value2: true }] } }, notes: "绑定完整才继续生成代码" };
workflow.nodes.push(prep, resolve, normalize, valid);

const normalizedQuery = node("标准化技术点批次字段");
normalizedQuery.name = "标准化二维表批次字段";
normalizedQuery.parameters.jsCode = `const binding = $('校验代码生成绑定').first()?.json || {};
const upstream = $input.all().map((item) => item.json || {});
const failed = upstream.find((item) => item.status === 'failed');
if (failed) return [{ json: failed }];
const batches = upstream.map((item) => ({ ...binding, ...item }))
  .filter((item) => Array.isArray(item.cases) && item.cases.length && Number(item.project_id) > 0 && Number(item.command_id) > 0);
if (!batches.length) return [{ json: { status: 'failed', error_message: '当前命令没有可处理的最新检证用例二维表批次' } }];
return batches.map((item) => ({ json: { ...item, generation_mode: item.generation_mode || 'case_table' } }));`;
normalizedQuery.notes = "标准化二维表批次字段；唯一输入为 cases。";

const request = node("准备Xelora技术点请求");
request.name = "准备二维表检证用例代码请求";
request.parameters.jsCode = `const batches = $input.all().map((item) => item.json || {});
if (!batches.length) throw new Error('verification-case-table batch allocation is missing');
const timestamp = $('解析参数').first().json.command_run_timestamp || new Date().toISOString().replace(/[-:TZ.]/g, '').slice(0, 14);
function safe(value) { return String(value || '').trim().replace(/[^a-zA-Z0-9_.-]+/g, '_') || '_'; }
function safeCategory(value) { return String(value || 'UNCLASSIFIED').trim().replace(/[<>:"/\\|?*]/g, '_').replace(/\s+/g, '_').slice(0, 100) || 'UNCLASSIFIED'; }
function table(cases) {
  const header = '| 用例番号 | テスト分類 | テスト目的 | パラメータ設定 | 入力データ | 期待結果 | 優先度 | 備考 |';
  const divider = '|---|---|---|---|---|---|---|---|';
  const rows = cases.map((item) => [item.case_number,item.test_category,item.test_purpose,item.parameter_config,item.input_data,item.expected_result,item.priority,item.notes]
    .map((value) => String(value ?? '').replace(/\\|/g, '\\\\|').replace(/\\r?\\n/g, '<br>')).join(' | '));
  return [header, divider, ...rows.map((row) => '| ' + row + ' |')].join('\\n');
}
return batches.map((batch) => {
  const cases = Array.isArray(batch.cases) ? batch.cases : [];
  if (!cases.length) throw new Error('verification-case-table batch has no cases');
  const baseNo = Number(batch.base_no || 0);
  if (!Number.isInteger(baseNo) || baseNo < 0 || baseNo >= 36 * 36) throw new Error('program base number must be in base36 range 0..1295');
  const isMove = batch.generation_mode === 'move_master';
  const programBaseCode = 'BO' + String(batch.command_id);
  const base36 = '0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ';
  const fileGroupSeq = base36[Math.floor(baseNo / 36)] + base36[baseNo % 36];
  const categoryDirectory = safeCategory(batch.test_category);
  const outputDirectory = isMove
    ? '/home/n8n/.n8n/verification_cases_code/ASP/COBOL/' + Number(batch.command_id) + '_' + safe(batch.command_name) + '/' + categoryDirectory + '/' + timestamp
    : '/home/n8n/.n8n/verification_cases_code/' + safe(batch.machine_type) + '/' + safe(batch.language) + '/' + Number(batch.command_id) + '_' + safe(batch.command_name) + '/' + categoryDirectory + '/' + timestamp;
  const query = isMove
    ? ['GENERATION_MODE: MOVE_MASTER','PROJECT_ID: ' + batch.project_id,'COMMAND_ID: ' + batch.command_id,'COMMAND_NAME: ' + batch.command_name,'TEST_CATEGORY: ' + batch.test_category,'CASE_TABLE_VERSION: ' + batch.case_table_version,'BATCH_INDEX: ' + batch.case_table_batch_no + '/' + batch.case_table_total_batches,'PROGRAM_BASE_CODE: ' + programBaseCode,'FILE_GROUP_SEQ: ' + fileGroupSeq,'PGM_BASE_CODE: ' + programBaseCode,'PGM_FILE_SEQ: ' + fileGroupSeq,'TEST_CASES_TABLE BEGIN',table(cases),'TEST_CASES_TABLE END','OUTPUT_FILES: ' + programBaseCode + 'C' + fileGroupSeq + '.CL, ' + programBaseCode + 'B' + fileGroupSeq + '.COBOL, ' + programBaseCode + 'F' + fileGroupSeq + '.SF','Return exactly these three headings: ## ' + programBaseCode + 'C' + fileGroupSeq + '.CL, ## ' + programBaseCode + 'B' + fileGroupSeq + '.COBOL, ## ' + programBaseCode + 'F' + fileGroupSeq + '.SF. Generate every case in this batch and no other case.'].join('\\n\\n')
    : ['GENERATION_MODE: case_table','PROJECT_ID: ' + batch.project_id,'COMMAND_ID: ' + batch.command_id,'COMMAND_NAME: ' + batch.command_name,'MACHINE_TYPE: ' + batch.machine_type,'LANGUAGE_TYPE: ' + batch.language,'TEST_CATEGORY: ' + batch.test_category,'CASE_TABLE_VERSION: ' + batch.case_table_version,'BATCH_INDEX: ' + batch.case_table_batch_no + '/' + batch.case_table_total_batches,'CATEGORY_BATCH_INDEX: ' + batch.category_batch_no + '/' + batch.category_total_batches,'PROGRAM_BASE_CODE: ' + programBaseCode,'FILE_GROUP_SEQ: ' + fileGroupSeq,'PGM_BASE_CODE: ' + programBaseCode,'PGM_FILE_SEQ: ' + fileGroupSeq,'TEST_CASES_TABLE BEGIN',table(cases),'TEST_CASES_TABLE END','仅按本批同一 TEST_CATEGORY 的 TEST_CASES_TABLE 逐行生成完整 CL、COBOL、SF 三文件；本批最多五个二维表 case，不得混入其他测试类别。'].join('\\n\\n');
  return { json: { ...batch, generation_mode: batch.generation_mode || 'case_table', cases, program_base_code: programBaseCode, file_group_seq: fileGroupSeq, output_directory: outputDirectory, base_dir_container: outputDirectory, xelora_api_base_url: $('校验代码生成绑定').first().json.xelora_api_base_url, xelora_api_key: $('校验代码生成绑定').first().json.xelora_api_key, xelora_agent_id: $('校验代码生成绑定').first().json.xelora_agent_id, xelora_kb_id: $('校验代码生成绑定').first().json.xelora_kb_ids[0], xelora_knowledge_ids: $('校验代码生成绑定').first().json.xelora_kb_ids, xelora_kb_name: $('校验代码生成绑定').first().json.xelora_kb_name, xelora_session_url: $('校验代码生成绑定').first().json.xelora_api_base_url + '/sessions', xelora_agent_url: $('校验代码生成绑定').first().json.xelora_api_base_url + '/agent-chat', xelora_query: query, request_time: new Date().toISOString() } };
});`;
request.notes = "将最新版本 8 列二维表按 test_category 分组、每批最多五条构造成 TEST_CASES_TABLE；可用 test_category 只回归一个类别；禁止将技术点或 SOURCE 传给智能体。";

const sfInfo = node("准备SF文件信息");
sfInfo.parameters.jsCode = sfInfo.parameters.jsCode.replace("const canonical = makePayload(source.sf_file_name, rawContent);", "const canonical = makePayload(source.sf_file_name, rawContent);\nif (source.generation_mode === 'case_table' && !/^\\s*FILE\\s+[A-Za-z0-9][A-Za-z0-9_.-]*\\s*\\r?\\nORG\\s+SF\\s*\\r?\\nRL\\s+\\d+\\s*\\r?\\nSIZE\\s+\\d+\\s*\\r?\\nRECN\\s+\\d+/i.test(rawContent)) throw new Error('case-table SF must use FILE/ORG SF/RL/SIZE/RECN storage format: ' + source.sf_file_name);");
sfInfo.notes = "二维表模式要求 SF 使用实际保存契约 FILE/ORG SF/RL/SIZE/RECN，拒绝说明性 DEFINE FILE 格式。";

const callAgent = node("调用CL技术点代码智能体");
callAgent.name = "调用二维表检证用例代码智能体";
callAgent.parameters.jsonBody = "={{ {query:$json.xelora_query,agent_enabled:true,agent_id:$json.xelora_agent_id,knowledge_base_ids:[$json.xelora_kb_id],knowledge_ids:$json.xelora_knowledge_ids,mentioned_items:[{id:$json.xelora_kb_id,name:$json.xelora_kb_name || 'knowledge-base',type:'kb',kb_type:'document'}],web_search_enabled:false,channel:'api'} }}";
callAgent.notes = "调用按机型/语言绑定的二维表检证用例代码智能体";
for (const name of ["创建Xelora会话", callAgent.name]) {
  const http = node(name);
  http.parameters.options = { ...(http.parameters.options || {}), timeout: name === callAgent.name ? 900000 : 120000 };
}
node("创建Xelora会话").parameters.jsonBody = "={{ {title: 'verification-case-table batch ' + $json.case_table_batch_no + '/' + $json.case_table_total_batches + ' ' + $json.command_name + ' ' + $json.request_time} }}";

const parser = node("解析三文件代码段");
parser.parameters.jsCode = `const responses = $input.all();
const sessions = $('提取Xelora会话ID').all();
const requests = $('准备二维表检证用例代码请求').all();
function pair(item) { const p = item.pairedItem; return Array.isArray(p) ? Number(p[0]?.item) : Number(p?.item); }
function answerOf(value) {
  let answer = ''; let complete = false; let error = null;
  function add(event) { if (!event || typeof event !== 'object') return; const type = String(event.response_type || event.event || event.type || '').toLowerCase(); if (typeof event.answer === 'string') answer += event.answer; if ((!type || ['answer','message','agent_message','final_answer','final_answer_chunk'].includes(type)) && typeof event.content === 'string') answer += event.content; if (type === 'complete' || type === 'done' || event.done === true) complete = true; if (type === 'error') error = String(event.error || event.content || 'Xelora agent error'); }
  const raw = typeof value === 'string' ? value : value?.data;
  if (typeof raw === 'string') for (const line of raw.split(/\\r?\\n/)) { const data = line.trimStart(); if (!data.startsWith('data:')) continue; const payload = data.slice(5).trim(); if (!payload || payload === '[DONE]') { complete = true; continue; } try { add(JSON.parse(payload)); } catch {} }
  else { add(value); add(value?.data); add(value?.result); }
  return { answer, complete, error };
}
function section(text, filename) {
  const source = String(text || '');
  const heading = '## ' + filename;
  const alternateHeading = '## TOINSRC/' + filename;
  let headingStart = source.indexOf(heading);
  if (headingStart < 0) headingStart = source.indexOf(alternateHeading);
  if (headingStart < 0) return '';
  const lineBreak = String.fromCharCode(10);
  const bodyStart = source.indexOf(lineBreak, headingStart);
  if (bodyStart < 0) return '';
  const nextHeading = source.indexOf(lineBreak + '## ', bodyStart + 1);
  const body = source.slice(bodyStart + 1, nextHeading < 0 ? source.length : nextHeading);
  const lines = body.split(lineBreak).map((line) => line.endsWith(String.fromCharCode(13)) ? line.slice(0, -1) : line);
  let startLine = 0;
  let endLine = lines.length;
  if (lines[startLine] !== undefined && lines[startLine].trim() === '') startLine += 1;
  const fence = String.fromCharCode(96).repeat(3);
  if (lines[startLine] !== undefined && lines[startLine].trim().startsWith(fence)) startLine += 1;
  if (lines[endLine - 1] !== undefined && lines[endLine - 1].trim() === '') endLine -= 1;
  if (lines[endLine - 1] !== undefined && lines[endLine - 1].trim() === fence) endLine -= 1;
  return lines.slice(startLine, endLine).join(lineBreak);
}
function tcCoverage(code, cases, isMove = false) { for (const item of cases) { const tc = String(item.case_number || '').trim().toUpperCase(); const pattern = isMove ? /^MOVE_TC[0-9]+$/i : /^TC[0-9]+$/i; if (!pattern.test(tc)) return 'invalid input case number: ' + tc; const upperCode = String(code).toUpperCase(); const n = upperCode.split(tc).length - 1; if (n < 1) return 'CL must contain ' + tc + '; found=' + n; } return null; }
function moveFormatError(meta, cl, cobol, sf) {
  const base = String(meta.program_base_code || '').toUpperCase();
  const seq = String(meta.file_group_seq || '').toUpperCase();
  const cb = base + 'B' + seq;
  const cf = base + 'F' + seq;
  const requiredCl = ['PGM', 'DLTFILE FILE-' + cf + '.SRTEST', 'CRTFILE FILE-' + cf + '.SRTEST', 'OVRF   FILE-' + cf + ',TOFILE-' + cf + '.SRTEST', 'CALL   PGM-' + cb + '.SRTEST', '?PGMEC := @PGMEC', 'RETURN PGMEC-?PGMEC'];
  for (const marker of requiredCl) if (!String(cl).includes(marker)) return 'MOVE SR6 CL missing required template marker: ' + marker;
  const normalizedCobol = String(cobol).replace(/\\s+/g, ' ').toUpperCase();
  if (!normalizedCobol.includes('PROGRAM-ID. ' + cb + '.')) return 'MOVE fast SR6 COBOL missing required template marker: PROGRAM-ID. ' + cb + '.';
  if (!normalizedCobol.includes('SELECT LOGFILE ASSIGN TO DA-' + cf)) return 'MOVE fast SR6 COBOL missing required template marker: SELECT LOGFILE ASSIGN TO DA-' + cf;
  const requiredCobol = [/IDENTIFICATION\\s+DIVISION\\./i, /AUTHOR\\.\\s+SR\\./i, /ENVIRONMENT\\s+DIVISION/i, /CONFIGURATION\\s+SECTION/i, /SOURCE-COMPUTER\\.\\s+FACOM\\./i, /OBJECT-COMPUTER\\.\\s+FACOM\\./i, /INPUT-OUTPUT\\s+SECTION/i, /FILE-CONTROL\\./i, /01\\s+LOGFILE-REC\\./i, /03\\s+PGMID/i, /03\\s+CASENO/i, /03\\s+RESULT1/i, /03\\s+RESULT2/i, /03\\s+RESULT3/i, /03\\s+RESULT4/i, /03\\s+RESULT5/i, /03\\s+RESULT6/i, /WORKING-STORAGE/i, /01\\s+STA01\\s+PIC\\s+X\\(2\\)/i, /PROCEDURE\\s+DIVISION\\./i, /INIT-RTN\\s+SECTION\\./i, /INITIALIZE\\s+LOGFILE-REC\\./i, /OPEN\\s+OUTPUT\\s+LOGFILE\\./i, /MAIN-RTN\\s+SECTION\\./i, /MOVE-METHOD-0-RTN\\s+SECTION\\./i, /MOVE-METHOD-0-EXIT\\.\\s+EXIT\\./i, /END-RTN\\s+SECTION\\./i, /WRITE\\s+LOGFILE-REC\\./i];
  for (const marker of requiredCobol) if (!marker.test(String(cobol))) return 'MOVE fast SR6 COBOL missing required template marker: ' + marker;
  if (/LINKAGE\\s+SECTION|RESULT\\s+PIC|RMSG1|MOVE_T[BCF]\\d|DA-MOVETC\\d+F01/i.test(String(cobol))) return 'MOVE fast SR6 COBOL contains legacy naming/template';
  const requiredSf = ['/FILE ' + cf, 'S0220002', '/RECORD RCD', 'F1000', 'F1110', 'F1120', 'F1130', 'F1140', 'F1150', 'F1160', 'F1170', 'F1180', '/END'];
  for (const marker of requiredSf) if (!String(sf).includes(marker)) return 'MOVE fast SR6 SF missing required template marker: ' + marker;
  if (/S0042001|\\*SRTEST|DDS/i.test(String(sf))) return 'MOVE fast SR6 canonical SF must be the SR6 PF layout';
  const caseText = JSON.stringify(meta.cases || []);
  if (/VAR_TYPE_GP\\s*\\(\\s*String\\s*\\)/i.test(caseText)) {
    if (!/MOVE\\s+HIGH-VALUES\\s+TO\\s+P01\\./i.test(String(cobol))) return 'VAR_TYPE_GP(String) must assign HIGH-VALUES to 01-level P01';
    if (!/MOVE\\s+P01\\s+TO\\s+RESULT6\\./i.test(String(cobol))) return 'VAR_TYPE_GP(String) must store the value in RESULT6';
    if (/MOVE\\s+HIGH-VALUES\\s+TO\\s+P0\\d{2,}\\./i.test(String(cobol))) return 'VAR_TYPE_GP(String) must not assign HIGH-VALUES to 02-level fields';
    if (/MOVE\\s+P01\\s+TO\\s+RESULT[1-5]\\./i.test(String(cobol))) return 'VAR_TYPE_GP(String) must use RESULT6';
  }
  return null;
}

const output = [];
for (let index = 0; index < responses.length; index += 1) {
  const response = answerOf(responses[index].json);
  const paired = pair(responses[index]);
  const meta = sessions[Number.isInteger(paired) && sessions[paired] ? paired : index]?.json || requests[Number.isInteger(paired) && requests[paired] ? paired : index]?.json;
  if (!meta) continue;
  const isMove = meta.generation_mode === 'move_master';
  const base = String(meta.program_base_code || '').toUpperCase(); const seq = String(meta.file_group_seq || '').toUpperCase();
   const clName = base + 'C' + seq + '.CL';
   const cobolName = base + 'B' + seq + '.COBOL';
   const sfName = base + 'F' + seq + '.SF';
  const cl = section(response.answer, clName); const cobol = section(response.answer, cobolName) || section(response.answer, cobolName.replace(/\.COBOL$/i, '.CBL')); const sf = section(response.answer, sfName);
  // MOVE_MASTER is intentionally landing-first: keep file-section/SSE checks, defer static SR6 content gates to post-write inspection.\n  const moveError = null;
  const error = response.error || (!response.complete ? 'Xelora SSE incomplete' : null) || (!cl || !cobol || !sf ? '智能体响应缺少文件段: ' + [!cl && 'CL',!cobol && 'COBOL',!sf && 'SF'].filter(Boolean).join(', ') : null) || moveError || (!isMove ? tcCoverage(cl, meta.cases || [], false) : null);
  output.push({ json: error ? { ...meta, status: 'failed', error_message: error, full_response: response.answer } : { ...meta, status: 'success', stream_complete: true, cl_code: cl, cobol_code: cobol, sf_code: sf, cl_file_name: clName, cobol_file_name: cobolName, sf_file_name: sfName, cl_file_path: meta.output_directory + '/' + clName, cobol_file_path: meta.output_directory + '/' + cobolName, sf_file_path: meta.output_directory + '/' + sfName, full_response: response.answer } });
}
for (const item of requests) if (!output.some((result) => result.json.generation_key === item.json.generation_key)) output.push({ json: { ...item.json, status: 'failed', error_message: 'missing Xelora response for table batch ' + item.json.case_table_batch_no } });
return output;`;
parser.notes = "解析二维表批次的 Xelora SSE 三文件输出；每个输入 case 编号必须在 CL 中出现，否则禁止保存。MOVE_MASTER 与普通二维表统一使用 BO+command_id+C/B/F+两位大写 base36 文件名。";

const registration = node("准备生成状态和SOURCE备份登记");
registration.name = "准备二维表生成登记";
registration.parameters.jsCode = `const directories = $input.all().map((item) => item.json || {}).filter((item) => item && item.status !== 'failed');
if (!directories.length) return [{ json: { status: 'failed', error_message: '没有可登记的二维表生成目录' } }];
function sql(value) { return String(value == null ? '' : value).replaceAll("'", "''"); }
const statements = [];
let first = null;
for (const directory of directories) {
  const batches = Array.isArray(directory.cases) && directory.cases.length ? directory.cases : [directory];
  for (const batch of batches) {
    const m = { ...directory, ...batch };
    if (!first) first = m;
    const trace = { ...(m.normalization_result || {}), generation_mode: m.generation_mode || 'case_table', test_category: m.test_category || null, category_batch_no: m.category_batch_no || null, category_total_batches: m.category_total_batches || null, case_table_version: m.case_table_version || null, case_table_batch_no: m.case_table_batch_no || null, case_table_total_batches: m.case_table_total_batches || null, case_numbers: (m.cases || []).map((entry) => String(entry.case_number || '').trim()).filter(Boolean) };
    const status = m.status === 'success' ? 'completed' : 'failed';
    const programPrefix = 'BO';
    if (m.base_no == null || m.base_no === '') throw new Error('no available base36 file group sequence for ' + String(m.generation_key || ''));
    const baseNumber = Number(m.base_no);
    if (!Number.isInteger(baseNumber) || baseNumber < 0 || baseNumber >= 1296) throw new Error('program base number must be in base36 range 0..1295 for ' + String(m.generation_key || ''));
    statements.push("INSERT INTO analyzes.cl_tech_point_code_generations (generation_run_id,project_id,language,command_id,command_name,tech_point_no,sub_category,status,program_prefix,base_no,agent_id,cl_file_path,cobol_file_path,sf_file_path,output_directory,normalization_result,error_message,started_at,completed_at,updated_at) VALUES (gen_random_uuid()," + Number(m.project_id) + ",'" + sql(m.language) + "'," + Number(m.command_id) + ",'" + sql(m.command_name) + "','" + sql(m.generation_key) + "','verification_case_table','" + status + "','" + sql(programPrefix) + "'," + baseNumber + ",'" + sql(m.xelora_agent_id) + "','" + sql(m.cl_file_path) + "','" + sql(m.cobol_file_path) + "','" + sql(m.sf_file_path) + "','" + sql(m.output_directory) + "','" + sql(JSON.stringify(trace)) + "'::jsonb," + (m.error_message ? "'" + sql(m.error_message) + "'" : 'NULL') + ",current_timestamp," + (status === 'completed' ? 'current_timestamp' : 'NULL') + ",current_timestamp) ON CONFLICT (project_id,language,command_id,tech_point_no) DO UPDATE SET status=EXCLUDED.status,program_prefix=EXCLUDED.program_prefix,base_no=EXCLUDED.base_no,agent_id=EXCLUDED.agent_id,cl_file_path=EXCLUDED.cl_file_path,cobol_file_path=EXCLUDED.cobol_file_path,sf_file_path=EXCLUDED.sf_file_path,output_directory=EXCLUDED.output_directory,normalization_result=EXCLUDED.normalization_result,error_message=EXCLUDED.error_message,completed_at=EXCLUDED.completed_at,updated_at=current_timestamp;");
  }
}
return [{ json: { ...(first || directories[0]), registration_sql: statements.join('\\n'), registration_batch_count: statements.length } }];`;
registration.notes = "只登记二维表版本、批次和 TC 编号的三文件生成结果；不写入技术点来源或 SOURCE 快照。";

for (const workflowNode of workflow.nodes) {
  if (typeof workflowNode.parameters?.jsCode === "string") {
    workflowNode.parameters.jsCode = workflowNode.parameters.jsCode
      .replaceAll("$('准备Xelora技术点请求')", "$('准备二维表检证用例代码请求')")
      .replaceAll("$('选择技术点或二维表代码生成批次')", "$('按二维表遍历代码生成批次')")
      .replaceAll("$('准备生成状态和SOURCE备份登记')", "$('准备二维表生成登记')");
    if (workflowNode.name === "准备COBOL文件信息") workflowNode.parameters.jsCode = workflowNode.parameters.jsCode
      .replace("const content = rawContent.replace(/(^|\\r?\\n)(?=.)/g, '$1        ');", "const content = source.generation_mode === 'move_master' ? rawContent : rawContent.replace(/(^|\\r?\\n)(?=.)/g, '$1        ');");
  }
  if (typeof workflowNode.notes === "string") workflowNode.notes = workflowNode.notes.replaceAll("技术点", "二维表批次").replaceAll("SOURCE", "来源数据");
}
const hasPending = node("判断是否有待处理技术点");
hasPending.name = "判断是否有二维表批次";
hasPending.notes = "存在最新二维表批次时才进入程序基号分配。";
const c = workflow.connections;
const response = node("返回结果");
response.parameters.responseBody = "={{JSON.stringify({success:$json.status === 'completed' || $json.status === 'success',status:$json.status === 'completed' ? 'success' : ($json.status || 'failed'),project_id:$json.project_id || null,command_id:$json.command_id || null,test_category:$json.test_category || null,category_batch_no:$json.category_batch_no || null,category_total_batches:$json.category_total_batches || null,case_table_version:$json.case_table_version || null,case_table_batch_no:$json.case_table_batch_no || null,output_directory:$json.output_directory || null,error_message:$json.error_message || null})}}";
c["解析参数"] = { main: [[{ node: prep.name, type: "main", index: 0 }]] };
c[prep.name] = { main: [[{ node: resolve.name, type: "main", index: 0 }]] };
c[resolve.name] = { main: [[{ node: normalize.name, type: "main", index: 0 }]] };
c[normalize.name] = { main: [[{ node: valid.name, type: "main", index: 0 }]] };
c[valid.name] = { main: [[{ node: query.name, type: "main", index: 0 }], [{ node: "返回结果", type: "main", index: 0 }]] };
function renameConnection(oldName, newName) {
  if (c[oldName]) { c[newName] = c[oldName]; delete c[oldName]; }
  for (const routes of Object.values(c)) for (const branches of routes.main || []) for (const edge of branches || []) if (edge.node === oldName) edge.node = newName;
}
renameConnection(oldQueryName, query.name);
renameConnection("调用CL技术点代码智能体", callAgent.name);
renameConnection("准备Xelora技术点请求", request.name);
renameConnection("按PGM优先抽取最多五条SOURCE", sampler.name);
renameConnection("合并抽样SOURCE与程序基号", mergeAllocation.name);
renameConnection("判断是否有待处理技术点", hasPending.name);
renameConnection("标准化技术点批次字段", normalizedQuery.name);
renameConnection("准备生成状态和SOURCE备份登记", registration.name);

fs.writeFileSync(outputPath, JSON.stringify(workflow, null, 2) + "\n", "utf8");
console.log(`Wrote ${outputPath}`);





