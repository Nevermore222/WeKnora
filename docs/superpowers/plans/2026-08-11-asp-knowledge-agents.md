# ASP Knowledge-Base Agents Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Idempotently provision and verify two administrator-owned, strictly single-knowledge-base ASP expert agents on `192.168.9.177`.

**Architecture:** Run a short-lived PowerShell provisioner on the remote Windows host. It mints an in-memory tenant API key using the running app's `TENANT_AES_KEY`, clones `SR_ASP` through the agent API, applies a narrow set of target overrides, and uses the same API for create/update/read-back; because API-key authentication intentionally creates tenant-owned agents, one tightly scoped SQL update sets `created_by` to the requested administrator after each API write. No product source, baseline agent, model, knowledge-base, or unrelated agent is changed.

**Tech Stack:** PowerShell 5.1, OpenSSH, Docker, Go 1.26 cross-compilation, Xelora REST API, PostgreSQL `psql`, SSE agent-chat responses.

---

## File Structure

- Retain: `docs/superpowers/specs/2026-08-11-asp-knowledge-agents-design.md` as the approved behavioral contract.
- Create and retain: `docs/superpowers/plans/2026-08-11-asp-knowledge-agents.md` as the execution record.
- Create temporarily: `.tmp/asp-agent-api-key/main.go` to mint a valid tenant API key without exposing `TENANT_AES_KEY` outside the remote process.
- Build temporarily: `.tmp/asp-agent-api-key/mint-tenant-key.exe`; copy it to `D:\Knavis\tmp\asp-agent-provision\` and delete both temporary copies after verification.
- Do not modify product source files or existing deployment configuration.

### Task 1: Run A Read-Only Remote Preflight

- [ ] **Step 1: Confirm the exact baseline, knowledge bases, models, administrator, and current target count**

Run the following SQL through SSH standard input so command-line quoting cannot alter it:

```powershell
$key = Join-Path $env:USERPROFILE '.ssh\knavis_9177_release'
$sql = @'
SELECT 'admin|' || id || '|' || tenant_id FROM users
 WHERE id = '8b4b0fc8-b801-49eb-8d20-e3021ea0b24f'
   AND email = 'xuwh@softroad.com.cn';
SELECT 'baseline|' || id || '|' || tenant_id FROM custom_agents
 WHERE id = '99a63205-a8f4-412f-a284-ebc043838e62' AND deleted_at IS NULL;
SELECT 'kb|' || name || '|' || id || '|' || tenant_id FROM knowledge_bases
 WHERE id IN ('1cf679e6-be1f-4a63-a825-b8d4b284879a',
              'f106b6d6-a268-4fc1-8fb5-15f42f408951')
   AND deleted_at IS NULL ORDER BY name;
SELECT 'model|' || id || '|' || type || '|' || status FROM models
 WHERE id IN ('cd3394b4-99d6-4f10-9c5b-12f773add8f7',
              'f9d119b8-1e46-478e-8379-147839170bd3',
              '3efff119-1faa-463f-9d29-15a8215c18dc') ORDER BY id;
SELECT 'target|' || name || '|' || id || '|' || COALESCE(created_by, '') FROM custom_agents
 WHERE tenant_id = 10000 AND deleted_at IS NULL
   AND name IN ('ASP-JP1 专家', 'ASP-BSP-RM 账票专家') ORDER BY name, id;
'@
$sql | ssh.exe -i $key -o IdentitiesOnly=yes -o BatchMode=yes `
  softroad@192.168.9.177 `
  'docker exec -i Xelora-postgres psql -U postgres -d WeKnora -t -A'
```

Expected: one administrator, one `SR_ASP`, two knowledge bases in tenant `10000`, three active models, and zero or one administrator-owned row per target name. Stop on a missing/duplicate/inactive invariant.

- [ ] **Step 2: Snapshot unrelated records for post-write comparison**

```sql
SELECT md5(string_agg(id || ':' || updated_at::text, '|' ORDER BY id))
FROM custom_agents
WHERE tenant_id = 10000 AND deleted_at IS NULL
  AND name NOT IN ('ASP-JP1 专家', 'ASP-BSP-RM 账票专家');

SELECT md5(string_agg(id || ':' || updated_at::text, '|' ORDER BY id))
FROM knowledge_bases
WHERE tenant_id = 10000 AND deleted_at IS NULL;
```

Expected: record both hashes without printing any credential columns.

### Task 2: Prepare The Short-Lived API-Key Helper

- [ ] **Step 1: Create the minimal helper source**

Create `.tmp/asp-agent-api-key/main.go` with this complete program:

```go
package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

