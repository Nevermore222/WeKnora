import crypto from "node:crypto";

function text(value) {
  return String(value ?? "").trim();
}

function pgmSortValue(value) {
  const valueText = text(value);
  return valueText ? valueText : null;
}

function compareRows(a, b) {
  const ap = pgmSortValue(a.pgmid);
  const bp = pgmSortValue(b.pgmid);
  if (ap === null && bp !== null) return 1;
  if (ap !== null && bp === null) return -1;
  if (ap !== bp) return (ap || "").localeCompare(bp || "");
  return Number(a.id || 0) - Number(b.id || 0);
}

export function technicalPointKey(row) {
  return [row.project_id, row.language, row.command_id, text(row.command_name), text(row.tech_point_no), text(row.sub_category)].join("\u0000");
}

export function groupAndSample(rows, max = 5) {
  if (!Number.isInteger(max) || max < 1) throw new Error("max must be a positive integer");
  const groups = new Map();
  for (const row of rows || []) {
    const key = technicalPointKey(row);
    if (!groups.has(key)) groups.set(key, []);
    groups.get(key).push({ ...row });
  }

  return [...groups.values()].map((group) => {
    group.sort(compareRows);
    const buckets = new Map();
    for (const row of group) {
      const bucket = text(row.pgmid);
      if (!buckets.has(bucket)) buckets.set(bucket, []);
      buckets.get(bucket).push(row);
    }
    const references = [];
    for (let offset = 0; references.length < max; offset += 1) {
      let added = false;
      for (const bucket of buckets.values()) {
        if (!bucket[offset]) continue;
        references.push({ ...bucket[offset], selection_order: references.length + 1 });
        added = true;
        if (references.length >= max) break;
      }
      if (!added) break;
    }
    return {
      project_id: group[0].project_id,
      language: group[0].language,
      command_id: group[0].command_id,
      command_name: group[0].command_name,
      tech_point_no: group[0].tech_point_no,
      sub_category: group[0].sub_category,
      references,
    };
  });
}

