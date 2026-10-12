import { execFileSync } from "node:child_process";
import { mkdirSync, readFileSync, writeFileSync } from "node:fs";
import { dirname, resolve } from "node:path";
import { fileURLToPath } from "node:url";

const supportedLanguages = new Set(["CL", "COBOL"]);
const columnOrder = [
  "command_id",
  "pgmid",
  "language",
  "sub_category",
  "tech_point_no",
  "source_content",
  "command_code",
  "command_name",
  "location_count",
  "project_id",
  "variable_declaration",
];

function normalize(value) {
  return value === null || value === undefined ? "" : String(value).trim();
}

function sqlLiteral(value) {
  if (value === null || value === undefined) return "NULL";
  if (typeof value === "number") return String(value);
  return `'${String(value).replaceAll("'", "''")}'`;
}

export function buildImportRows(sourceRows, commandIds) {
  const rows = [];

  for (const source of sourceRows) {
    const language = normalize(source.language).toUpperCase();
    if (!supportedLanguages.has(language)) {
      throw new Error(`unsupported source language: ${language || "<empty>"}`);
    }
    const techPointNo = normalize(source.TP_DTL_NO);
    const subCategory = normalize(source.SETUMEI);
    const variableDeclaration = normalize(source.VAR_DEF_SRC);
    const commandName = normalize(source.CMD_NAME);
    if (commandName === "MOVE" || commandName === "IF") continue;
    if (!techPointNo || !subCategory || !commandName) {
      throw new Error(`required source field is empty: ${language}/${techPointNo || "<no TP_DTL_NO>"}`);
    }

    rows.push({
      command_id: commandIds.get(`${language}\u0000${commandName}`) ?? null,
      pgmid: normalize(source.PGMID) || null,
      language,
      sub_category: subCategory,
      tech_point_no: techPointNo,
      source_content: source.SOURCE_CONTENT === null || source.SOURCE_CONTENT === undefined
        ? null
        : String(source.SOURCE_CONTENT),
      command_code: null,
      command_name: commandName,
      location_count: 1,
      project_id: 1,
      variable_declaration: variableDeclaration || null,
    });
  }
  return rows;
}

export function buildReplaceSql(rows) {
  const expected = Object.fromEntries(["CL", "COBOL"].map((language) => [
    language,
    rows.filter((row) => row.language === language).length,
  ]));
  const values = rows.map((row) =>
    `(${columnOrder.map((column) => sqlLiteral(row[column])).join(", ")})`,
  ).join(",\n");
  return `BEGIN;

DELETE FROM analyzes.cl_tech_point_case_references AS child
USING analyzes.tech_point AS source
WHERE child.source_tech_point_id = source.id
  AND source.project_id = 1
  AND source.language IN ('CL', 'COBOL');

DELETE FROM analyzes.cl_tech_point_code_generation_sources AS child
USING analyzes.tech_point AS source
WHERE child.source_tech_point_id = source.id
  AND source.project_id = 1
  AND source.language IN ('CL', 'COBOL');

DELETE FROM analyzes.tech_point
WHERE project_id = 1
  AND language IN ('CL', 'COBOL');

INSERT INTO analyzes.tech_point (${columnOrder.join(", ")})
VALUES
${values};

DO $$
BEGIN
  IF (SELECT count(*) FROM analyzes.tech_point WHERE project_id=1 AND language='CL') <> ${expected.CL} THEN
    RAISE EXCEPTION 'expected ${expected.CL} CL rows';
  END IF;
  IF (SELECT count(*) FROM analyzes.tech_point WHERE project_id=1 AND language='COBOL') <> ${expected.COBOL} THEN
    RAISE EXCEPTION 'expected ${expected.COBOL} COBOL rows';
  END IF;
  IF EXISTS (
    SELECT 1 FROM analyzes.tech_point
    WHERE project_id=1 AND language IN ('CL','COBOL') AND command_code IS NOT NULL
  ) THEN
    RAISE EXCEPTION 'command_code must remain null';
  END IF;
END $$;

COMMIT;
`;
}

function readSourceRows(databasePath) {
  const script = String.raw`
import json, sqlite3, sys
db = sqlite3.connect(sys.argv[1])
db.row_factory = sqlite3.Row
for language, table in [('CL', 'ALL_CL_TECHICAL_POINTS_TBL'), ('COBOL', 'ALL_COBOL_TECHICAL_POINTS_TBL')]:
    for row in db.execute(f'SELECT * FROM "{table}"'):
        item = dict(row)
        item['language'] = language
        print(json.dumps(item, ensure_ascii=False))
`;
  const output = execFileSync("python", ["-c", script, databasePath], {
    encoding: "utf8",
    maxBuffer: 256 * 1024 * 1024,
    env: { ...process.env, PYTHONIOENCODING: "utf-8" },
  });
  return output.trimEnd().split("\n").filter(Boolean).map((line) => JSON.parse(line));
}

