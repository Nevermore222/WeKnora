#!/usr/bin/env node
import { execFileSync } from "node:child_process";
import fs from "node:fs";
import path from "node:path";

const repoRoot = process.cwd();
const sourceAgentPath = path.resolve(
  repoRoot,
  "..",
  "n8n",
  "检证用例代码",
  "COBOL用例代码生成器-SR6.yml",
);
const knowledgeBaseName = "Manual_ASP";
const knowledgeTagName = "ASP_COBOL";
const agentName = "COBOL检证用例代码生成智能体-SR6";
const agentDescription =
  "根据检证用例二维表生成完整 CL、COBOL、PF 三文件的 SR6 智能体；输出规范完全由智能体系统提示词约束。";

function extractDoubleQuotedYamlScalar(source, key, nextKey) {
  const pattern = new RegExp(`  ${key}: "([\\s\\S]*?)"\\r?\\n  ${nextKey}:`);
  const match = source.match(pattern);
  if (!match) throw new Error(`Unable to locate YAML scalar: ${key}`);
  const joined = match[1].replace(/\\\r?\n\s*(?:\\ ?)?/g, "");
  return JSON.parse(`"${joined}"`);
}

const sourceAgentYaml = fs.readFileSync(sourceAgentPath, "utf8");
function normalizeLegacyCommentExamples(prompt) {
  return String(prompt || "").replace(/^ {5}(?=\*)/gm, "      ");
}

const sourcePrompt = normalizeLegacyCommentExamples(
  extractDoubleQuotedYamlScalar(sourceAgentYaml, "pre_prompt", "prompt_type"),
);
for (const marker of ["DLTFILE", "WORKING-STORAGE", "SUB_CATEGORY", "PACKED-DECIMAL", "DISPLAY"]) {
  if (!sourcePrompt.includes(marker)) throw new Error(`SR6 source prompt is missing marker: ${marker}`);
}

