#!/usr/bin/env node
import { execFileSync } from "node:child_process";
import fs from "node:fs";
import path from "node:path";

const repoRoot = process.cwd();
const envOutputPath = path.resolve(
  repoRoot,
  "..",
  "n8n",
  "参数解析",
  "Xelora-CL-Parameter-Parse-Parallel.env",
);

const agentName = "CL命令参数解析智能体";
const agentDescription =
  "专用于 n8n 并行工作流的 CL 命令参数解析智能体。只基于 Manual_ASP 知识库输出严格 JSON。";

const systemPrompt = `You are the dedicated CL command parameter parsing agent for the Xelora n8n parallel workflow.

Scope:
- Parse only the requested CL command.
- Use only the bound Manual_ASP knowledge base as the factual source.
- Do not use database SOURCE/source_content input.
- Do not generate verification cases, project filtering results, CL code, COBOL code, PF/SF files, or prose explanations.

Retrieval discipline:
- Search the bound knowledge base for the exact command name first.
- Deep-read the relevant chunks before extracting parameters.
- If the knowledge base does not contain enough evidence, return an empty parameters array.
- Do not fabricate parameters, enum values, defaults, or dependency rules.

Output contract:
- Return exactly one JSON object.
- Do not return Markdown fences, comments, citations, or explanatory text.
- The JSON root must contain: command, language, parameters.
- parameters must be an array.
- Every parameter object must contain exactly these fields:
  parameter_name, parameter_type, data_type, enum_value, value_range, default_value, description, is_required, relationship_notes.
- Use empty strings for unknown optional string fields.
- Use false for unknown is_required.
- Keep one logical command parameter per object.

Required JSON shape:
{
  "command": "SORTD",
  "language": "CL",
  "parameters": [
    {
      "parameter_name": "PARAM",
      "parameter_type": "SIMPLE",
      "data_type": "STRING",
      "enum_value": "",
      "value_range": "",
      "default_value": "",
      "description": "",
      "is_required": false,
      "relationship_notes": ""
    }
  ]
}`;

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
WHERE deleted_at IS NULL AND name = 'Manual_ASP'
ORDER BY updated_at DESC
LIMIT 1;
`);

if (!kbResult) {
  throw new Error("Manual_ASP knowledge base was not found in Xelora database.");
}

const [manualAspKbId, tenantId, modelIdFromKb, creatorId] = kbResult.split("|");

const existingAgentConfig = runPsql(`
SELECT COALESCE(config->>'rerank_model_id', '') || '|' || COALESCE(config->>'model_id', '')
FROM custom_agents
WHERE deleted_at IS NULL
  AND tenant_id = ${Number(tenantId)}
  AND config->'knowledge_bases' ? ${sqlLiteral(manualAspKbId)}
ORDER BY updated_at DESC
LIMIT 1;
`);

const [rerankModelIdFromAgent = "", modelIdFromAgent = ""] = existingAgentConfig
  ? existingAgentConfig.split("|")
  : [];

const modelId = modelIdFromAgent || modelIdFromKb;
const rerankModelId = rerankModelIdFromAgent || "";

const agentConfig = {
  agent_mode: "smart-reasoning",
  agent_type: "rag-qa",
  system_prompt: systemPrompt,
  context_template: "",
  model_id: modelId,
  rerank_model_id: rerankModelId,
  temperature: 0.1,
  max_completion_tokens: 2048,
  thinking: false,
  max_iterations: 8,
  llm_call_timeout: 120,
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
  knowledge_bases: [manualAspKbId],
  retrieve_kb_only_when_mentioned: false,
  retain_retrieval_history: false,
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
  multi_turn_enabled: false,
  history_turns: 1,
  embedding_top_k: 8,
  keyword_threshold: 0.3,
  vector_threshold: 0.5,
  rerank_top_k: 5,
  rerank_threshold: 0,
  enable_query_expansion: false,
  enable_rewrite: false,
  rewrite_prompt_system: "",
  rewrite_prompt_user: "",
  fallback_strategy: "model",
  fallback_response: "",
  fallback_prompt: "",
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
    avatar = 'C',
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
    'C',
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
  throw new Error("Failed to upsert Xelora parameter parsing agent.");
}

fs.mkdirSync(path.dirname(envOutputPath), { recursive: true });
const existingApiKey = readExistingEnvValue(envOutputPath, "N8N_XELORA_API_KEY");
const apiKeyForEnv =
  existingApiKey && existingApiKey !== "configure-in-n8n-runtime"
    ? existingApiKey
    : "configure-in-n8n-runtime";
fs.writeFileSync(
  envOutputPath,
  [
    "XELORA_API_BASE_URL=http://localhost/api/v1",
    `N8N_XELORA_API_KEY=${apiKeyForEnv}`,
    `XELORA_PARAMETER_AGENT_ID=${upsertResult}`,
    `XELORA_MANUAL_ASP_KB_ID=${manualAspKbId}`,
    "",
  ].join("\n"),
  "utf8",
);

console.log(`Agent: ${agentName}`);
console.log(`XELORA_PARAMETER_AGENT_ID=${upsertResult}`);
console.log(`XELORA_MANUAL_ASP_KB_ID=${manualAspKbId}`);
console.log(`Wrote ${envOutputPath}`);
