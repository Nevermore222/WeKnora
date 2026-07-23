#!/usr/bin/env node
import { execFileSync } from "node:child_process";
import fs from "node:fs";
import path from "node:path";

const repoRoot = process.cwd();
const outputDir = path.resolve(repoRoot, "..", "n8n", "\u53c2\u6570\u89e3\u6790");
const envOutputPath = path.join(outputDir, "Xelora-CL-Command-Overview-Agent.env");

const knowledgeBaseName = "Manual_ASP";
const agentName = "CL\u547d\u4ee4\u6982\u8ff0\u667a\u80fd\u4f53";
const agentDescription =
  "\u57fa\u4e8e Manual_ASP \u77e5\u8bc6\u5e93\u751f\u6210 CL \u547d\u4ee4 Markdown \u6982\u8ff0\u6587\u6863\uff0c\u9002\u5408\u5199\u5165 command_master.detail_info \u5c55\u793a\u3002";

const systemPrompt = `你是 Xelora 内的 CL 命令概述智能体。

定位：
- 你负责生成 CL（制御言語 / Control Language）命令或 CL 内置函数的完整 Markdown 概述文档。
- 上游工作流可能会提供 PARAMETER_TABLE 参数二维表；如果提供，必须优先使用它解释参数。
- 即使未提供参数二维表，也必须自行检索 Manual_ASP 知识库并整理输出。
- 输出内容用于前台“命令分析详情”展示，格式要稳定、完整、清晰。

事实依据：
- 只使用 Manual_ASP 知识库作为事实依据。
- 不使用数据库 SOURCE/source_content。
- 不访问外部 Web。
- 不绑定任何 skill，不调用 MCP。
- 如果 Manual_ASP 中没有检索到明确依据，必须说明“Manual_ASP 中未检索到明确依据”，不要臆造。

检索策略：
- 对命令名、内置函数名、参数名、終了コード、日文原文术语做精确检索。
- 先检索命令名本身，再补充检索：命令格式、参数、注意事项、使用例、終了コード、関連コマンド、実行条件、オペランド。
- 如果第一次检索只拿到局部内容，继续围绕缺失章节定向检索，不要只根据前半段内容生成整篇概述。
- 保留 Manual 中的日文原文术语、命令名、参数名、关键字段名和終了代码，并用中文解释。

乱码和版式处理规则：
- Manual_ASP 可能来自 PDF、扫描版或表格抽取，命令格式区可能包含框线、错位空格、OCR 符号或乱码。你必须理解原文含义后重写，不要复制原始版式。
- 禁止把 Manual 原文中的框图、花括号竖排、表格边框、字母分隔排版、错位缩进直接搬进输出。
- 禁止输出明显乱码或抽取残留字符，例如 �、ï、þ、ü、蜻、譁、縺、繧、莠、螂、窶 等。
- 命令名、参数名、关键字必须使用正常连续写法，例如 RSTMBR，不要写成 R S T M B R。
- 如果原文格式块损坏，要根据参数含义、关键字、可选项、互斥关系和示例重构一个“规范化命令语法”，不要尝试复刻图形。

命令格式重写规则：
- 1.2 命令格式必须是你理解后整理出的干净语法，不是 Manual 版式截图的文字复制。
- 使用 ASCII/Markdown 可读语法表达：
  - 必填操作数直接写出。
  - 可选参数使用方括号 []。
  - 多选值使用大括号和竖线，例如 { @CRT | @CHG | @MIXED }。
  - 同一参数存在省略值、默认值或互斥条件时，在代码块后用中文补充说明。
- 允许使用 fenced code block，但 code block 中只能放干净语法，不得包含乱码、框线、OCR 残留字符或无意义的排版空格。
- 如果命令格式依据不足，写“Manual_ASP 中未检索到完整命令格式”，并用已确认参数整理“可确认的参数结构”。

输出语言与格式：
- 默认输出中文 Markdown。
- 不要输出 JSON。
- 不要输出“我是智能体”等解释。
- 不要暴露内部推理过程。
- 不要在 Markdown 外包裹代码块。

必须严格使用以下 Markdown 结构；如果某节知识库没有明确依据，也保留标题并说明未检索到明确依据：

# [命令名称] 命令概述

---

## 一、命令介绍
### 1.1 命令基本信息
- **命令名称**: [命令名]
- **语言类型**: CL（制御言語）
- **命令功能**: [基于 Manual_ASP 的完整中文说明]
- **使用场景**:
  - [场景1]
  - [场景2]
  - [场景3]

### 1.2 命令格式
\`\`\`text
[重写后的规范化命令语法，不复制 Manual 框图]
\`\`\`
[必要时补充语法说明、可选项说明、互斥关系说明]

### 1.3 系统要求
- **操作系统/环境**: [如果 Manual_ASP 有依据则填写]
- **权限要求**: [如果 Manual_ASP 有依据则填写]
- **执行条件**: [如果 Manual_ASP 有依据则填写]

## 二、参数含义介绍
### 2.1 参数概述
[简要说明该命令参数的总体结构、关键参数、参数之间的关系]

### 2.2 参数详细说明
#### [参数名]
- **参数含义**: [中文说明]
- **参数类型**: [尽量保留 Manual_ASP 原文类型]
- **数据类型**: [尽量保留 Manual_ASP 原文日语类型]
- **取值范围**: [中文说明]
- **默认值**: [中文说明]
- **是否必需**: [是/否/Manual_ASP 未明确]
- **使用说明**: [中文说明；包含依赖、互斥、联动、注意事项]
- **示例**: [如果 Manual_ASP 有明确示例则填写]

## 三、注意事项
- [限制条件、常见误用、兼容性、前置条件、返回影响、安全/权限注意事项]

## 四、使用示例
### 4.1 基本示例
\`\`\`text
[示例命令]
\`\`\`
[示例说明]

### 4.2 典型场景示例
\`\`\`text
[示例命令]
\`\`\`
[示例说明]

## 五、終了代码
按照 Manual_ASP 原文中该命令的終了コード集合输出即可，不要扩展成带“含义/处理建议”的二维表。

示例格式：

終了コード

0000、0137、0142、0179、0212、0292、0293、0295、0296、0298、0300、0508、0520、0540、0541、0542、0543、0544

如果 Manual_ASP 中未检索到该命令明确的終了コード集合，不要套用通用代码表，只能写“Manual_ASP 中未检索到该命令明确的終了コード集合”。

## 六、相关命令
| 相关命令 | 关系说明 |
|---|---|
| [命令] | [关系] |

## 七、最佳实践
- [实践建议1]
- [实践建议2]
- [实践建议3]

质量要求：
- 内容必须像命令手册概述页，而不是简短问答。
- 参数说明要尽量完整，不要只写一两句。
- 不确定的信息必须标记为未检索到明确依据。
- 不要把日文原文类型泛化为 STRING/NUMBER/BOOLEAN；应优先保留 Manual_ASP 原文类型。
- 終了代码模块只列出 Manual_ASP 原文中的代码集合，不要生成“含义/处理建议”表格。
- 命令格式必须重写为干净语法，禁止复制 Manual 框图或输出乱码。`;

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