const xeloraContract = `

# Xelora execution contract

SR6_PROMPT_SOURCE:
- The complete source contract above is imported from COBOL用例代码生成器-SR6.yml and remains authoritative.
- Runtime values are provided in the user query as COMMAND_NAME, PGM_BASE_CODE, PGM_FILE_SEQ, BATCH_INDEX, and TEST_CASES_TABLE.
- Replace the corresponding {{...}} placeholders with those runtime values before generating the answer.

MANUAL_ASP_SCOPE:
- Use Manual_ASP as the only knowledge base.
- For COBOL tasks, use only the ASP_COBOL documents supplied in knowledge_ids by the workflow.
- The verification-case table is the authoritative source for each case number, tech point number, sub-category, type, input, and expected result.

OUTPUT_LANGUAGE_JAPANESE:
- Generated source comments and explanatory text must be Japanese. Do not output Chinese source comments.
- COBOL identifiers, CL tokens, PF tokens, literals, and values remain unchanged where required.

COMMENT_INDICATOR_COLUMN_7_OVERRIDE:
- This rule overrides every historical example in the imported prompt: each physical COBOL comment line must contain exactly six spaces in columns 1 through 6 and the comment indicator * in column 7.
- A line beginning with five spaces followed by * is always invalid. Never output that form.
- Before returning, scan the complete CBL source line by line. For every border, TP_NO, SUB_CATEGORY, TC, AUTHOR, section heading, and explanatory comment, verify that line.indexOf('*') equals 6.
- Comment borders must be exactly six spaces followed by 44 asterisks. Continuation comment text also begins with exactly six spaces followed by *.

COBOL_PHYSICAL_LINE_WIDTH_60_FINAL_OVERRIDE:
- This is the final and highest-priority width rule. Ignore and override every earlier 50-column comment rule and every earlier 72-column COBOL code rule in the imported prompt.
- In .COBOL and .CBL output, every physical line, including comments and non-comment code, must be at most 60 display columns from column 1 through the final character.
- Count ASCII characters, digits, and spaces as one display column. Count Japanese and other full-width characters as two display columns.
- Wrap long Japanese comments into multiple column-7 comment lines. Split long COBOL declarations and statements at valid fixed-format continuation points.
- For example, never emit the over-width declaration "03 RESULT3 PIC S9(10)V9(8) PACKED-DECIMAL." on one physical line. Put the PIC clause and PACKED-DECIMAL clause on separate valid lines when needed.
- This override applies only to .COBOL and .CBL. It does not change the imported CL or PF line-width rules.

LEVEL_NUMBER_SEPARATOR_FINAL_OVERRIDE:
- Every COBOL data-description level number must be followed by at least one ASCII space before its data name or special entry.
- Forms such as 03RESULT4 and 01NAME are always invalid. Emit 03 RESULT4 and 01 NAME with fixed-format indentation.
- Before returning, scan every data-description line and correct any level number that touches the following token.

OCCURS_ACTUAL_STRUCTURE_FINAL_OVERRIDE:
- Level 01 is a root group and must never carry OCCURS. Never append OCCURS as a continuation of a level-01 elementary PIC item.
- Do not force one universal 01/05/10 hierarchy. Derive levels, names, occurrence counts, dimensions, PIC clauses, signs, decimal positions, and USAGE from the current verification row's SUB_CATEGORY, parameter configuration, input data, and expected result.
- OCCURS may be on a subordinate group item, such as "02 WK-GOKEI-TBL OCCURS 4.", or on a subordinate elementary PIC item, such as "05 WK-GOKEI-T3 PIC 9(04) OCCURS 29.", when that is the actual required structure.
- Nested OCCURS is allowed. Preserve the required 01/02/03/05 or other valid subordinate hierarchy instead of inventing convenience tables.
- Definition names and Procedure Division indexed references must match exactly. Use the actual typed elementary item and the correct number of subscripts.
- Apply PACKED-DECIMAL, COMP, signed PIC, and decimal PIC independently to each typed elementary item according to the current verification row.

DIRECT_MOVE_TO_TYPED_RESULT_FIELD:
- Write observable results by direct MOVE to the matching RESULT1 through RESULT6 field. Never use DISPLAY or STRING into a shared result buffer.

NO_WORKFLOW_SEMANTIC_VALIDATION:
- n8n only extracts and stores the three file sections. Therefore you must perform the complete semantic and formatting self-review defined above before returning.
- Return all three complete file sections. Do not omit code, use placeholders, or explain that validation should happen downstream.

TECH_POINT_SEMANTIC_EXECUTION_FINAL_OVERRIDE:
- TP_NO is traceability metadata only. Printing the correct TP_NO or SUB_CATEGORY in comments is never evidence that the technical point is covered.
- The local coverage tool parses the executable .COBOL/.CBL source. For every TEST_CASES_TABLE row, the generated DATA DIVISION declarations, PROCEDURE DIVISION statement, concrete inputs, and observable result must implement the complete SUB_CATEGORY semantics.
- Decompose and preserve the receiving-item type, every operand type, operator sequence, parentheses, literal kind, Round有/無, SIZE有/無, and ON SIZE ERROR behavior. Never simplify a mixed or packed-decimal expression into a convenient DISPLAY-only expression.
- COMP-3 requires an actual packed-decimal declaration and use in the tested expression. S9 and S9V9 require signed declarations; 9 and 9V9 require unsigned declarations with the correct decimal scale. ROUNDED and ON SIZE ERROR must appear exactly when required and must affect an observable expected result or branch.
- Do not count comments, variable names, copied source text, or dead/unreached statements as implementation. Each case must be reached by the generated driver and execute the statement corresponding to that row.
- Before returning, perform an internal semantic audit row by row: after ignoring TP_NO, SUB_CATEGORY, and all comments, the remaining executable source must still uniquely demonstrate every required feature. Correct any row that fails this audit.

PROJECT_SCOPED_SOURCE_CONTEXT:
- TECH_POINT_CONTEXT contains project-scoped blocks keyed by 技術点NO.
- Match each verification-case row only to the block with the same 技術点NO.

REAL_DECLARATION_PRESERVATION:
- When a block contains 変数宣言, preserve its data names, levels, PIC clauses, USAGE, sign, scale, group hierarchy, and subordinate items in generated COBOL.
- Preserve the associated SOURCE_CONTENT statement semantics.
- Do not replace real declarations with generic working-storage declarations.

NO_CROSS_TECH_POINT_SOURCE_REUSE:
- Never use one block's 変数宣言 or SOURCE_CONTENT to implement another 技術点NO.
- parameter_config remains the immediate case specification; TECH_POINT_CONTEXT is the factual source used to confirm it.
`;

for (const marker of [
  "COBOL_PHYSICAL_LINE_WIDTH_60_FINAL_OVERRIDE",
  "LEVEL_NUMBER_SEPARATOR_FINAL_OVERRIDE",
  "OCCURS_ACTUAL_STRUCTURE_FINAL_OVERRIDE",
	"TECH_POINT_SEMANTIC_EXECUTION_FINAL_OVERRIDE",
]) {
  if (!xeloraContract.includes(marker)) throw new Error(`Missing final contract marker: ${marker}`);
}

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

const kbResult = runPsql(`
SELECT id || '|' || tenant_id || '|' || COALESCE(summary_model_id, '') || '|' || COALESCE(creator_id, '')
FROM knowledge_bases
WHERE deleted_at IS NULL AND name = ${sqlLiteral(knowledgeBaseName)}
ORDER BY updated_at DESC
LIMIT 1;
`);
if (!kbResult) throw new Error(`${knowledgeBaseName} knowledge base was not found.`);

