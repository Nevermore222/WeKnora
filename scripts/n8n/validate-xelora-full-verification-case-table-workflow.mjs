#!/usr/bin/env node
import fs from "node:fs";
import path from "node:path";

const p = path.resolve(process.cwd(), "..", "n8n", "二维表", "Xelora通用全量检证用例二维表工作流_1.0.0.json");
const w = JSON.parse(fs.readFileSync(p, "utf8"));
const text = JSON.stringify(w);
const required = [
  "XeloraFullVerTableA8B247V1",
  "xelora-full-verification-case-table",
  "verification_case_table_parse",
  "analyzes.mac_language_agent",
  "analyzes.agent_config",
  "analyzes.cl_verification_cases",
  "TargetDB",
];
for (const value of required) if (!text.includes(value)) throw new Error("missing contract: " + value);
for (const forbidden of ["Dify", "project_id", "tech_point", "cl_verification_cases_master_proj"]) {
  if (text.includes(forbidden)) throw new Error("forbidden legacy content remains: " + forbidden);
}
const names = new Set(w.nodes.map((node) => node.name));
for (const [source, value] of Object.entries(w.connections)) {
  if (!names.has(source)) throw new Error("connection source missing: " + source);
  for (const branch of value.main || []) for (const target of branch || []) {
    if (!names.has(target.node)) throw new Error("connection target missing: " + source + " -> " + target.node);
  }
}
const mustExist = ["准备查询全量二维表绑定", "查询全量二维表绑定", "校验全量二维表绑定", "创建Xelora全量二维表会话", "调用Xelora全量检证用例智能体", "处理Xelora全量二维表响应", "解析检证用例", "保存检证用例"];
for (const name of mustExist) if (!names.has(name)) throw new Error("missing node: " + name);
if (w.active !== false) throw new Error("workflow must be inactive before import");
console.log("PASS: " + w.name + "; nodes=" + w.nodes.length + "; connections=" + Object.keys(w.connections).length + "; no evaluation/project/tech-point chain");
