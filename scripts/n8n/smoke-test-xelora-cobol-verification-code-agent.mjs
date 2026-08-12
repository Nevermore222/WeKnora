#!/usr/bin/env node
import { execFileSync } from "node:child_process";
import fs from "node:fs";

const builder = fs.readFileSync(
  new URL("./build-xelora-cobol-verification-code-workflow.mjs", import.meta.url),
  "utf8",
);

function readConstant(name) {
  const match = builder.match(new RegExp(`const ${name} = "([^"]+)";`));
  if (!match) throw new Error(`Unable to read ${name} from workflow builder.`);
  return match[1];
}

function displayWidth(line) {
  return Array.from(line).reduce(
    (width, character) => width + (character.codePointAt(0) <= 0x7f ? 1 : 2),
    0,
  );
}

function runPsql(sql) {
  return execFileSync(
    "docker",
    ["exec", "-i", "Xelora-postgres", "psql", "-U", "postgres", "-d", "Xelora", "-t", "-A"],
    { input: sql, encoding: "utf8" },
  ).trim();
}

const apiBase = readConstant("xeloraApiBaseUrl");
const apiKey = readConstant("xeloraApiKey");
const agentId = runPsql(`SELECT id FROM custom_agents WHERE deleted_at IS NULL AND name = 'COBOL检证用例代码生成智能体-SR6' ORDER BY updated_at DESC LIMIT 1;`);
const kbId = runPsql(`SELECT id FROM knowledge_bases WHERE deleted_at IS NULL AND name = 'Manual_ASP' ORDER BY updated_at DESC LIMIT 1;`);
const knowledgeIds = runPsql(`
SELECT k.id FROM knowledges k
JOIN knowledge_tags t ON t.id = k.tag_id
WHERE k.deleted_at IS NULL AND t.deleted_at IS NULL
  AND k.knowledge_base_id = '${kbId}' AND t.knowledge_base_id = '${kbId}'
  AND t.name = 'ASP_COBOL'
ORDER BY k.title;`).split(/\r?\n/).filter(Boolean);

if (!agentId || !kbId || !knowledgeIds.length) throw new Error("Live agent or Manual_ASP/ASP_COBOL binding is missing.");

async function post(path, body) {
  const response = await fetch(`${apiBase}${path}`, {
    method: "POST",
    headers: { "X-API-Key": apiKey, "Content-Type": "application/json" },
    body: JSON.stringify(body),
    signal: AbortSignal.timeout(600000),
  });
  const text = await response.text();
  if (!response.ok) throw new Error(`${path} returned HTTP ${response.status}: ${text.slice(0, 2000)}`);
  return text;
}

const sessionRaw = await post("/sessions", { title: "COBOL SR6 agent smoke test" });
const session = JSON.parse(sessionRaw);
const sessionId = session.session_id || session.id || session.data?.session_id || session.data?.id;
if (!sessionId) throw new Error(`Session ID missing: ${sessionRaw}`);

const table = [
  "| 技術点NO | 小分類 | 用例番号 | テスト分類 | テスト目的 | パラメータ設定 | 入力データ | 期待結果 | 優先度 | 備考 |",
  "|---|---|---|---|---|---|---|---|---|---|",
  "| CMPTE-SMOKE-0001 | COMPUTE【(受取り側項目が単一変数(9型,Round無変数(9型,加算,変数(9型),SIZE無))】 | TC001 | 正常系 | 9型同士の加算結果を確認する。 | 01 SO-HEIKIN PIC 9(4).;<br>01 KU-HEIKIN PIC 9(3).;<br>COMPUTE SO-HEIKIN = SO-HEIKIN + KU-HEIKIN | SO-HEIKIN=1000<br>KU-HEIKIN=25 | SO-HEIKIN=1025 | P0 | 単一の構造確認用ケース。 |",
].join("\n");
const occursTableRow = [
  "| CMPTE-SMOKE-0002",
  "COMPUTE【受取り側項目が表要素(9型,二重OCCURS,添字参照,SIZE無)】",
  "TC002",
  "正常系",
  "実際の階層に従う二重表要素の加算を確認する。",
  "01 WORK-GOKEI.;<br>  02 WK-GOKEI-TBL OCCURS 4.;<br>    03 WK-G-3.;<br>      05 WK-GOKEI-T3 PIC 9(04) OCCURS 29.;<br>COMPUTE WK-GOKEI-T3(1 1) = WK-GOKEI-T3(1 1) + 1",
  "WK-GOKEI-T3(1 1)=1000",
  "WK-GOKEI-T3(1 1)=1001",
  "P0",
  "01はルート、OCCURSは下位階層に定義する。",
].join(" | ") + " |";
const query = [
  "COMMAND_NAME: COMPUTE",
  "PGM_BASE_CODE: BT999",
  "PGM_FILE_SEQ: 01",
  "BATCH_INDEX: 1/1",
  "LANGUAGE_TYPE: COBOL",
  "以下の TEST_CASES_TABLE から、システムプロンプトに従って完全な CL、COBOL、PF を生成してください。",
  "TEST_CASES_TABLE BEGIN",
  [table, occursTableRow].join("\n"),
  "TEST_CASES_TABLE END",
].join("\n\n");

