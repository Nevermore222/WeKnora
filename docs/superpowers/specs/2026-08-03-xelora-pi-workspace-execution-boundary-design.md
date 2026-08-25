# Xelora Pi Workspace Execution Boundary Design

**Date:** 2026-08-03

## Decision

Xelora keeps the Go agent loop as the only intelligence and orchestration
layer. Pi is adopted only as the Personal edition's workspace execution
backend. It must not own user conversation state, model calls, retrieval,
planning, tool selection, citations, or SSE lifecycle.

The purpose is to let one Xelora agent turn combine its existing knowledge-base
retrieval with reliable, workspace-scoped file, shell, and Skill execution.

## Scope

This design applies to Personal desktop sessions bound to a local folder.

It covers:

- Pi SDK-backed execution of workspace file, shell, and approved Skill actions.
- The durable `sessions.workspace_binding` contract.
- A single Xelora tool and SSE surface for retrieval and workspace operations.
- Xelora-owned authorization, destructive-operation confirmation, job records,
  and artifact registration.

It does not cover:

- Replacing the Go agent loop with Pi.
- Moving Progressive RAG, knowledge search, reranking, citations, MCP, or
  business tools into Pi.
- Starting a Pi sidecar in Docker or Enterprise editions.
- Introducing an independent Pi conversation, prompt, model, or session store.

## Architecture

```text
Frontend / Wails
  -> Xelora API and SSE
       -> Go Agent Loop
            -> Progressive RAG, KB tools, MCP, answer construction
            -> Workspace Runtime Facade
                 -> Pi SDK sidecar over versioned NDJSON stdio
                      -> bound local folder only
```

The Go agent decides whether to retrieve knowledge, operate on files, execute a
Skill, or combine those actions. Pi receives a single, already-authorized
workspace operation and returns a structured result. It has no authority to
call Xelora retrieval tools or make another model decision.

## Workspace Binding Contract

Personal clients select a local folder through the Wails directory picker and
create a session with `workspace_binding.root_path`. The server normalizes and
validates that path, derives a `local:<hash>` workspace ID, and persists the
binding with `binding_kind=local_folder`.

Every Pi call obtains its `cwd` from the persisted session binding. The model,
frontend, and tool arguments cannot select another root. A missing, invalid, or
inaccessible binding fails before the sidecar is called.

Docker and Enterprise retain their managed-workspace flow. They neither start
the sidecar nor accept client-supplied absolute roots.

## Responsibility Boundary

| Xelora Go owns | Pi SDK sidecar owns |
| --- | --- |
| Agent planning and tool selection | Execution of one workspace operation |
| Knowledge retrieval and answer generation | File read/write/edit/list under `cwd` |
| Session binding resolution and path authority | Shell process launched with `cwd` |
| Skill lookup and script allowlisting | Execution of an already-prepared Skill script |
| SSE events, user-visible history, job and artifact records | Structured stdout, stderr, exit status, touched paths |
| Delete/rename/move approval and final apply | No independent approval policy |

Pi SDK is used for its mature coding-workspace tool implementation, not for its
agent loop. The sidecar must not initialize Pi model providers, session
persistence, autonomous planning, or a second user-facing permission system.

## Tool Flow

For `workspace_read`, `workspace_write`, `workspace_edit`, `workspace_list`,
and `workspace_shell`:

1. The Go tool resolves the durable binding from the session ID.
2. The Workspace Runtime Facade validates `status=bound` and the absolute root.
3. The Facade sends tool name, validated arguments, `cwd`, timeout, and a
   correlation ID to the sidecar.
4. The Pi SDK tool implementation executes only inside `cwd`.
5. Go maps the result into the existing `ToolResult` and EventBus/SSE contract.

For destructive mutations, Go continues to create the proposal, emit the
existing confirmation SSE event, wait for the user decision, and apply only
after approval. Pi must not bypass this flow.

## Skill Execution Flow

`execute_skill_script` follows the same session binding but keeps Xelora's
existing security and artifact ownership:

1. Go resolves the requested skill through the existing Skill Manager.
2. Go validates the relative script path and prepares any approved input files.
3. Go constructs a trusted request containing `script_abs`, sanitized arguments,
   staged input metadata, bound `cwd`, and timeout.
4. The Pi sidecar executes only that prepared script with `cwd` fixed to the
   session workspace. It must reject arbitrary executable paths from model
   input.
5. Go snapshots the workspace before and after execution, registers changed
   files as artifacts, records job status, and emits ordinary tool events.

This preserves the current Xelora artifact and audit model while replacing only
the local workspace execution mechanism.

## Execution Isolation Policy

Pi local file and shell tools run with the desktop user's filesystem access;
they are not an execution sandbox. Go therefore selects the execution provider
after resolving the Skill metadata. Only an explicit allowlist of trusted,
Personal-local Skills may run through Pi. Skills that require container
isolation, restricted network access, Office tooling, or another existing
provider continue through the Xelora Executor Gateway and its controlled
provider.

The model cannot select the provider or weaken that policy through tool
arguments. Both routes use the same persisted binding, job, artifact, and SSE
contracts.

## Sidecar Integration Requirements

- Add and pin the official Pi coding-agent package in
  `sidecars/pi-workspace/package.json`.
- Replace the handwritten file, shell, and Skill dispatch internals with Pi SDK
  tool primitives or factories behind the existing NDJSON protocol.
- Keep the Go protocol types and tool names stable unless an explicit API
  migration is approved.
- Preserve path-confinement tests for reads, writes, edits, lists, shell, and
  scripts.
- Return structured result fields for content, stdout, stderr, exit code,
  duration, and touched paths. Go remains responsible for artifact records.

## Failure Behavior

| Condition | Required behavior |
| --- | --- |
| Unbound or invalid session workspace | `workspace_required` or `binding_invalid`; no Pi call |
| Path outside `cwd` | `path_out_of_workspace`; no filesystem side effect |
| Sidecar missing, stopped, or protocol mismatch | `runtime_unavailable` or `protocol_mismatch`; no unconstrained fallback |
| Script not prepared by Xelora | Reject before sidecar execution |
| User rejects destructive mutation | `mutation_rejected`; no Pi mutation |
| Pi action fails | Preserve stdout/stderr and mark the Xelora tool/job as failed |

## Acceptance Criteria

1. A Personal session can retrieve from the existing knowledge base and then
   create or modify a file in its bound local folder during the same turn.
2. Knowledge retrieval events, workspace tool events, artifacts, final answer,
   and failures remain on the existing Xelora SSE stream.
3. Pi receives no model credentials, retrieval context, or authority to select
   tools beyond the single request sent by Go.
4. A forged `cwd`, `root_path`, or script path cannot escape the persisted
   session workspace or Skill allowlist.
5. A Skill requiring a controlled provider cannot be forced into Pi execution
   by model or client input.
6. Docker and Enterprise behavior remains unchanged and does not launch Pi.
7. Closing Personal stops the sidecar without leaving an orphan process.

## Migration Order

1. Add Pi SDK dependency and verify its tool primitives behind the existing
   sidecar protocol.
2. Replace file/list/shell internals while retaining Go Facade tests.
3. Add an execution-provider policy and a Go-prepared `execute_skill_script`
   handoff to the sidecar for eligible Skills.
4. Verify artifact registration and destructive-confirmation regression cases.
5. Run an end-to-end Personal smoke: bind folder, retrieve KB evidence, write a
   file, execute an approved Skill, inspect artifact, and reject one mutation.
