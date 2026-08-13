import assert from "node:assert/strict";
import test from "node:test";
import { selectReferences } from "./toin-cl-reference-selection.mjs";

const row = (id, pgmid) => ({ project_id: 1, language: "CL", command_name: "CALL", tech_point_no: "TP1", sub_category: "desc", id, pgmid, source_content: `S${id}`, variable_declaration: null });

test("covers distinct PGMIDs before round-robin fill", () => {
  const result = selectReferences([row(1, "P1"), row(2, "P1"), row(3, "P2"), row(4, "P3"), row(5, "P4"), row(6, "P5")])[0].references;
  assert.deepEqual(result.map((x) => x.pgmid), ["P1", "P2", "P3", "P4", "P5"]);
});

test("round-robin fills from existing PGM buckets", () => {
  const result = selectReferences([row(1, "P1"), row(2, "P1"), row(3, "P1"), row(4, "P2"), row(5, "P2"), row(6, "P3")])[0].references;
  assert.deepEqual(result.map((x) => x.id), [1, 4, 6, 2, 5]);
});

test("excludes consumed rows and keeps all remaining when fewer than five", () => {
  const result = selectReferences([row(1, "P1"), row(2, "P2"), row(3, "P3")], new Set([2]))[0].references;
  assert.deepEqual(result.map((x) => x.id), [1, 3]);
});