func main() {
	secret := []byte(os.Getenv("TENANT_AES_KEY"))
	if len(secret) != 16 && len(secret) != 24 && len(secret) != 32 {
		panic("TENANT_AES_KEY must be a valid AES key length")
	}
	block, err := aes.NewCipher(secret)
	if err != nil { panic(err) }
	gcm, err := cipher.NewGCM(block)
	if err != nil { panic(err) }
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil { panic(err) }
	plain := make([]byte, 8)
	binary.LittleEndian.PutUint64(plain, 10000)
	sealed := gcm.Seal(nil, nonce, plain, nil)
	combined := append(nonce, sealed...)
	fmt.Print("sk-" + base64.RawURLEncoding.EncodeToString(combined))
}
```

- [ ] **Step 2: Cross-compile and copy the helper without running it locally**

```powershell
$env:GOOS = 'windows'
$env:GOARCH = 'amd64'
go build -trimpath -o '.tmp\asp-agent-api-key\mint-tenant-key.exe' `
  '.\.tmp\asp-agent-api-key\main.go'
Remove-Item Env:GOOS, Env:GOARCH

$key = Join-Path $env:USERPROFILE '.ssh\knavis_9177_release'
ssh.exe -i $key -o IdentitiesOnly=yes -o BatchMode=yes `
  softroad@192.168.9.177 `
  'if not exist D:\Knavis\tmp\asp-agent-provision mkdir D:\Knavis\tmp\asp-agent-provision'
scp.exe -i $key -o IdentitiesOnly=yes `
  '.tmp\asp-agent-api-key\mint-tenant-key.exe' `
  'softroad@192.168.9.177:D:/Knavis/tmp/asp-agent-provision/mint-tenant-key.exe'
```

Expected: build and copy exit `0`; no key or secret appears in output.

### Task 3: Provision Both Agents Through The API

- [ ] **Step 1: Run the remote PowerShell provisioner**

Send a PowerShell script over SSH standard input and execute it with `powershell -NoProfile -File -`. The script must:

1. Read `TENANT_AES_KEY` from `docker inspect Xelora-app` into a process variable.
2. Set it only for `mint-tenant-key.exe`, capture the generated API key, then clear `TENANT_AES_KEY`.
3. GET `/api/v1/agents/99a63205-a8f4-412f-a284-ebc043838e62` and deep-clone its `config` via JSON round-trip.
4. For each target, overwrite only the following values:

```json
{
  "agent_mode": "smart-reasoning",
  "agent_type": "rag-qa",
  "kb_selection_mode": "selected",
  "knowledge_bases": ["$($target.KBID)"],
  "web_search_enabled": false,
  "web_fetch_enabled": false,
  "mcp_selection_mode": "none",
  "mcp_services": [],
  "skills_selection_mode": "none",
  "selected_skills": [],
  "allowed_tools": [
    "knowledge_search",
    "grep_chunks",
    "list_knowledge_chunks",
    "get_document_info",
    "thinking"
  ],
  "retrieve_kb_only_when_mentioned": false,
  "retain_retrieval_history": false
}
```

Append this agent-specific block to the cloned `system_prompt`, substituting the target name and knowledge-base name:

```text
### Agent-Specific Bound Scope (ABSOLUTE)
You are $($target.Name). For every factual or domain-specific question, use only the currently bound $($target.KBName) knowledge base. Perform fresh retrieval and mandatory deep reading before answering. Answer in the same language as the user's current question, preserve exact product terms, command names, parameters, and message IDs, and place knowledge-base citations inline with the claims they support. Web search, web fetch, database access, workspace access, mutation tools, and unbound knowledge bases are forbidden. If the bound knowledge base does not provide enough evidence, state that clearly in the user's language and do not fill the gap from parametric knowledge.
```

Use these API payload identities:

```text
ASP-JP1 专家
仅基于 ASP-JP1 知识库进行深度检索、引用式回答和证据不足说明的 JP1 专家。
KB: 1cf679e6-be1f-4a63-a825-b8d4b284879a

ASP-BSP-RM 账票专家
仅基于 ASP-BSP-RM账票KnowDB 进行深度检索、引用式回答和证据不足说明的账票专家。
KB: f106b6d6-a268-4fc1-8fb5-15f42f408951
```

Represent those identities in the script as the only entries in `$targets`:

```powershell
$targets = @(
  [pscustomobject]@{
    Name = 'ASP-JP1 专家'
    Description = '仅基于 ASP-JP1 知识库进行深度检索、引用式回答和证据不足说明的 JP1 专家。'
    KBName = 'ASP-JP1'
    KBID = '1cf679e6-be1f-4a63-a825-b8d4b284879a'
  },
  [pscustomobject]@{
    Name = 'ASP-BSP-RM 账票专家'
    Description = '仅基于 ASP-BSP-RM账票KnowDB 进行深度检索、引用式回答和证据不足说明的账票专家。'
    KBName = 'ASP-BSP-RM账票KnowDB'
    KBID = 'f106b6d6-a268-4fc1-8fb5-15f42f408951'
  }
)
```

For each exact target name, issue `PUT /api/v1/agents/:id` when exactly one tenant target exists, otherwise `POST /api/v1/agents` when none exists. Stop when more than one exists. Immediately after the successful API response, validate and capture the ID, then set ownership using ID/name-scoped SQL:

