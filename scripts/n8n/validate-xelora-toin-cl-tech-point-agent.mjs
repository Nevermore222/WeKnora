#!/usr/bin/env node
import { execFileSync } from "node:child_process";

const agentId = process.env.XELORA_CL_TECH_POINT_AGENT_ID || "ea572e95-405b-43da-b8e7-70a4a22e9cce";
const query = `SELECT json_build_object('id', id, 'name', name, 'description', description, 'config', config) FROM custom_agents WHERE id='${agentId.replaceAll("'", "''")}' AND deleted_at IS NULL;`;
const raw = execFileSync("docker", ["exec", "Xelora-postgres", "psql", "-U", "postgres", "-d", "Xelora", "-t", "-A", "-v", "ON_ERROR_STOP=1", "-c", query], { encoding: "utf8" }).trim();
if (!raw) throw new Error(`agent not found: ${agentId}`);
const agent = JSON.parse(raw);
if (agent.name !== "Xelora TOIN CL技术点二维表生成智能体") throw new Error(`unexpected agent name: ${agent.name}`);
const prompt = agent.config?.system_prompt || "";
for (const fragment of [
  "references 是同一技术点",
  "SOURCE 与变量保真协议",
  "variable_declaration 非空时",
  "正常系",
  "境界値",
  "異常系",
  "只输出一个 Markdown 主表",
  "| 技術点NO | 用例番号 | 小分類 | テスト分類 | テスト目的 | パラメータ設定 | 入力データ | 期待結果 | 優先度 | 備考 |",
  "used_reference_ids",
  "source_tech_point_id",
  "PGMID",
]) {
  if (!prompt.includes(fragment)) throw new Error(`system prompt missing: ${fragment}`);
}
if (!Array.isArray(agent.config?.knowledge_bases) || agent.config.knowledge_bases.length !== 1) throw new Error("agent must bind exactly one Manual_ASP knowledge base");
if (agent.config.web_search_enabled !== false || agent.config.web_fetch_enabled !== false) throw new Error("external web access must remain disabled");
console.log(JSON.stringify({ valid: true, id: agent.id, name: agent.name, promptLength: prompt.length, knowledgeBaseId: agent.config.knowledge_bases[0] }));
