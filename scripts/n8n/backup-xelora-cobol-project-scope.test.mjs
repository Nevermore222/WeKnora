import assert from "node:assert/strict";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";
import test from "node:test";
import {
  AGENT_NAMES,
  buildManifest,
  validateAgentRecord,
  verifyBackupSet,
  writeBackupSet,
} from "./backup-xelora-cobol-project-scope.mjs";

function agent(name, id) {
  return {
    id,
    tenant_id: 1,
    name,
    description: `${name} description`,
    config: { system_prompt: `${name} prompt`, knowledge_bases: ["kb-1"] },
    created_at: "2026-08-01T00:00:00Z",
    updated_at: "2026-08-11T00:00:00Z",
  };
}

test("writes two agents, two workflows, and a verifiable manifest", (t) => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "xelora-backup-"));
  t.after(() => fs.rmSync(root, { recursive: true, force: true }));
  const workflows = [
    { name: "verification-case-1.0.0.json", content: '{"name":"case"}' },
    { name: "verification-code-1.0.0.json", content: '{"name":"code"}' },
  ];
  const records = AGENT_NAMES.map((name, index) => agent(name, `agent-${index + 1}`));

  writeBackupSet(root, records, workflows);

  assert.equal(verifyBackupSet(root).ok, true);
  assert.equal(buildManifest(root).files.length, 4);
});

test("rejects an empty agent config", () => {
  assert.throws(
    () => validateAgentRecord({ ...agent(AGENT_NAMES[0], "agent-1"), config: {} }),
    /system_prompt/,
  );
});

test("rejects a missing required agent", (t) => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "xelora-backup-"));
  t.after(() => fs.rmSync(root, { recursive: true, force: true }));
  assert.throws(
    () => writeBackupSet(root, [agent(AGENT_NAMES[0], "agent-1")], []),
    /required agent/i,
  );
});
