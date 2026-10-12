import test from "node:test";
import assert from "node:assert/strict";
import {
  deduplicateReferences,
  groupAndSample,
  projectPrefix,
  sourceStructureSignature,
  variableSignature,
} from "./cl-tech-point-codegen-lib.mjs";

function row(id, pgmid, source = "ADD 1 I GIVING J", declaration = "03 I PIC 9(03). 03 J PIC 9(03).") {
  return { id, project_id: 3, language: "CL", command_id: 229, command_name: "ADD", tech_point_no: "ADD-TP", sub_category: "ADD sample", pgmid, source_content: source, variable_declaration: declaration };
}

test("samples one source per PGM before filling the fifth slot", () => {
  const groups = groupAndSample([row(5, "P5"), row(1, "P1"), row(3, "P3"), row(2, "P2"), row(4, "P4"), row(6, "P1")]);
  assert.equal(groups.length, 1);
  assert.deepEqual(groups[0].references.map((r) => r.pgmid), ["P1", "P2", "P3", "P4", "P5"]);
  assert.deepEqual(groups[0].references.map((r) => r.selection_order), [1, 2, 3, 4, 5]);
});

test("fills from an earlier PGM when fewer than five PGM buckets exist", () => {
  const refs = groupAndSample([row(1, "P1"), row(2, "P1"), row(3, "P2"), row(4, "P2")])[0].references;
  assert.deepEqual(refs.map((r) => r.id), [1, 3, 2, 4]);
});

test("keeps technical points and projects isolated", () => {
  const groups = groupAndSample([row(1, "P1"), { ...row(2, "P1"), project_id: 1, tech_point_no: "OTHER" }]);
  assert.equal(groups.length, 2);
  assert.equal(projectPrefix(1), "CO");
  assert.equal(projectPrefix(3), "CF");
});

test("variable signature ignores names but retains declaration shape", () => {
  assert.equal(variableSignature("03 I PIC 9(03). 03 J PIC 9(03)."), variableSignature("03 LEFT PIC 9(03). 03 RIGHT PIC 9(03)."));
  assert.notEqual(variableSignature("03 I PIC 9(03)."), variableSignature("03 I PIC S9(03)."));
  assert.equal(variableSignature("DCL VAR(&A) TYPE(*CHAR) LEN(8)"), variableSignature("DCL VAR(&B) TYPE(*CHAR) LEN(8)"));
  assert.notEqual(variableSignature("DCL VAR(&A) TYPE(*CHAR) LEN(8)"), variableSignature("DCL VAR(&A) TYPE(*CHAR) LEN(12)"));
});

test("source structure signature ignores names and concrete values", () => {
  assert.equal(sourceStructureSignature("ADD 1 I GIVING J"), sourceStructureSignature("ADD 9 LEFT GIVING RIGHT"));
  assert.equal(sourceStructureSignature("DCL VAR(&A) TYPE(*CHAR) LEN(8)"), sourceStructureSignature("DCL VAR(&B) TYPE(*CHAR) LEN(12)"));
  assert.notEqual(sourceStructureSignature("ADD 1 I GIVING J"), sourceStructureSignature("SUB 1 I GIVING J"));
});

test("source structure signature normalizes literal CRLF escape text", () => {
  const oneLine = "CNVFILE FILE-A.LIB,TOFILE-B.LIB,ADD-@YES";
  const escapedLineBreak = "CNVFILE FILE-A.LIB,\\r\\nTOFILE-B.LIB,ADD-@YES";
  assert.equal(sourceStructureSignature(oneLine), sourceStructureSignature(escapedLineBreak));
});

test("deduplication requires both declaration and source structure to match", () => {
  const refs = [row(1, "P1"), row(2, "P2", "ADD 9 LEFT GIVING RIGHT"), row(3, "P3", "ADD 1 I GIVING J", "03 I PIC S9(03). 03 J PIC S9(03).")];
  const result = deduplicateReferences(refs);
  assert.equal(result.unique.length, 2);
  assert.equal(result.all.filter((r) => r.is_duplicate).length, 1);
  assert.equal(result.unique[0].structure_id, "S01");
});
