#!/usr/bin/env node
import fs from "node:fs";
import path from "node:path";

const repoRoot = process.cwd();
const agentScriptPath = path.join(
  repoRoot,
  "scripts",
  "n8n",
  "upsert-xelora-cobol-verification-code-agent.mjs",
);
const workflowPath = path.resolve(
  repoRoot,
  "..",
  "n8n",
  "检证用例代码",
  "Xelora版COBOL检证用例代码生成工作流_1.0.0.json",
);

const agentScript = fs.readFileSync(agentScriptPath, "utf8");
const workflowText = fs.readFileSync(workflowPath, "utf8");
const workflow = JSON.parse(workflowText);

function requireNode(name) {
  const node = workflow.nodes.find((entry) => entry.name === name);
  if (!node) throw new Error(`Generated workflow is missing node: ${name}`);
  return node;
}

for (const node of workflow.nodes.filter((entry) => entry.type === "n8n-nodes-base.code")) {
  const code = node.parameters?.jsCode || "";
  try {
    new Function(code);
  } catch (error) {
    throw new Error(`Code node has invalid JavaScript (${node.name}): ${error.message}`);
  }
}

for (const name of [
  "AI批次汇总并校验",
  "保存CL文件到服务器",
  "保存COBOL文件到服务器",
  "保存PF文件到服务器",
  "调用文件转码服务",
  "创建Xelora会话",
  "提取Xelora会话ID",
  "调用Xelora代码生成智能体",
  "处理Xelora智能体响应",
]) {
  requireNode(name);
}

const createSessionNode = requireNode("创建Xelora会话");
const callAgentNode = requireNode("调用Xelora代码生成智能体");
const normalizeFilesNode = requireNode("调用文件转码服务");
const collectNormalizePathNode = requireNode("收集转码路径");
const normalizeServiceUrl = normalizeFilesNode.parameters?.url;
if (normalizeServiceUrl !== "=http://192.168.8.247:8765/normalize") {
  throw new Error(`File normalize service must use the fixed workflow URL: ${normalizeServiceUrl}`);
}
if (normalizeServiceUrl.includes("$env")) {
  throw new Error("File normalize service URL must not access n8n environment variables.");
}
const collectNormalizePathCode = collectNormalizePathNode.parameters?.jsCode || "";
for (const forbidden of ["process.env", "FILE_NORMALIZE_"]) {
  if (collectNormalizePathCode.includes(forbidden)) {
    throw new Error(`File normalize path collector must not depend on environment variables: ${forbidden}`);
  }
}
if (workflowText.includes("FILE_NORMALIZE_")) {
  throw new Error("Generated workflow must not contain obsolete file-normalize environment variable guidance.");
}

const webhookParametersNode = requireNode("解析Webhook参数");
const batchSizeAssignment = webhookParametersNode.parameters?.assignments?.assignments?.find(
  (entry) => entry.name === "code_gen_batch_size",
);
if (!String(batchSizeAssignment?.value || "").includes("|| 14")) {
  throw new Error("The 1.2.4 default code generation batch size must be 14.");
}

const aggregateNode = requireNode("AI批次汇总并校验");
const aggregateCode = aggregateNode.parameters?.jsCode || "";
for (const marker of [
  "buildUnifiedClCode",
  "replaceOpenMode",
  "replaceLogfileTarget",
  "shared_first_batch",
  "effective_cl_code",
  "effective_cobol_code",
  "effective_pf_code",
]) {
  if (!aggregateCode.includes(marker)) {
    throw new Error(`The 1.2.4 multi-batch aggregator is missing marker: ${marker}`);
  }
}
for (const forbidden of [
  "normalizeResultMoves",
  "buildVariableResultFieldMap",
  "detectResultFieldByDefinition",
]) {
  if (aggregateCode.includes(forbidden)) {
    throw new Error(`Workflow aggregation must not semantically rewrite generated COBOL: ${forbidden}`);
  }
}

