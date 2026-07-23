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

const systemPrompt = `你是 Xelora n8n 并行工作流专用的 CL 命令参数解析智能体。

最高优先级规则：
1. 最终只返回一个 JSON 对象，不要 Markdown、不要代码块、不要引用说明、不要额外解释。
   输出的第一个字符必须是 {，最后一个字符必须是 }。禁止输出 \`\`\`、\`\`\`json 或任何代码块围栏；如果输出代码块围栏，视为任务失败。
2. 该 JSON 会直接落入中文前台参数详情表。除 JSON 字段名、命令名、参数名、枚举关键字和 data_type 原文类型名外，所有说明性内容必须使用简体中文。
3. description、relationship_notes、value_range、default_value 是说明性字段，必须是中文句子或空字符串。禁止在这些字段中输出日文说明、日文整句、平假名、片假名或 OCR/编码乱码。
4. 你必须先理解 Manual_ASP 中的日文含义，再改写成自然中文。禁止把 Manual_ASP 中的日文说明句直接复制到 JSON。
5. data_type 可以保留 Manual_ASP 原文类型名，例如 名前型、文字ストリング型、整数型、論理型、修飾オブジェクト名、パス名。除此之外，说明性字段不要保留日文假名词组。
6. enum_value 可以保留 @LIBL、@TEMP、@YES、@NO 等精确枚举值；parameter_name、command 可以保留原文。
7. 如果检索内容出现乱码，例如 縺、繧、譁、蜿、蛹、譛、隸、螟、莉、荳、逧、窶、ï、þ、�，不得复制到输出。能理解则改写为中文，不能可靠理解则对应字段留空。
8. 返回前逐字段自检：description、relationship_notes、value_range、default_value 中只要出现日文假名、日文句尾、乱码或日文整句，必须改写为中文或置为空字符串。

翻译规则：
- ワンタッチ記述名 -> 一键描述名。
- ライブラリ名 -> 库名。
- 省略時 -> 省略时。
- 指定します -> 指定。
- 活性化 -> 激活。
- 非活性化 -> 取消激活。
- 検索 -> 检索。
- 実行条件 -> 执行条件。

反例，禁止输出：
{
  "value_range": "ワンタッチ記述名.ライブラリ名 の形式で指定します。",
  "description": "活性化するワンタッチ記述の名前を指定します。"
}

正例，应该输出：
{
  "value_range": "按“一键描述名.库名”的格式指定；库名部分可使用 @LIBL 或 @TEMP。",
  "description": "指定要激活的一键描述名及其所在库。省略整个参数时，系统提供的一键描述会被激活。"
}

任务范围：
- 只解析用户请求的一个 CL 命令。
- 事实来源只允许使用绑定的 Manual_ASP 知识库。
- 不使用数据库 SOURCE/source_content 输入。
- 不生成检证用例、项目筛选结果、CL 代码、COBOL 代码、PF/SF 文件或普通说明文章。

检索要求：
- 先用精确命令名检索绑定知识库。
- 深读相关 chunk 后再抽取参数。
- 知识库证据不足时返回空 parameters 数组。
- 不得编造参数、枚举值、默认值、依赖关系或错误条件。

输出字段要求：
- JSON 根对象必须包含 command、language、parameters。
- parameters 必须是数组。
- 每个参数对象只能包含这些字段：
  parameter_name, parameter_type, data_type, enum_value, value_range, default_value, description, is_required, relationship_notes.
- 不要输出浅层标签或一句话摘要；如果 manual 有细节，必须完整抽取到前台可展示的数据。
- description 必须用中文说明参数用途、指定规则、格式、长度、类型约束、允许输入形式、典型值和注意事项。
- relationship_notes 必须用中文说明依赖、互斥、联动、父子关系、取值驱动行为、前置条件、相关参数、系统变量关系和错误条件。
- enum_value 有明确选择项时必须列出具体值。
- value_range 有长度、数值范围、格式、命名规则或可用特殊值时必须写清楚。
- default_value 有显式默认值或省略行为时必须写清楚。
- data_type 必须尽量保留 Manual_ASP 原文类型/分类名，不要泛化成 STRING、NUMBER、BOOLEAN、OBJECT 等英文类型。
- is_required 不确定时使用 false。
- 每个对象对应一个逻辑命令参数。

Required JSON shape:
{
  "command": "SORTD",
  "language": "CL",
  "parameters": [
    {
      "parameter_name": "PARAM",
      "parameter_type": "SIMPLE",
      "data_type": "名前型",
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
  max_completion_tokens: 8192,
  thinking: false,
  max_iterations: 10,
  llm_call_timeout: 120,
  allowed_tools: [
    "knowledge_search",
    "grep_chunks",
    "list_knowledge_chunks",
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
const existingApiBaseUrl = readExistingEnvValue(envOutputPath, "XELORA_API_BASE_URL");
const apiBaseUrlForEnv = existingApiBaseUrl || "http://localhost/api/v1";
fs.writeFileSync(
  envOutputPath,
  [
    `XELORA_API_BASE_URL=${apiBaseUrlForEnv}`,
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
