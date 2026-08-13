import { readFileSync } from "node:fs";

const file = process.argv[2];
if (!file) throw new Error("usage: node validate-toin-cl-reference-workflow.mjs <workflow.json>");
const workflow = JSON.parse(readFileSync(file, "utf8"));
const names = new Set(workflow.nodes.map((node) => node.name));
for (const required of ["按小分類查询tech_point代表行", "按批次分组小分類", "汇总并准备用例生成请求", "保存最终检证用例"]) {
  if (!names.has(required)) throw new Error(`missing node: ${required}`);
}
const query = workflow.nodes.find((node) => node.name === "按小分類查询tech_point代表行").parameters.query;
const selector = workflow.nodes.find((node) => node.name === "按批次分组小分類").parameters.jsCode;
const request = workflow.nodes.find((node) => node.name === "汇总并准备用例生成请求").parameters.jsCode;
const save = workflow.nodes.find((node) => node.name === "保存最终检证用例").parameters.query;
for (const [label, text, fragments] of [
  ["query", query, ["project_id", "language", "MOVE", "IF", "cl_tech_point_case_references", "NOT EXISTS"]],
  ["selector", selector, ["references", "refs.length < 5", "buckets", "round"]],
  ["request", request, ["REFERENCES", "SOURCE_CONTENT", "variable_declaration"]],
  ["save", save, ["BEGIN;", "cl_verification_cases_master_proj", "cl_tech_point_case_references", "ON CONFLICT DO NOTHING", "COMMIT;"]],
]) {
  for (const fragment of fragments) if (!text.includes(fragment)) throw new Error(`${label} missing ${fragment}`);
}
console.log(JSON.stringify({ valid: true, nodes: workflow.nodes.length, connections: Object.keys(workflow.connections || {}).length }));