for (const name of ["准备CL文件信息", "准备COBOL文件信息", "准备PF文件信息"]) {
  const code = requireNode(name).parameters?.jsCode || "";
  if (!code.includes("effective_")) {
    throw new Error(`Shared multi-batch artifact data is not consumed by node: ${name}`);
  }
}
for (const name of ["保存CL文件到服务器", "保存PF文件到服务器"]) {
  const command = requireNode(name).parameters?.command || "";
  if (!command.includes("shared_first_batch") || !command.includes("跳过非首批")) {
    throw new Error(`Shared CL/PF persistence guard is missing from node: ${name}`);
  }
}
for (const marker of ["C01.CL", "B' + seq", "F01.SF", "shared_first_batch"]) {
  if (!collectNormalizePathCode.includes(marker)) {
    throw new Error(`Normalization manifest does not describe actual shared artifacts: ${marker}`);
  }
}

const aggregateRunner = new Function("$input", "$", aggregateCode);
const aggregateResult = aggregateRunner({
  all: () => [
    {
      json: {
        status: "success",
        batch_index: 1,
        batch_total: 2,
        pgm_file_seq: "01",
        pgm_base_code: "BT999",
        cl_code: "PGM\n         CALL   PGM-BT999B01.SRTEST\n         ?PGMEC := @PGMEC\n         RETURN PGMEC-?PGMEC",
        cobol_code: "       SELECT LOGFILE ASSIGN TO DA-BT999F01.\n           OPEN  OUTPUT  LOGFILE.\n       01 WS-AMOUNT PIC S9(4)V9(2).\n           MOVE WS-AMOUNT TO RESULT1.",
        pf_code: "/FILE BT999F01\n/END",
      },
    },
    {
      json: {
        status: "success",
        batch_index: 2,
        batch_total: 2,
        pgm_file_seq: "02",
        pgm_base_code: "BT999",
        cl_code: "PGM\n         CALL   PGM-BT999B02.SRTEST\n         ?PGMEC := @PGMEC\n         RETURN PGMEC-?PGMEC",
        cobol_code: "       SELECT LOGFILE ASSIGN TO DA-BT999F02.\n           OPEN  OUTPUT  LOGFILE.\n       01 WS-AMOUNT PIC S9(4)V9(2).\n           MOVE WS-AMOUNT TO RESULT1.",
        pf_code: "/FILE BT999F02\n/END",
      },
    },
  ],
}, () => ({ first: () => ({ json: {} }) }));
const aggregatedPayloads = aggregateResult?.[0]?.json?._batch_payloads;
if (!Array.isArray(aggregatedPayloads) || aggregatedPayloads.length !== 2) {
  throw new Error("Multi-batch aggregation did not preserve both batches.");
}
if (!aggregatedPayloads[0].effective_cl_code.includes("BT999B01") ||
    !aggregatedPayloads[0].effective_cl_code.includes("BT999B02")) {
  throw new Error("Shared C01 content must call every Bxx batch program.");
}
if (!aggregatedPayloads[1].effective_cobol_code.includes("DA-BT999F01") ||
    !/OPEN\s+EXTEND\s+LOGFILE/i.test(aggregatedPayloads[1].effective_cobol_code)) {
  throw new Error("The second COBOL batch must append to shared F01 with OPEN EXTEND.");
}
if (!aggregatedPayloads[1].effective_cobol_code.includes("MOVE WS-AMOUNT TO RESULT1.")) {
  throw new Error("Workflow aggregation must preserve agent source semantics without rewriting RESULT fields.");
}
const expectedSessionBatching = createSessionNode.parameters?.options?.batching?.batch;
if (expectedSessionBatching?.batchSize !== 5 || expectedSessionBatching?.batchInterval !== 300) {
  throw new Error("Session creation must be throttled in batches of 5 with a 300ms interval.");
}
const expectedAgentBatching = callAgentNode.parameters?.options?.batching?.batch;
if (expectedAgentBatching?.batchSize !== 2 || expectedAgentBatching?.batchInterval !== 1000) {
  throw new Error("Agent calls must be throttled in batches of 2 with a 1000ms interval.");
}