const stream = await post(`/agent-chat/${sessionId}`, {
  query,
  agent_enabled: true,
  agent_id: agentId,
  knowledge_base_ids: [kbId],
  knowledge_ids: knowledgeIds,
  mentioned_items: [{ id: kbId, name: "Manual_ASP", type: "kb", kb_type: "document" }],
  web_search_enabled: false,
  channel: "api",
});

let answer = "";
for (const line of stream.split("\n")) {
  if (!line.trimStart().startsWith("data:")) continue;
  const payload = line.trimStart().slice(5).trim();
  if (!payload || payload === "[DONE]") continue;
  try {
    const event = JSON.parse(payload);
    if (typeof event.answer === "string") answer += event.answer;
    const type = String(event.response_type || event.event || event.type || "").toLowerCase();
    if ((!type || ["answer", "message", "agent_message", "final_answer", "final_answer_chunk"].includes(type)) && typeof event.content === "string") {
      answer += event.content;
    }
  } catch (error) {}
}

const checks = {
  cl_heading: /##\s+TOINSRC\/BT999C01\.CL/i.test(answer),
  cobol_heading: /##\s+TOINSRC\/BT999B01\.CBL/i.test(answer),
  pf_heading: /##\s+TOINSRC\/BT999F01\.PF/i.test(answer),
  cl_source: /\bDLTFILE\b[\s\S]*\bCRTFILE\b[\s\S]*\bCALL\b/i.test(answer),
  cobol_source: /IDENTIFICATION\s+DIVISION/i.test(answer),
  pf_source: /\/FILE\s+BT999F01/i.test(answer),
};
const cobolMatch = answer.match(/##\s+TOINSRC\/BT999B01\.CBL\s*\r?\n```[^\r\n]*\r?\n([\s\S]*?)```/i);
const cobolSource = cobolMatch?.[1] || "";
const cobolLines = cobolSource.split(/\r?\n/);
const dataDivisionSource =
  cobolSource.match(/\bDATA\s+DIVISION\b[\s\S]*?(?=\bPROCEDURE\s+DIVISION\b)/i)?.[0] || "";
const dataDivisionLines = dataDivisionSource.split(/\r?\n/);
const structuralCommentLines = cobolLines.filter((line) =>
  /^\s*\*(?:\*{5,}|\s*(?:TP_NO|SUB_CATEGORY|TC\d+|AUTHOR|---))/i.test(line),
);
const levelNameConcatenation = /^\s*(?:01|02|03|05|07|09|10|15|20|49|66|77|88)[A-Z0-9]/;
const levelOneOccurs = /^\s*01\b.*\bOCCURS\b/i;
checks.cobol_first_line_indent = /^ {7}IDENTIFICATION/i.test(cobolLines[0] || "");
checks.comment_indicator_column_7 =
  structuralCommentLines.length > 0 && structuralCommentLines.every((line) => line.indexOf("*") === 6);
checks.cobol_display_width_60 = cobolLines.every((line) => displayWidth(line) <= 60);
checks.level_name_separator =
  dataDivisionSource.length > 0 && dataDivisionLines.every((line) => !levelNameConcatenation.test(line));
checks.no_level_01_occurs =
  dataDivisionSource.length > 0 && dataDivisionLines.every((line) => !levelOneOccurs.test(line));
checks.subordinate_occurs_group =
  /^\s*(?:02|03|05)\s+WK-GOKEI-TBL\s+OCCURS\s+4\b/im.test(cobolSource);
checks.typed_occurs_element =
  /^\s*(?:03|05|07)\s+WK-GOKEI-T3\s+PIC\s+9\(04\)\s+OCCURS\s+29\b/im.test(cobolSource);
checks.indexed_element_reference = /WK-GOKEI-T3\s*\([^\r\n)]+\)/i.test(cobolSource);
const failed = Object.entries(checks).filter(([, ok]) => !ok).map(([name]) => name);
if (failed.length) {
  throw new Error(`Live SR6 response is missing structural sections: ${failed.join(", ")}\n${answer.slice(0, 6000)}`);
}

console.log(JSON.stringify({
  session_id: sessionId,
  answer_length: answer.length,
  max_cobol_display_width: Math.max(0, ...cobolLines.map(displayWidth)),
  checks,
}, null, 2));
