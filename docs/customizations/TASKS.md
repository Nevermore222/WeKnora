# Xelora Secondary Development Task Board

This file is the shared cross-machine task board for Xelora secondary
development. Sync it through normal `git pull` and `git push`.

Status legend: `[ ]` pending, `[~]` in progress, `[x]` done

Task format:

```text
- [ ] T-### Title - Short description (@machine-id)
```

## Pending

<!-- Add new tasks here and keep IDs stable. -->

- [x] T-007 Executor gateway baseline - implement the first Xelora-owned gateway contract for session workspaces, jobs, logs, artifacts, and policy decisions (@win-main)
- [x] T-008 Workspace and artifact model - establish persistent session workspace ownership plus artifact-first output handling for runtime tasks (@win-main)
- [x] T-010 Experimental OpenSandbox provider - retain the OpenSandbox adapter behind the provider layer, document the current command-proxy failure, and keep it available for future provider evaluation (@win-main)
- [ ] T-012 Browser automation path - add the first browser automation provider path around the selected browser reference while preserving Xelora-owned task and artifact semantics (@win-main)
- [ ] T-013 Runtime observability and audit - add provider health, job history, artifact traceability, and policy decision auditing for the new runtime layers (@win-main)

## Done

- [x] T-001 Repository baseline - establish the forked secondary-development repository control structure (@win-main)
- [x] T-002 Collaboration workflow - write `WORKFLOW.md` for multi-machine collaboration rules (@win-main)
- [x] T-003 Development guide - write `README-dev.md` for deployment and development flow (@win-main)
- [x] T-004 Source deployment - build from source and replace the upstream images, then verify the full chain (@win-main)
- [x] T-005 Shared task board - create the `xelora-tasks` skill and `TASKS.md` workflow (@win-main)
- [x] T-006 Runtime reference architecture - finalize the broad runtime reference architecture and module ownership model for sandbox execution, gateway orchestration, workspace ownership, artifacts, browser automation, and file capability layers (@win-main)
- [x] T-009 Controlled Docker executor - build the first usable local execution provider to validate Xelora-owned workspace, job, log, and artifact contracts; verified app-container Docker CLI, Docker socket access, `--volumes-from Xelora-app`, and host-visible Markdown output via `scripts/controlled-docker-smoke.ps1` (@win-main)

- [x] T-008a Session workspace binding MVP (US1) - new chats bind to the active tenant workspace at creation time, the binding persists as durable session state, and the chat view renders bound/unbound status with i18n; backend domain types, repository persistence, service validation, handler wiring, and frontend store/view/API are all landed; see specs/002-session-workspace-binding/tasks.md for task-level detail (@win-main)
- [x] T-008b Session workspace binding full delivery (US2+US3) - runtime output routing to bound workspace, boundary enforcement, path-escape rejection, invalid-binding recovery, and legacy unbound compatibility; all 34 tasks in specs/002-session-workspace-binding/tasks.md complete; migration 000063 applied and verified in running app (@win-main)
- [x] T-011 File capability bridge - connect Markdown, PDF, spreadsheet, and presentation capability paths through the runtime artifact model using the current reference modules; OfficeCLI sandbox POC is verified via `scripts/officecli-smoke.ps1`, the preloaded Office wrapper skill `officecli-document-editing` is verified via Python unit tests and Chrome E2E, and the structured text artifact skill `workspace-file-writer` is verified via Python unit tests and Chrome E2E. 2026-07-11 evidence: local workspace `Browser E2E 20260711` bound to session `90a848be-de2b-4aa8-8e21-fbcb8e4c14f7`, host files `browser-e2e-report.md`, `browser-e2e-brief.docx`, `browser-e2e-sheet.xlsx`, and `browser-e2e-slides.pptx` created under `data/workspaces-e2e`; reopened Office edit persisted, `../escape.md` was blocked, unbound session `b6b9e257-f54b-4eec-aaf8-bfcf6937b411` returned `workspace_required`, and no E2E output appeared under `skills/preloaded` (@win-main)
- [x] T-014 Office and XLSX output hardening - add high-level `write_docx` and `write_xlsx` actions, route generic `xlsx/scripts/create_xlsx.py` calls into the same workbook writer, document the preferred skill path, and verify Python plus container smoke tests for generated Excel files (@win-main)
