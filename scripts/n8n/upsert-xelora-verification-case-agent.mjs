#!/usr/bin/env node
import { execFileSync } from "node:child_process";
import fs from "node:fs";
import path from "node:path";

const repoRoot = process.cwd();
const outputDir = path.resolve(repoRoot, "..", "n8n", "\u4e8c\u7ef4\u8868");
const envOutputPath = path.join(outputDir, "Xelora-CL-Verification-Case-Agent.env");

const knowledgeBaseName = "ASP_ALL_KNOWLEDGE";
const agentName = "CL\u547d\u4ee4\u68c0\u8bc1\u7528\u4f8b\u751f\u6210\u667a\u80fd\u4f53";
const agentDescription =
  "\u57fa\u4e8e ASP_ALL_KNOWLEDGE \u548c\u53c2\u6570\u89e3\u6790\u4e8c\u7ef4\u8868\u751f\u6210 CL \u68c0\u8bc1\u7528\u4f8b\u4e8c\u7ef4\u8868\uff0c\u9002\u7528\u4e8e command_master.detail_info \u4ee5\u5916\u7684\u6d4b\u8bd5\u7528\u4f8b\u6d41\u7a0b\u3002";

const systemPrompt = `你是 Xelora 内的 CL 命令检证用例生成智能体。

定位：
- 你负责一次性生成 CL（制御言語 / Control Language）命令或 CL 内置函数的完整检证用例二维表。
- 这是唯一生成阶段，不需要评估智能体补齐。
- 输出结果用于 n8n 继续解析并落库到 cl_verification_cases。

事实依据：
- 优先使用上游传入的 PARAMETER_TABLE、RETURN_CODE_LIST、RETURN_CODE_MESSAGES_RAW、RETURN_CODE_SOURCE。
- 当 RETURN_CODE_SOURCE = database 时，RETURN_CODE_MESSAGES_RAW 是异常系 CASE 的唯一权威依据。
- 当 RETURN_CODE_SOURCE = none 时，使用 ASP_ALL_KNOWLEDGE 补充结束码语义与异常场景。
- 不使用数据库 SOURCE/source_content。
- 不访问外部 Web。

生成目标：
- 一次性覆盖正常系、异常系、边界值、组合、依赖、互斥、默认值、权限、顺序、缺省、返回码驱动场景。
- 你的输出必须足够完整，避免把缺口留给评估智能体。
- 如果参数表或结束码原文有缺失，也要生成尽可能完整的用例，并把不确定部分写成保守、可执行的描述。

输出格式：
- 只输出一个 Markdown 表格，不要输出前言、解释、评估意见或代码块之外的说明。
- 表头固定 7 列：
  用例番号 / テスト分類 / テスト目的 / パラメータ設定 / 期待結果 / 優先度 / 備考
- 不要输出「入力データ」列。
- 所有单元格文本必须使用日文。
- 用例番号必须以 `TC` 开头，例如 `TC001`、`TC002`。
- 期待結果中的结束码必须与 `RETURN_CODE_LIST`、`RETURN_CODE_MESSAGES_RAW` 一致，不得臆造。
- `パラメータ設定` 中可以写前置条件、参数组合、文件准备、命令示例，但要保持在同一列。

构造规则：
- 正常系：覆盖必要参数、默认值、省略值、典型值。
- 异常系：优先使用 RETURN_CODE_MESSAGES_RAW 中可触发的结束码。
- 边界值：覆盖最小值、最大值、长度边界、枚举边界、组合边界。
- 组合/依赖：覆盖互斥、联动、前置条件、参数顺序与重复指定。
- 若命令为内置函数，也要覆盖函数特有的返回值验证与场景验证。

质量要求：
- 用例必须可执行、可复现。
- 优先写具体参数值，不要写空泛描述。
- 若信息不足，宁可保守也不要胡乱扩展。
- 不要输出评估报告，不要输出改进提示词，不要输出第二阶段补充建议。`;

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

function readExistingEnvValue(filePath, key) {
  if (!fs.existsSync(filePath)) return "";
  const line = fs
    .readFileSync(filePath, "utf8")
    .split(/\r?\n/)
    .find((entry) => entry.startsWith(`${key}=`));
  return line ? line.slice(key.length + 1).trim() : "";
}

const kbResult = runPsql(`
SELECT id || '|' || tenant_id || '|' || COALESCE(summary_model_id, '') || '|' || COALESCE(creator_id, '')
FROM knowledge_bases
WHERE deleted_at IS NULL AND name = ${sqlLiteral(knowledgeBaseName)}
ORDER BY updated_at DESC
LIMIT 1;
`);

if (!kbResult) {
  throw new Error(`${knowledgeBaseName} knowledge base was not found in Xelora database.`);
}

const [knowledgeBaseId, tenantId, modelIdFromKb, creatorId] = kbResult.split("|");
if (!modelIdFromKb) {
  throw new Error(`${knowledgeBaseName} knowledge base has no summary_model_id.`);
}

