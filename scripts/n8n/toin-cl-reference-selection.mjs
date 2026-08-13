function compareRows(a, b) {
  const ap = a.pgmid == null || String(a.pgmid).trim() === "" ? "\uffff" : String(a.pgmid);
  const bp = b.pgmid == null || String(b.pgmid).trim() === "" ? "\uffff" : String(b.pgmid);
  return ap.localeCompare(bp) || Number(a.id) - Number(b.id);
}

function groupKey(row) {
  return [row.project_id, row.language, row.command_name, row.tech_point_no, row.sub_category]
    .map((value) => value == null ? "" : String(value).trim()).join("\u0000");
}

export function selectReferences(rows, consumedIds = new Set(), limit = 5) {
  const consumed = new Set([...consumedIds].map((id) => String(id)));
  const groups = new Map();
  for (const row of rows) {
    if (consumed.has(String(row.id))) continue;
    const key = groupKey(row);
    if (!groups.has(key)) groups.set(key, []);
    groups.get(key).push(row);
  }
  return [...groups.entries()].map(([key, candidates]) => {
    const buckets = new Map();
    for (const row of candidates.sort(compareRows)) {
      const pgmid = row.pgmid == null ? "" : String(row.pgmid).trim();
      if (!buckets.has(pgmid)) buckets.set(pgmid, []);
      buckets.get(pgmid).push(row);
    }
    const bucketList = [...buckets.values()];
    const references = [];
    for (let index = 0; references.length < limit; index += 1) {
      let added = false;
      for (const bucket of bucketList) {
        if (bucket[index]) {
          references.push(bucket[index]);
          added = true;
          if (references.length === limit) break;
        }
      }
      if (!added) break;
    }
    return { key, group: candidates[0], references };
  });
}
