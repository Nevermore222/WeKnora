import { readFileSync } from "node:fs";

const file = process.argv[2];
if (!file) throw new Error("usage: node validate-toin-cl-agent-config.mjs <agent.yml>");
const text = readFileSync(file, "utf8");
for (const fragment of ["references", "source_tech_point_id", "SOURCE_CONTENT", "variable_declaration", "used_reference_ids", "PGMID"]) {
  if (!text.includes(fragment)) throw new Error(`agent config missing ${fragment}`);
}
console.log(JSON.stringify({ valid: true, bytes: Buffer.byteLength(text) }));
