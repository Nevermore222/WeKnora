import { readFileSync } from "node:fs";

const file = process.argv[2];
if (!file) throw new Error("usage: node validate-toin-cl-reference-workflow.mjs <workflow.json>");
const workflow = JSON.parse(readFileSync(file, "utf8"));
const names = new Set(workflow.nodes.map((node) => node.name));
for (const required of ["按小分類查询tech_point代表行", "按批次分组小分類", "汇总并准备用例生成请求", "保存最终检证用例"]) {
  if (!names.has(required)) throw new Error(`missing node: ${required}`);
}
for (const required of ["创建Xelora会话", "提取Xelora会话ID", "调用Xelora用例生成智能体", "处理本批智能体响应"]) {
  if (!names.has(required)) throw new Error(`missing Xelora node: ${required}`);
}
const query = workflow.nodes.find((node) => node.name === "按小分類查询tech_point代表行").parameters.query;
const selector = workflow.nodes.find((node) => node.name === "按批次分组小分類").parameters.jsCode;
const request = workflow.nodes.find((node) => node.name === "汇总并准备用例生成请求").parameters.jsCode;
const save = workflow.nodes.find((node) => node.name === "保存最终检证用例").parameters.query;
for (const [label, text, fragments] of [
  ["query", query, ["project_id", "language", "MOVE", "IF", "cl_tech_point_case_references", "NOT EXISTS"]],
  ["selector", selector, ["references", "refs.length < 5", "buckets", "round"]],
  ["request", request, ["XELORA_API_BASE_URL", "xelora_session_url", "xelora_agent_request_body", "REFERENCES", "SOURCE_CONTENT", "variable_declaration"]],
  ["save", save, ["BEGIN;", "cl_verification_cases_master_proj", "_reference_rows", "cl_tech_point_case_references", "ON CONFLICT DO NOTHING", "COMMIT;"]],
]) {
  for (const fragment of fragments) if (!text.includes(fragment)) throw new Error(`${label} missing ${fragment}`);
}
const serialized = JSON.stringify(workflow);
for (const forbidden of ["/v1/chat-messages", "dify_request_body", "Authorization", "Bearer app-"]) {
  if (serialized.includes(forbidden)) throw new Error(`workflow still contains Dify artifact: ${forbidden}`);
}
if (!serialized.includes("/sessions") || !serialized.includes("/agent-chat/")) throw new Error("workflow must call Xelora session and agent-chat endpoints");
if (!serialized.includes("XELORA_CL_TECH_POINT_AGENT_ID")) throw new Error("workflow must use the dedicated CL tech-point agent env var");
if (serialized.includes("7286db41-b3e9-4289-ac4d-e6382d6be6e2")) throw new Error("workflow must not use the COBOL tech-point agent");
console.log(JSON.stringify({ valid: true, nodes: workflow.nodes.length, connections: Object.keys(workflow.connections || {}).length }));
