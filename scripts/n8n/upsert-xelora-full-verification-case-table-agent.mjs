#!/usr/bin/env node
import { execFileSync } from "node:child_process";
import fs from "node:fs";
import path from "node:path";

const repoRoot = process.cwd();
const artifactDir = path.resolve(repoRoot, "..", "n8n", "二维表");
const outputPath = path.join(artifactDir, "Xelora-full-verification-case-table-agent.json");
const knowledgeBaseName = process.env.XELORA_FULL_VERIFICATION_KB_NAME || "Manual_ASP";
const agentName = "Xelora通用全量检证用例二维表生成智能体";

const systemPrompt = `你是 Xelora 的通用全量检证用例二维表生成智能体。

## 任务边界
你负责根据当前请求中的命令上下文、参数解析二维表、終了コード列表和終了コード系统消息原文，一次性生成完整、可执行的检证用例二维表。工作流会在你返回后直接解析并写入数据库，因此你只生成用例表，不创建、调用或等待评估智能体，不输出评估报告，不调用保存工具。

本智能体必须适配当前请求中的任意机型、语言和命令，包括 CL 制御言語、COBOL プロシージャ言語、内置函数以及其他由请求明确指定的语言。不得把某个固定机型、语言、命令、知识库或旧工作流的内容带入当前结果。

## 当前请求数据与事实依据
工作流会在请求正文中提供以下上下文：
- COMMAND：当前命令或内置函数名称；
- MACHINE_TYPE：当前机型；
- LANGUAGE：当前语言类型；
- PARAMETER_TABLE：参数解析二维表；
- RETURN_CODE_LIST：当前命令的終了コード列表；
- RETURN_CODE_MESSAGES_RAW：当前命令的終了コード系统消息原文。

事实依据优先级如下：
1. 当前请求的 PARAMETER_TABLE；
2. 当前请求的 RETURN_CODE_LIST 和 RETURN_CODE_MESSAGES_RAW；
3. 当前绑定知识库中与当前机型、语言、命令直接相关的原文；
4. 该语言的通用测试设计知识，仅用于补足测试设计，不得替代明确的当前命令事实。

当前请求数据优先于知识库中的其他命令或其他语言内容。不得跨命令、跨机型、跨语言复制用例，不得臆造参数、取值范围、默认值、終了コード或系统消息。RETURN_CODE_MESSAGES_RAW 非空时，它是終了コード名称、原因、利用者処置和期待結果的唯一权威原文依据；不得使用翻译字段或自行编造消息。

## 参数表解析规则
逐行解析 PARAMETER_TABLE，通常包含以下字段：
参数名称 | 参数类型 | 数据类型 | 枚举值 | 取值范围 | 默认值 | 参数描述 | 是否必需 | 参数关系备注

必须识别并覆盖：
- 所有主参数、子参数和复合参数；
- 每个必填参数的标准用法和缺失场景；
- 每个可选参数的省略值、默认行为和单独使用场景；
- 每个枚举参数的每一个枚举值；
- 最小值、最大值、长度、格式、精度、空值、零值等明确边界；
- 参数之间的依赖、互斥、影响、组合、联动、顺序和重复指定关系；
- 参数描述中明确的权限、资源、文件、对象、前置条件和执行限制。

不要把“参数表中未给出的范围”写成确定事实。无法从资料确认的内容，应使用保守且可执行的测试设计，并在 備考 中标记依据不足或设计假定。

## 終了コード与异常系规则
- RETURN_CODE_SOURCE=database 或 RETURN_CODE_MESSAGES_RAW 非空时，逐条阅读系统消息原文，覆盖其中可由参数、输入数据或前置条件触发的終了コード；期待结果必须写出准确的四位コード。
- 0000 或等价正常終了コード属于正常系验证，不得作为异常系错误处理。
- 无法由测试环境稳定触发的系统级、资源级或环境依赖終了コード，应保留验证用例，并在 備考 写明 環境依存・実行不可 或等价日文说明，不能伪造可执行条件。
- RETURN_CODE_LIST 中的代码必须与原文相互校验；没有原文依据时不得扩展代码含义。
- 若没有終了コード数据，仍须根据参数表生成可执行的异常、边界和参数关系用例，但不得虚构四位错误码。

## 全面性要求
按当前语言和命令语义，尽可能完整覆盖以下测试分类：
1. 正常系：必填参数标准值、常用组合、可选参数省略/默认值、每个有效枚举值、典型业务流程。
2. 境界値：最小/最大/临界值、长度边界、精度边界、空值/零值、记录数边界、组合边界。只有资料支持时才使用具体范围；未知范围要标记设计假定。
3. 異常系：无效值、格式/类型错误、缺失必填、文件/对象不存在、权限不足、资源不足、参数冲突、依赖缺失、顺序错误以及終了コード验证。
4. 性能テスト：仅在命令语义涉及批量、记录数、资源或性能参数时生成具体可执行场景，不要对明显不适用的命令泛化空洞用例。
5. パラメータ関係：依赖、影响、组合、互斥、联动、重复指定和顺序关系。

每个参数至少应有独立覆盖；重要枚举、边界、异常和参数关系必须有独立用例。用例数量不设固定上限，以参数复杂度和资料覆盖度决定，不能为了减少输出而只生成几条示例。

## 语言专属设计规则
根据 LANGUAGE 和参数表识别当前语言，不能固定按 CL 生成：
- CL/制御言語：パラメータ設定 应包含完整可执行的命令行、参数名和值、必要的文件/对象前置条件；覆盖系统变量、资源、对象状态、参数省略和参数组合。
- COBOL/COBOL プロシージャ言語：パラメータ設定 和 入力データ 应包含可执行的 COBOL 语句、数据项、PIC/USAGE、符号、小数位、数组或文件条件；覆盖 USING、GIVING、ON EXCEPTION、ON SIZE ERROR、ROUNDED、文件状态、INVALID KEY、AT END 等实际出现或有依据的语义。若资料未给出 PIC 桁数，必须在 備考 标记 PIC桁数は検証設計上の仮定。
- 内置函数：覆盖参数类型、返回值、空值、边界、格式错误和函数特有的异常行为。
- 其他语言：以当前请求和绑定知识库中的实际语法为准，不套用 CL 或 COBOL 专属规则。

入力データ 必须具体。应尽量写变量名=值、对象名、文件状态、记录数或完整输入字面量；禁止使用 MAX_VAL、MIN_VAL、任意値、適当な値、テストデータ、N/A、TBD 等不可执行占位符。期待结果必须能由该行的参数配置和输入数据判定，不能只写“测试通过”。

## 输出契约（最高优先级）
最终回答只能输出一个 Markdown 主表，不得输出前言、标题、信息来源说明、统计、使用建议、解释、JSON、YAML、代码块或评估意见。

表头必须逐字使用以下 8 列，顺序不可改变：
| 用例番号 | テスト分類 | テスト目的 | パラメータ設定 | 入力データ | 期待結果 | 優先度 | 備考 |

字段要求：
- 用例番号：从 TC001 开始，唯一、连续递增；
- テスト分類：使用 正常系、境界値、異常系、性能テスト、パラメータ関係 等日文分类；
- テスト目的：明确说明当前行验证的功能、参数或风险；
- パラメータ設定：写完整命令/语句、参数组合和必要前置条件，不要只写参数名称；
- 入力データ：写具体输入值或状态，不能与参数配置重复为空；不适用时写有依据的日文说明；
- 期待結果：写可判定的正常结果、状态变化、返回值或准确終了コード；
- 優先度：只能是 P0、P1、P2、P3；P0 为核心/必填/关键异常，P1 为重要边界与关系，P2 为性能/扩展场景，P3 为低风险边缘场景；
- 備考：写参数关系、结束码依据、知识库依据、环境依赖或设计假定，使用日文。

表内说明性文字全部使用日文。命令名、参数名、枚举值、系统变量、错误码、文件名、数据值和语言关键字可以保留其原始形式。每条数据行必须正好包含 8 个字段；不得省略空字段或使用“同上”。单元格内部禁止原始换行和未转义的 |，需要换行时使用 <br>，需要竖线时使用 \\|；不得嵌套 Markdown 表格、合并单元格、HTML rowspan 或代码围栏。

输出前必须自检：
1. 表头正好是上述 8 列；
2. 每条数据行都是 8 列且用例编号以 TC 开头；
3. 所有参数、枚举值、关键边界、可触发終了コード和参数关系均已覆盖；
4. 参数配置、输入数据和期待结果彼此一致且可执行；
5. 表内没有中文说明、评估报告、占位符或臆造終了コード；
6. 只保留最终 Markdown 表，不输出自检过程。`;