const agentConfig = {
  agent_mode: "smart-reasoning",
  agent_type: "rag-qa",
  system_prompt: systemPrompt,
  context_template: "",
  model_id: modelIdFromKb,
  rerank_model_id: "",
  temperature: 0.1,
  max_completion_tokens: 8192,
  thinking: true,
  max_iterations: 10,
  llm_call_timeout: 180,
  allowed_tools: [
    "knowledge_search",
    "grep_chunks",
    "list_knowledge_chunks",
    "get_document_info",
    "thinking",
  ],
  mcp_selection_mode: "none",
  mcp_services: [],
  skills_selection_mode: "none",
  selected_skills: [],
  kb_selection_mode: "selected",
  knowledge_bases: [knowledgeBaseId],
  retrieve_kb_only_when_mentioned: false,
  retain_retrieval_history: true,
  image_upload_enabled: false,
  audio_upload_enabled: false,
  supported_file_types: [],
  data_analysis_enabled: false,
  faq_priority_enabled: true,
  faq_direct_answer_threshold: 0.9,
  faq_score_boost: 1.2,
  web_search_enabled: false,
  web_fetch_enabled: false,
  web_search_max_results: 5,
  multi_turn_enabled: true,
  history_turns: 5,
  embedding_top_k: 12,
  keyword_threshold: 0.25,
  vector_threshold: 0.45,
  rerank_top_k: 8,
  rerank_threshold: 0,
  enable_query_expansion: true,
  enable_rewrite: true,
  rewrite_prompt_system: "",
  rewrite_prompt_user: "",
  fallback_strategy: "model",
  fallback_response: "",
  fallback_prompt: "",
  suggested_prompts: [
    "请生成 CHGCMVAR 命令的检证用例",
    "请基于参数表与结束码原文输出完整检证用例二维表",
    "请补全异常系和边界值检证用例",
  ],
};

const createdBySql = creatorId ? sqlLiteral(creatorId) : "NULL";
const agentConfigSql = sqlLiteral(JSON.stringify(agentConfig));

const upsertResult = runPsql(`
WITH existing AS (
  SELECT id
  FROM custom_agents
  WHERE deleted_at IS NULL
    AND tenant_id = ${Number(tenantId)}
    AND name = ${sqlLiteral(agentName)}
  LIMIT 1
),
updated AS (
  UPDATE custom_agents
  SET
    description = ${sqlLiteral(agentDescription)},
    avatar = 'O',
    config = ${agentConfigSql}::jsonb,
    created_by = COALESCE(created_by, ${createdBySql}),
    runnable_by_viewer = true,
    updated_at = CURRENT_TIMESTAMP
  WHERE id IN (SELECT id FROM existing)
    AND tenant_id = ${Number(tenantId)}
  RETURNING id
),
inserted AS (
  INSERT INTO custom_agents (
    name,
    description,
    avatar,
    is_builtin,
    tenant_id,
    created_by,
    config,
    runnable_by_viewer
  )
  SELECT
    ${sqlLiteral(agentName)},
    ${sqlLiteral(agentDescription)},
    'O',
    false,
    ${Number(tenantId)},
    ${createdBySql},
    ${agentConfigSql}::jsonb,
    true
  WHERE NOT EXISTS (SELECT 1 FROM existing)
  RETURNING id
)
SELECT id FROM updated
UNION ALL
SELECT id FROM inserted
LIMIT 1;
`);

if (!upsertResult) {
  throw new Error("Failed to upsert CL verification-case agent.");
}

fs.mkdirSync(path.dirname(envOutputPath), { recursive: true });
const existingApiKey = readExistingEnvValue(envOutputPath, "N8N_XELORA_API_KEY");
const apiKeyForEnv =
  existingApiKey && existingApiKey !== "configure-in-n8n-runtime"
    ? existingApiKey
    : "configure-in-n8n-runtime";
const existingApiBaseUrl = readExistingEnvValue(envOutputPath, "XELORA_API_BASE_URL");
const apiBaseUrlForEnv = existingApiBaseUrl || "http://192.168.2.61:18080/api/v1";
fs.writeFileSync(
  envOutputPath,
  [
    `XELORA_API_BASE_URL=${apiBaseUrlForEnv}`,
    `N8N_XELORA_API_KEY=${apiKeyForEnv}`,
    `XELORA_VERIFICATION_CASE_AGENT_ID=${upsertResult}`,
    `XELORA_TEST_MATRIX_KB_ID=${knowledgeBaseId}`,
    "",
  ].join("\n"),
  "utf8",
);

console.log(`Agent: ${agentName}`);
console.log(`XELORA_VERIFICATION_CASE_AGENT_ID=${upsertResult}`);
console.log(`XELORA_TEST_MATRIX_KB_ID=${knowledgeBaseId}`);
console.log(`tenant_id=${tenantId}`);
console.log(`model_id=${modelIdFromKb}`);
console.log("skills_selection_mode=none");
console.log("output_format=markdown");
console.log(`Wrote ${envOutputPath}`);
