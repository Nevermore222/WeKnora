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

- [ ] T-007 Executor gateway baseline - implement the first Xelora-owned gateway contract for session workspaces, jobs, logs, artifacts, and policy decisions (@win-main)
- [ ] T-008 Workspace and artifact model - establish persistent session workspace ownership plus artifact-first output handling for runtime tasks (@win-main)
- [ ] T-009 Local provider stub - build a restricted local execution provider to validate runtime contracts before CubeSandbox integration (@win-main)
- [ ] T-010 CubeSandbox adapter - integrate CubeSandbox as the first sandbox baseline through a replaceable provider layer and document the WSL/Linux local development path (@win-main)
- [ ] T-011 File capability bridge - connect Markdown, PDF, spreadsheet, and presentation capability paths through the runtime artifact model using the current reference modules (@win-main)
- [ ] T-012 Browser automation path - add the first browser automation provider path around the selected browser reference while preserving Xelora-owned task and artifact semantics (@win-main)
- [ ] T-013 Runtime observability and audit - add provider health, job history, artifact traceability, and policy decision auditing for the new runtime layers (@win-main)

## Done

- [x] T-001 Repository baseline - establish the forked secondary-development repository control structure (@win-main)
- [x] T-002 Collaboration workflow - write `WORKFLOW.md` for multi-machine collaboration rules (@win-main)
- [x] T-003 Development guide - write `README-dev.md` for deployment and development flow (@win-main)
- [x] T-004 Source deployment - build from source and replace the upstream images, then verify the full chain (@win-main)
- [x] T-005 Shared task board - create the `xelora-tasks` skill and `TASKS.md` workflow (@win-main)
- [x] T-006 Runtime reference architecture - finalize the broad runtime reference architecture and module ownership model for sandbox execution, gateway orchestration, workspace ownership, artifacts, browser automation, and file capability layers (@win-main)
