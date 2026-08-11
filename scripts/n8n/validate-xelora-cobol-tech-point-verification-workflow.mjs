#!/usr/bin/env node
import fs from "node:fs";
import path from "node:path";

const repoRoot = process.cwd();
const agentScriptPath = path.join(repoRoot, "scripts", "n8n", "upsert-xelora-cobol-tech-point-verification-agent.mjs");
const workflowPath = path.resolve(repoRoot, "..", "n8n", "二维表", "Xelora版COBOL技术点检证用例工作流（基于tech_point）_1.0.0.json");

const agentScript = fs.readFileSync(agentScriptPath, "utf8");
const workflow = JSON.parse(fs.readFileSync(workflowPath, "utf8"));
const workflowCode = workflow.nodes
  .filter((node) => node.type === "n8n-nodes-base.code")
  .map((node) => node.parameters?.jsCode || "")
  .join("\n");

for (const node of workflow.nodes.filter((entry) => entry.type === "n8n-nodes-base.code")) {
  const code = node.parameters?.jsCode || "";
  try {
    new Function(code);
  } catch (error) {
    throw new Error(`Code node has invalid JavaScript (${node.name}): ${error.message}`);
  }
}

const requiredAgentMarkers = [
	"SEMANTIC_COVERAGE_NOT_TP_NO_FINAL_CONTRACT",
  "SUB_CATEGORY_TYPE_CONTRACT",
  "CODEGEN_READY_PARAMETER_SETTING",
  "CONCRETE_INPUT_ASSIGNMENTS",
  "VERIFICATION_DESIGN_ASSUMPTION",
  "CANONICAL_COBOL_DECLARATION_ORDER",
  "JAPANESE_ONLY_OUTPUT",
];

const requiredWorkflowMarkers = [
  "SUB_CATEGORY_TYPE_CONTRACT",
  "CODEGEN_READY_PARAMETER_SETTING",
  "CONCRETE_INPUT_ASSIGNMENTS",
  "VERIFICATION_DESIGN_ASSUMPTION",
  "CANONICAL_COBOL_DECLARATION_ORDER",
  "JAPANESE_ONLY_OUTPUT",
];

for (const marker of requiredAgentMarkers) {
  if (!agentScript.includes(marker)) throw new Error(`Agent script is missing contract marker: ${marker}`);
}
for (const marker of [
  "PROJECT_SCOPED_TECH_POINT_INPUT",
  "VARIABLE_DECLARATION_AUTHORITY",
  "NO_CROSS_TECH_POINT_DECLARATION_REUSE",
  "MISSING_VARIABLE_DECLARATION_FALLBACK",
]) {
  if (!agentScript.includes(marker)) {
    throw new Error(`Agent script is missing project-scope marker: ${marker}`);
  }
}
if (agentScript.includes("说明段落可以使用中文")) {
  throw new Error("Agent script still permits Chinese explanatory output.");
}

for (const marker of requiredWorkflowMarkers) {
  if (!workflowCode.includes(marker)) throw new Error(`Generated workflow is missing contract marker: ${marker}`);
}
if (workflowCode.includes("event.done === true")) {
	throw new Error("Generated workflow must not treat every done event as a successful SSE completion.");
}
if (!workflowCode.includes("| 技術点NO | 用例番号 | 小分類 | テスト分類 | テスト目的 | パラメータ設定 | 入力データ | 期待結果 | 優先度 | 備考 |")) {
  throw new Error("Generated workflow does not require the Japanese-only Markdown header.");
}

const parserNode = workflow.nodes.find((node) => node.name === "解析本批检证用例");
if (!parserNode) throw new Error("Generated workflow is missing parser node: 解析本批检证用例");
for (const forbiddenMarker of ["validateCodegenContract", "type_contract_failed"]) {
  if (parserNode.parameters.jsCode.includes(forbiddenMarker)) {
    throw new Error(`Parser must not enforce agent output contract: ${forbiddenMarker}`);
  }
}
if (!parserNode.parameters.jsCode.includes("isValidTestCaseCore")) {
  throw new Error("Parser must retain basic non-empty row parsing before database insertion.");
}

const previouslyRejectedResponse = [
  "### 检证用例表",
  "| 技术点NO | 用例番号 | 小分類 | テスト分類 | テスト目的 | パラメータ設定 | 入力データ | 期待結果 | 優先度 | 備考 |",
  "| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |",
  "| CMPTE0V0000030V0000033V0000050 | TC001 | COMPUTE【(受取り側項目が単一変数(9型,Round無変数(9型,乗算,変数(9V9型,SIZE無))】 | 正常系 | 乗算結果を確認する。 | PIC 9(5) RD1130-JYURYO;<br>PIC 9(3) RD1130-CASESU;<br>PIC 9(3)V9(2) MSHIN-R4020;<br>COMPUTE RD1130-JYURYO = RD1130-CASESU * MSHIN-R4020 | RD1130-CASESU=100<br>MSHIN-R4020=1.50 | RD1130-JYURYO=00150 | P1 | PIC桁数は検証設計上の仮定。 |",
].join("\n");
const runParser = new Function("$input", parserNode.parameters.jsCode);
const parsedRows = runParser({
  first: () => ({
    json: {
      command_master_id: 232,
      full_verification_cases_response: previouslyRejectedResponse,
    },
  }),
});
if (!Array.isArray(parsedRows) || parsedRows.length !== 1 || parsedRows[0].json.case_number !== "TC001") {
  throw new Error("Parser did not accept the previously rejected agent table row for database insertion.");
}
for (const field of ["parameter_config", "input_data", "expected_result", "notes"]) {
  const value = String(parsedRows[0].json[field] || "");
  if (/<br\s*\/?\s*>|&lt;br\s*\/?\s*&gt;/i.test(value)) {
    throw new Error(`Parser left a visible HTML line-break marker in ${field}.`);
  }
}
if (!parsedRows[0].json.parameter_config.includes("\n") || !parsedRows[0].json.input_data.includes("\n")) {
  throw new Error("Parser did not convert agent line-break markers into plain-text newlines.");
}

console.log("COBOL tech-point code-generation contract validation passed.");