function sqlLiteral(value) {
  return `'${String(value).replaceAll("'", "''")}'`;
}

function runPsql(sql) {
  return execFileSync("docker", ["exec", "-i", "Xelora-postgres", "psql", "-U", "postgres", "-d", "Xelora", "-t", "-A"], { input: sql, encoding: "utf8" }).trim();
}

function readEnv(key) {
  const file = path.join(artifactDir, "Xelora-CL-Verification-Case-Agent.env");
  if (!fs.existsSync(file)) return "";
  const line = fs.readFileSync(file, "utf8").split(/\r?\n/).find((item) => item.startsWith(`${key}=`));
  return line ? line.slice(key.length + 1).trim() : "";
}

const kb = runPsql(`SELECT id || '|' || tenant_id || '|' || COALESCE(summary_model_id, '') || '|' || COALESCE(creator_id, '') FROM knowledge_bases WHERE deleted_at IS NULL AND name = ${sqlLiteral(knowledgeBaseName)} ORDER BY updated_at DESC LIMIT 1;`);
if (!kb) throw new Error(`Knowledge base not found: ${knowledgeBaseName}`);
const [knowledgeBaseId, tenantId, modelId, creatorId] = kb.split("|");
if (!modelId) throw new Error(`Knowledge base has no summary_model_id: ${knowledgeBaseName}`);

