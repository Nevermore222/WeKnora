#!/usr/bin/env node
import { execFileSync } from "node:child_process";
import fs from "node:fs";
import path from "node:path";

const agentName = "Xelora TOIN CL技术点二维表生成智能体";
const knowledgeBaseName = "Manual_ASP";
const outputPath = path.resolve(process.cwd(), "..", "n8n", "二维表", "Xelora-TOIN-CL-Tech-Point-Agent.env");

function sql(value) {
  return `'${String(value).replaceAll("'", "''")}'`;
}

function psql(query) {
  return execFileSync("docker", ["exec", "-i", "Xelora-postgres", "psql", "-U", "postgres", "-d", "Xelora", "-t", "-A", "-v", "ON_ERROR_STOP=1"], { input: query, encoding: "utf8" }).trim();
}

const kb = psql(`SELECT id || '|' || tenant_id || '|' || COALESCE(summary_model_id, '') || '|' || COALESCE(creator_id, '') FROM knowledge_bases WHERE deleted_at IS NULL AND name = ${sql(knowledgeBaseName)} ORDER BY updated_at DESC LIMIT 1;`);
if (!kb) throw new Error("Manual_ASP knowledge base not found");
const [kbId, tenantId, modelId, creatorId] = kb.split("|");
if (!modelId) throw new Error("Manual_ASP has no summary model");

const systemPrompt = `# 角色
你是 Xelora 内专用于 TOIN 项目的 CL（富士通オフコン制御言語）技术点检证用例二维表生成智能体。你只处理 CL，不处理 COBOL。你的输出将由 n8n 解析并写入 analyzes.cl_verification_cases_master_proj，之后还会被用于生成真实 CL 检证程序，因此信息必须完整、可追踪、可执行。

# 输入契约
每次请求只包含一个技术点，字段包括 COMMAND_NAME、LANGUAGE_TYPE=CL、TECH_POINT_NO、SUB_CATEGORY，以及最多五条 references。每条 reference 包含：
- source_tech_point_id
- PGMID
- command_name
- tech_point_no
- sub_category
- SOURCE_CONTENT
- variable_declaration

references 是同一技术点在不同 PGM 或同一 PGM 不同位置的证据集合，不是五个必须分别生成 case 的任务。只为这个技术点生成一批必要且不重复的 case。

# 事实优先级
1. 当前 reference 的 variable_declaration（非空时为该 reference 的权威变量定义）。
2. 当前 reference 的 SOURCE_CONTENT、TECH_POINT_NO、SUB_CATEGORY、PGMID。
3. Manual_ASP 中 CL 相关资料，用于补充命令语义、参数约束、结束码和异常行为。
4. 专业常识仅用于保守补足；不得覆盖或改写前两级事实。

禁止跨 reference 借用变量定义。禁止把不同 PGM 的 SOURCE_CONTENT 拼接成源数据中不存在的 CL 片段。相同小分类也不代表变量可以互换。

# SOURCE 与变量保真协议
- 保持每段 SOURCE_CONTENT 的命令结构、参数顺序、系统变量、续行和关键字语义。
- variable_declaration 非空时，必须保留原变量名、层级、类型、长度、精度、符号、初始值、数组、组项和子项关系。不得改名、缩短长度、改变数据类型或用通用占位变量替代。
- variable_declaration 为空时，只能依据当前 reference 的 SOURCE_CONTENT 和 SUB_CATEGORY 设计最小可执行输入；備考必须写明“変数宣言は検証設計上の仮定”。不得声称推断值来自原源码。
- 若五条引用存在不同变量定义或 SOURCE 形态，应设计能体现差异的 case；如果差异不影响技术点行为，不要复制同质 case。

# 用例覆盖
根据技术点实际语义选择必要场景，不以数量为目标：
- 正常系：典型有效参数和正常结束。
- 境界値：最小/最大长度、数值边界、空值、临界枚举或资源边界，仅在该技术点适用时生成。
- 異常系：缺失必填项、非法格式、非法枚举、资源不存在、权限/状态冲突、知识库明确的结束码。
- 参数关系：依赖、互斥、组合、顺序和系统变量联动。
- PGM 差异：只有当引用间的 SOURCE 或变量定义导致可观察行为差异时生成。

不得生成与 SOURCE_CONTENT、SUB_CATEGORY 或 Manual_ASP 无依据的场景。不得使用 MAX_VAL、MIN_VAL、N/A、TBD、任意值、同上、略等不可执行占位符。

# 可生成真实代码的字段要求
- パラメータ設定：写清命令参数、原变量定义或明确标注的设计假设、前置状态、依赖资源以及被检证的完整 CL 命令片段。单元格内换行使用 <br>。
- 入力データ：为每个输入变量和参数给出 NAME=value 形式的具体值；值必须符合原类型、长度、精度和枚举约束。
- 期待結果：写出可判定的结束状态、结束码、资源变化、变量变化或输出内容；不得只写“成功”“报错”。
- 備考：列出该行实际使用的 source_tech_point_id 与 PGMID，格式为 references=id@PGMID,id@PGMID；并标明事实来自 SOURCE_CONTENT、変数宣言、Manual_ASP 或検証設計上の仮定。

# 输出语言与格式
除标识符、CL 语法、变量名、技术点号、PGMID、数值和结束码外，主表所有说明必须使用自然日文。不要输出中文说明。

只输出一个 Markdown 主表，表头必须逐字为：
| 技術点NO | 用例番号 | 小分類 | テスト分類 | テスト目的 | パラメータ設定 | 入力データ | 期待結果 | 優先度 | 備考 |

规则：
- 每个数据行必须恰好 10 列。
- 技術点NO 必须完整等于输入 TECH_POINT_NO，禁止省略、截断或改写。
- 小分類必须完整等于输入 SUB_CATEGORY。
- 用例番号从 TC001 开始连续编号；n8n 会在落库前改为全局编号。
- テスト分類使用 正常系、境界値、異常系、組合せ、性能/資源 等明确分类。
- 優先度仅允许 P0、P1、P2、P3。核心正常和关键异常用 P0/P1，次要边界用 P2，低风险补充用 P3。
- 单元格中的竖线必须转义，禁止 rowspan、空白重复单元格、“同上”或省略号。

主表之后只允许追加一行机器可读元数据：
{"used_reference_ids":[1,2]}
used_reference_ids 必须是本次实际用于设计 case 的 source_tech_point_id 去重数组，只能包含输入 references 中的 ID。至少包含一个 ID。

# 输出前自检
在内部完成以下检查，不要输出检查过程：
1. 所有行都是当前 TECH_POINT_NO 和 SUB_CATEGORY。
2. 每行十列且编号、优先级合法。
3. 每个变量值符合 variable_declaration；没有跨 PGM 借用定义。
4. SOURCE_CONTENT 结构未被改造为不存在的源码。
5. 参数配置、输入数据、期待结果可以共同支撑真实 CL 检证代码。
6. 備考中的引用 ID/PGMID 与 used_reference_ids 一致。
`;

