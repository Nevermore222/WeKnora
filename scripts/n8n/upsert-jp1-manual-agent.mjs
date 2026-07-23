#!/usr/bin/env node
import { execFileSync } from "node:child_process";
import fs from "node:fs";
import path from "node:path";

const repoRoot = process.cwd();
const outputDir = path.resolve(repoRoot, "..", "n8n", "\u53c2\u6570\u89e3\u6790");
const envOutputPath = path.join(outputDir, "Xelora-JP1-Manual-Agent.env");

const knowledgeBaseName = "JP1";
const agentName = "JP1文档咨询智能体";
const agentDescription =
  "面向 JP1 Manual 文档知识库的智能推理咨询智能体。使用 JP1 知识库进行 RAG 检索和多步推理，不绑定任何 skill、MCP 或 Web 搜索。";

const systemPrompt = `你是 JP1 Manual 文档咨询智能体。

职责范围：
- 只基于已绑定的 JP1 知识库回答用户问题。
- 面向大量 Manual 文档内容做检索、归纳、对比、步骤梳理、参数/命令/配置项解释和故障排查建议。
- 如果问题涉及多个 JP1 组件、版本、命令或配置项，先检索相关文档，再综合回答。
- 优先保留 Manual 中的日文原文术语、命令名、参数名、画面名、消息 ID 和配置项名称，并用中文解释含义。
- 不绑定任何 skill，不执行代码，不访问外部网络，不臆造未检索到的文档内容。

检索与推理要求：
- 对明确的产品名、命令名、消息 ID、参数名、文件名、画面名，必须优先精确检索。
- 对宽泛问题，先拆解检索主题，再按主题综合。
- 当文档证据不足时，明确说明“当前 JP1 知识库中没有找到足够依据”，并给出建议的进一步检索关键词。
- 需要对比多个条目时，按表格或分点输出，说明适用场景、前提条件、差异和注意事项。
- 回答操作步骤时，区分前提条件、操作步骤、检查点、风险/注意事项。

回答格式：
- 默认使用中文回答。
- 引用或解释 JP1 专有名词时保留原文日语/英文术语。
- 不输出 Markdown 代码块，除非用户明确要求命令或配置样例。
- 不暴露内部推理过程，只输出结论、依据和可执行建议。`;

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

const existingAgentConfig = runPsql(`
SELECT COALESCE(config->>'rerank_model_id', '') || '|' || COALESCE(config->>'model_id', '')
FROM custom_agents
WHERE deleted_at IS NULL
  AND tenant_id = ${Number(tenantId)}
  AND config->'knowledge_bases' ? ${sqlLiteral(knowledgeBaseId)}
ORDER BY updated_at DESC
LIMIT 1;
`);

const [rerankModelIdFromAgent = "", modelIdFromAgent = ""] = existingAgentConfig
  ? existingAgentConfig.split("|")
  : [];

const modelId = modelIdFromAgent || modelIdFromKb;
const rerankModelId = rerankModelIdFromAgent || "";

if (!modelId) {
  throw new Error(`${knowledgeBaseName} knowledge base has no summary_model_id and no reusable agent model_id.`);
}

const thinking = true;
const agentConfig = {
  agent_mode: "smart-reasoning",
  agent_type: "rag-qa",
  system_prompt: systemPrompt,
  context_template: "",
  model_id: modelId,
  rerank_model_id: rerankModelId,
  temperature: 0.2,
  max_completion_tokens: 8192,
  thinking,
  max_iterations: 12,
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
    "请说明 JP1 中某个命令或参数的用途、格式和注意事项",
    "请对比两个 JP1 功能/配置项的适用场景和差异",
    "请根据错误消息或现象检索 Manual 并给出排查步骤",
    "请整理某个 JP1 操作流程的前提条件、步骤和检查点",
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
    avatar = 'J',
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
    'J',
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
  throw new Error("Failed to upsert JP1 manual consultation agent.");
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
    `XELORA_JP1_AGENT_ID=${upsertResult}`,
    `XELORA_JP1_KB_ID=${knowledgeBaseId}`,
    "",
  ].join("\n"),
  "utf8",
);

console.log(`Agent: ${agentName}`);
console.log(`XELORA_JP1_AGENT_ID=${upsertResult}`);
console.log(`XELORA_JP1_KB_ID=${knowledgeBaseId}`);
console.log(`tenant_id=${tenantId}`);
console.log(`model_id=${modelId}`);
console.log(`thinking=${thinking}`);
console.log("skills_selection_mode=none");
console.log(`Wrote ${envOutputPath}`);
