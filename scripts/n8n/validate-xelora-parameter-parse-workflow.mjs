#!/usr/bin/env node
import fs from "node:fs";
import path from "node:path";

const repoRoot = process.cwd();
const workflowPath =
  process.argv[2] ||
  path.resolve(
    repoRoot,
    "..",
    "n8n",
    "\u53c2\u6570\u89e3\u6790",
    "Xelora-CL-Parameter-Parse-Parallel_1.0.0.json",
  );

function fail(message) {
  console.error(`FAIL: ${message}`);
  process.exitCode = 1;
}

function assert(condition, message) {
  if (!condition) fail(message);
}

let raw;
try {
  raw = fs.readFileSync(workflowPath, "utf8");
} catch (error) {
  fail(`workflow file cannot be read: ${error.message}`);
  process.exit();
}

let workflow;
try {
  workflow = JSON.parse(raw);
} catch (error) {
  fail(`workflow is not valid JSON: ${error.message}`);
  process.exit();
}

const serialized = JSON.stringify(workflow);
const nodeNames = new Set((workflow.nodes || []).map((node) => node.name));
const nodeTypes = new Set((workflow.nodes || []).map((node) => node.type));
const hasNodeNamedLike = (parts) =>
  Array.from(nodeNames).some((name) => parts.every((part) => name.includes(part)));

assert(
  typeof workflow.name === "string" &&
    workflow.name.includes("Xelora") &&
    workflow.name.includes("1.0.0"),
  "workflow name must identify the Xelora parallel flow",
);
assert(workflow.active === false, "workflow must be inactive before manual validation");
assert(Array.isArray(workflow.nodes), "workflow.nodes must be an array");
assert(workflow.nodes.length >= 10, "workflow must contain the core parallel parsing nodes");
assert(nodeTypes.has("n8n-nodes-base.webhook"), "workflow must contain a webhook node");
assert(nodeTypes.has("n8n-nodes-base.httpRequest"), "workflow must contain HTTP request nodes");
assert(nodeTypes.has("n8n-nodes-base.postgres"), "workflow must contain PostgreSQL nodes");

assert(hasNodeNamedLike(["Webhook"]), "missing webhook node");
assert(hasNodeNamedLike(["Xelora", "\u4f1a\u8bdd"]), "missing Xelora session nodes");
assert(hasNodeNamedLike(["Xelora", "\u667a\u80fd\u4f53"]), "missing Xelora agent nodes");
assert(hasNodeNamedLike(["AI"]), "missing AI response processing node");
assert(hasNodeNamedLike(["\u53c2\u6570"]), "missing parameter parsing or persistence nodes");
assert(hasNodeNamedLike(["\u5931\u8d25"]), "missing failure persistence nodes");

assert(!serialized.includes("192.168.8.247"), "workflow must not call the old Dify host");
assert(!serialized.includes("Authorization"), "workflow must not use Dify Authorization header");
assert(!serialized.includes("Bearer app-"), "workflow must not embed Dify app tokens");
assert(!serialized.includes("source_content"), "workflow must not read or pass source_content");
assert(!serialized.includes("SOURCE"), "workflow must not pass a SOURCE input");
assert(
  !serialized.includes("f786036e-bce5-4fe2-ad96-76a83ab2f78e"),
  "workflow must not reuse the old webhook path",
);

const hasEnvConfig =
  serialized.includes("N8N_XELORA_API_KEY") &&
  serialized.includes("XELORA_API_BASE_URL") &&
  serialized.includes("XELORA_PARAMETER_AGENT_ID") &&
  serialized.includes("XELORA_CL_OVERVIEW_AGENT_ID") &&
  serialized.includes("XELORA_MANUAL_ASP_KB_ID");
const hasInlineConfig =
  serialized.includes("X-API-Key") &&
  /https?:\/\/[^"]+\/api\/v1/.test(serialized) &&
  serialized.includes("/agent-chat/") &&
  !serialized.includes("$env.");
assert(hasEnvConfig || hasInlineConfig, "workflow must contain env-based or inline Xelora API config");
assert(serialized.includes("/agent-chat/"), "workflow must call the registered Xelora agent-chat endpoint");
assert(!serialized.includes("/agent-qa"), "workflow must not call the stale agent-qa swagger route");
assert(
  (workflow.nodes || []).some((node) => typeof node.notes === "string" && node.notes.length > 0),
  "workflow must keep node notes",
);

assert(!serialized.includes("CREATE TABLE"), "workflow must not create or mutate database schema");
assert(
  serialized.includes("DELETE FROM analyzes.command_parameters"),
  "workflow must clear existing command parameters before inserting new rows",
);
assert(
  serialized.includes("INSERT INTO analyzes.command_parameters"),
  "workflow must write to the formal command_parameters table",
);
assert(
  serialized.includes("parameter_table_markdown"),
  "workflow must build a Markdown parameter table for the overview agent",
);
assert(
  serialized.includes("XELORA_CL_OVERVIEW_AGENT_ID") || serialized.includes("overview_agent_request_body"),
  "workflow must call the CL overview agent",
);
assert(
  serialized.includes("UPDATE analyzes.command_master") && serialized.includes("detail_info"),
  "workflow must persist overview Markdown to command_master.detail_info",
);
assert(
  serialized.includes("xelora_parameter_parse_failures"),
  "workflow must write failures to failure table",
);
assert(
  serialized.includes("startsWith") && serialized.includes("data:"),
  "workflow must parse SSE data lines with or without a space",
);
assert(serialized.includes("response_type"), "workflow must inspect Xelora SSE response_type");
assert(
  serialized.includes("description") && serialized.includes("relationship_notes"),
  "workflow prompt must demand detailed parameter descriptions and relationships",
);
assert(
  serialized.includes("description\u548crelationship_notes") &&
    (serialized.includes("\u4e2d\u6587\u8bf4\u660e") || serialized.includes("\u4e2d\u6587\u8865\u5145\u89e3\u91ca")),
  "workflow prompt must require Chinese descriptions and relationship notes",
);
assert(
  serialized.includes("non_chinese_explanation") &&
    serialized.includes("needsChineseRewrite") &&
    serialized.includes("hasJapaneseSentence"),
  "workflow must reject Japanese explanatory sentences in description and relationship_notes",
);
assert(
  serialized.includes("\u7d42\u4e86\u4ee3\u7801") &&
    serialized.includes("Manual_ASP\u539f\u6587") &&
    serialized.includes("\u4e8c\u7ef4\u8868"),
  "overview prompt must keep exit codes as the manual code set instead of an advice table",
);
assert(
  serialized.includes("\u5fc5\u987b\u7406\u89e3Manual\u539f\u6587\u540e\u91cd\u5199") &&
    serialized.includes("\u91cd\u5199\u540e\u7684\u89c4\u8303\u5316\u8bed\u6cd5") &&
    serialized.includes("R S T M B R"),
  "overview prompt must rewrite corrupted manual command format into clean syntax",
);
assert(
  serialized.includes("hasGarbledText") &&
    serialized.includes("overview_has_garbled_text") &&
    serialized.includes("detail_info was not updated"),
  "workflow must block garbled overview Markdown from updating detail_info",
);
assert(serialized.includes("retry_reason"), "workflow must preserve retry reason");
assert(
  serialized.includes("Number($json.attempt_count || 1) < 2"),
  "workflow must retry at most once",
);

if (process.exitCode) process.exit();
console.log(`PASS: ${workflowPath}`);
