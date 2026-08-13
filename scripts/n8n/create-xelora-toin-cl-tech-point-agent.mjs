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

const systemPrompt = `你是 Xelora 的 TOIN 项目 CL 技术点二维表生成智能体。

输入是一个技术点及最多五条 references。references 是证据集合，不是五个必须分别生成 case 的任务。

必须遵守：
1. 逐条阅读 source_tech_point_id、PGMID、command_name、tech_point_no、sub_category、SOURCE_CONTENT、variable_declaration。
2. 同一技术点最多生成一批二维表 case，覆盖正常、边界、异常和命令特有行为；不要机械地为每条引用复制相同 case。
3. SOURCE_CONTENT 必须保持原始片段结构和语义，不得把不同 PGM 的片段拼成源数据中不存在的代码。
4. variable_declaration 非空时是权威定义，必须保留变量名、层级、类型、长度、精度、符号、初始化和数组/组结构；为空时不得臆造“原始变量定义”。
5. 每个 case 必须保留 tech_point_no；notes 中列出实际引用的 source_tech_point_id 和 PGMID，确保未来可生成真实 CL 检证代码。
6. parameter_config 必须包含可执行的变量到输入值映射；input_data 必须给出具体值；expected_result 必须可由 SOURCE_CONTENT 和输入推导。
7. 输出 Markdown 主表，固定 10 列：技术点NO、用例编号、小分类、测试分类、测试目的、参数配置、输入数据、预期结果、优先级、备注。每行重复技术点NO和小分类，不使用省略号、同上或空白代替。
8. 最后输出一行 JSON 元数据：{"used_reference_ids":[...]}，数组只能包含输入 references 中的 source_tech_point_id。
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
const result = psql(`WITH inserted AS (INSERT INTO custom_agents (name, description, avatar, is_builtin, tenant_id, created_by, config, runnable_by_viewer) SELECT ${sql(agentName)}, ${sql(description)}, 'T', false, ${Number(tenantId)}, ${creatorId ? sql(creatorId) : "NULL"}, ${sql(JSON.stringify(config))}::jsonb, true WHERE NOT EXISTS (SELECT 1 FROM custom_agents WHERE tenant_id=${Number(tenantId)} AND name=${sql(agentName)} AND deleted_at IS NULL) RETURNING id) SELECT id FROM inserted UNION ALL SELECT id FROM custom_agents WHERE tenant_id=${Number(tenantId)} AND name=${sql(agentName)} AND deleted_at IS NULL LIMIT 1;`);
if (!result) throw new Error("failed to create TOIN CL tech-point agent");
fs.mkdirSync(path.dirname(outputPath), { recursive: true });
fs.writeFileSync(outputPath, [`XELORA_API_BASE_URL=http://192.168.2.61:18080/api/v1`, `XELORA_CL_TECH_POINT_AGENT_ID=${result}`, `XELORA_MANUAL_ASP_KB_ID=${kbId}`, ""].join("\n"), "utf8");
console.log(`XELORA_CL_TECH_POINT_AGENT_ID=${result}`);
console.log(`XELORA_MANUAL_ASP_KB_ID=${kbId}`);
console.log(`Wrote ${outputPath}`);