```powershell
$agentID = [string]$response.data.id
if ($agentID -notmatch '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$') {
  throw 'API returned an invalid agent ID'
}
$escapedName = $target.Name.Replace("'", "''")
$ownerSQL = @"
UPDATE custom_agents
SET created_by = '8b4b0fc8-b801-49eb-8d20-e3021ea0b24f'
WHERE id = '$agentID'
  AND tenant_id = 10000
  AND name = '$escapedName'
  AND deleted_at IS NULL;
"@
$ownerResult = $ownerSQL | docker exec -i Xelora-postgres `
  psql -U postgres -d WeKnora -t -A
if (($ownerResult | Select-Object -Last 1).Trim() -ne 'UPDATE 1') {
  throw 'Ownership update did not affect exactly one target'
}
```

Require `UPDATE 1`, GET the agent back through the API, and clear the in-memory API key after both targets finish. Output only target names, IDs, and `created`/`updated` actions.

- [ ] **Step 2: Re-run the provisioner to prove idempotency**

Expected: both actions are `updated`, IDs are unchanged, and no duplicate target rows exist.

### Task 4: Verify Persisted Configuration And Isolation

- [ ] **Step 1: Validate all persisted invariants in one read-only SQL query**

```sql
SELECT name,
       id,
       tenant_id,
       created_by,
       config->>'agent_mode',
       config->>'agent_type',
       config->>'kb_selection_mode',
       config->'knowledge_bases',
       config->>'model_id',
       config->>'rerank_model_id',
       config->>'vlm_model_id',
       config->>'web_search_enabled',
       config->>'web_fetch_enabled',
       config->'allowed_tools',
       config->>'mcp_selection_mode',
       config->>'skills_selection_mode'
FROM custom_agents
WHERE tenant_id = 10000 AND deleted_at IS NULL
  AND name IN ('ASP-JP1 专家', 'ASP-BSP-RM 账票专家')
ORDER BY name;
```

Expected: exactly two rows; each has the administrator owner, one distinct expected KB ID, the three baseline model IDs, both Web flags false, exactly five allowed retrieval/reasoning tools, and no MCP or skills selection.

- [ ] **Step 2: Compare the unrelated-record hashes from Task 1**

Expected: both hashes are byte-for-byte unchanged.

### Task 5: Verify Real Agent Responses

- [ ] **Step 1: Ask one representative question per agent through `/api/v1/agent-chat/:session_id`**

Use fresh UUID session IDs, `agent_enabled=true`, `web_search_enabled=false`, and these questions:

```text
ASP-JP1 专家:
JP1/AJS3 中如何确认作业网络的执行状态？请列出相关命令、主要输出项和注意事项，并在相关段落内引用依据。

ASP-BSP-RM 账票专家:
ASP-BSP-RM 中账票输出失败时，应如何根据消息代码和打印机状态进行排查？请给出步骤并在相关段落内引用依据。
```

Capture the SSE stream to a temporary file on the remote host. Require an `answer` event with `done=true`, at least one inline `<kb doc=` citation in the assembled answer, and at least one reference/tool result.

- [ ] **Step 2: Prove every returned document belongs to the bound KB**

Extract referenced `knowledge_id` values from each SSE stream, reject any value that is not a UUID, and build the read-only query as follows:

```powershell
$quotedIDs = $referenceIDs | ForEach-Object {
  if ($_ -notmatch '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$') {
    throw 'SSE returned an invalid knowledge ID'
  }
  "'$_'"
}
$referenceSQL = @"
SELECT id, knowledge_base_id
FROM knowledges
WHERE id IN ($($quotedIDs -join ','));
"@
$referenceSQL | docker exec -i Xelora-postgres `
  psql -U postgres -d WeKnora -t -A
```

Expected: every JP1 reference maps only to `1cf679e6-be1f-4a63-a825-b8d4b284879a`; every BSP-RM reference maps only to `f106b6d6-a268-4fc1-8fb5-15f42f408951`.

- [ ] **Step 3: Verify evidence-gap behavior**

Ask each agent in Chinese:

```text
请仅依据当前绑定知识库说明 Apache Kafka 4.0 的 KRaft 控制器配置；如果库内没有依据，请明确说明证据不足，不要使用外部知识。
```

Expected: the answer states that the bound knowledge base lacks sufficient evidence, contains no `<web url=` citation, and does not present Kafka configuration as fact.

### Task 6: Clean Up And Report

- [ ] **Step 1: Remove only the verified temporary artifacts**

```powershell
$remoteTemp = 'D:\Knavis\tmp\asp-agent-provision'
if ((Resolve-Path $remoteTemp).Path -eq $remoteTemp) {
  Remove-Item -LiteralPath $remoteTemp -Recurse -Force
}
Remove-Item -LiteralPath '.tmp\asp-agent-api-key' -Recurse -Force
```

Expected: both exact temporary paths are gone; no application or database files are removed.

- [ ] **Step 2: Record the final result**

Report the two agent IDs, create/update actions, bound KB names and IDs, persisted model/tool/Web settings, representative-question citation and reference checks, evidence-gap checks, and any verification limitation. Do not report API keys, AES keys, raw authentication headers, or full internal prompts.
