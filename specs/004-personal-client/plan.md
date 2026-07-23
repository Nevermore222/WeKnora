# 004 — Xelora Personal Client: Implementation Plan

Status: Draft
Created: 2026-07-21
Depends on: 001 (executor gateway), 002 (workspace binding)

## Technical Context

- Backend: Go 1.26, Gin, GORM, SQLite + sqlite-vec (already supported via DB_DRIVER=sqlite)
- Frontend: Vue 3.5 + TypeScript + Pinia (existing frontend/ directory)
- Desktop shell: Wails v2 (existing cmd/desktop/, wails.json, app.go, main.go)
- Existing desktop app name: "Xelora Lite" → rename to "Xelora Personal"
- Build target: Windows amd64 single EXE

## Staged Delivery

### Stage 1: Single-User Local Core (P0)

Goal: Personal client boots into a fully functional local-only mode with zero
configuration. No enterprise server needed.

Tasks:

- [ ] T-101 Rename product identity from "Xelora Lite" to "Xelora Personal"
  - wails.json: name, outputfilename, info.productName
  - cmd/desktop/main.go: menu labels, about dialog
  - cmd/desktop/prefs.go: config dir name
  - Frontend: title, branding strings

- [ ] T-102 Single-user auto-initialization
  - On first launch (empty SQLite), auto-create default tenant + admin user
  - Skip login/registration flow; auto-authenticate with local session token
  - Add `internal/runtime/personal_bootstrap.go`
  - Frontend: detect personal mode, hide auth pages, show direct chat

- [ ] T-103 Personal mode config profile
  - Add `config/config.personal.yaml` with SQLite defaults
  - cmd/desktop loads personal profile by default
  - Disable multi-tenant features (org management, invitations, RBAC UI)
  - Keep knowledge base, agent, skill, workspace features active

- [ ] T-104 Local skill execution mode selector
  - Add ExecutionMode to executor gateway config
  - Default: local process execution (existing LocalProvider)
  - Optional: Docker sandbox (existing ControlledDockerProvider)
  - Frontend: skill execution settings panel (local vs sandbox toggle)
  - Persist preference in desktop-prefs.json

- [ ] T-105 Windows EXE build pipeline
  - Verify `wails build -platform windows/amd64` produces working EXE
  - Embed frontend dist via Wails asset server
  - Test SQLite + sqlite-vec in packaged binary
  - Add `scripts/build-personal.ps1` build script

### Stage 2: Enterprise Connector (P0)

Goal: Connect to LAN Xelora Server, browse and use enterprise knowledge bases,
agents, and skills in real-time.

Tasks:

- [ ] T-201 Enterprise server data model
  - Create `internal/enterprise/types.go`: ServerConfig, ConnectionStatus,
    ServerCapabilities, RemoteKnowledgeBase, RemoteAgent, RemoteSkill
  - Create `internal/enterprise/store.go`: SQLite persistence for server configs
  - Migration: enterprise_servers table, enterprise_resource_cache table

- [ ] T-202 Enterprise connector core
  - Create `internal/enterprise/connector.go`: Connector struct, connection
    lifecycle (connect/disconnect/reconnect), health check loop
  - HTTP client with configurable timeout and retry
  - API token authentication header injection
  - Graceful degradation: mark resources offline on connection loss

- [ ] T-203 Enterprise capability discovery
  - On connect: GET /api/v1/system/info → server version, capabilities
  - GET /api/v1/knowledgebases → enterprise KB list
  - GET /api/v1/custom-agents → enterprise agent list
  - GET /api/v1/skills → enterprise skill list
  - Cache results in enterprise_resource_cache for instant UI on reconnect

- [ ] T-204 Enterprise API proxy handlers
  - Create `internal/handler/enterprise.go`: EnterpriseHandler
  - POST /api/v1/enterprise/servers (CRUD)
  - POST /api/v1/enterprise/servers/:id/test
  - POST /api/v1/enterprise/servers/:id/connect | disconnect
  - GET /api/v1/enterprise/servers/:id/status
  - GET /api/v1/enterprise/resources (aggregated)
  - Register routes in router.go

- [ ] T-205 Enterprise chat proxy
  - POST /api/v1/enterprise/chat: proxy agent-chat to remote server
  - Stream SSE responses back to local frontend
  - Attach enterprise session context (server_id, remote_session_id)
  - Handle auth errors, timeouts, server offline gracefully

- [ ] T-206 Enterprise retrieval proxy
  - POST /api/v1/enterprise/retrieval: proxy knowledge search to remote
  - Return citations with source server attribution
  - Support streaming chunk responses

- [ ] T-207 Enterprise skill execution proxy
  - POST /api/v1/enterprise/skill/execute: proxy skill execution to remote
  - Return artifacts with download URLs pointing to remote server
  - Timeout and error handling

### Stage 3: Frontend Integration (P0)

Goal: UI clearly distinguishes local vs enterprise resources, provides server
connection management, and unified interaction experience.

