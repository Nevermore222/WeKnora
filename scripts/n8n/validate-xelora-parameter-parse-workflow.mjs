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
    "参数解析",
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

assert(
  workflow.name === "Xelora - CL Parameter Parse - Parallel",
  "workflow name must identify the Xelora parallel flow",
);
assert(workflow.active === false, "workflow must be inactive before manual validation");
assert(Array.isArray(workflow.nodes), "workflow.nodes must be an array");
assert(workflow.nodes.length >= 10, "workflow must contain the core parallel parsing nodes");
assert(nodeTypes.has("n8n-nodes-base.webhook"), "workflow must contain a webhook node");
assert(nodeTypes.has("n8n-nodes-base.httpRequest"), "workflow must contain HTTP request nodes");
assert(nodeTypes.has("n8n-nodes-base.postgres"), "workflow must contain PostgreSQL nodes");

assert(nodeNames.has("Webhook - Xelora Parameter Parse"), "missing webhook node");
assert(nodeNames.has("Prepare Xelora Session Request"), "missing session request preparation node");
assert(nodeNames.has("Create Xelora Session"), "missing session creation node");
assert(nodeNames.has("Prepare Xelora Agent Request"), "missing agent request preparation node");
assert(nodeNames.has("Call Xelora Agent"), "missing agent call node");
assert(nodeNames.has("Parse Xelora Stream"), "missing stream parser node");
assert(nodeNames.has("Validate Parameter JSON"), "missing JSON validation node");
assert(nodeNames.has("Should Retry Xelora Parse"), "missing retry decision node");
assert(nodeNames.has("Prepare Retry Attempt"), "missing retry preparation node");
assert(nodeNames.has("Ensure Xelora Staging Tables"), "missing staging table node");
assert(nodeNames.has("Build Parameter Insert SQL"), "missing insert SQL builder node");
assert(nodeNames.has("Insert Xelora Parameter Rows"), "missing parameter insert node");
assert(nodeNames.has("Build Failure Insert SQL"), "missing failure SQL builder node");
assert(nodeNames.has("Insert Xelora Failure Row"), "missing failure insert node");

assert(!serialized.includes("192.168.8.247"), "workflow must not call the old Dify host");
assert(!serialized.includes("Authorization"), "workflow must not use Dify Authorization header");
assert(!serialized.includes("Bearer app-"), "workflow must not embed Dify app tokens");
assert(!serialized.includes("source_content"), "workflow must not read or pass source_content");
assert(!serialized.includes("SOURCE"), "workflow must not pass a SOURCE input");
assert(
  !serialized.includes("f786036e-bce5-4fe2-ad96-76a83ab2f78e"),
  "workflow must not reuse the old webhook path",
);
assert(serialized.includes("N8N_XELORA_API_KEY"), "workflow must use N8N_XELORA_API_KEY");
assert(serialized.includes("XELORA_PARAMETER_AGENT_ID"), "workflow must use XELORA_PARAMETER_AGENT_ID");
assert(serialized.includes("XELORA_MANUAL_ASP_KB_ID"), "workflow must use XELORA_MANUAL_ASP_KB_ID");
assert(
  serialized.includes("xelora_command_parameters_staging"),
  "workflow must write to staging parameter table",
);
assert(
  serialized.includes("xelora_parameter_parse_failures"),
  "workflow must write failures to failure table",
);
assert(serialized.includes("retry_reason"), "workflow must preserve retry reason");
assert(
  serialized.includes("Number($json.attempt_count || 1) < 2"),
  "workflow must retry at most once",
);

if (process.exitCode) process.exit();
console.log(`PASS: ${workflowPath}`);