const extractSessionNode = requireNode("提取Xelora会话ID");
const extractSessionRunner = new Function("$input", "$", extractSessionNode.parameters.jsCode);
const sessionRequests = [1, 2, 3].map((batchIndex) => ({
  json: {
    batch_index: batchIndex,
    batch_total: 3,
    xelora_api_base_url: "http://example.test/api/v1",
    pgm_base_code: "BT999",
    pgm_file_seq: "01",
    command_master_id: 999,
    command: "COMPUTE",
    version: 1,
  },
}));
const partialOutOfOrderResponses = [
  {
    json: { data: { id: "session-3", title: "COBOL SR6 code COMPUTE batch 3/3" }, success: true },
    pairedItem: { item: 2 },
  },
  {
    json: { data: { id: "session-1", title: "COBOL SR6 code COMPUTE batch 1/3" }, success: true },
    pairedItem: { item: 0 },
  },
];
const correlatedSessions = extractSessionRunner(
  { all: () => partialOutOfOrderResponses },
  () => ({ all: () => sessionRequests }),
);
if (!Array.isArray(correlatedSessions) || correlatedSessions.length !== 2) {
  throw new Error("Session extractor must return the two successful, correlated session batches without throwing.");
}
const sessionByBatch = new Map(correlatedSessions.map((item) => [item.json.batch_index, item.json]));
if (sessionByBatch.get(1)?.xelora_session_id !== "session-1" || sessionByBatch.get(3)?.xelora_session_id !== "session-3") {
  throw new Error("Session extractor did not correlate out-of-order responses by their original batch.");
}

for (const forbidden of [
  "Dify",
  "调用Dify智能体",
  "/v1/chat-messages",
  "app-94Nmqnx3zHeqRbYj5ToZuxIJ",
]) {
  if (workflowText.includes(forbidden)) throw new Error(`Generated workflow still contains Dify marker: ${forbidden}`);
}

for (const marker of [
  "http://192.168.2.61:18080/api/v1",
  "X-API-Key",
  "Manual_ASP",
  "ASP_COBOL",
  "knowledge_ids",
  "agent-chat/",
]) {
  if (!workflowText.includes(marker)) throw new Error(`Generated workflow is missing Xelora marker: ${marker}`);
}

const responseNode = requireNode("处理Xelora智能体响应");
const responseCode = responseNode.parameters?.jsCode || "";
for (const marker of ["cl_code", "cobol_code", "pf_code", "full_response", "response_type"]) {
  if (!responseCode.includes(marker)) throw new Error(`Response parser is missing structural marker: ${marker}`);
}
for (const forbidden of [
  "validateCobol",
  "validateCodegenContract",
  "PACKED-DECIMAL is required",
  "DISPLAY is forbidden",
  "lineWidthError",
]) {
  if (responseCode.includes(forbidden)) throw new Error(`Workflow must not semantically validate generated code: ${forbidden}`);
}
for (const forbidden of ["fenced[1].trim()", "pfCode.trim()"] ) {
  if (responseCode.includes(forbidden)) {
    throw new Error(`Source extraction must preserve first-line indentation: ${forbidden}`);
  }
}

const sampleAnswer = [
  "## TOINSRC/BT999C01.CL",
  "```cl",
  "PGM",
  "         DLTFILE FILE-BT999F01.SRTEST",
  "         CRTFILE FILE-BT999F01.SRTEST",
  "         CALL PGM-BT999B01.SRTEST",
  "```",
  "## TOINSRC/BT999B01.CBL",
  "```cobol",
  "       IDENTIFICATION DIVISION.",
  "       PROGRAM-ID. BT999B01.",
  "```",
  "## TOINSRC/BT999F01.PF",
  "```",
  "/FILE BT999F01",
  "/END",
  "```",
].join("\n");
const sampleSse = `data: ${JSON.stringify({ response_type: "answer", content: sampleAnswer })}\n` +
  `data: ${JSON.stringify({ response_type: "complete", done: true })}\n`;
