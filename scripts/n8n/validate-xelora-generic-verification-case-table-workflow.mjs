#!/usr/bin/env node
import fs from "node:fs";
import path from "node:path";
const p = path.resolve(process.cwd(), "..", "n8n", "二维表", "Xelora通用检证用例二维表工作流_1.0.0.json");
const w = JSON.parse(fs.readFileSync(p, "utf8"));
const text = JSON.stringify(w);
for (const v of ["XeloraGenVerTableA8B247V1", "xelora-verification-case-table-generic", "verification_case_table_parse", "analyzes.mac_language_agent", "analyzes.cl_verification_cases_master_proj", "TargetDB"]) if (!text.includes(v)) throw new Error(`missing contract: ${v}`);
if (w.active !== false) throw new Error("workflow must be inactive before import");
if (text.includes("XELORA_API_KEY = $env") || text.includes("192.168.2.61")) throw new Error("legacy hard-coded Xelora config remains");
if (!w.nodes.some((n) => n.type === "n8n-nodes-base.webhook") || !w.nodes.some((n) => n.type === "n8n-nodes-base.postgres")) throw new Error("missing core nodes");
console.log("PASS: generic verification-case table workflow contract is valid");
