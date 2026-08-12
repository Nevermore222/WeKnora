#!/usr/bin/env node
import assert from "node:assert/strict";
import fs from "node:fs";
import path from "node:path";

const repoRoot = process.cwd();
const workflowPath = path.resolve(
  repoRoot,
  "..",
  "n8n",
  "检证用例代码",
  "Xelora版COBOL检证用例代码生成工作流_1.1.0.json",
);
const workflow = JSON.parse(fs.readFileSync(workflowPath, "utf8"));
const organizeNode = workflow.nodes.find((node) => node.name === "组织检证用例二维表");
assert.ok(organizeNode, "missing code-generation organize node");
const organize = new Function("$input", "$", organizeNode.parameters.jsCode);

const rows = [
  {
    id: 6093,
    project_id: 3,
    project_name: "福山",
    tech_point_no: "ACEPTV0000214",
    sub_category: "ACCEPT【変数(GROUP型,TIME)】",
    source_tech_point_id: 6093,
    source_content: "ACCEPT W-TIME FROM TIME",
    variable_declaration: "01 W-TIME.",
    case_number: "F-001",
    test_category: "正常系",
  },
  {
    id: 6094,
    project_id: 3,
    project_name: "福山",
    tech_point_no: "ACEPTV0000217",
    sub_category: "ACCEPT【変数(GROUP型,SPECIAL)】",
    source_tech_point_id: 6094,
    source_content: "ACCEPT WK-TENCD FROM CSL",
    variable_declaration: "03 WK-TENCD.",
    case_number: "F-002",
    test_category: "正常系",
  },
  {
    id: 6095,
    project_id: 3,
    project_name: "福山",
    tech_point_no: "ADD-1L00000030V000003V0000030",
    sub_category: "ADD【定数(All無,数字)TO無変数(9型,変数(9型,ROUND無)】",
    source_tech_point_id: 6095,
    source_content: "ADD 1 , I GIVING J",
    variable_declaration: "03 I PIC 9(03). 03 J PIC 9(03).",
    case_number: "F-003",
    test_category: "正常系",
  },
  {
    id: 6096,
    project_id: 3,
    project_name: "福山",
    tech_point_no: "ADD-1L00000030V000017V0000170",
    sub_category: "ADD【定数(All無,数字)TO無変数(COMP-3型,変数(COMP-3型,ROUND無)】",
    source_tech_point_id: 6096,
    source_content: "ADD 1 I1 GIVING I2",
    variable_declaration: "01 I1 PIC 9(03) PACKED-DECIMAL. 01 I2 PIC 9(03) PACKED-DECIMAL.",
    case_number: "F-004",
    test_category: "正常系",
  },
];

function run(inputRows) {
  return organize(
    { all: () => inputRows.map((json) => ({ json })) },
    () => ({ first: () => ({ json: {
      command_id: 228,
      command: "ADD",
      version: 1,
      language: "COBOL",
      project_id: 3,
      code_gen_batch_size: 14,
    } }) }),
  );
}

const [batch] = run(rows);
assert.equal(batch.json.project_id, 3);
assert.equal(batch.json.pgm_base_code, "BF228");
assert.equal(batch.json.tech_point_context.match(/TECH_POINT_CONTEXT BEGIN/g).length, 4);
for (const row of rows) {
  assert.ok(batch.json.tech_point_context.includes(`TECH_POINT_NO: ${row.tech_point_no}`));
  assert.ok(batch.json.tech_point_context.includes(row.variable_declaration));
  assert.ok(batch.json.xelora_agent_request_body.query.includes(row.source_content));
}

const missingDeclaration = { ...rows[0], variable_declaration: "" };
const fallbackBatch = run([missingDeclaration])[0].json;
assert.match(fallbackBatch.tech_point_context, /VARIABLE_DECLARATION_MODE: CURRENT_TECH_POINT_FALLBACK/);
assert.doesNotMatch(fallbackBatch.tech_point_context, /03 WK-TENCD/);

assert.throws(
  () => run([
    rows[0],
    { ...rows[0], source_content: "ACCEPT OTHER FROM TIME" },
  ]),
  /同一技术点的 source context 不一致/,
);

console.log("COBOL project-scope fixture validation passed.");