const responseRunner = new Function("$input", "$", responseCode);
const sampleMeta = {
  command_master_id: 999,
  command: "COMPUTE",
  version: 1,
  batch_index: 1,
  batch_total: 1,
  cases_in_batch: 1,
  total_cases_all_rows: 1,
  pgm_base_code: "BT999",
  pgm_file_seq: "01",
  test_cases_table: "sample",
  xelora_session_id: "session-1",
};
const parsedFiles = responseRunner(
  { all: () => [{ json: { data: sampleSse } }] },
  () => ({ all: () => [{ json: sampleMeta }] }),
);
if (!Array.isArray(parsedFiles) || parsedFiles.length !== 1) {
  throw new Error("Xelora response parser did not return one batch.");
}
const parsed = parsedFiles[0].json;
if (parsed.status !== "success" || !parsed.cl_code || !parsed.cobol_code || !parsed.pf_code) {
  throw new Error(`Xelora response parser did not extract all three file sections: ${JSON.stringify(parsed)}`);
}
if (!parsed.cobol_code.startsWith("       IDENTIFICATION")) {
  throw new Error(`COBOL first-line indentation was not preserved: ${JSON.stringify(parsed.cobol_code.slice(0, 40))}`);
}

function sampleSseForSession(sessionId) {
  return `data: ${JSON.stringify({ response_type: "answer", session_id: sessionId, content: sampleAnswer })}\n` +
    `data: ${JSON.stringify({ response_type: "complete", session_id: sessionId, done: true })}\n`;
}
const partialAgentResponses = [
  { json: { data: sampleSseForSession("session-3") }, pairedItem: { item: 0 } },
  { json: { data: sampleSseForSession("session-1") }, pairedItem: { item: 1 } },
];
const allBatchResults = responseRunner(
  { all: () => partialAgentResponses },
  (name) => ({
    all: () => name === "提取Xelora会话ID" ? correlatedSessions : sessionRequests,
  }),
);
if (!Array.isArray(allBatchResults) || allBatchResults.length !== 3) {
  throw new Error("Response processor must restore a result record for every originally requested batch.");
}
const resultByBatch = new Map(allBatchResults.map((item) => [item.json.batch_index, item.json]));
if (resultByBatch.get(1)?.status !== "success" || resultByBatch.get(3)?.status !== "success") {
  throw new Error("Response processor did not preserve successful out-of-order agent responses.");
}
if (resultByBatch.get(1)?.response_session_id !== "session-1" || resultByBatch.get(3)?.response_session_id !== "session-3") {
  throw new Error("Response processor must correlate out-of-order agent responses by SSE session_id.");
}
if (resultByBatch.get(2)?.status !== "failed" || !String(resultByBatch.get(2)?.error_message).includes("缺少")) {
  throw new Error("Response processor must emit an explicit failed result for the batch lost before agent invocation.");
}

for (const marker of [
  "COBOL用例代码生成器-SR6.yml",
  "Manual_ASP",
  "ASP_COBOL",
  'skills_selection_mode: "none"',
  "SR6_PROMPT_SOURCE",
  "COBOL_PHYSICAL_LINE_WIDTH_60_FINAL_OVERRIDE",
  "LEVEL_NUMBER_SEPARATOR_FINAL_OVERRIDE",
  "OCCURS_ACTUAL_STRUCTURE_FINAL_OVERRIDE",
  "DIRECT_MOVE_TO_TYPED_RESULT_FIELD",
  "NO_WORKFLOW_SEMANTIC_VALIDATION",
	"TECH_POINT_SEMANTIC_EXECUTION_FINAL_OVERRIDE",
  "normalizeLegacyCommentExamples",
  "COMMENT_INDICATOR_COLUMN_7_OVERRIDE",
]) {
  if (!agentScript.includes(marker)) throw new Error(`Agent upsert script is missing contract marker: ${marker}`);
}
for (const marker of [
  "PROJECT_SCOPED_SOURCE_CONTEXT",
  "REAL_DECLARATION_PRESERVATION",
  "NO_CROSS_TECH_POINT_SOURCE_REUSE",
]) {
  if (!agentScript.includes(marker)) throw new Error(`Code agent is missing: ${marker}`);
}
if (workflowText.includes("event.done === true")) {
	throw new Error("Generated workflow must not treat every done event as a successful SSE completion.");
}
for (const obsoleteMarker of ["COMMENT_DISPLAY_WIDTH_50:", "CODE_DISPLAY_WIDTH_72:"]) {
  if (agentScript.includes(obsoleteMarker)) {
    throw new Error(`Agent upsert script still defines obsolete contract marker: ${obsoleteMarker}`);
  }
}

console.log("Xelora COBOL verification-code workflow validation passed.");
