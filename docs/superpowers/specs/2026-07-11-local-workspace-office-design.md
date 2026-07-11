# Local Workspace And Office Editing Design

**Date:** 2026-07-11

**Status:** Approved for planning

## Goal

Allow a user to bind a newly created conversation to a persistent folder under
an administrator-approved Windows host root. File-producing tools must create,
read, and edit real files in that folder throughout the conversation. The same
workspace contract applies to Markdown, JSON, CSV, Word, Excel, PowerPoint,
browser captures, and later file-capability providers.

## Current Gap

The existing session `workspace_binding` records tenant-oriented metadata, but
the frontend does not let the user select a host folder. Skill scripts still run
from skill-private directories, and controlled Docker execution inherits the app
container's mounts through `--volumes-from`. A successful text response therefore
does not prove that a file was persisted in a user-visible workspace.

## Selected Approach

Use a configured host workspace root and bind-mount it into the app container.
Conversations select existing subdirectories of that root or create new ones.
The browser never submits an arbitrary host path, and the backend never exposes
the Windows path to agents or sandbox providers.

Example deployment mapping:

```text
Windows host: F:\XeloraWorkspaces
App container: /workspaces
Conversation:  /workspaces/quarterly-review
```

The deployment root is an administrator setting. A conversation binding stores
a stable workspace ID, display name, and canonical container path. It does not
store credentials or an unrestricted host path.

## Ownership Boundaries

Xelora owns workspace identity, session binding, authorization, canonical path
resolution, boundary enforcement, job identity, artifact registration, and
user-visible history. Controlled Docker, OpenSandbox, OfficeCLI, and later file
providers only execute operations against a workspace supplied by Xelora.

Provider replacement must not change session or artifact contracts.

## Configuration

Add these deployment settings:

```dotenv
XELORA_WORKSPACE_HOST_ROOT=F:\XeloraWorkspaces
XELORA_WORKSPACE_CONTAINER_ROOT=/workspaces
```

Compose mounts `${XELORA_WORKSPACE_HOST_ROOT}` at
`${XELORA_WORKSPACE_CONTAINER_ROOT}` with read-write access in the app service.
The app fails workspace API requests with a configuration error when either the
mount or container root is unavailable. Ordinary chat remains usable.

Only one root is required for the first release. Multiple named roots and
per-user roots are deferred until there is a demonstrated need.

## Workspace Identity And Paths

A workspace is one direct or nested directory below the configured container
root. Its API representation contains:

- `id`: stable opaque ID derived from persisted workspace metadata, not a path
- `name`: user-facing directory name
- `relative_path`: normalized slash-separated path below the allowed root
- `status`: `available`, `missing`, `access_denied`, or `archived`

`SessionWorkspaceBinding.RootPath` is the canonical container path resolved by
the backend. Clients send only a workspace ID when creating or rebinding a
session. Existing `tenant:*` bindings remain readable but are treated as legacy
unbound bindings until explicitly migrated or rebound.

Workspace resolution rejects absolute client paths, `..`, empty segments,
device names, paths outside the configured root, and paths that escape through
symbolic links. Resolution uses the real parent path before allowing creation,
and the real target path before every execution.

## Backend API

Add tenant- and user-scoped endpoints:

```text
GET  /api/v1/workspaces
POST /api/v1/workspaces
GET  /api/v1/workspaces/:id
```

`GET` lists accessible directories and their status. `POST` creates a directory
under the allowed root after validating a short display name or relative path.
The session create and update APIs continue to carry `workspace_binding`, but
new clients submit only `workspace_id`; the service resolves authoritative name,
root path, status, and binding audit fields.

Directory deletion, rename, recursive browsing, file management, and arbitrary
host path entry are outside the first release.

## New Conversation Flow

The new-conversation screen includes a workspace selector near the agent and
model controls. It lists available directories and offers a `New folder` command.
The selected workspace is included when the first message creates the session.

The selector remembers the user's most recent workspace as a UI preference, but
the session binding is authoritative after creation. Reopening a conversation
shows the bound workspace. Rebinding is allowed only before a file-producing job
is running; the first release does not move existing files when rebinding.

If no workspace is selected, normal question answering remains available, but
file-producing tools return a clear `workspace_required` error instead of using
a hidden skill directory.

## Execution And File Tools

The executor gateway resolves the session binding before dispatch and supplies
the canonical workspace root as the job working directory. A file-producing job
must not choose its own persistent root.

