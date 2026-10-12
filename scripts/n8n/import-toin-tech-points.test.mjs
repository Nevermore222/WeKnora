import assert from "node:assert/strict";
import test from "node:test";

import {
  buildImportRows,
  buildReplaceSql,
} from "./import-toin-tech-points.mjs";

test("maps every CL and COBOL source field to the tech_point contract", () => {
  const sourceRows = [
    {
      language: "COBOL",
      PGMID: "PGM001",
      PGM_PATH: "src/PGM001.COBOL",
      SEQ: 1,
      AFT_PGN_NM: "PGM001",
      STATS: "OK",
      START_LINE: "10",
      STOP_LINE: "11",
      SYNTAX_ST_TOKEN_INDEX: "20-30",
      TP_NO: null,
      TP_DTL_NO: "CALL001",
      SOURCE_CONTENT: "CALL PGM-A",
      SETUMEI: "CALL description",
      REF_HEAD_SKCD: "CALL0",
      VAR_DEF_SRC: "01 A PIC X.",
      CMD_NAME: "CALL",
      TEST_COUNT: 0,
      TEST_STATUS: null,
    },
  ];
  const commands = new Map([["COBOL\u0000CALL", 227]]);

  assert.deepEqual(buildImportRows(sourceRows, commands), [
    {
      command_id: 227,
      pgmid: "PGM001",
      language: "COBOL",
      sub_category: "CALL description",
      tech_point_no: "CALL001",
      source_content: "CALL PGM-A",
      command_code: null,
      command_name: "CALL",
      location_count: 1,
      project_id: 1,
      variable_declaration: "01 A PIC X.",
    },
  ]);
});

test("preserves every source row for non-excluded commands", () => {
  const base = {
    language: "CL",
    PGMID: "PGM001",
    TP_DTL_NO: "CALL001",
    SOURCE_CONTENT: "CALL PGM-A",
    SETUMEI: "CALL description",
    REF_HEAD_SKCD: "CALL0",
    VAR_DEF_SRC: "01 CALL-AREA PIC X.",
    CMD_NAME: "CALL",
  };
  const rows = buildImportRows(
    [base, { ...base, PGMID: "PGM002", SOURCE_CONTENT: "CALL PGM-B" }],
    new Map([["CL\u0000CALL", 2]]),
  );

  assert.equal(rows.length, 2);
  assert.equal(rows[0].pgmid, "PGM001");
  assert.equal(rows[0].source_content, "CALL PGM-A");
  assert.equal(rows[0].location_count, 1);
  assert.equal(rows[1].pgmid, "PGM002");
  assert.equal(rows[1].source_content, "CALL PGM-B");
  assert.equal(rows[1].location_count, 1);
});

test("excludes MOVE and IF rows entirely", () => {
  const base = {
    language: "CL",
    PGMID: "PGM001",
    TP_DTL_NO: "TP001",
    SOURCE_CONTENT: "source",
    SETUMEI: "description",
  };
  const rows = buildImportRows(
    [
      { ...base, CMD_NAME: "MOVE", VAR_DEF_SRC: "move var" },
      { ...base, CMD_NAME: "IF", VAR_DEF_SRC: "if var" },
      { ...base, CMD_NAME: "CALL", VAR_DEF_SRC: "call var" },
    ],
    new Map([["CL\u0000MOVE", 1], ["CL\u0000IF", 2], ["CL\u0000CALL", 3]]),
  );
  assert.equal(rows.length, 1);
  assert.equal(rows[0].command_name, "CALL");
  assert.equal(rows[0].variable_declaration, "call var");
});

test("leaves command_id null when CMD_NAME is absent from command_master", () => {
  const rows = buildImportRows(
    [{
      language: "CL",
      PGMID: "PGM001",
      TP_DTL_NO: "FSTRG001",
      SOURCE_CONTENT: "%STRING(A)",
      SETUMEI: "function",
      REF_HEAD_SKCD: "FSTRG",
      VAR_DEF_SRC: "01 WS-VALUE PIC X.",
      CMD_NAME: "%STRING",
    }],
    new Map(),
  );

  assert.equal(rows[0].command_id, null);
});

test("rejects MED rows instead of silently importing them", () => {
  assert.throws(
    () => buildImportRows([{ language: "MED" }], new Map()),
    /unsupported source language: MED/,
  );
});

test("replace SQL only removes project 1 CL and COBOL rows", () => {
  const sql = buildReplaceSql([{
    command_id: null,
    pgmid: "PGM001",
    language: "CL",
    sub_category: "CALL description",
    tech_point_no: "CALL001",
    source_content: "CALL PGM-A",
    command_code: null,
    command_name: "CALL",
    location_count: 1,
    project_id: 1,
    variable_declaration: "CALL0",
  }]);

  assert.match(sql, /BEGIN;/);
  assert.match(sql, /DELETE FROM analyzes\.cl_tech_point_case_references/);
  assert.match(sql, /DELETE FROM analyzes\.cl_tech_point_code_generation_sources/);
  assert.match(sql, /DELETE FROM analyzes\.tech_point\s+WHERE project_id = 1\s+AND language IN \('CL', 'COBOL'\)/);
  assert.ok(
    sql.indexOf("DELETE FROM analyzes.cl_tech_point_case_references")
      < sql.indexOf("DELETE FROM analyzes.tech_point"),
  );
  assert.ok(
    sql.indexOf("DELETE FROM analyzes.cl_tech_point_code_generation_sources")
      < sql.indexOf("DELETE FROM analyzes.tech_point"),
  );
  assert.doesNotMatch(sql, /'MED'/);
  assert.match(sql, /expected 1 CL rows/);
  assert.match(sql, /expected 0 COBOL rows/);
  assert.match(sql, /command_code must remain null/);
  assert.match(sql, /COMMIT;/);
});
