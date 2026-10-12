#!/usr/bin/env node
import fs from "node:fs";
import path from "node:path";

const p = path.resolve(process.cwd(), "..", "n8n", "检证用例代码", "Xelora通用检证用例代码生成工作流_1.0.0.json");
const w = JSON.parse(fs.readFileSync(p, "utf8"));
const text = JSON.stringify(w);
for (const required of [
  "XeloraGenVerCodeA8B247V1", "xelora-verification-code-generic", "verification_case_code_generate",
  "analyzes.cl_verification_cases", "analyzes.cl_verification_cases_master", "TEST_CASES_TABLE",
  "MOVE_MASTER", "test_category", "category_batch_no", "category_total_batches", "verification_cases_code",
  "PROGRAM_BASE_CODE", "FILE_GROUP_SEQ", "PGM_FILE_SEQ", "BO"
]) if (!text.includes(required)) throw new Error(`missing contract: ${required}`);
if (w.active !== false) throw new Error("workflow must be inactive before import");
if (text.includes("192.168.2.61") || text.includes("sk-5NB4aUS")) throw new Error("legacy hard-coded Xelora config remains");
if (!w.nodes.some((n) => n.type === "n8n-nodes-base.webhook") || !w.nodes.some((n) => n.type === "n8n-nodes-base.executeCommand")) throw new Error("missing core nodes");
for (const n of w.nodes.filter((entry) => entry.type === "n8n-nodes-base.code")) {
  try { new Function(n.parameters.jsCode); } catch (error) { throw new Error(`invalid Code node ${n.name}: ${error.message}`); }
}
const names = new Set(w.nodes.map((n) => n.name));
for (const [from, routes] of Object.entries(w.connections || {})) {
  if (!names.has(from)) throw new Error(`connection source missing: ${from}`);
  for (const branch of routes.main || []) for (const edge of branch || []) if (edge && !names.has(edge.node)) throw new Error(`connection target missing: ${from} -> ${edge.node}`);
}
const sampler = w.nodes.find((n) => n.name === "按二维表遍历代码生成批次");
const requestNode = w.nodes.find((n) => n.name === "准备二维表检证用例代码请求");
const parser = w.nodes.find((n) => n.name === "解析三文件代码段");
if (!sampler?.parameters?.jsCode || !requestNode?.parameters?.jsCode || !parser?.parameters?.jsCode) throw new Error("missing batch/request/parser nodes");
for (const required of ["const chunkSize = 5", "request.test_category", "safeCategory", "TEST_CASES_TABLE BEGIN", "OUTPUT_FILES", "base36", "36 * 36", "VAR_TYPE_GP", "RESULT6"]) {
  if (!text.includes(required)) throw new Error(`missing implementation safeguard: ${required}`);
}
for (const forbidden of ["MOVE_CASE BEGIN", "MOVE_TC001.CL", "MOVE_TB001.COBOL", "MOVE_TF001.SF"]) {
  if (text.includes(forbidden)) throw new Error(`legacy MOVE output naming remains: ${forbidden}`);
}

const rows = [];
for (let i = 1; i <= 6; i++) rows.push({ project_id: 1, language: "CL", command_id: 9, command_name: "CMD", case_table_version: 3, case_number: `TC${String(i).padStart(3, "0")}`, test_category: i === 6 ? "ZERO_VALUE" : "FIG_HIGH_VALUE", test_purpose: "p", parameter_config: "c", input_data: "i", expected_result: "e", priority: "1" });
const samplerFactory = new Function("$input", "$", sampler.parameters.jsCode);
const normalBatches = samplerFactory({ all: () => rows.map((json) => ({ json })) }, (name) => name === "解析参数" ? { first: () => ({ json: { generation_mode: "case_table", command: "CMD" } }) } : { first: () => ({ json: { machine_type: "ASP" } }) });
if (normalBatches.length !== 2) throw new Error(`expected 2 category batches, got ${normalBatches.length}`);
if (normalBatches.some((x) => x.json.cases.length > 5)) throw new Error("batch exceeds five cases");
if (normalBatches.some((x) => new Set(x.json.cases.map((c) => c.test_category)).size !== 1)) throw new Error("batch crosses test_category");
if (normalBatches[0].json.cases.map((c) => c.case_number).join(",") !== "TC001,TC002,TC003,TC004,TC005") throw new Error("category sort/batch failed");

const moveRows = [
  { project_id: 1, language: "CL", command_id: 227, command_name: "MOVE", case_table_version: 1, case_number: "MOVE_TC001", test_category: "FIG_HIGH_VALUE", test_purpose: "p", parameter_config: "VAR_TYPE_GP(String)", input_data: "i", expected_result: "e", priority: "1" },
  { project_id: 1, language: "CL", command_id: 227, command_name: "MOVE", case_table_version: 1, case_number: "MOVE_TC002", test_category: "FIG_HIGH_VALUE", test_purpose: "p", parameter_config: "VAR_TYPE_GP(String)", input_data: "i", expected_result: "e", priority: "1" },
];
const moveBatches = samplerFactory({ all: () => moveRows.map((json) => ({ json })) }, (name) => name === "解析参数" ? { first: () => ({ json: { generation_mode: "move_master", command: "MOVE" } }) } : { first: () => ({ json: { machine_type: "ASP" } }) });
if (moveBatches.length !== 1 || moveBatches[0].json.cases.length !== 2) throw new Error("MOVE FIG_HIGH_VALUE must be one five-case batch");

const requestFactory = new Function("$input", "$", requestNode.parameters.jsCode);
function makeRequest(baseNo, generationMode = "move_master") {
  const batch = { ...moveBatches[0].json, base_no: baseNo, generation_mode: generationMode };
  return requestFactory({ all: () => [{ json: batch }] }, (name) => name === "解析参数" ? { first: () => ({ json: { command_run_timestamp: "20260915120000" } }) } : { first: () => ({ json: { machine_type: "ASP", xelora_api_base_url: "http://x", xelora_api_key: "k", xelora_agent_id: "a", xelora_kb_ids: ["kb"], xelora_kb_name: "kb" } }) });
}
for (const [baseNo, expected] of [[0, "00"], [1, "01"], [9, "09"], [10, "0A"], [35, "0Z"], [36, "10"], [359, "9Z"], [1295, "ZZ"]]) {
  const out = makeRequest(baseNo)[0].json;
  if (out.program_base_code !== "BO227" || out.file_group_seq !== expected) throw new Error(`base36 naming failed for ${baseNo}: ${out.program_base_code}/${out.file_group_seq}`);
  for (const suffix of ["C", "B", "F"]) if (!out.xelora_query.includes(`BO227${suffix}${expected}`)) throw new Error(`query filename missing BO227${suffix}${expected}`);
}
let overflow = false;
try { makeRequest(1296); } catch { overflow = true; }
if (!overflow) throw new Error("base36 overflow was not rejected");

console.log("PASS: generic verification workflow, five-case category batches, MOVE FIG_HIGH_VALUE, VAR_TYPE_GP(Result6), and BO/base36 naming contracts are valid");

