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
- [x] T-101 Personal client rebrand - rename product identity from "Xelora Lite" to "Xelora Personal" across wails.json, cmd/desktop, CI workflow, frontend, and logger (@win-main)
- [x] T-102 Personal single-user auto-init - extend AutoSetup edition gate to accept "personal", frontend router guard auto-setup flow works unchanged, menu badge shows dynamic edition label (@win-main)
- [x] T-103 Personal mode config profile - add .env.personal.example with SQLite/memory-stream/local-storage defaults, add personal_defaults.go runtime injection, desktop main.go loads .env.personal fallback (@win-main)
- [x] T-104 Skill execution mode selector - selectProvider respects XELORA_SANDBOX_MODE env, desktop prefs persist sandbox_mode, Wails bindings Get/SetDesktopSandboxMode, frontend settings UI with i18n (@win-main)
- [x] T-105 Windows EXE build script - add scripts/build-personal.ps1 with frontend build, wails build (edition=personal ldflags), and standalone packaging (config, migrations, skills) (@win-main)
- [x] T-201 Enterprise connector core - internal/enterprise/ package: ServerConfig, Connector lifecycle, health check, capability discovery, API proxy; DI wiring in container.go; routes registered in router.go (@win-main)
- [x] T-301 Enterprise frontend integration - API client, Pinia store, EnterpriseServerManager.vue, ResourceOriginBadge enterprise variant, ConnectionStatusIndicator.vue, full i18n (zh-CN + en-US), vue-tsc passes (@win-main)
- [x] T-303 Unified resource lists - enterprise KBs merged into KnowledgeBaseSelector with prefixed IDs and origin badges; enterprise agents merged into AgentSelector as a dedicated group (@win-main)
- [x] T-304 Enterprise chat integration - AgentSelector emits select-enterprise with server context; KnowledgeBaseSelector tracks enterprise KB selection for proxy routing (@win-main)
- [x] T-305 Personal mode UI simplification - reuses existing lite-mode menu hiding (logout/organizations), tenant switcher gated by canAccessAllTenants (false for single user) (@win-main)
- [x] T-401 Credential security - DPAPI token encryption (credentials_windows.go via crypt32 CryptProtectData/CryptUnprotectData), non-Windows fallback, integrated into store Create/Update/Get/List (@win-main)
- [x] T-402 mDNS server discovery - internal/enterprise/discovery.go via grandcat/zeroconf (_xelora._tcp), GET /enterprise/discover endpoint (@win-main)
- [x] T-403 Offline resilience - exponential backoff auto-reconnect (5s→5min), capability refresh on recovery, LastSyncedAt staleness exposed via /enterprise/servers/:id/status (@win-main)
- [x] T-404 Personal auto-update feed - XELORA_UPDATE_FEED_URL env override in update.go, defaults to upstream GitHub releases (@win-main)
- [x] T-405 Packaging and installer - scripts/installer-personal.nsi (NSIS, LZMA, shortcuts, uninstaller preserving user data), pairs with build-personal.ps1 (@win-main)
- [x] T-501 Client-server sync Phase 1 (auto-provisioning) - server: ProvisionClientUser service + handler + Admin-gated POST /tenants/:id/client-users; client: ServerConfig linked-identity fields (DPAPI-encrypted), connector.ProvisionUser (discover tenant → provision → login → JWT), proxy JWT auth + 401 refresh, ConnectServer auto-provisions local user, frontend linked-status display; see specs/005-client-server-sync (@win-main)
- [x] T-502 Client-server sync Phase 2 (access shared resources) - connector.RefreshSharedResources fetches /shared-knowledge-bases + /shared-agents with the linked JWT (called after provisioning), merges into capabilities with Shared/Permission/OrgName; /enterprise/resources surfaces them; selectors show shared resources with org-name badge; chat routes through proxy (server auto-resolves shared resource) (@win-main)
- [ ] T-503 Client-server sync Phase 3 (push local data) - one-way push of local agents/KBs (with documents) to the server, with push-state tracking (@win-main)
- [ ] T-504 Client user identity model fix (006) - change ProvisionClientUser to give the provisioned client user an independent home tenant (Owner), not membership in the admin tenant, so the admin can share arbitrary organization spaces with the user via SearchTenantsForInvite and per-space OrgMemberRole permissions take effect; see specs/006-client-user-identity-and-sharing (@win-main)
- [ ] T-505 Enterprise proxy CRUD for KB/agent (006) - add /enterprise/knowledge-bases and /enterprise/agents proxy routes (create/list/get/update/delete + KB document upload) forwarded with the linked JWT; reorder connect flow to provision-then-discover-with-JWT so the client sees own home-tenant resources (@win-main)
- [ ] T-506 Frontend server-side KB/agent creation (006) - enterprise API client + origin switch in create dialogs posting to /enterprise/*; own-server ResourceOriginBadge variant; i18n (@win-main)

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
- [x] T-014 Office and XLSX output hardening - add high-level `write_docx`, `write_xlsx`, staged `run_python`, and controlled `officecli` passthrough actions, route generic `xlsx/scripts/create_xlsx.py` calls into the same workbook writer, add concise agent prompt rules to keep Office file work on the OfficeCLI bridge, surface failed script stderr/stdout summaries, handle locked Office files with pending-output diagnostics, and verify Python plus container smoke tests for generated and styled Excel files (@win-main)
