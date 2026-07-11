# Workspace-Bound File Output Progress

Updated: 2026-07-12

This document summarizes the current implementation status for workspace-bound
agent file output, Office document generation, and the local runtime execution
path used during Xelora secondary development.

## Current Status

The workspace binding and file output path is now functionally complete for the
local Docker-backed validation target.

- New chat creation can carry a durable `workspace_binding` payload.
- Session read/update paths persist and return the binding state.
- The chat UI displays bound, unbound, and invalid binding states.
- File-producing skills receive the conversation workspace context.
- Runtime jobs stage input files under `.xelora/jobs/<job-id>` inside the
  bound workspace and route detected artifacts back to that workspace.
- Unsafe output paths and invalid bindings are blocked instead of silently
  falling back to a skill-private directory.
- Legacy sessions without a binding remain supported and explicitly unbound.

## Runtime Architecture

The active local execution path is:

1. The session service stores the conversation workspace binding.
2. The agent tool layer attaches that binding to skill execution requests.
3. The executor gateway validates the binding and prepares a job workspace.
4. Controlled Docker runs the skill script with shared access to the app data
   volume via `--volumes-from Xelora-app`.
5. The gateway scans post-run changes, records artifacts, and reports the
   workspace context in the tool result.

OpenSandbox remains wired as an experimental provider path. Controlled Docker is
the validated local provider for this delivery.

## File Capabilities

### Structured Text Files

`workspace-file-writer` provides safe Markdown, text, JSON, and CSV file
creation or append operations through JSON request files. It rejects absolute
paths and `..` escapes.

### Office Files

`officecli-document-editing` provides a stable wrapper for real Office files:

- `write_docx` creates Word documents from title and paragraphs.
- `write_xlsx` creates Excel workbooks from structured sheets, headers, and
  rows in one script call.
- Lower-level OfficeCLI actions are still available for existing document
  inspection and edits.

The `xlsx` skill now has a compatibility wrapper at `scripts/create_xlsx.py`.
If the model selects the generic spreadsheet skill and calls that script, the
wrapper delegates to the same `write_xlsx` path instead of failing with a
missing script or path error.

## Validation Evidence

Local checks that passed during this delivery:

```powershell
python -m unittest discover -s skills\preloaded\officecli-document-editing\scripts -p '*_test.py'
python -m unittest discover -s skills\preloaded\xlsx\scripts -p '*_test.py'
```

Container check that passed:

```powershell
docker run --rm --volumes-from Xelora-app -w /workspaces/Test0711 `
  wechatopenai/xelora-sandbox:latest `
  python3 /app/skills/preloaded/xlsx/scripts/create_xlsx.py generated-compat.md
```

Browser E2E evidence:

- The `write_xlsx` smoke prompt generated `excel-write-xlsx-smoke.xlsx`.
- The UI reduced the flow to `3` thinking rounds and `2` tool calls.
- The workbook was readable with `openpyxl` and contained the expected sheet,
  headers, and row data.

## Important Gotchas

- When the frontend or app image is rebuilt, rebuild the frontend `dist` before
  rebuilding the frontend container.
- Changes under `skills/preloaded/` are mounted into the app container, but the
  app may need a restart for the skill manager to reload instructions.
- PowerShell `Set-Content -Encoding UTF8` can emit a BOM on older Windows paths;
  skill JSON request files should be written without BOM or read with
  `utf-8-sig`.
- Do not commit `data/` smoke-test workspaces or generated Office artifacts.
- Do not rely on `builtin-smart-reasoning` for file-writing validation; use a
  custom agent with file/script tools enabled.

## Remaining Work

- Browser automation provider work is still in progress under
  `specs/003-browser-automation/`.
- Runtime observability and audit history are still pending.
- OpenSandbox needs real live-provider credentials and a successful end-to-end
  smoke run before it can replace Controlled Docker as the default local proof.
