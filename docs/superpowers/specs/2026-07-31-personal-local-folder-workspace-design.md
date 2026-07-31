# Personal Local Folder Workspace Design

**Date:** 2026-07-31  
**Status:** Draft for implementation planning  
**Branch context:** Extends `002-session-workspace-binding` for Personal edition only

## Problem

Current workspace binding on Personal still follows the managed-workspace model: directories under a configured host root (`data/workspaces` / AppData). Users cannot point a conversation at an arbitrary local project folder the way Cursor / AI IDEs do. AI read/write is therefore limited to Xelora-managed trees rather than the user’s real project tree.

## Goals

1. On **Personal desktop only**, a workspace is a **single local folder** chosen by the user.
2. AI can **list, read, write, create, delete, rename, and move** files inside that folder (with confirmation for destructive ops).
3. UX matches “Open Folder”: native directory picker + recent folders list.
4. Docker / Enterprise **managed workspace** semantics remain unchanged.

## Non-Goals

- Multi-root workspaces
- Terminal / shell execution inside the workspace (v1)
- Browser/Docker/Enterprise binding of arbitrary host paths
- Remote Enterprise AI reading a user’s laptop via reverse tunnel
- Automatic migration of files from old managed workspaces into local folders
- System-wide sensitive-path denylist (optional follow-up)

## Locked Decisions

| Topic | Choice |
|---|---|
| Runtime | Personal desktop only |
| Relation to managed workspaces | Personal **replaces** managed workspaces; managed remains for Docker/Enterprise |
| Shape | Single root per session |
| File ops | Full file management (incl. delete/rename/move) |
| Destructive safety | Frontend confirmation before apply |
| Reuse | Recent folders list (client-local), not a server registry of named workspaces |

## Architecture (Approach 1)

Keep `sessions.workspace_binding` as the durable session contract and keep executor `WorkDir = binding.root_path`. On Personal, change **where the root comes from** and **how the user selects it**:

```text
User → Wails folder dialog / recent list
     → bind session (workspace_id + root_path)
     → LocalRegistry / Resolve (Personal path mode)
     → File APIs + Agent tools (WorkDir = root_path)
     → Destructive ops: propose → UI confirm → apply
```

Non-Personal builds continue to reject arbitrary absolute paths from clients and only resolve managed workspace IDs under the configured workspace root.

## Data Model

### SessionWorkspaceBinding (existing JSONB, Personal semantics)

| Field | Personal meaning |
|---|---|
| `workspace_id` | Stable id: `local:` + deterministic hash of the normalized absolute path |
| `workspace_name` | Folder basename for display |
| `root_path` | Normalized absolute local path (server-authoritative) |
| `status` | `bound` \| `unbound` \| `invalid` \| `access_denied` \| `archived` |
| `binding_kind` | New optional field: `local_folder` (Personal) \| `managed` (non-Personal). Absent/`managed` = legacy managed |

Client bind input (Personal):

- Open new folder or re-select from recent: `{ "root_path": "D:\\projects\\foo" }` (Personal only)
- Server normalizes the path, derives `workspace_id` (`local:` + hash) and `workspace_name` (basename), sets `binding_kind=local_folder`
- Client may also send `workspace_id` as a hint; if present it **must** match the hash of the normalized `root_path`, otherwise bind fails
- Because recent list is client-only, Personal bind always carries `root_path` — the server does not look up paths by `workspace_id` alone

Validation:

- Path must exist and be a directory at bind/validate time
- Normalize (absolute, clean; root symlink policy: resolve to canonical directory before hashing)
- All subsequent file ops must stay within the normalized root (relative paths only from tools; reject `..`, absolute injection, symlink escape)

### Recent local workspaces (client preference, not server DB)

Stored in Personal local preferences (e.g. preferenceStorage / desktop settings):

```ts
type RecentLocalWorkspace = {
  workspace_id: string
  workspace_name: string
  root_path: string
  last_used_at: string // ISO
}
```