const config = {
  agent_mode: "smart-reasoning",
  agent_type: "rag-qa",
  system_prompt: systemPrompt,
  context_template: "",
  model_id: modelId,
  rerank_model_id: "",
  temperature: 0.1,
  max_completion_tokens: 12000,
  thinking: true,
  max_iterations: 8,
  llm_call_timeout: 240,
  allowed_tools: ["knowledge_search", "grep_chunks", "list_knowledge_chunks", "thinking"],
  mcp_selection_mode: "none",
  mcp_services: [],
  skills_selection_mode: "none",
  selected_skills: [],
  kb_selection_mode: "selected",
  knowledge_bases: [kbId],
  retrieve_kb_only_when_mentioned: false,
  retain_retrieval_history: false,
  image_upload_enabled: false,
  audio_upload_enabled: false,
  supported_file_types: [],
  data_analysis_enabled: false,
  web_search_enabled: false,
  web_fetch_enabled: false,
  multi_turn_enabled: false,
  history_turns: 1,
  embedding_top_k: 12,
  keyword_threshold: 0.25,
  vector_threshold: 0.45,
  rerank_top_k: 8,
  rerank_threshold: 0,
  enable_query_expansion: true,
  enable_rewrite: true,
  fallback_strategy: "model",
  fallback_response: "",
  suggested_prompts: ["生成 CL 技术点二维表", "按 references 保留 SOURCE_CONTENT 和变量定义"],
};

const description = "TOIN 项目 CL 技术点二维表专用智能体。按 PGM 覆盖优先的最多五条 references 生成可用于真实检证代码的 Markdown case。";
const result = psql(`WITH existing AS (SELECT id FROM custom_agents WHERE tenant_id=${Number(tenantId)} AND name=${sql(agentName)} AND deleted_at IS NULL LIMIT 1), updated AS (UPDATE custom_agents SET description=${sql(description)}, config=${sql(JSON.stringify(config))}::jsonb, runnable_by_viewer=true, updated_at=CURRENT_TIMESTAMP WHERE id IN (SELECT id FROM existing) RETURNING id), inserted AS (INSERT INTO custom_agents (name, description, avatar, is_builtin, tenant_id, created_by, config, runnable_by_viewer) SELECT ${sql(agentName)}, ${sql(description)}, 'T', false, ${Number(tenantId)}, ${creatorId ? sql(creatorId) : "NULL"}, ${sql(JSON.stringify(config))}::jsonb, true WHERE NOT EXISTS (SELECT 1 FROM existing) RETURNING id) SELECT id FROM updated UNION ALL SELECT id FROM inserted LIMIT 1;`);
if (!result) throw new Error("failed to create TOIN CL tech-point agent");
fs.mkdirSync(path.dirname(outputPath), { recursive: true });
fs.writeFileSync(outputPath, [`XELORA_API_BASE_URL=http://192.168.2.61:18080/api/v1`, `XELORA_CL_TECH_POINT_AGENT_ID=${result}`, `XELORA_MANUAL_ASP_KB_ID=${kbId}`, ""].join("\n"), "utf8");
console.log(`XELORA_CL_TECH_POINT_AGENT_ID=${result}`);
console.log(`XELORA_MANUAL_ASP_KB_ID=${kbId}`);
console.log(`Wrote ${outputPath}`);
