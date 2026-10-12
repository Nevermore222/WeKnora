import assert from "node:assert/strict";
import test from "node:test";
import { rows, sql } from "./append-toin-tech-points-add260909.mjs";

test("maps appended COBOL rows field-by-field", () => {
  const result = rows([{
    PGMID: "PGM001", TP_DTL_NO: "TP001", SOURCE_CONTENT: "EVALUATE X",
    SETUMEI: "desc", VAR_DEF_SRC: "01 X PIC 9.", CMD_NAME: "EVALUATE",
  }], new Map([["COBOL\u0000EVALUATE", 237]]));
  assert.deepEqual(result[0], {
    command_id: 237, pgmid: "PGM001", language: "COBOL", sub_category: "desc",
    tech_point_no: "TP001", source_content: "EVALUATE X", command_code: null,
    command_name: "EVALUATE", location_count: 1, project_id: 1,
    variable_declaration: "01 X PIC 9.",
  });
});

test("append SQL inserts without deleting and validates baseline plus additions", () => {
  const statement = sql(rows([{
    PGMID: "P", TP_DTL_NO: "T", SOURCE_CONTENT: "S", SETUMEI: "D",
    VAR_DEF_SRC: "", CMD_NAME: "EVALUATE",
  }], new Map()), 4982);
  assert.match(statement, /INSERT INTO analyzes\.tech_point/);
  assert.doesNotMatch(statement, /DELETE FROM/);
  assert.match(statement, /expected 4983 rows/);
  assert.match(statement, /NULL, 'EVALUATE'/);
});