function normalizeDeclarationLine(line) {
  let value = String(line || "").replace(/^\s*[*!].*$/, "").trim();
  if (!value) return "";
  value = value.toUpperCase().replace(/[.]+$/, "").replace(/\s+/g, " ");
  // A source export can place several level/name pairs on one physical line.
  value = value.replace(/(^|\s)(\d{1,2})\s+([A-Z?][A-Z0-9_-]*)\b/g, "$1$2 <NAME>");
  // CL declarations carry the variable name in DCL VAR(&NAME), which is not
  // part of the declaration type/length shape used for deduplication.
  value = value.replace(/\b(DCL(?:-[A-Z]+)?\s+VAR\()&?[A-Z?][A-Z0-9_?*-]*/g, "$1&<NAME>");
  return value;
}

export function variableSignature(declaration) {
  const lines = String(declaration || "")
    .split(/\r?\n|(?=\b\d{1,2}\s+[A-Z?][A-Z0-9_-]*\b)/i)
    .map(normalizeDeclarationLine)
    .filter(Boolean);
  return lines.join("|");
}

const STRUCTURE_KEYWORDS = new Set([
  "ACCEPT", "ADD", "CALL", "CHGVAR", "CLOSE", "COMMIT", "COPY", "CRTPF", "CRTDUPOBJ",
  "DCL", "DCLF", "DCL-PROC", "DELETE", "DOWHILE", "DOUNTIL", "ELSE", "ENDDO", "ENDIF",
  "ENDPGM", "ENDSUBR", "GOTO", "IF", "LEAVE", "MONMSG", "MOVE", "OVRDBF", "RCVF", "RETURN",
  "RMVM", "RUNSQL", "SBMJOB", "SNDPGMMSG", "STRCMTCTL", "THEN", "UPDATE", "WHEN", "WHILE",
  "FROM", "TO", "BY", "CMD", "FILE", "LIB", "MBROPT", "LVLCHK", "TYPE", "RCDFMT", "OPTION",
  "PGM", "ENDPGM", "DO", "ENDDO", "CPF0000", "*CHAR", "*DEC", "*LDA", "*LIBL", "*JOB",
]);

function structureToken(token, index) {
  if (!token) return "";
  if (/^'.*'$/.test(token) || /^\".*\"$/.test(token)) return "STR";
  if (/^[+-]?(?:\d+(?:\.\d*)?|\.\d+)$/.test(token)) return "NUM";
  const upper = token.toUpperCase();
  if (STRUCTURE_KEYWORDS.has(upper)) return upper;
  if (/^\*[A-Z0-9_-]+$/.test(upper)) return "SYSTEM_CONST";
  if (/^(?:FILE|LIB|OBJ|MBR)[-_A-Z0-9]+$/.test(upper)) return "RESOURCE";
  if (index === 0) return upper;
  if (/^&?[A-Z?][A-Z0-9_?*-]*$/.test(upper)) return "VAR";
  return upper;
}

export function sourceStructureSignature(source) {
  const normalized = String(source || "")
    // PostgreSQL exports can preserve line breaks as literal escape text.
    .replace(/\\r\\n/g, " ")
    .replace(/\\n/g, " ")
    .replace(/\\r/g, " ")
    .replace(/\r\n?|\n/g, " ")
    .replace(/\/\*[^]*?\*\//g, " ")
    .replace(/\/\/.*$/gm, " ")
    .replace(/--.*$/gm, " ")
    .replace(/'[^']*'/g, (match) => ` '${match.slice(1, -1)}' `)
    .replace(/"[^"]*"/g, ' "STR" ')
    .replace(/([(),=:+*/<>])/g, " $1 ")
    .replace(/\s+/g, " ")
    .trim()
    .toUpperCase();
  if (!normalized) return "";
  return normalized.split(" ").map(structureToken).filter(Boolean).join(" ");
}

function fingerprint(variable, source) {
  return crypto.createHash("sha256").update(`${variable}\n---\n${source}`, "utf8").digest("hex");
}

export function deduplicateReferences(references) {
  const seen = new Map();
  const structures = new Map();
  const output = [];
  for (const reference of references || []) {
    const variable = variableSignature(reference.variable_declaration);
    const source = sourceStructureSignature(reference.source_content);
    const pairFingerprint = fingerprint(variable, source);
    if (!structures.has(pairFingerprint)) structures.set(pairFingerprint, `S${String(structures.size + 1).padStart(2, "0")}`);
    const structureId = structures.get(pairFingerprint);
    const duplicateOf = seen.get(pairFingerprint) || null;
    const enriched = {
      ...reference,
      variable_signature: variable,
      source_structure_signature: source,
      structure_fingerprint: pairFingerprint,
      structure_id: structureId,
      is_duplicate: Boolean(duplicateOf),
      duplicate_of_source_tech_point_id: duplicateOf,
    };
    if (!duplicateOf) {
      seen.set(pairFingerprint, reference.id);
      output.push(enriched);
    }
  }
  return { unique: output, all: (references || []).map((reference) => {
    const variable = variableSignature(reference.variable_declaration);
    const source = sourceStructureSignature(reference.source_content);
    const pairFingerprint = fingerprint(variable, source);
    const first = (references || []).find((candidate) => fingerprint(variableSignature(candidate.variable_declaration), sourceStructureSignature(candidate.source_content)) === pairFingerprint);
    return {
      ...reference,
      variable_signature: variable,
      source_structure_signature: source,
      structure_fingerprint: pairFingerprint,
      structure_id: structures.get(pairFingerprint),
      is_duplicate: first && first.id !== reference.id,
      duplicate_of_source_tech_point_id: first && first.id !== reference.id ? first.id : null,
    };
  }) };
}

export function projectPrefix(projectId) {
  const id = Number(projectId);
  if (id === 1) return "CO";
  if (id === 3) return "CF";
  throw new Error(`unsupported project_id: ${projectId}`);
}

export function safePathSegment(value) {
  const segment = String(value ?? "").trim().replace(/[^a-zA-Z0-9_.-]+/g, "_");
  return segment || "_";
}