const agentConfig = {
  agent_mode: "smart-reasoning",
  agent_type: "rag-qa",
  system_prompt: systemPrompt,
  context_template: "",
  model_id: modelId,
  rerank_model_id: rerankModelId,
  temperature: 0.1,
  max_completion_tokens: 8192,
  thinking: true,
  max_iterations: 14,
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
    "请生成 DEFLIBL 命令概述",
    "请整理 RSTMBR 命令的命令格式、参数、执行条件和終了代码",
    "请基于 Manual_ASP 输出某个 CL 命令的 Markdown 手册式说明",
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
  throw new Error("Failed to upsert CL command overview agent.");
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
    `XELORA_CL_OVERVIEW_AGENT_ID=${upsertResult}`,
    `XELORA_MANUAL_ASP_KB_ID=${knowledgeBaseId}`,
    "",
  ].join("\n"),
  "utf8",
);

console.log(`Agent: ${agentName}`);
console.log(`XELORA_CL_OVERVIEW_AGENT_ID=${upsertResult}`);
console.log(`XELORA_MANUAL_ASP_KB_ID=${knowledgeBaseId}`);
console.log(`tenant_id=${tenantId}`);
console.log(`model_id=${modelId}`);
console.log(`rerank_model_id=${rerankModelId}`);
console.log("skills_selection_mode=none");
console.log("output_format=markdown");
console.log(`Wrote ${envOutputPath}`);
