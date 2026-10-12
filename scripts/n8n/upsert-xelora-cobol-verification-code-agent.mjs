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
const fastSkillPath = path.resolve(repoRoot, "..", "..", "PI", ".pi", "skills", "cobol-case-pipeline-fast", "SKILL.md");
const fastContractPath = path.resolve(repoRoot, "..", "..", "PI", ".pi", "skills", "cobol-case-pipeline-fast", "references", "sr6-code-contract.md");
if (!fs.existsSync(fastSkillPath) || !fs.existsSync(fastContractPath)) throw new Error("cobol-case-pipeline-fast references are missing.");
const fastSkillPrompt = fs.readFileSync(fastSkillPath, "utf8");
const fastCodeContract = fs.readFileSync(fastContractPath, "utf8");
const moveMasterContract = `

MOVE_MASTER_EXECUTION_ADAPTER:
- This Xelora agent is called by n8n with GENERATION_MODE: MOVE_MASTER and a TEST_CASES_TABLE containing up to five rows from one TEST_CATEGORY. Apply the TOIN SR6 fast-pipeline template and static contract above to every row. Do not use the old HANLE/LINKAGE/RESULT-RMSG1 template and do not invent a simplified template.
- File naming is mandatory: PROGRAM_BASE_CODE is BO + COMMAND_ID and FILE_GROUP_SEQ is the supplied two-character uppercase base36 sequence (00..ZZ, 36x36=1296 values). Emit exactly BO{COMMAND_ID}C{FILE_GROUP_SEQ}.CL, BO{COMMAND_ID}B{FILE_GROUP_SEQ}.COBOL, and BO{COMMAND_ID}F{FILE_GROUP_SEQ}.SF. Never emit MOVE_TC/MOVE_TB/MOVE_TF filenames. PROGRAM-ID is BO{COMMAND_ID}B{FILE_GROUP_SEQ}; SELECT/physical result file uses BO{COMMAND_ID}F{FILE_GROUP_SEQ}; the canonical SF header is /FILE BO{COMMAND_ID}F{FILE_GROUP_SEQ}.
- The CL is the standard SR6 driver: first line PGM; VAR    ?PGMEC,INTEGER; DLTFILE FILE-BO{COMMAND_ID}F{FILE_GROUP_SEQ}.SRTEST; CRTFILE FILE-BO{COMMAND_ID}F{FILE_GROUP_SEQ}.SRTEST,PUBAUT-@ALL,ORG-@SF,BF-2,RL-220, then the comma-wrapped RECN-100,ERECN-10 line; OVRF   FILE-BO{COMMAND_ID}F{FILE_GROUP_SEQ},TOFILE-BO{COMMAND_ID}F{FILE_GROUP_SEQ}.SRTEST; CALL   PGM-BO{COMMAND_ID}B{FILE_GROUP_SEQ}.SRTEST; ?PGMEC := @PGMEC; RETURN PGMEC-?PGMEC. Use exact spacing and single-quoted CL literals from the SR6 contract.
- The COBOL must be the fast SR6 one-driver form. Use this exact procedure skeleton and do not rearrange it: PROCEDURE DIVISION. then PERFORM INIT-RTN., PERFORM MAIN-RTN., PERFORM END-RTN.; INIT-RTN SECTION.; MAIN-RTN SECTION. containing only PERFORM MOVE-METHOD-0-RTN.; one MOVE-METHOD-0-RTN SECTION. case; one MOVE-METHOD-0-EXIT. EXIT.; and END-RTN SECTION. Never put PERFORM END-RTN inside MAIN-RTN.
- The COBOL identification/environment skeleton must contain IDENTIFICATION DIVISION., PROGRAM-ID. BO{COMMAND_ID}B{FILE_GROUP_SEQ}., AUTHOR. SR., ENVIRONMENT DIVISION., CONFIGURATION SECTION., SOURCE-COMPUTER. FACOM., OBJECT-COMPUTER. FACOM., INPUT-OUTPUT SECTION., and FILE-CONTROL. in that order. Use 01 STA01 PIC X(2)., not PIC XX.
- The COBOL must be the fast SR6 one-driver form: all divisions, FILE-CONTROL, SELECT LOGFILE ASSIGN TO DA-BO{COMMAND_ID}F{FILE_GROUP_SEQ} FILE STATUS IS STA01, LOGFILE-REC with PGMID X(10), CASENO X(10), RESULT1 X(20), RESULT2 S9(10)V9(8), RESULT3 S9(10)V9(8) PACKED-DECIMAL, RESULT4 S9(18) COMP, RESULT5 N(12), RESULT6 X(120), WORKING-STORAGE declarations required by the current MOVE rows, PROCEDURE DIVISION with PERFORM INIT-RTN, MAIN-RTN, END-RTN, and no LINKAGE SECTION.
- INIT-RTN contains the level-01 INITIALIZE statements required by the SR6 contract and opens OUTPUT LOGFILE before the first WRITE. Do not put INITIALIZE in a case section. MAIN-RTN only PERFORMs MOVE-METHOD-0-RTN (or the exact command-name equivalent) and each case has exactly one MOVE-METHOD-n-EXIT paragraph followed by EXIT.
- Each MOVE case must have the complete SR6 50-column header with exactly these metadata labels: TP_NO, SUB_CATEGORY, PGMID, SOURCE_ID, STRUCTURE_ID, CASE_TYPE, DUPLICATE_SOURCE_IDS, plus a non-empty case purpose. Copy SUB_CATEGORY verbatim from the input row. Preserve every row's source/target declaration, input_data, expected_result, and MOVE semantics. The MOVE statement and operands are executable COBOL, not comments. For VAR_TYPE_GP(String), assign only the 01-level group variable P01 with MOVE HIGH-VALUES TO P01.; never expand to P0201/P0202 or other 02-level fields. Before output, inspect RESULT field lengths and store the value in the largest defined field RESULT6 using MOVE P01 TO RESULT6.; never store it in RESULT1-RESULT5 for this case. For other types, move the typed result directly to the matching RESULT field according to the SR6 type mapping, then WRITE LOGFILE-REC. Do not use DISPLAY, STRING summaries, or MOVE "OK".
- The canonical SF is the fast SR6 PF layout: /FILE BO{COMMAND_ID}F{FILE_GROUP_SEQ}, a line beginning with one space then S0220002, /RECORD RCD, F1000, one-space-prefixed F1110 through F1180 with exact type codes from sr6-code-contract.md, and /END. Do not emit S0042001, four-field SF stubs, DDS, or *SRTEST in the canonical SF.
- COBOL fixed-format prefix is literal, not descriptive. Copy these line starts exactly, including all leading ASCII spaces: IDENTIFICATION DIVISION., PROGRAM-ID. MOVE_TBxxx., AUTHOR. SR., ENVIRONMENT DIVISION., CONFIGURATION SECTION., SOURCE-COMPUTER. FACOM., OBJECT-COMPUTER. FACOM., INPUT-OUTPUT SECTION., FILE-CONTROL., DATA DIVISION., FILE SECTION., FD  LOGFILE., 01  LOGFILE-REC., WORKING-STORAGE SECTION., PROCEDURE DIVISION., INIT-RTN SECTION., MAIN-RTN SECTION., MOVE-METHOD-0-RTN SECTION., MOVE-METHOD-0-EXIT., END-RTN SECTION. The first seven spaces are required Area A indentation; never let Markdown rendering or sentence generation remove them.
- Before returning COBOL, perform this mechanical check: every non-empty non-comment line starts with seven ASCII spaces; every comment line starts with six ASCII spaces followed by *; if any line fails, regenerate the complete COBOL section before answering. Do not return a prose explanation of the check.
- Return only these three complete headings and source sections: ## BO{COMMAND_ID}C{FILE_GROUP_SEQ}.CL, ## BO{COMMAND_ID}B{FILE_GROUP_SEQ}.COBOL, ## BO{COMMAND_ID}F{FILE_GROUP_SEQ}.SF. Include every case in TEST_CASES_TABLE exactly once. Never return a JSON wrapper, prose, placeholders, old MOVE_TC/MOVE_TB/MOVE_TF headings, or an extra case.
`;


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