const [knowledgeBaseId, tenantId, modelId, creatorId] = kbResult.split("|");
if (!modelId) throw new Error(`${knowledgeBaseName} has no summary_model_id.`);

const knowledgeCount = Number(runPsql(`
SELECT COUNT(*)
FROM knowledges k
JOIN knowledge_tags t ON t.id = k.tag_id
WHERE k.deleted_at IS NULL
  AND t.deleted_at IS NULL
  AND k.knowledge_base_id = ${sqlLiteral(knowledgeBaseId)}
  AND t.knowledge_base_id = ${sqlLiteral(knowledgeBaseId)}
  AND t.name = ${sqlLiteral(knowledgeTagName)};
`));
if (!knowledgeCount) throw new Error(`No documents found under Manual_ASP tag ${knowledgeTagName}.`);

const rerankModelId = runPsql(`
SELECT id
FROM models
WHERE deleted_at IS NULL
  AND tenant_id = ${Number(tenantId)}
  AND type = 'Rerank'
  AND status = 'active'
ORDER BY updated_at DESC, created_at DESC
LIMIT 1;
`);
if (!rerankModelId) throw new Error(`No active Rerank model found for tenant ${tenantId}.`);

const agentConfig = {
  agent_mode: "smart-reasoning",
  agent_type: "rag-qa",
  system_prompt: `${sourcePrompt}${xeloraContract}`,
  context_template: "",
  model_id: modelId,
  rerank_model_id: rerankModelId,
  temperature: 0.1,
  max_completion_tokens: 30000,
  thinking: false,
  max_iterations: 10,
  llm_call_timeout: 600,
  allowed_tools: ["knowledge_search", "grep_chunks", "list_knowledge_chunks"],
  mcp_selection_mode: "none",
  mcp_services: [],
  skills_selection_mode: "none",
  selected_skills: [],
  kb_selection_mode: "selected",
  knowledge_bases: [knowledgeBaseId],
  retrieve_kb_only_when_mentioned: false,
  retain_retrieval_history: false,
  image_upload_enabled: false,
  audio_upload_enabled: false,
  supported_file_types: [],
  data_analysis_enabled: false,
  faq_priority_enabled: false,
  web_search_enabled: false,
  web_fetch_enabled: false,
  multi_turn_enabled: false,
  history_turns: 1,
  embedding_top_k: 12,
  keyword_threshold: 0.2,
  vector_threshold: 0.35,
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
    "二维表から SR6 形式の CL、COBOL、PF を一式生成してください",
    "COMPUTE 検証ケースから固定形式 COBOL ソースを生成してください",
  ],
};

const createdBySql = creatorId ? sqlLiteral(creatorId) : "NULL";
const configSql = sqlLiteral(JSON.stringify(agentConfig));
const agentId = runPsql(`
WITH existing AS (
  SELECT id FROM custom_agents
  WHERE deleted_at IS NULL
    AND tenant_id = ${Number(tenantId)}
    AND name = ${sqlLiteral(agentName)}
  LIMIT 1
), updated AS (
  UPDATE custom_agents
  SET description = ${sqlLiteral(agentDescription)},
      avatar = 'S',
      config = ${configSql}::jsonb,
      created_by = COALESCE(created_by, ${createdBySql}),
      runnable_by_viewer = true,
      updated_at = CURRENT_TIMESTAMP
  WHERE id IN (SELECT id FROM existing)
  RETURNING id
), inserted AS (
  INSERT INTO custom_agents (name, description, avatar, is_builtin, tenant_id, created_by, config, runnable_by_viewer)
  SELECT ${sqlLiteral(agentName)}, ${sqlLiteral(agentDescription)}, 'S', false,
         ${Number(tenantId)}, ${createdBySql}, ${configSql}::jsonb, true
  WHERE NOT EXISTS (SELECT 1 FROM existing)
  RETURNING id
)
SELECT id FROM updated
UNION ALL
SELECT id FROM inserted
LIMIT 1;
`);
if (!agentId) throw new Error("Failed to upsert Xelora COBOL verification-code agent.");

console.log(`Agent: ${agentName}`);
console.log(`XELORA_COBOL_VERIFICATION_CODE_AGENT_ID=${agentId}`);
console.log(`XELORA_MANUAL_ASP_KB_ID=${knowledgeBaseId}`);
console.log(`XELORA_MANUAL_ASP_TAG=${knowledgeTagName}`);
console.log(`XELORA_MANUAL_ASP_DOCUMENT_COUNT=${knowledgeCount}`);
console.log(`SR6_SYSTEM_PROMPT_LENGTH=${agentConfig.system_prompt.length}`);
console.log(`model_id=${modelId}`);
console.log(`rerank_model_id=${rerankModelId}`);
