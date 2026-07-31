# Personal Pi Workspace Runtime Design

**Date:** 2026-07-31  
**Status:** Draft for review  
**Scope:** Personal edition only — workspace-scoped Skill / file / shell execution via Pi coding-agent SDK sidecar  
**Non-scope:** Docker / Enterprise sidecar; replacing Go agent loop; migrating KB retrieval to Pi; replacing UI with Pi-web

---

## 1. Problem

Personal now binds a local folder as the session workspace (`workspace_binding.root_path`, `binding_kind=local_folder`). Skill scripts, file ops, and shell still rely on a homegrown Go executor path that is harder to harden and keep aligned with mature coding-agent practice.

Open-source stacks such as Pi (coding-agent SDK) and Pi-web show a stable pattern: **cwd = workspace**, progressive skills, and permissioned file/shell tools. We want that execution maturity **without** discarding WeKnora’s Progressive Agentic RAG and existing tool orchestration, which are tuned for knowledge-base answer accuracy.

---

## 2. Goals

1. On Personal, execute workspace Skill / file / shell through a Pi-based runtime.
2. Keep the Go agent as the **only decision loop** (KB, MCP, business tools, SSE contract unchanged).
3. Preserve path confinement: all Pi ops use server-resolved `root_path` as cwd.
4. Keep destructive ops behind the existing frontend confirmation UX.
5. Package and lifecycle-manage the sidecar with Personal; no orphan Node processes.

## 3. Non-goals

- Pi as a second agent / sub-planner with its own conversation loop for normal turns.
- Moving `knowledge_search` / Progressive RAG orchestration / Chat Pipeline into Pi.
- Enabling the sidecar on Docker or Enterprise in this phase.
- Replacing the WeKnora frontend with Pi-web.
- Rewriting session or SSE protocols.
- A server-side registry of local folders (recent list stays client-side).

---

## 4. Locked decisions

| Topic | Decision |
|---|---|
| Architecture | **Go main agent + Pi as embedded tool backend** (not Pi-as-main-loop) |
| Tool surface via Pi | Skill + file + shell (full cwd tool surface) |
| Edition | **Personal only**; other editions keep current Go executor |
| KB / RAG | **Stay in Go**; no behavioral change required this phase |
| Integrity model | One tool table, one SSE stream; Pi is a dumb workspace runtime |
| Transport | Prefer **stdio** child process; localhost socket acceptable fallback |
| cwd source | `sessions.workspace_binding.root_path` only (server-enforced) |
| Destructive ops | Frontend confirm before apply (existing mutation confirm path) |

---

## 5. Architecture

```text
[Frontend / Wails]
        │  SSE / API (unchanged)
        ▼
[Go Agent Loop]  ←── KB / RAG / MCP / business tools (unchanged)
        │
        │  Personal + status=bound + valid root_path
        ▼
[Workspace Tool Facade]  ← Go: bind resolve, path checks, permission gate, error map
        │  stdio RPC (versioned protocol)
        ▼
[Pi Workspace Sidecar · Node]  ← coding-agent SDK tool factories, skills, shell
        │
        ▼
[Local folder = root_path]
```

### 5.1 Component responsibilities

| Unit | Responsibility | Depends on |
|---|---|---|
| Go Agent Loop | Plan, call tools, stream events, answer with citations | Existing agent engine, EventBus |
| Workspace Tool Facade | Decide Personal vs legacy path; inject cwd; call sidecar; map results to `ToolResult` | Session binding resolver, sidecar client |
| Pi Sidecar | Execute cwd-scoped skill/file/shell; no independent multi-turn planning for the user session | Pi coding-agent SDK, local filesystem |
| Mutation / permission gate | Propose destructive ops; wait for UI apply/reject | Existing mutation store + SSE events |
| Binding layer | Open Folder / recent list / `local:` ids (already shipped) | Unchanged |

### 5.2 Boundary rules

**Goes to Pi (Personal only):**

- Skill read / script execute (and skill discovery layout aligned with Pi where practical)
- File read / write / edit / list
- File delete / rename / move (after confirm)
- Shell in workspace

**Never goes to Pi:**

- `knowledge_search`, `grep_chunks`, `list_knowledge_chunks`, graph/wiki/KB tools
- MCP and other non-cwd business tools
- Session title, thinking/answer streaming machinery

**Non-Personal:** Facade short-circuits to the existing Go executor; sidecar is not started.

---

## 6. Tool protocol

### 6.1 LLM-facing surface

The model continues to see **one** WeKnora tool registry. Tool names may stay as today or be normalized to a small cwd set; schemas remain owned by Go. Pi is an implementation detail behind the Facade.

### 6.2 Facade → Sidecar request (minimum)

```text
session_id
correlation_id
protocol_version
tool_name
args                  # JSON, tool-specific
cwd                   # absolute root_path from server binding only
permission_mode       # e.g. ask | elevated after user confirm
timeout_ms
```

Rules:

- Model/client **cannot** supply an alternate root. Any `cwd`/`root_path` in tool args is ignored or rejected.
- Unbound / invalid binding → fail in Facade with `workspace_required` / `binding_invalid` without calling Pi.
- Response maps to existing `types.ToolResult` (`Success`, `Output`, `Error`, `Data`) and existing EventBus tool events so the UI stays unchanged.