MOVE_MASTER_FAST_SR6_TEMPLATE:
- GENERATION_MODE MOVE_MASTER uses the TOIN SR6 fast-pipeline code template below. Do not use the historical HANLE/LINKAGE/RESULT-RMSG1 template.
- The MOVE adapter changes only the external names: MOVE_TCxxx.CL, MOVE_TBxxx.COBOL, MOVE_TFxxx.SF. For sequence xxx, the COBOL PROGRAM-ID is MOVE_TBxxx, the CL CALL target is PGM-MOVE_TBxxx.SRTEST, the logical result file is MOVE_TFxxx, the physical result file is MOVE_TFxxx.SRTEST, the COBOL SELECT target is DA-MOVE_TFxxx, and the canonical SF first line is /FILE MOVE_TFxxx.
- CL must start with PGM and use exactly the SR6 driver shape: VAR ?PGMEC,INTEGER; DLTFILE FILE-MOVE_TFxxx.SRTEST; CRTFILE FILE-MOVE_TFxxx.SRTEST,PUBAUT-@ALL,ORG-@SF,BF-2,RL-220, followed by RECN-100,ERECN-10; OVRF FILE-MOVE_TFxxx,TOFILE-MOVE_TFxxx.SRTEST; CALL PGM-MOVE_TBxxx.SRTEST; ?PGMEC := @PGMEC; RETURN PGMEC-?PGMEC. Use single-quoted CL literals and no IBM i PARM/FILE/TYPE forms.
- COBOL must contain all divisions, FILE-CONTROL with SELECT LOGFILE ASSIGN TO DA-MOVE_TFxxx FILE STATUS IS STA01, 01 LOGFILE-REC with exactly the SR6 result layout PGMID, CASENO, RESULT1 X(20), RESULT2 S9(10)V9(8), RESULT3 S9(10)V9(8) PACKED-DECIMAL, RESULT4 S9(18) COMP, RESULT5 N(12), RESULT6 X(120), and the required WORKING-STORAGE declarations.
- COBOL must use INIT-RTN, MAIN-RTN, and END-RTN. Put one INITIALIZE statement for each level-01 item in INIT-RTN before OPEN OUTPUT LOGFILE. MAIN-RTN only PERFORMs CMD-METHOD-n-RTN sections. Each case section has one CMD-METHOD-n-EXIT paragraph and EXIT. Never add LINKAGE SECTION, parameters, RESULT/RMSG1 fields, RESULT-only layouts, or a generated simplified driver.
- Each MOVE case must include the complete 50-column SR6 header: TC number and purpose, TP_NO, SUB_CATEGORY copied verbatim, PGMID, SOURCE_ID, STRUCTURE_ID, CASE_TYPE, DUPLICATE_SOURCE_IDS. Copy the MOVE operation as executable COBOL, preserve source declaration/type semantics, perform the requested MOVE, move the typed result directly to RESULT1 through RESULT6 according to the SR6 type mapping, and finish with WRITE LOGFILE-REC. Do not use DISPLAY, STRING summaries, or meaningless MOVE "OK".
- The canonical MOVE SF must be the SR6 PF layout, not a DDS or four-field stub: /FILE MOVE_TFxxx, S0220002, /RECORD RCD, F1000, F1110 X 0010, F1120 X 0010, F1130 X 0020, F1140 US 001808, F1150 PS 001808, F1160 BS 0018, F1170 N 0012, F1180 X 0120, /END. Keep the leading spaces exactly as the SR6 reference.
- Fixed-format constraints: comment indicator * in COBOL column 7, comment borders exactly 50 display columns, comments <=50, source/code lines <=72 display columns, no Markdown fences, no tabs, no .SRTEST in PROGRAM-ID, no CASE INITIALIZE, and no extra LINKAGE section.
- Return exactly the three headings ## MOVE_TCxxx.CL, ## MOVE_TBxxx.COBOL, ## MOVE_TFxxx.SF with complete source only. Do not explain the output.

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
  system_prompt: `${fastSkillPrompt}\n\n${fastCodeContract}\n${moveMasterContract}`,
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