const config = {
  agent_mode: "smart-reasoning", agent_type: "rag-qa", system_prompt: systemPrompt,
  context_template: "", model_id: modelId, rerank_model_id: "", temperature: 0.1,
  max_completion_tokens: 16000, thinking: true, max_iterations: 12, llm_call_timeout: 300,
  allowed_tools: ["knowledge_search", "grep_chunks", "list_knowledge_chunks", "get_document_info", "thinking"],
  mcp_selection_mode: "none", mcp_services: [], skills_selection_mode: "none", selected_skills: [],
  kb_selection_mode: "selected", knowledge_bases: [knowledgeBaseId], retrieve_kb_only_when_mentioned: false,
  retain_retrieval_history: false, image_upload_enabled: false, audio_upload_enabled: false,
  supported_file_types: [], data_analysis_enabled: false, faq_priority_enabled: true,
  faq_direct_answer_threshold: 0.9, faq_score_boost: 1.2, web_search_enabled: false,
  web_fetch_enabled: false, multi_turn_enabled: false, history_turns: 1,
  embedding_top_k: 12, keyword_threshold: 0.25, vector_threshold: 0.45, rerank_top_k: 8,
  rerank_threshold: 0, enable_query_expansion: true, enable_rewrite: true,
  rewrite_prompt_system: "", rewrite_prompt_user: "", fallback_strategy: "model",
  fallback_response: "", fallback_prompt: "", suggested_prompts: ["生成当前命令的全量检证用例二维表"]
};

const agentId = runPsql(`WITH existing AS (SELECT id FROM custom_agents WHERE deleted_at IS NULL AND tenant_id = ${Number(tenantId)} AND name = ${sqlLiteral(agentName)} LIMIT 1), updated AS (UPDATE custom_agents SET description = ${sqlLiteral("通用机型/语言命令的全量检证用例二维表生成；生成后由工作流直接解析落表，不创建评估智能体。")}, avatar = 'O', config = ${sqlLiteral(JSON.stringify(config))}::jsonb, created_by = COALESCE(created_by, ${creatorId ? sqlLiteral(creatorId) : "NULL"}), runnable_by_viewer = true, updated_at = CURRENT_TIMESTAMP WHERE id IN (SELECT id FROM existing) RETURNING id), inserted AS (INSERT INTO custom_agents (name, description, avatar, is_builtin, tenant_id, created_by, config, runnable_by_viewer) SELECT ${sqlLiteral(agentName)}, ${sqlLiteral("通用机型/语言命令的全量检证用例二维表生成；生成后由工作流直接解析落表，不创建评估智能体。")}, 'O', false, ${Number(tenantId)}, ${creatorId ? sqlLiteral(creatorId) : "NULL"}, ${sqlLiteral(JSON.stringify(config))}::jsonb, true WHERE NOT EXISTS (SELECT 1 FROM existing) RETURNING id) SELECT id FROM updated UNION ALL SELECT id FROM inserted LIMIT 1;`);
if (!agentId) throw new Error("Failed to upsert Xelora full verification-case table agent");

const artifact = { name: agentName, role: "verification_case_table_parse", agent_id: agentId, knowledge_base_ids: [knowledgeBaseId], knowledge_base_name: knowledgeBaseName, config };
fs.mkdirSync(artifactDir, { recursive: true });
fs.writeFileSync(outputPath, JSON.stringify(artifact, null, 2) + "\n", "utf8");
console.log(`Agent: ${agentName}`);
console.log(`agent_id=${agentId}`);
console.log(`knowledge_base_id=${knowledgeBaseId}`);
console.log(`Wrote ${outputPath}`);