function runPsql(sql, options = {}) {
  return execFileSync("docker", [
    "exec", "-i", "-e", `PGPASSWORD=${options.password || "postgres"}`,
    options.container || "Xelora-app", "psql",
    "-v", "ON_ERROR_STOP=1", "-h", options.host || "192.168.8.247",
    "-p", options.port || "5432", "-U", options.user || "postgres",
    "-d", options.database || "n8n",
  ], { input: sql, encoding: "utf8", maxBuffer: 256 * 1024 * 1024 });
}

function loadCommandIds() {
  const output = runPsql(
    "COPY (SELECT language, command, id FROM analyzes.command_master WHERE language IN ('CL','COBOL')) TO STDOUT WITH (FORMAT csv, DELIMITER E'\\t');\n",
  );
  return new Map(output.trimEnd().split("\n").filter(Boolean).map((line) => {
    const [language, command, id] = line.split("\t");
    return [`${language}\u0000${command}`, Number(id)];
  }));
}

function main() {
  const args = process.argv.slice(2);
  const apply = args.includes("--apply");
  const databaseArg = args.find((arg) => !arg.startsWith("--"));
  if (!databaseArg) throw new Error("usage: node import-toin-tech-points.mjs <sqlite-db> [--apply]");

  const databasePath = resolve(databaseArg);
  const timestamp = new Date().toISOString().replaceAll(/[-:.]/g, "").replace("Z", "Z");
  const outputDir = resolve(dirname(fileURLToPath(import.meta.url)), "backups", `toin-tech-point-${timestamp}`);
  mkdirSync(outputDir, { recursive: true });

  const rows = buildImportRows(readSourceRows(databasePath), loadCommandIds());
  const replaceSql = buildReplaceSql(rows);
  const backupSql = "COPY (SELECT * FROM analyzes.tech_point WHERE project_id=1 AND language IN ('CL','COBOL') ORDER BY id) TO STDOUT WITH (FORMAT csv, HEADER true);\n";
  const backupCsv = runPsql(backupSql);
  const caseReferencesBackupCsv = runPsql("COPY (SELECT child.* FROM analyzes.cl_tech_point_case_references AS child JOIN analyzes.tech_point AS source ON source.id=child.source_tech_point_id WHERE source.project_id=1 AND source.language IN ('CL','COBOL')) TO STDOUT WITH (FORMAT csv, HEADER true);\n");
  const codeGenerationSourcesBackupCsv = runPsql("COPY (SELECT child.* FROM analyzes.cl_tech_point_code_generation_sources AS child JOIN analyzes.tech_point AS source ON source.id=child.source_tech_point_id WHERE source.project_id=1 AND source.language IN ('CL','COBOL')) TO STDOUT WITH (FORMAT csv, HEADER true);\n");
  writeFileSync(resolve(outputDir, "before.csv"), backupCsv, "utf8");
  writeFileSync(resolve(outputDir, "before-case-references.csv"), caseReferencesBackupCsv, "utf8");
  writeFileSync(resolve(outputDir, "before-code-generation-sources.csv"), codeGenerationSourcesBackupCsv, "utf8");
  writeFileSync(resolve(outputDir, "replace.sql"), replaceSql, "utf8");

  const summary = {
    apply,
    outputDir,
    sourceDatabase: databasePath,
    rows: Object.fromEntries(["CL", "COBOL"].map((language) => [
      language,
      rows.filter((row) => row.language === language).length,
    ])),
    unmatchedCommands: [...new Set(rows.filter((row) => row.command_id === null)
      .map((row) => `${row.language}:${row.command_name}`))].sort(),
    missingVariableDeclarations: Object.fromEntries(["CL", "COBOL"].map((language) => [
      language,
      rows.filter((row) => row.language === language && row.variable_declaration === null).length,
    ])),
  };
  writeFileSync(resolve(outputDir, "summary.json"), `${JSON.stringify(summary, null, 2)}\n`, "utf8");

  if (apply) runPsql(replaceSql);
  process.stdout.write(`${JSON.stringify(summary, null, 2)}\n`);
}

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  main();
}