- Cap: 20 entries
- Selecting an entry binds the session by re-sending that entry’s `root_path` (+ matching `workspace_id` hint) and bumps `last_used_at`
- If the path is missing when opening the picker list, mark the entry stale in UI (do not auto-delete); user can dismiss/remove it. Session validate against a missing path returns `invalid`

## Binding UX

1. **Primary CTA:** “打开文件夹…” → native directory picker (Wails).
2. **Secondary:** “最近使用” list under the picker / workspace selector.
3. **Session chrome:** show bound name + truncated path; if `invalid`, show rebind affordance.
4. **Unbound sessions:** chat allowed; file tools still return `workspace_required` (unchanged contract).
5. **Personal UI:** hide “create managed workspace” and any managed-root creation flow.

## File & AI Capabilities

| Operation | Behavior |
|---|---|
| List / read / preview / download / raw | Immediate |
| Write / append / create (text, JSON, office skills) | Immediate; `WorkDir = root_path` |
| Delete / rename / move | **Propose → confirm → apply** |

### Destructive mutation flow

1. Agent/tool emits a **pending mutation** (`op`, `from`, `to`, human preview).
2. Frontend shows confirm dialog (Chinese/English via i18n).
3. On confirm: `POST` apply endpoint with short-lived single-use token / proposal id.
4. On reject: tool result reports user rejection; no disk change.
5. Token expiry or path no longer under root → apply fails safely.

Reuse existing workspace file listing/preview/raw routes by resolving Personal bindings to absolute roots. Extend with mutation propose/apply (or equivalent names) rather than inventing a parallel FS protocol.

Skills (`workspace-file-writer`, `officecli-document-editing`, browser artifacts) continue to use the executor gateway’s bound `WorkDir` and artifact snapdiff under that root.

## Security

- Path confinement under normalized `root_path` for every read/write/mutation
- No client-supplied absolute paths for file ops after bind (relative to root only)
- Symlink escape denied
- Destructive ops require explicit user confirmation
- Confirm tokens: short TTL, single use, bound to session + proposal content
- Choosing a folder **is** the authorization grant for that root (v1); no extra OS ACL UI

## Edition Split

| Edition | Workspace source | Client may submit absolute `root_path`? |
|---|---|---|
| Personal | Local folder via dialog / recent | Yes (bind only) |
| Docker / Enterprise | Managed registry under configured root | No |

Personal must not expose managed workspace creation. Managed APIs may remain compiled but unused in Personal UI; creating managed workspaces on Personal is out of product scope (disable or no-op with clear error).

## Migration (Personal)

- Existing sessions with managed bindings: on validate, set `status=invalid` with message guiding user to open a local folder and rebind
- Do **not** copy/move files from old managed trees
- Recent list starts empty (do not seed from managed registry)
- Artifacts recorded against old managed roots remain historical; new writes require a valid local binding

## Error Codes (aligned with existing)

- `workspace_required` — unbound session file op
- `binding_invalid` — path missing / not a directory
- `access_denied` — OS permission failure
- `workspace_path_escape` — path escapes root
- `mutation_rejected` — user declined confirm
- `mutation_expired` — confirm token / proposal expired

## Testing

1. Open folder → bind → list/read/write succeed under that tree  
2. Delete/rename/move: no disk change without confirm; apply after confirm; reject leaves disk unchanged  
3. Delete bound folder on disk → next validate/file op → `invalid` + rebind UX  
4. Non-Personal build: absolute `root_path` bind rejected; managed flow unchanged  
5. Unbound session: file tools return `workspace_required`  
6. Symlink / `..` escape attempts fail with `workspace_path_escape`

## Rollout

1. Spec → implementation plan → Personal-only code path  
2. Rebuild Personal client; manual Open Folder smoke  
3. No Docker frontend/app behavior change required for v1  

## Open Follow-ups (explicitly out of v1)

- Multi-root  
- In-workspace terminal  
- Sensitive path denylist (e.g. system dirs)  
- Git-aware UI  
- Syncing recent list across machines
