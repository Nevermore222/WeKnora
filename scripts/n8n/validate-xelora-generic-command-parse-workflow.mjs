#!/usr/bin/env node
import fs from "node:fs";
import path from "node:path";

const repoRoot = process.cwd();
const workflowPath = path.resolve(
  repoRoot,
  "..",
  "n8n",
  "参数解析",
  "Xelora通用命令参数与概述解析工作流_1.0.3.json",
);

if (!fs.existsSync(workflowPath)) throw new Error(`Workflow does not exist: ${workflowPath}`);
const workflowText = fs.readFileSync(workflowPath, "utf8");
const workflow = JSON.parse(workflowText);
if (workflow.id !== "XeloraGenericA8B247V3") throw new Error("Unexpected workflow ID");
const names = new Set(workflow.nodes.map((node) => node.name));

for (const node of workflow.nodes) {
  const jsCode = node.parameters?.jsCode;
  if (typeof jsCode !== "string") continue;
  try {
    new Function(jsCode);
  } catch (error) {
    throw new Error(`Invalid JavaScript in node ${node.name}: ${error.message}`);
  }
}

for (const [sourceName, outputGroups] of Object.entries(workflow.connections)) {
  if (!names.has(sourceName)) throw new Error(`Connection source does not exist: ${sourceName}`);
  for (const outputs of Object.values(outputGroups)) {
    for (const branch of outputs) {
      for (const connection of branch || []) {
        if (!names.has(connection.node)) throw new Error(`Connection target does not exist: ${connection.node}`);
      }
    }
  }
}

const requiredNodes = [
  "Webhook接收参数",
  "解析Webhook参数",
  "准备查询通用智能体绑定",
  "查询通用智能体与知识库绑定",
  "校验并整理通用绑定",
  "判断通用绑定是否完整",
  "调用Xelora参数解析智能体",
  "解析参数二维表",
  "保存参数到表",
  "调用概述智能体",
  "更新命令主表",
  "保存失败记录",
];
for (const name of requiredNodes) if (!names.has(name)) throw new Error(`Missing node: ${name}`);

const requiredText = [
  "xelora-command-parse-generic",
  "analyzes.mac_language_agent",
  "analyzes.agent_config",
  "analyzes.mac_list",
  "workflow_role",
  "command_parameter_parse",
  "command_overview_parse",
  "machine_type",
  "language",
  "knowledge_base_ids",
  "parameter_base_url",
  "parameter_api_key",
  "overview_base_url",
  "overview_api_key",
  "workflow_binding_not_found",
  "analyzes.command_parameters",
  "analyzes.command_master",
  "detail_info",
  "analyzes.xelora_parameter_parse_failures",
  "data:",
  "response_type",
];
for (const fragment of requiredText) if (!workflowText.includes(fragment)) throw new Error(`Missing contract: ${fragment}`);

const forbiddenText = [
  "XELORA_PARAMETER_AGENT_ID",
  "XELORA_CL_OVERVIEW_AGENT_ID",
  "XELORA_MANUAL_ASP_KB_ID",
  "51c574e9-0445-4c41-b0c7-6eecda9c4638",
  "d2ee025f-c41c-4ecd-9b1e-7136df4551c7",
  "a9beec2d-4c56-4314-99a6-b44da92b1863",
  "Manual_ASP",
  "custom_agents",
  "tenant_id",
  "N8N_XELORA_API_KEY",
  "sk-",
];
for (const fragment of forbiddenText) if (workflowText.includes(fragment)) throw new Error(`Legacy hard-code remains: ${fragment}`);

const webhook = workflow.nodes.find((node) => node.name === "Webhook接收参数");
if (webhook?.parameters?.path !== "xelora-command-parse-generic") throw new Error("Unexpected webhook path");
const bindingNode = workflow.nodes.find((node) => node.name === "查询通用智能体与知识库绑定");
if (bindingNode?.credentials?.postgres?.name !== "TargetDB") throw new Error("Binding lookup must use the target database credential");
const httpNodes = workflow.nodes.filter((node) => node.type === "n8n-nodes-base.httpRequest");
for (const node of httpNodes) {
  const apiKeyHeader = node.parameters?.headerParameters?.parameters?.find((header) => String(header.name).toLowerCase() === "x-api-key");
  const expected = ["创建概述会话", "调用概述智能体"].includes(node.name)
    ? "={{ $json.overview_api_key }}"
    : "={{ $json.parameter_api_key }}";
  if (apiKeyHeader?.value !== expected) throw new Error(`HTTP node ${node.name} does not use its mapped API key`);
  if (node.parameters?.options?.timeout !== 600000) throw new Error(`HTTP node ${node.name} must use the bounded 10-minute timeout`);
}
if (workflow.active !== false) throw new Error("Generated workflow must remain inactive until live verification");
const bindingBranches = workflow.connections["判断通用绑定是否完整"]?.main || [];
if (bindingBranches[0]?.[0]?.node !== "准备Xelora会话请求") throw new Error("Valid binding branch is incorrect");
if (bindingBranches[1]?.[0]?.node !== "准备失败记录SQL") throw new Error("Missing binding branch is incorrect");

console.log("PASS: generic command parameter and overview workflow contract is valid");