### 6.3 Sidecar response (minimum)

```text
ok
error_code            # path_out_of_workspace | timeout | runtime_unavailable | ...
content / stdout
artifacts             # optional structured paths touched
duration_ms
```

### 6.4 Skills

- Phase 1: Go may still inject Level-1 skill metadata into the system prompt (Progressive Disclosure UX unchanged).
- `read_skill` / `execute_skill_script` **execution** runs in Pi with `cwd = root_path`.
- Prefer Pi-aligned on-disk layout (e.g. project `.agents/skills`, agent skills dirs) to avoid dual conventions long term.
- Skill script outputs and file side effects are confined to cwd (same confinement as file/shell tools).

---

## 7. Permissions and destructive ops

Product rule (unchanged): **frontend confirmation before applying** delete / rename / move (and similarly dangerous shell when gated).

Flow:

1. Facade/Pi prepares a proposal (or Pi signals ask-mode).
2. Go emits the existing confirmation SSE (e.g. `workspace_mutation_required`).
3. User apply/reject via current APIs/UI cards.
4. Only after apply does the destructive op execute in the sidecar.

Pi’s ask/full permission modes may be used **internally**; users only see the existing WeKnora confirmation UX — not a second permission system.

---

## 8. Process lifecycle and packaging

### 8.1 Lifecycle

1. Personal app start → spawn sidecar → protocol handshake (version + ready).
2. On first cwd-tool use: if not ready, brief wait + one restart; then `runtime_unavailable`.
3. Heartbeat; consecutive failures mark unhealthy and trigger relaunch on next use.
4. App quit (and best-effort on crash): terminate child; no orphan Node.

Preferred IPC: **stdio**. Localhost TCP/Unix socket is a fallback if stdio proves impractical with Wails packaging.

### 8.2 Packaging

- Ship sidecar (and Node runtime if required) **only** with Personal.
- Pin versions: Facade `protocol_version` must match sidecar; mismatch → refuse exec + upgrade hint.
- Docker/Enterprise builds omit sidecar binaries and never start the process.

### 8.3 Binding interaction

- Continue using `workspace_binding.root_path` from the Personal local-folder design.
- Sidecar is **stateless** across calls regarding project registry; every call carries cwd.
- Missing directory is detected in Facade (`Stat`) before RPC.

---

## 9. Error handling

| Condition | Behavior |
|---|---|
| No / invalid binding | `workspace_required` / `binding_invalid`; no Pi call |
| Sidecar down / crash | One relaunch; then `runtime_unavailable` tool error |
| Path escapes cwd | `path_out_of_workspace` |
| User rejects mutation | `mutation_rejected`; agent loop continues |
| Tool timeout | Cancel that job; keep sidecar alive unless unresponsive |
| Protocol mismatch | Refuse execution; prompt Personal upgrade |

Do not silently fall back to unconstrained local execution. Do not dump sidecar stacks into user-visible answers; log with `correlation_id`.

---

## 10. Knowledge-base posture (explicit)

Analysis of the current agent KB path shows accuracy depends on:

- Progressive RAG prompt (`grep_chunks` → `knowledge_search` → `list_knowledge_chunks` → cite)
- HybridSearch + RRF + rerank + MMR
- Multi-tenant `SearchTargets` and history redact of prior KB tool results
- SSE/EventBus entanglement with the Go ReAct loop

**This design does not migrate that stack.** Optional future work (out of scope here): extract HybridSearch/Grep/List as a Go KB Tool RPC so a later “Pi-as-main-loop” experiment could call them without forking retrieval math. That extraction is **not** required to ship the Personal Pi workspace runtime.

---

## 11. Testing

- **Unit (Go):** Facade routing (Personal vs not), cwd injection, path checks, error mapping, protocol mismatch.
- **Contract:** Golden JSON fixtures for Facade ↔ sidecar RPC.
- **Integration (Personal):** Open Folder → create/edit file → run skill → shell → confirm delete/rename.
- **Regression:** Existing Progressive RAG / KB agent tests remain green; sample QA unchanged.
- **Edition guard:** Non-Personal CI asserts sidecar is not started and cwd tools use Go executor.

---

## 12. Success criteria

1. Personal: Skill / file / shell for a bound folder execute via Pi with no path escape.
2. Same session can interleave KB tools and workspace tools without a mode switch.
3. KB Progressive RAG behavior matches pre-change baseline.
4. Quitting Personal leaves no leftover sidecar process; sidecar failure does not crash the app.

---

## 13. Rollout

1. Define versioned RPC + Go sidecar client stub (fake sidecar for tests).
2. Implement Facade behind a Personal feature flag; route file/skill/shell tools.
3. Bundle real Pi sidecar; handshake + heartbeat.
4. Wire destructive ops through existing mutation confirm.
5. Manual Personal smoke + KB regression; then enable by default on Personal.

---

## 14. Open points (resolved for v1)

| Item | Resolution |
|---|---|
| Pi main loop vs tool backend | Tool backend only |
| Editions | Personal only |
| KB migration | Not in v1 |
| IPC | stdio preferred |
| Permission UX | Existing WeKnora confirm cards |

Remaining implementation choices (exact npm package pin, stdio framing codec, which Go tool names map 1:1 to Pi tool factories) belong in the implementation plan, not this spec.
