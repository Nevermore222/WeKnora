#!/usr/bin/env node
import crypto from "node:crypto";
import fs from "node:fs";
import path from "node:path";
import { execFileSync } from "node:child_process";

const repoRoot = process.cwd();
const workflowId = "XeloraGenVerCodeA8B247V1";
const workflowPath = path.resolve(repoRoot, "..", "n8n", "检证用例代码", "Xelora通用检证用例代码生成工作流_1.0.0.json");
const backupRoot = path.resolve(repoRoot, "..", "n8n", "backups", "remote-generic-verification-code");
const dbPassword = process.env.N8N_DB_PASSWORD || process.env.PGPASSWORD;
if (!dbPassword) throw new Error("Set N8N_DB_PASSWORD or PGPASSWORD before deployment.");
if (!fs.existsSync(workflowPath)) throw new Error(`Generated workflow not found: ${workflowPath}`);

function psql(query) {
  return execFileSync("docker", ["exec", "-i", "-e", `PGPASSWORD=${dbPassword}`, "Xelora-postgres", "psql", "-h", "192.168.8.247", "-p", "5432", "-U", "postgres", "-d", "n8n", "-v", "ON_ERROR_STOP=1", "-At"], { input: query, encoding: "utf8", maxBuffer: 128 * 1024 * 1024 }).trim();
}
function fetchEntity() {
  const value = psql(`SELECT json_build_object('id',id,'name',name,'active',active,'nodes',nodes,'connections',connections,'settings',settings,'staticData',"staticData",'pinData',"pinData",'meta',meta,'versionId',"versionId",'versionCounter',"versionCounter",'activeVersionId',"activeVersionId",'description',description)::text FROM public.workflow_entity WHERE id='${workflowId}';`);
  if (!value) throw new Error(`Remote workflow not found: ${workflowId}`);
  return JSON.parse(value);
}
function fetchHistory(versionId) {
  const value = psql(`SELECT json_build_object('workflowId',"workflowId",'versionId',"versionId",'authors',authors,'createdAt',"createdAt",'updatedAt',"updatedAt",'nodes',nodes,'connections',connections,'name',name,'autosaved',autosaved,'description',description,'nodeGroups',"nodeGroups")::text FROM public.workflow_history WHERE "workflowId"='${workflowId}' AND "versionId"='${versionId}';`);
  if (!value) throw new Error(`Active workflow history not found: ${workflowId}/${versionId}`);
  return JSON.parse(value);
}
function b64(value) { return Buffer.from(JSON.stringify(value), "utf8").toString("base64"); }
function updateEntity(workflow) {
  return `UPDATE public.workflow_entity SET nodes=convert_from(decode('${b64(workflow.nodes)}','base64'),'UTF8')::json, connections=convert_from(decode('${b64(workflow.connections)}','base64'),'UTF8')::json, "updatedAt"=CURRENT_TIMESTAMP WHERE id='${workflowId}';`;
}
function updateHistory(workflow) {
  return `UPDATE public.workflow_history SET nodes=convert_from(decode('${b64(workflow.nodes)}','base64'),'UTF8')::json, connections=convert_from(decode('${b64(workflow.connections)}','base64'),'UTF8')::json, "updatedAt"=CURRENT_TIMESTAMP WHERE "workflowId"='${workflowId}' AND "versionId"='${workflow.versionId}';`;
}
function keySet(workflow) { return new Set((workflow.nodes || []).map((node) => node.name)); }
function verify(workflow) {
  const names = keySet(workflow);
  const required = ["查询最新检证用例二维表", "按二维表遍历代码生成批次", "准备二维表检证用例代码请求", "调用二维表检证用例代码智能体", "准备二维表生成登记"];
  for (const name of required) if (!names.has(name)) throw new Error(`deployed workflow missing node: ${name}`);
  const text = JSON.stringify(workflow);
  for (const forbidden of ["analyzes.tech_point", "source_content", "variable_declaration", "source_tech_point_id"]) if (text.includes(forbidden)) throw new Error(`deployed workflow contains forbidden source contract: ${forbidden}`);
  const sf = (workflow.nodes || []).find((node) => node.name === "准备SF文件信息");
  if (!sf?.parameters?.jsCode?.includes("FILE/ORG SF/RL/SIZE/RECN")) throw new Error("deployed SF storage gate is missing");
}

const current = fetchEntity();
if (!current.active) throw new Error("Refusing to update inactive workflow.");
const history = fetchHistory(current.versionId);
const generated = JSON.parse(fs.readFileSync(workflowPath, "utf8"));
verify(generated);
const stamp = new Date().toISOString().replace(/[-:TZ.]/g, "").slice(0, 14);
const backupDir = path.join(backupRoot, stamp);
fs.mkdirSync(backupDir, { recursive: true });
fs.writeFileSync(path.join(backupDir, `${workflowId}.entity.before.json`), `${JSON.stringify(current, null, 2)}\n`, "utf8");
fs.writeFileSync(path.join(backupDir, `${workflowId}.history.before.json`), `${JSON.stringify(history, null, 2)}\n`, "utf8");
const digest = crypto.createHash("sha256").update(fs.readFileSync(workflowPath)).digest("hex");
psql(`BEGIN;\n${updateEntity(generated)}\n${updateHistory({ ...generated, versionId: current.versionId })}\nCOMMIT;`);
const after = fetchEntity();
verify(after);
if (!after.active || after.versionId !== current.versionId || after.activeVersionId !== current.activeVersionId) throw new Error("Remote workflow activation/version invariants changed unexpectedly.");
const deployment = { deployed_at: new Date().toISOString(), workflow_id: workflowId, generated_workflow_sha256: digest, backup_dir: backupDir, before: { active: current.active, versionId: current.versionId, activeVersionId: current.activeVersionId }, after: { active: after.active, versionId: after.versionId, activeVersionId: after.activeVersionId, nodes: after.nodes.length } };
fs.writeFileSync(path.join(backupDir, "deployment.json"), `${JSON.stringify(deployment, null, 2)}\n`, "utf8");
console.log(JSON.stringify(deployment, null, 2));