Tasks:

- [ ] T-301 Server connection manager UI
  - New settings panel: "Enterprise Servers"
  - Add/edit/delete server entries (name, URL, token)
  - Test connection button with status feedback
  - Connection status indicator in top bar (green/red/reconnecting)
  - Auto-connect on startup toggle

- [ ] T-302 Resource origin badges
  - Knowledge base list: "Local" / "Enterprise: {server_name}" badge
  - Agent list: same origin badges
  - Skill list: same origin badges
  - Enterprise resources: read-only indicator, no edit/delete buttons

- [ ] T-303 Unified resource lists
  - Merge local + enterprise resources in KB/agent/skill list views
  - Filter tabs: All / Local / Enterprise
  - Enterprise resources sorted after local, grouped by server
  - Loading states while fetching enterprise resources

- [ ] T-304 Enterprise chat integration
  - Agent selector includes enterprise agents (with origin badge)
  - KB selector includes enterprise KBs (with origin badge)
  - Chat stream handles enterprise-proxied responses
  - Citation display shows enterprise source attribution

- [ ] T-305 Personal mode UI simplification
  - Hide: organization management, tenant switching, member invitations
  - Hide: user registration/login pages
  - Show: simplified settings (model config, skill execution mode, servers)
  - Keep: knowledge base CRUD, agent CRUD, skill studio, workspace files

- [ ] T-306 Skill execution mode UI
  - Skill execution dialog: mode selector (Local / Docker Sandbox)
  - Show Docker availability status (detected / not installed)
  - Persist last-used mode preference
  - Warning when selecting Docker mode without Docker installed

### Stage 4: Hardening & Polish (P1)

Tasks:

- [ ] T-401 Credential security
  - Store API tokens in Windows Credential Manager (via keyring library)
  - Never persist tokens in plain text SQLite
  - Clear token on server removal

- [ ] T-402 mDNS server discovery (optional)
  - Broadcast/receive mDNS service announcements for Xelora Server
  - Auto-populate server list with discovered instances
  - User confirms before connecting

- [ ] T-403 Offline resilience
  - Enterprise resources cached from last successful connection
  - Stale cache indicator with "last synced" timestamp
  - Retry connection with exponential backoff
  - Queue enterprise requests during brief disconnections

- [ ] T-404 Auto-update for personal client
  - Separate update feed from server edition
  - Delta update support (bsdiff)
  - Changelog display before applying update

- [ ] T-405 Packaging and installer
  - NSIS or WiX installer for Windows
  - Desktop shortcut, Start Menu entry
  - Uninstaller with data preservation option
  - Code signing (if certificate available)

## Source Code Layout (new files)

```
internal/enterprise/
├── types.go              # ServerConfig, ConnectionStatus, capabilities
├── store.go              # SQLite persistence for server configs
├── connector.go          # Connection lifecycle, health check
├── discovery.go          # mDNS discovery (Stage 4)
├── proxy.go              # HTTP proxy for enterprise API calls
└── connector_test.go

internal/handler/
└── enterprise.go         # Enterprise CRUD + proxy handlers

internal/runtime/
└── personal_bootstrap.go # Single-user auto-init

frontend/src/
├── api/enterprise/
│   └── index.ts          # Enterprise API client
├── stores/
│   └── enterprise.ts     # Enterprise connection state (Pinia)
├── components/
│   ├── EnterpriseServerManager.vue
│   ├── ResourceOriginBadge.vue
│   └── ConnectionStatusIndicator.vue
└── views/settings/
    └── EnterpriseServersPanel.vue

config/
└── config.personal.yaml  # Personal mode defaults

scripts/
└── build-personal.ps1    # Windows EXE build script
```

## Milestones

| Milestone | Deliverable | Est. |
|-----------|-------------|------|
| M1 | Local-only personal client boots, creates KB/agent/skill | Stage 1 |
| M2 | Enterprise server connects, resources visible | Stage 2 |
| M3 | Full UI integration, unified experience | Stage 3 |
| M4 | Hardened, packaged, distributable EXE | Stage 4 |

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| sqlite-vec CGO in Wails build | Build failure | Test early in T-105; fallback to pure-Go vec if needed |
| Enterprise server API version mismatch | Proxy errors | Version negotiation on connect; graceful degradation |
| WebView2 not available on old Windows | App won't start | Bundle Evergreen Bootstrapper in installer |
| Docker not installed | Sandbox mode unavailable | Detect and disable; show install guidance |
| Large enterprise KB retrieval latency | Poor UX | Streaming responses; timeout with partial results |

## Verification

- Stage 1: Launch EXE → auto-login → create KB → upload doc → query → create agent → chat → execute skill locally
- Stage 2: Add server → connect → see enterprise KBs/agents → query enterprise KB → chat with enterprise agent
- Stage 3: Verify origin badges, filter tabs, unified lists, connection status indicator
- Stage 4: Kill server → verify graceful degradation → restart → verify reconnect
