# Progress: Personal local workspace + Pi runtime

**Branch:** `002-session-workspace-binding`  
**Date:** 2026-07-31  
**Remote:** `origin/002-session-workspace-binding`

## Specs & plans

| Doc | Path |
|---|---|
| Local folder design | `docs/superpowers/specs/2026-07-31-personal-local-folder-workspace-design.md` |
| Local folder plan | `docs/superpowers/plans/2026-07-31-personal-local-folder-workspace.md` |
| Pi workspace runtime design | `docs/superpowers/specs/2026-07-31-personal-pi-workspace-runtime-design.md` |
| Pi workspace runtime plan | `docs/superpowers/plans/2026-07-31-personal-pi-workspace-runtime.md` |

## Locked architecture

- Go agent loop remains the only decision brain (KB / Progressive RAG / MCP unchanged).
- Personal: cwd file/shell tools go through Go Facade → Node sidecar (NDJSON stdio).
- Docker/Enterprise: no sidecar; managed workspace binding only.
- Destructive delete/rename/move: existing frontend confirm → Go apply.

## Shipped (high-signal commits)

- Local folder bind + Open Folder + mutation confirm UI
- `internal/piruntime` protocol / client / facade / lifecycle
- `sidecars/pi-workspace` Node runtime
- Personal agent tools: `workspace_read|write|edit|list|shell`
- Fix: clear sticky Personal mode when API edition is not Personal (`6096d4de`)

## Follow-ups

1. Route `execute_skill_script` through Facade/`script_abs` after prepare.
2. Optionally swap sidecar internals to Pi coding-agent SDK tool factories (RPC surface stays).
3. Rebuild Docker **app** image if server-side `root_path` rejection / Personal gates are required in containers (frontend fix already deployed for sticky UI).
4. Create GitHub PR when `gh auth login` is available.

## Ops reminder

| Client | Workspace to select |
|---|---|
| Docker / Xelora Server web | Managed workspace (e.g. `/workspaces/...`) |
| Personal desktop | Local folder via Open Folder (`root_path`) |