For controlled Docker, the app mount is inherited by the child sandbox through
the current `--volumes-from` mechanism. The child working directory is set to the
current conversation workspace. Gateway validation and tool-level relative-path
validation both remain mandatory because the child can technically see the
larger mounted root.

The first implementation keeps the existing preloaded skill packages immutable.
Requests may be materialized in a job-temporary directory, but all requested
input and output document paths resolve relative to the conversation workspace.
The temporary request file is removed after execution.

File snapshots are taken against the conversation workspace before and after a
job. New or modified files become artifacts linked to the session and job. A
tool result is successful only when expected files exist and artifact
registration succeeds; model text alone never counts as file-work success.

## Office Capability

OfficeCLI remains the default Office provider because one structured command
surface covers `.docx`, `.xlsx`, and `.pptx` without requiring Microsoft Office.
The existing bridge is changed to resolve document paths against the conversation
workspace rather than its skill directory.

The first release supports:

- create, inspect, batch edit, and validate Word, Excel, and PowerPoint files
- text and HTML preview where supported by OfficeCLI
- repeated edits to the same file across turns
- artifact registration after each meaningful modification

Format-specific SDKs are provider-level fallbacks, not parallel user-facing
tools. `python-docx`, `openpyxl`, and `python-pptx` may be added later for
operations OfficeCLI cannot perform reliably. Univer or ONLYOFFICE integration
is deferred until in-browser collaborative editing is requested.

Every meaningful Office edit runs format validation. A failed validation leaves
the original file intact where the provider can make the edit transactionally;
otherwise the provider writes to a temporary sibling file and replaces the
original only after validation.

## Error Handling

Errors use stable codes and actionable messages:

- `workspace_required`: no workspace is bound for a file-producing action
- `workspace_not_found`: the configured directory no longer exists
- `workspace_access_denied`: the directory cannot be read or written
- `workspace_path_escape`: a requested path leaves the workspace boundary
- `workspace_busy`: a rebind conflicts with an active file-producing job
- `office_validation_failed`: the generated Office package is invalid
- `artifact_registration_failed`: output exists but was not recorded

The agent receives the same structured failure shown to the user and must not
rewrite it as success.

## Security

- Mount only the administrator-approved root, never an entire drive.
- Do not expose host paths to browser clients, prompts, or provider payloads.
- Resolve real paths before each execution to detect symbolic-link escapes.
- Run sandbox processes as non-root with existing capability and network limits.
- Restrict tool paths to relative paths under the resolved workspace.
- Record workspace ID, session ID, user ID, job ID, operation, and artifact paths.
- Do not silently fall back to skill-private directories.

The first release accepts that `--volumes-from` exposes the configured root to
the child container. Xelora boundary checks are the enforcement layer. A later
hardening phase may replace this with a per-job bind mount of only the selected
workspace without changing product-facing contracts.

## Compatibility And Migration

Existing sessions without a valid host-backed binding continue as normal chat
sessions. Their file-producing actions require the user to bind a workspace.
Legacy `tenant:*` bindings are displayed as needing selection and are not mapped
to a host directory automatically.

No existing files are copied or deleted. Skill-private files produced during
earlier experiments remain untouched and are not presented as workspace files.

## Verification

Automated checks:

1. Workspace service accepts an in-root directory and rejects traversal,
   absolute paths, missing roots, and symbolic-link escapes.
2. Session create, get, update, and reopen preserve the selected workspace ID
   while deriving the authoritative canonical path server-side.
3. Gateway jobs use the bound workspace as their working directory and reject
   missing or invalid bindings without falling back.
4. File snapshot detection registers new and modified files from the bound
   workspace only.
5. OfficeCLI creates, edits, previews, and validates representative `.docx`,
   `.xlsx`, and `.pptx` files in the workspace.

End-to-end acceptance on Docker Desktop:

1. Configure and mount a temporary Windows workspace root.
2. Create a folder from the new-conversation workspace selector.
3. Start a conversation bound to that folder.
4. Ask the agent to create a Markdown file and verify it on the Windows host.
5. Ask the same conversation to edit that file and verify the persisted change.
6. Create and edit one Word, Excel, and PowerPoint file, validate each format,
   and verify all files persist on the Windows host.
7. Reopen the conversation and edit an existing Office file again.
8. Attempt traversal and an unbound file-producing request and verify both are
   blocked with the expected structured errors.

## Non-Goals

- Selecting arbitrary directories anywhere on the host
- Installing a desktop synchronization daemon
- Collaborative in-browser Office editing
- Full filesystem explorer, rename, move, or delete operations
- Automatic migration of legacy skill-private outputs
- Multi-root administration and per-user storage quotas

