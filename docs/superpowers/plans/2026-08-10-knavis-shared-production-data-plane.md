# Knavis Shared Production Data Plane Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Merge the local and `192.168.9.177` Knavis data sets with local-wins conflict handling, make `192.168.9.177` the sole active internal production server, and retain a safe local development workflow.

**Architecture:** `192.168.9.177` owns production PostgreSQL/pgvector, Redis/Asynq, MinIO, and the workspace share. The local machine uses `docker-compose.dev.yml` for ordinary development. A local shared/passive profile may connect to the production services only for controlled administrative verification; its process role prevents migrations and autonomous background runners from starting.

**Tech Stack:** Go 1.26, Docker Compose v2, PostgreSQL/pgvector, Redis/Asynq, MinIO, Windows PowerShell 5.1+, Node built-in test runner, Windows SMB share.

---

## Global Safety Rules

- Work from an isolated implementation worktree because the current
  `knavis-internal` checkout contains unrelated uncommitted changes.
- Never merge PostgreSQL Docker volumes or WSL VHDX files. Use `pg_dump` and
  `pg_restore` archives only.
- Never place database, MinIO, or workspace credentials in Git, release
  manifests, command lines, or test fixtures.
- Do not run a production write until both source snapshots, their SHA-256
  manifests, the schema comparison report, the conflict report, and the file
  reference report exist and have been reviewed.
- `192.168.9.177` is the only `active` process after cutover. The local shared
  profile is not a LAN endpoint and may not run migrations, Asynq workers,
  data-source scheduling, housekeeping, audit retention, or IM consumers.
- The existing Go AgentEngine, Progressive RAG, Controlled Docker, Skill,
  workspace authorization, and SSE paths remain unchanged.

## File Structure

| Path | Responsibility |
| --- | --- |
| `internal/container/process_role.go` | Parses and validates the new active/passive process role. |
| `internal/container/process_role_test.go` | Locks down role defaults and invalid-role failure behavior. |
| `internal/container/container.go` | Gates Asynq, scheduler, housekeeping, audit retention, and IM channel startup by process role. |
| `.env.example` | Documents the default `active` process role and controlled migration behavior. |
| `docker-compose.yml` | Keeps current defaults while allowing DB host, Redis address, and process role to be overridden. |
| `docker-compose.knavis-production.yml` | Adds the internal production MinIO profile and restricted maintenance ports on `192.168.9.177`. |
| `docker-compose.knavis-passive.yml` | Defines the local-only shared/passive application profile. |
| `.env.knavis-production.example` | Production template without secrets. |
| `.env.knavis-passive.example` | Local passive template without secrets. |
| `scripts/start-knavis-passive.ps1` | Validates and launches only local `app` and `frontend` with `--no-deps`. |
| `scripts/verify-shared-workspace.ps1` | Verifies that the host share is mounted as `/workspaces` and supports a round-trip write. |
| `scripts/data-plane/common.ps1` | Shared PowerShell validation, Docker, hashing, and artifact helpers. |
| `scripts/data-plane/preflight.ps1` | Produces a read-only source inventory and rejects unsafe migration prerequisites. |
| `scripts/data-plane/snapshot.ps1` | Creates PostgreSQL, data-file-volume, workspace, and configuration snapshots after maintenance freeze. |
| `scripts/data-plane/build-merge-plan.ps1` | Restores snapshots into disposable databases and creates schema, table, FK, conflict, and file-reference reports. |
| `scripts/data-plane/execute-merge.ps1` | Produces a merged logical PostgreSQL archive with local records winning collisions. |
| `scripts/data-plane/import-files-to-minio.ps1` | Uploads validated source files to MinIO and rewrites only reviewed `local://` knowledge references. |
| `scripts/data-plane/verify-merge.ps1` | Checks rows, constraints, object hashes, workspace manifests, and application smoke results. |
| `scripts/data-plane/promote-177.ps1` | Performs the explicit `192.168.9.177` promotion after all validation gates pass. |
| `scripts/data-plane/rollback-177.ps1` | Restores the pre-cutover configuration and snapshots. |
| `scripts/data-plane/data-plane.test.mjs` | Prevents lifecycle scripts from losing required safety gates. |
| `scripts/run-knavis-production-migration.ps1` | Runs versioned migrations once on the active server with a recorded backup ID. |
| `scripts/shared-data-plane-compose.test.mjs` | Verifies profile defaults and that passive mode cannot expose a LAN service. |
| `docs/deployment/knavis-shared-production-data-plane.md` | Operator runbook for dry run, cutover, verification, and rollback. |
| `scripts/package-knavis-internal.ps1` | Includes the new non-secret templates and lifecycle scripts in future internal releases. |

## Task 1: Add an Explicit Process Role

**Files:**
- Create: `internal/container/process_role.go`
- Create: `internal/container/process_role_test.go`
- Modify: `internal/container/container.go:270-303`
- Modify: `internal/container/container.go:369-373`
- Modify: `internal/container/container.go:384-391`
- Modify: `.env.example:230-250`
- Modify: `docker-compose.yml:112-238`

- [ ] **Step 1: Write the failing process-role tests**

```go
package container

import "testing"

func TestProcessRoleFromEnv(t *testing.T) {
    cases := []struct {
        name string
        raw  string
        want processRole
        bad  bool
    }{
        {name: "default active", raw: "", want: processRoleActive},
        {name: "explicit active", raw: "active", want: processRoleActive},
        {name: "case insensitive passive", raw: "PASSIVE", want: processRolePassive},
        {name: "unknown rejected", raw: "writer", bad: true},
    }
    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            t.Setenv(processRoleEnv, tc.raw)
            got, err := processRoleFromEnv()
            if tc.bad {
                if err == nil { t.Fatal("expected error") }
                return
            }
            if err != nil { t.Fatalf("processRoleFromEnv() error = %v", err) }
            if got != tc.want { t.Fatalf("processRoleFromEnv() = %q, want %q", got, tc.want) }
        })
    }
}
```

- [ ] **Step 2: Run the focused test to verify it fails**

Run: `go test ./internal/container -run '^TestProcessRoleFromEnv$' -count=1`

Expected: FAIL because `processRole`, `processRoleEnv`, and
`processRoleFromEnv` do not exist.

- [ ] **Step 3: Implement strict role parsing**

Create `internal/container/process_role.go`:

```go
package container

import (
    "fmt"
    "os"
    "strings"
)

const processRoleEnv = "XELORA_PROCESS_ROLE"

type processRole string

const (
    processRoleActive  processRole = "active"
    processRolePassive processRole = "passive"
)

func processRoleFromEnv() (processRole, error) {
    switch strings.ToLower(strings.TrimSpace(os.Getenv(processRoleEnv))) {
    case "", string(processRoleActive):
        return processRoleActive, nil
    case string(processRolePassive):
        return processRolePassive, nil
    default:
        return "", fmt.Errorf("%s must be active or passive", processRoleEnv)
    }
}

func backgroundServicesEnabled(role processRole) bool {
    return role == processRoleActive
}
```

- [ ] **Step 4: Gate all autonomous runners in container construction**

At the beginning of `BuildContainer`, resolve the role once and fail startup
for an invalid value:

```go
role, err := processRoleFromEnv()
must(err)
activeProcess := backgroundServicesEnabled(role)
logger.Infof(ctx, "[Container] process role=%s", role)
```

Apply `activeProcess` to every side-effecting bootstrap call:

```go
if activeProcess {
    must(container.Invoke(startDataSourceScheduler))
    must(container.Invoke(startAuditLogRetention))
    must(container.Invoke(startHousekeepingService))
}

if redisAvailable && activeProcess {
    must(container.Invoke(router.RunAsynqServer))
} else if !redisAvailable && activeProcess {
    must(container.Invoke(router.RegisterSyncHandlers))
}
```

Keep the task enqueuer and task inspector registrations in place when Redis is
available. A passive API request may enqueue a task for the active server to
process, but the passive process may not consume it.

Split IM adapter registration from IM channel startup:

```go
func registerIMAdapterFactories(imService *imPkg.Service, role processRole) {
    imService.RegisterAdapterFactory("wecom", wecom.NewFactory())
    imService.RegisterAdapterFactory("feishu", feishu.NewFactory())
    imService.RegisterAdapterFactory("slack", slack.NewFactory())
    imService.RegisterAdapterFactory("telegram", telegram.NewFactory())
    imService.RegisterAdapterFactory("dingtalk", dingtalk.NewFactory())
    imService.RegisterAdapterFactory("mattermost", mattermost.NewFactory())
    imService.RegisterAdapterFactory("wechat", wechat.NewFactory())
    if !backgroundServicesEnabled(role) {
        logger.Infof(context.Background(), "[IM] channel consumers disabled for passive process")
        return
    }
    if err := imService.LoadAndStartChannels(); err != nil {
        logger.Warnf(context.Background(), "[IM] Failed to load channels from database: %v", err)
    }
}
```

Provide `processRole` through Dig with a provider that calls
`processRoleFromEnv`, then invoke the revised function. Do not conditionally
omit the IM handler itself; administrators must still be able to inspect
configuration from the passive process.

- [ ] **Step 5: Expose the role through Compose and the environment template**

Change the current hard-coded Compose defaults without changing normal
behavior:

```yaml
- DB_HOST=${DB_HOST:-postgres}
- DB_PORT=${DB_PORT:-5432}
- REDIS_ADDR=${REDIS_ADDR:-redis:6379}
- XELORA_PROCESS_ROLE=${XELORA_PROCESS_ROLE:-active}
```

Add this comment near the existing runtime settings in `.env.example`:

```dotenv
# active starts workers, schedulers, housekeeping, audit retention, and IM consumers.
# passive is for local controlled verification against a shared production data plane.
XELORA_PROCESS_ROLE=active
```

- [ ] **Step 6: Run focused regression tests**

Run:

```powershell
go test ./internal/container -run 'TestProcessRoleFromEnv|TestResetPending' -count=1
docker compose --env-file .env -f docker-compose.yml config --quiet
```

Expected: tests pass; existing `.env` still resolves `DB_HOST=postgres`,
`REDIS_ADDR=redis:6379`, and process role `active`.

- [ ] **Step 7: Commit the process-role boundary**

```powershell
git add internal/container/process_role.go internal/container/process_role_test.go internal/container/container.go .env.example docker-compose.yml
git commit -m "feat: add passive process role"
```

## Task 2: Add Production And Local Passive Compose Profiles

**Files:**
- Create: `docker-compose.knavis-production.yml`
- Create: `docker-compose.knavis-passive.yml`
- Create: `.env.knavis-production.example`
- Create: `.env.knavis-passive.example`
- Create: `scripts/start-knavis-passive.ps1`
- Create: `scripts/shared-data-plane-compose.test.mjs`
- Modify: `scripts/host-workspace-mount-smoke.ps1`

- [ ] **Step 1: Write failing static contract tests for the profiles**

Create `scripts/shared-data-plane-compose.test.mjs`:

```js
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const production = readFileSync(new URL('../docker-compose.knavis-production.yml', import.meta.url), 'utf8')
const passive = readFileSync(new URL('../docker-compose.knavis-passive.yml', import.meta.url), 'utf8')
const launcher = readFileSync(new URL('./start-knavis-passive.ps1', import.meta.url), 'utf8')

test('production profile centralizes MinIO and marks the app active', () => {
  assert.match(production, /XELORA_PROCESS_ROLE:\s*active/)
  assert.match(production, /STORAGE_TYPE:\s*minio/)
  assert.match(production, /MINIO_BUCKET_NAME/)
})

test('passive profile disables automatic migrations and binds locally only', () => {
  assert.match(passive, /XELORA_PROCESS_ROLE:\s*passive/)
  assert.match(passive, /AUTO_MIGRATE:\s*"false"/)
  assert.match(passive, /127\.0\.0\.1/)
  assert.match(launcher, /--no-deps/)
  assert.match(launcher, /app", "frontend"/)
})
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `node --test scripts/shared-data-plane-compose.test.mjs`

Expected: FAIL because the profile and launcher files do not exist.

- [ ] **Step 3: Create the non-secret production template and overlay**

Create `.env.knavis-production.example` with explicit values that operators
must replace outside Git:

```dotenv
XELORA_BRAND_PROFILE=internal
XELORA_PROCESS_ROLE=active
AUTO_MIGRATE=false
AUTO_RECOVER_DIRTY=false
DB_HOST=postgres
DB_PORT=5432
DB_NAME=Xelora
DB_USER=replace-me
DB_PASSWORD=replace-me
REDIS_ADDR=redis:6379
STORAGE_TYPE=minio
MINIO_ENDPOINT=minio:9000
MINIO_USE_SSL=false
MINIO_ACCESS_KEY_ID=replace-me
MINIO_SECRET_ACCESS_KEY=replace-me
MINIO_BUCKET_NAME=knavis-production
XELORA_WORKSPACE_HOST_ROOT=D:\Knavis\workspaces
XELORA_WORKSPACE_CONTAINER_ROOT=/workspaces
APP_PORT=8080
FRONTEND_PORT=80
KNAVIS_MAINTENANCE_BIND_IP=192.168.9.177
```

Create `docker-compose.knavis-production.yml`:

```yaml
services:
  app:
    environment:
      XELORA_PROCESS_ROLE: active
      AUTO_MIGRATE: "false"
      STORAGE_TYPE: minio
      MINIO_ENDPOINT: minio:9000
      MINIO_USE_SSL: "false"
  postgres:
    ports:
      - "${KNAVIS_MAINTENANCE_BIND_IP}:5432:5432"
  redis:
    ports:
      - "${KNAVIS_MAINTENANCE_BIND_IP}:6379:6379"
  minio:
    profiles: [minio]
    ports:
      - "${KNAVIS_MAINTENANCE_BIND_IP}:9000:9000"
      - "${KNAVIS_MAINTENANCE_BIND_IP}:9001:9001"
```

The deployment runbook must add Windows Firewall rules that allow ports `5432`,
`6379`, `9000`, and `9001` only from the local development host and deny other
LAN clients. Client-facing frontend/API ports remain separately governed by the
existing Knavis deployment configuration.

- [ ] **Step 4: Create the passive template and overlay**

Create `.env.knavis-passive.example`:

```dotenv
XELORA_BRAND_PROFILE=internal
XELORA_PROCESS_ROLE=passive
AUTO_MIGRATE=false
AUTO_RECOVER_DIRTY=false
DB_HOST=192.168.9.177
DB_PORT=5432
DB_NAME=Xelora
DB_USER=replace-me
DB_PASSWORD=replace-me
REDIS_ADDR=192.168.9.177:6379
STORAGE_TYPE=minio
MINIO_ENDPOINT=192.168.9.177:9000
MINIO_USE_SSL=false
MINIO_ACCESS_KEY_ID=replace-me
MINIO_SECRET_ACCESS_KEY=replace-me
MINIO_BUCKET_NAME=knavis-production
XELORA_WORKSPACE_HOST_ROOT=K:\KnavisWorkspaces
XELORA_WORKSPACE_CONTAINER_ROOT=/workspaces
APP_PORT=127.0.0.1:18080
FRONTEND_PORT=127.0.0.1:18081
```

Create `docker-compose.knavis-passive.yml`:

```yaml
services:
  app:
    environment:
      DB_HOST: ${DB_HOST:?set DB_HOST in .env.knavis-passive}
      DB_PORT: ${DB_PORT:-5432}
      DB_USER: ${DB_USER:?set DB_USER in .env.knavis-passive}
      DB_PASSWORD: ${DB_PASSWORD:?set DB_PASSWORD in .env.knavis-passive}
      DB_NAME: ${DB_NAME:?set DB_NAME in .env.knavis-passive}
      REDIS_ADDR: ${REDIS_ADDR:?set REDIS_ADDR in .env.knavis-passive}
      STORAGE_TYPE: minio
      MINIO_ENDPOINT: ${MINIO_ENDPOINT:?set MINIO_ENDPOINT in .env.knavis-passive}
      MINIO_USE_SSL: "false"
      MINIO_ACCESS_KEY_ID: ${MINIO_ACCESS_KEY_ID:?set MINIO_ACCESS_KEY_ID in .env.knavis-passive}
      MINIO_SECRET_ACCESS_KEY: ${MINIO_SECRET_ACCESS_KEY:?set MINIO_SECRET_ACCESS_KEY in .env.knavis-passive}
      MINIO_BUCKET_NAME: ${MINIO_BUCKET_NAME:?set MINIO_BUCKET_NAME in .env.knavis-passive}
      AUTO_MIGRATE: "false"
      AUTO_RECOVER_DIRTY: "false"
      XELORA_PROCESS_ROLE: passive
```

The passive profile intentionally does not publish ports beyond loopback. It
starts only `app` and `frontend` with `--no-deps`; it never starts local
PostgreSQL, Redis, MinIO, DocReader, or sandbox containers as a side effect.

- [ ] **Step 5: Add the passive launcher and share smoke test**

Create `scripts/start-knavis-passive.ps1`:

```powershell
[CmdletBinding()]
param([string]$EnvFile = ".env.knavis-passive")

$ErrorActionPreference = "Stop"
$repoRoot = Split-Path -Parent $PSScriptRoot
$base = Join-Path $repoRoot "docker-compose.yml"
$passive = Join-Path $repoRoot "docker-compose.knavis-passive.yml"
$envPath = Join-Path $repoRoot $EnvFile

foreach ($path in @($base, $passive, $envPath)) {
    if (-not (Test-Path -LiteralPath $path)) { throw "Missing required file: $path" }
}

& docker compose --env-file $envPath -f $base -f $passive config --quiet
if ($LASTEXITCODE -ne 0) { throw "Passive Compose validation failed" }

& docker compose --env-file $envPath -f $base -f $passive up -d --no-deps app frontend
if ($LASTEXITCODE -ne 0) { throw "Passive services failed to start" }
```

Create `scripts/verify-shared-workspace.ps1` by adapting the existing mount
smoke test. It must require a supplied `-HostRoot`, write
`.shared-mount-smoke` from `/workspaces`, verify its SHA-256 on the Windows
host, and remove only that exact probe file.

- [ ] **Step 6: Run profile verification**

Run:

```powershell
node --test scripts/shared-data-plane-compose.test.mjs
Copy-Item .env.knavis-passive.example .env.knavis-passive.local
docker compose --env-file .env.knavis-passive.local -f docker-compose.yml -f docker-compose.knavis-passive.yml config --quiet
```

Expected: Node tests pass and Compose validates without starting containers.
Do not start the passive profile against production during this task.

- [ ] **Step 7: Commit profile isolation**

```powershell
git add docker-compose.yml docker-compose.knavis-production.yml docker-compose.knavis-passive.yml .env.example .env.knavis-production.example .env.knavis-passive.example scripts/start-knavis-passive.ps1 scripts/verify-shared-workspace.ps1 scripts/shared-data-plane-compose.test.mjs scripts/host-workspace-mount-smoke.ps1
git commit -m "feat: add Knavis production and passive profiles"
```

## Task 3: Build Read-Only Inventory And Snapshot Tooling

**Files:**
- Create: `scripts/data-plane/common.ps1`
- Create: `scripts/data-plane/preflight.ps1`
- Create: `scripts/data-plane/snapshot.ps1`
- Create: `scripts/data-plane/data-plane.test.mjs`

- [ ] **Step 1: Write failing script-contract tests**

Create `scripts/data-plane/data-plane.test.mjs`:

```js
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

const preflight = readFileSync(new URL('./preflight.ps1', import.meta.url), 'utf8')
const snapshot = readFileSync(new URL('./snapshot.ps1', import.meta.url), 'utf8')

test('preflight is read-only and records required database facts', () => {
  assert.match(preflight, /information_schema\.tables/)
  assert.match(preflight, /pg_constraint/)
  assert.match(preflight, /pg_extension/)
  assert.match(preflight, /Get-FileHash/)
  assert.doesNotMatch(preflight, /DROP\s+DATABASE/i)
})

test('snapshot requires maintenance confirmation and uses logical archives', () => {
  assert.match(snapshot, /MaintenanceApproved/)
  assert.match(snapshot, /pg_dump/)
  assert.match(snapshot, /--format=custom/)
  assert.match(snapshot, /Get-FileHash/)
  assert.doesNotMatch(snapshot, /docker\s+volume\s+rm/i)
})
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `node --test scripts/data-plane/data-plane.test.mjs`

Expected: FAIL because the data-plane scripts do not exist.

- [ ] **Step 3: Implement common, non-destructive helpers**

Create `scripts/data-plane/common.ps1` with these helpers:

```powershell
Set-StrictMode -Version Latest

function Invoke-Checked {
    param([string]$FilePath, [string[]]$Arguments)
    & $FilePath @Arguments
    if ($LASTEXITCODE -ne 0) { throw "$FilePath failed with exit code $LASTEXITCODE" }
}

function Assert-ExistingDirectory {
    param([string]$Path, [string]$Label)
    if (-not (Test-Path -LiteralPath $Path -PathType Container)) {
        throw "$Label directory does not exist: $Path"
    }
    return (Resolve-Path -LiteralPath $Path).Path
}

function Write-ArtifactHash {
    param([string]$Path)
    $hash = Get-FileHash -LiteralPath $Path -Algorithm SHA256
    [pscustomobject]@{ path = $Path; sha256 = $hash.Hash; bytes = (Get-Item -LiteralPath $Path).Length }
}
```

Do not accept a recursive deletion path from script parameters. Every script
creates a new timestamped output directory with `New-Item -ItemType Directory`
and refuses an existing target directory.

- [ ] **Step 4: Implement source preflight**

Create `scripts/data-plane/preflight.ps1` with required parameters:

```powershell
param(
    [Parameter(Mandatory)][string]$SourceName,
    [Parameter(Mandatory)][string]$PostgresContainer,
    [Parameter(Mandatory)][string]$DatabaseName,
    [Parameter(Mandatory)][string]$DatabaseUser,
    [Parameter(Mandatory)][string]$DataFilesVolume,
    [Parameter(Mandatory)][string]$WorkspaceRoot,
    [Parameter(Mandatory)][string]$OutputRoot
)
```

It must:

1. verify Docker, the named PostgreSQL container, the named volume, and the
   workspace directory exist;
2. query the database through `docker exec` for server version, all installed
   extensions, migration version/dirty state, table names, row counts, primary
   keys, foreign keys, and non-empty tables without primary keys;
3. query all `text`, `varchar`, and `jsonb` columns for values containing
   `local://`, `minio://`, or legacy absolute `/data/files/` paths, recording
   table, primary key, column, path, and value hash without printing secrets;
4. inventory the data-files volume through a read-only Alpine container and
   the workspace root through `Get-ChildItem`, producing relative path, bytes,
   and SHA-256; and
5. write `preflight.json`, `database-schema.json`, `file-references.json`,
   `data-files-manifest.json`, `workspace-manifest.json`, and `SHA256SUMS.txt`.

Preflight must fail before writing its success marker when the migration table
is dirty, required PostgreSQL extensions differ from the expected source,
there are non-empty tables without primary keys, or a file reference escapes
the declared source root.

- [ ] **Step 5: Implement frozen-source snapshots**

Create `scripts/data-plane/snapshot.ps1` with:

```powershell
param(
    [Parameter(Mandatory)][string]$SourceName,
    [Parameter(Mandatory)][string]$PostgresContainer,
    [Parameter(Mandatory)][string]$DatabaseName,
    [Parameter(Mandatory)][string]$DatabaseUser,
    [Parameter(Mandatory)][string]$DataFilesVolume,
    [Parameter(Mandatory)][string]$WorkspaceRoot,
    [Parameter(Mandatory)][string]$OutputRoot,
    [switch]$MaintenanceApproved
)
if (-not $MaintenanceApproved) { throw "Pass -MaintenanceApproved after stopping both application nodes" }
```

Before calling `pg_dump`, assert that the corresponding `app`, `frontend`,
`docreader`, and worker-producing containers are stopped. Create a custom
logical archive with:

```powershell
Invoke-Checked "docker" @(
    "exec", $PostgresContainer, "pg_dump", "--username=$DatabaseUser",
    "--dbname=$DatabaseName", "--format=custom", "--no-owner", "--no-privileges",
    "--file=/tmp/$SourceName.dump"
)
```

Copy the archive out with `docker cp`, export the data-files volume with a
read-only helper container, archive the workspace root without following
reparse points, hash every archive, and record the `docker compose config`
and image digest inventory. The script must not call `docker volume rm`,
`docker compose down -v`, `docker system prune`, or a raw volume copy.

- [ ] **Step 6: Run script and syntax verification**

Run:

```powershell
node --test scripts/data-plane/data-plane.test.mjs
powershell.exe -NoProfile -Command "[void][scriptblock]::Create((Get-Content -Raw 'scripts/data-plane/common.ps1'))"
powershell.exe -NoProfile -Command "[void][scriptblock]::Create((Get-Content -Raw 'scripts/data-plane/preflight.ps1'))"
powershell.exe -NoProfile -Command "[void][scriptblock]::Create((Get-Content -Raw 'scripts/data-plane/snapshot.ps1'))"
```

Expected: tests and PowerShell parse checks pass. Do not invoke preflight or
snapshot against either data source in this code task.

- [ ] **Step 7: Commit inventory and snapshot tooling**

```powershell
git add scripts/data-plane/common.ps1 scripts/data-plane/preflight.ps1 scripts/data-plane/snapshot.ps1 scripts/data-plane/data-plane.test.mjs
git commit -m "feat: add Knavis data-plane inventory tooling"
```

## Task 4: Implement Dry-Run Database Merge Planning

**Files:**
- Create: `scripts/data-plane/build-merge-plan.ps1`
- Modify: `scripts/data-plane/data-plane.test.mjs`

- [ ] **Step 1: Extend tests for merge-plan safety**

Add:

```js
const mergePlan = readFileSync(new URL('./build-merge-plan.ps1', import.meta.url), 'utf8')

test('merge planning restores only disposable databases and reports collisions', () => {
  assert.match(mergePlan, /knavis_merge_/)
  assert.match(mergePlan, /pg_restore/)
  assert.match(mergePlan, /postgres_fdw/)
  assert.match(mergePlan, /conflicts\.json/)
  assert.match(mergePlan, /unsupported-tables\.json/)
  assert.doesNotMatch(mergePlan, /DROP\s+DATABASE\s+Xelora/i)
})
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `node --test scripts/data-plane/data-plane.test.mjs`

Expected: FAIL because `build-merge-plan.ps1` does not exist.

- [ ] **Step 3: Create the disposable merge environment**

Implement `scripts/data-plane/build-merge-plan.ps1` with required input
archives and a mandatory `-OutputRoot`. It creates a timestamped temporary
PostgreSQL container named `knavis_merge_<timestamp>`, creates separate
`server_snapshot`, `local_snapshot`, and `merged_preview` databases, and
restores each dump only into its matching temporary database.

The script must verify all of the following before generating any merge SQL:

```sql
SELECT extname, extversion FROM pg_extension ORDER BY extname;
SELECT version, dirty FROM schema_migrations;
SELECT table_name FROM information_schema.tables
WHERE table_schema = 'public' AND table_type = 'BASE TABLE'
ORDER BY table_name;
```

It must fail if schemas, extensions, migration versions, or table definitions
differ. It must also fail when a non-empty table lacks a primary key or the
foreign-key graph contains a cycle. These constraints make automatic local-wins
upserts safe and deterministic.

- [ ] **Step 4: Generate reviewable collision and merge artifacts**

For each supported table, compare `to_jsonb(row)` values keyed by its primary
key. Write one JSON object per conflict with:

```json
{
  "table": "knowledges",
  "primary_key": { "id": "..." },
  "local_sha256": "...",
  "server_sha256": "...",
  "winner": "local",
  "dependency_action": "upsert in topological order"
}
```

Write `table-order.json`, `conflicts.json`, `unsupported-tables.json`,
`schema-compare.json`, and a generated `merge.sql`. `merge.sql` must use
`INSERT ... ON CONFLICT (...) DO UPDATE` for supported tables so local rows
replace only matching server primary keys and server-only rows remain.

Use a generated column list rather than `SELECT *`. For identity columns use
`OVERRIDING SYSTEM VALUE`; after merge, reset every owned sequence from its
table maximum. The generated SQL must start with:

```sql
BEGIN;
SET lock_timeout = '10s';
SET statement_timeout = '0';
```

and end with `COMMIT;` only in the final executable archive, never during the
dry-run report phase.

- [ ] **Step 5: Verify the dry-run tooling**

Run:

```powershell
node --test scripts/data-plane/data-plane.test.mjs
powershell.exe -NoProfile -Command "[void][scriptblock]::Create((Get-Content -Raw 'scripts/data-plane/build-merge-plan.ps1'))"
```

Expected: tests and syntax parsing pass. The first real invocation is a dry
run against copied snapshots only and produces no writes to either source.

- [ ] **Step 6: Commit the dry-run merge planner**

```powershell
git add scripts/data-plane/build-merge-plan.ps1 scripts/data-plane/data-plane.test.mjs
git commit -m "feat: plan Knavis database reconciliation"
```

## Task 5: Implement Approved Database And File Reconciliation

**Files:**
- Create: `scripts/data-plane/execute-merge.ps1`
- Create: `scripts/data-plane/import-files-to-minio.ps1`
- Create: `scripts/data-plane/verify-merge.ps1`
- Modify: `scripts/data-plane/data-plane.test.mjs`

- [ ] **Step 1: Write tests for required execution gates**

Add:

```js
const executeMerge = readFileSync(new URL('./execute-merge.ps1', import.meta.url), 'utf8')
const importFiles = readFileSync(new URL('./import-files-to-minio.ps1', import.meta.url), 'utf8')
const verifyMerge = readFileSync(new URL('./verify-merge.ps1', import.meta.url), 'utf8')

test('execution requires explicit local-wins approval and emits a portable archive', () => {
  assert.match(executeMerge, /ApproveLocalWins/)
  assert.match(executeMerge, /merge-plan\.sha256/)
  assert.match(executeMerge, /pg_dump/)
  assert.doesNotMatch(executeMerge, /docker\s+volume\s+rm/i)
})

test('file migration preserves content hashes and only rewrites reviewed knowledge paths', () => {
  assert.match(importFiles, /local:\/\//)
  assert.match(importFiles, /minio:\/\//)
  assert.match(importFiles, /file-references\.json/)
  assert.match(importFiles, /Get-FileHash/)
  assert.match(verifyMerge, /foreign_key_check/)
  assert.match(verifyMerge, /object-manifest/)
})
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `node --test scripts/data-plane/data-plane.test.mjs`

Expected: FAIL because the execution and verification scripts do not exist.

- [ ] **Step 3: Implement controlled database archive generation**

`execute-merge.ps1` accepts a reviewed merge-plan directory and requires both
`-ApproveLocalWins` and `-ReviewedPlanHash <SHA256>`. It hashes the supplied
`merge.sql` and rejects a mismatch. It restores the server snapshot into a
new disposable database, attaches the local snapshot through `postgres_fdw`,
executes only the reviewed generated SQL, validates constraints, and emits a
new custom archive named `merged-local-wins.dump`.

The required guard is:

```powershell
param(
    [Parameter(Mandatory)][string]$MergePlanDir,
    [Parameter(Mandatory)][string]$ReviewedPlanHash,
    [switch]$ApproveLocalWins
)
if (-not $ApproveLocalWins) { throw "Pass -ApproveLocalWins only after reviewing conflicts.json" }
$actual = (Get-FileHash (Join-Path $MergePlanDir "merge.sql") -Algorithm SHA256).Hash
if ($actual -ine $ReviewedPlanHash) { throw "Reviewed merge.sql hash does not match" }
```

The script writes `merged-local-wins.dump`, `merge-execution.json`, and a hash
manifest. It never restores the archive into `192.168.9.177` production.

- [ ] **Step 4: Implement MinIO file import and reference conversion**

`import-files-to-minio.ps1` accepts the verified data-files archive, the
reviewed `file-references.json`, the MinIO endpoint and credentials through
the environment, and a target bucket. It uses the official `minio/mc` image
to create the target bucket if absent and uploads each relative file key
unchanged. The resulting target path is deterministic:

```text
local://<relative-key>  ->  minio://<target-bucket>/<relative-key>
```

It verifies the object size and SHA-256 before recording a conversion. It
updates only `knowledges.file_path` rows whose `id`, old path, and expected
hash match the reviewed report. It also updates
`knowledge_bases.storage_provider_config` to `{"provider":"minio"}` and
the tenant `storage_engine_config` default provider to `minio` only after all
referenced knowledge files for that tenant were copied successfully.

Because MinIO multipart ETags are not a SHA-256 contract, verify every upload
by streaming it with `mc cat` to a newly created staging file and comparing
`Get-FileHash -Algorithm SHA256` with the source manifest. Remove only that
new staging file after the comparison succeeds.

Any `local://` value found outside the approved review list blocks execution;
the script must not apply a global text replacement to JSON columns.

- [ ] **Step 5: Implement validation**

`verify-merge.ps1` runs against the temporary merged database and MinIO first,
then against production only after promotion. It verifies:

```sql
SELECT conname
FROM pg_constraint
WHERE contype = 'f' AND convalidated = false;

SELECT id, file_path
FROM knowledges
WHERE deleted_at IS NULL AND file_path LIKE 'local://%';
```

It compares source versus merged row counts, confirms every retained
`knowledges.file_path` object exists with the expected hash, compares
workspace manifests, checks the migration version is clean, and writes one
machine-readable `verification.json` plus a readable summary. It returns a
nonzero exit code on any mismatch.

- [ ] **Step 6: Run tests and syntax checks**

Run:

```powershell
node --test scripts/data-plane/data-plane.test.mjs
powershell.exe -NoProfile -Command "[void][scriptblock]::Create((Get-Content -Raw 'scripts/data-plane/execute-merge.ps1'))"
powershell.exe -NoProfile -Command "[void][scriptblock]::Create((Get-Content -Raw 'scripts/data-plane/import-files-to-minio.ps1'))"
powershell.exe -NoProfile -Command "[void][scriptblock]::Create((Get-Content -Raw 'scripts/data-plane/verify-merge.ps1'))"
```

Expected: tests and parsing pass. No production connection is made in this
task.

- [ ] **Step 7: Commit reconciliation tooling**

```powershell
git add scripts/data-plane/execute-merge.ps1 scripts/data-plane/import-files-to-minio.ps1 scripts/data-plane/verify-merge.ps1 scripts/data-plane/data-plane.test.mjs
git commit -m "feat: add Knavis data reconciliation tooling"
```

## Task 6: Add Controlled Production Migration And Release Packaging

**Files:**
- Create: `scripts/run-knavis-production-migration.ps1`
- Modify: `scripts/package-knavis-internal.ps1`
- Modify: `scripts/package-knavis-internal.test.mjs`
- Create: `docs/deployment/knavis-shared-production-data-plane.md`

- [ ] **Step 1: Add failing package contract assertions**

Extend `scripts/package-knavis-internal.test.mjs`:

```js
test('internal package includes shared-data-plane templates but excludes secrets and snapshots', () => {
  assert.match(source, /docker-compose\.knavis-production\.yml/)
  assert.match(source, /\.env\.knavis-production\.example/)
  assert.match(source, /run-knavis-production-migration\.ps1/)
  assert.match(source, /knavis-shared-production-data-plane\.md/)
  assert.doesNotMatch(source, /merged-local-wins\.dump/)
  assert.doesNotMatch(source, /data-plane\\.*\.zip/)
})
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `node --test scripts/package-knavis-internal.test.mjs`

Expected: FAIL because the package script does not yet copy the new material.

- [ ] **Step 3: Add the one-time migration runner**

Create `scripts/run-knavis-production-migration.ps1`. It requires a
`-BackupManifest` path and refuses to run unless it contains a SHA-256 record
for the current PostgreSQL backup. The existing application image already
contains `/usr/local/bin/migrate` and `/app/migrations/versioned`, so run that
tool in a one-off `app` container with no published ports:

```powershell
$migrationScript = @'
set -eu
encoded_password="$(python3 -c 'import os, urllib.parse; print(urllib.parse.quote(os.environ["DB_PASSWORD"], safe=""))')"
exec /usr/local/bin/migrate \
  -path /app/migrations/versioned \
  -database "postgres://${DB_USER}:${encoded_password}@${DB_HOST}:${DB_PORT}/${DB_NAME}?sslmode=disable" up
'@

& docker compose --env-file $EnvFile -f docker-compose.yml -f docker-compose.knavis-production.yml `
  run --rm --no-deps --entrypoint /bin/sh app -ec $migrationScript
if ($LASTEXITCODE -ne 0) { throw "Production migration failed" }
```

Run the one-off command while the normal app is stopped. It receives
credentials only from the target environment file and must not echo them. Read
the `schema_migrations` version and dirty flag through the PostgreSQL container
before allowing the active app to start.

The runner must stop on a dirty migration state instead of relying on
`AUTO_RECOVER_DIRTY`.

- [ ] **Step 4: Package only templates and lifecycle scripts**

Modify `scripts/package-knavis-internal.ps1` to copy:

```text
docker-compose.knavis-production.yml
.env.knavis-production.example
scripts/run-knavis-production-migration.ps1
scripts/data-plane/*.ps1
docs/deployment/knavis-shared-production-data-plane.md
```

Keep actual `.env` files, database dumps, MinIO archives, workspace archives,
and generated merge reports outside the package and outside Git. Add their
names to the package exclusion checks.

- [ ] **Step 5: Write the operator runbook**

Document these exact gates in
`docs/deployment/knavis-shared-production-data-plane.md`:

1. Create `D:\Knavis\workspaces` on `192.168.9.177` and publish the named
   SMB share with the minimum required ACLs.
2. Map the share to `K:\KnavisWorkspaces` on the local development host and
   run `scripts/verify-shared-workspace.ps1` on both hosts.
3. Create production secrets from `.env.knavis-production.example`; set stable
   `CRYPTO_MASTER_KEY` and `CRYPTO_SALT` before moving data, because the old
   local `data-files/.crypto_state.json` cannot be treated as the production
   secret source.
4. Use `docker compose --profile minio -f docker-compose.yml -f
   docker-compose.knavis-production.yml` on `192.168.9.177`.
5. Stop both application nodes, run preflight and snapshot on each source,
   transfer only hash-verified local archives to the server staging directory,
   then run dry merge, review, execute merge, file import, and temporary
   validation in that order.
6. Promote only after `verification.json` is successful, then run the
   one-time migration runner and start the active production stack with
   `--no-build`.
7. Set the Knavis client default origin to `http://192.168.9.177:8080` only
   after server health and login smoke tests pass.
8. Roll back using the pre-cutover `192.168.9.177` backup identifiers; do not
   attempt a reverse merge.

- [ ] **Step 6: Run packaging verification**

Run:

```powershell
node --test scripts/package-knavis-internal.test.mjs scripts/shared-data-plane-compose.test.mjs scripts/data-plane/data-plane.test.mjs
docker compose --env-file .env -f docker-compose.yml -f docker-compose.knavis-production.yml config --quiet
```

Expected: all tests pass and Compose validates. The command does not deploy
or replace local containers.

- [ ] **Step 7: Commit release material**

```powershell
git add scripts/run-knavis-production-migration.ps1 scripts/package-knavis-internal.ps1 scripts/package-knavis-internal.test.mjs docs/deployment/knavis-shared-production-data-plane.md
git commit -m "docs: add Knavis shared data-plane release runbook"
```

## Task 7: Run The Real Migration In Controlled Checkpoints

**Files:**
- Create outside Git: timestamped snapshot, merge, MinIO, workspace, and
  verification artifact directories on the local host and `192.168.9.177`.
- Modify outside Git: production `.env` on `192.168.9.177` and local passive
  `.env` only after successful promotion.

- [ ] **Step 1: Preflight both sources without writes**

Run on each host with actual container, volume, and workspace names:

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File scripts/data-plane/preflight.ps1 `
  -SourceName local `
  -PostgresContainer Xelora-postgres `
  -DatabaseName Xelora `
  -DatabaseUser postgres `
  -DataFilesVolume weknora_data-files `
  -WorkspaceRoot E:\Xelora\WeKnora\data\workspaces `
  -OutputRoot E:\KnavisMigration\preflight
```

Use the equivalent actual names on `192.168.9.177`; do not reuse local volume
names by assumption. Stop immediately if reports show a dirty schema, a
schema/version mismatch, unrecognized file references, insufficient space, or
workspace share mount failure.

- [ ] **Step 2: Freeze, snapshot, and verify source archives**

After the user approves the maintenance window, stop the two application
stacks while leaving PostgreSQL available for export. Run `snapshot.ps1` once
on each host with `-MaintenanceApproved`, verify every SHA-256 manifest, and
copy the local source archive to a new `D:\Knavis\migration\<timestamp>`
directory on `192.168.9.177`. Do not copy a Docker volume directory.

- [ ] **Step 3: Dry merge and review evidence**

On `192.168.9.177`, run `build-merge-plan.ps1` with the two verified archive
paths. Review `schema-compare.json`, `table-order.json`, `conflicts.json`,
`unsupported-tables.json`, and `file-references.json`. Confirm that every
conflict records `winner: local` and every server-only record remains in the
preview. Do not continue without explicit user approval of this generated
report.

- [ ] **Step 4: Create and validate the merged candidate**

Run `execute-merge.ps1 -ApproveLocalWins` with the reviewed `merge.sql` hash,
then `import-files-to-minio.ps1`, then `verify-merge.ps1` against the
temporary candidate. Required evidence is a clean migration version, no
unresolved `local://` knowledge paths, no unvalidated foreign keys, matching
object hashes, and successful sample retrieval.

- [ ] **Step 5: Promote `192.168.9.177`**

Only after all candidate checks pass, run `promote-177.ps1` with the exact
candidate archive, MinIO object manifest, workspace manifest, and pre-cutover
backup manifest. The script restores the candidate data, starts the Knavis
production Compose profile with `--no-build`, runs the one-time migration
runner, then checks `/health`, login, knowledge-base retrieval, agent chat,
document upload/download, and one preloaded workspace Skill.

- [ ] **Step 6: Switch client delivery and preserve local development**

Build the Knavis client only after promotion passes, packaging
`http://192.168.9.177:8080` as its default origin. Keep ordinary local changes
on `docker-compose.dev.yml`. Use the shared/passive profile only for controlled
administrative verification and never publish its ports to LAN users.

- [ ] **Step 7: Roll back on failed acceptance**

If any production acceptance check fails, run `rollback-177.ps1` with the
pre-cutover backup manifest, verify the prior app/frontend/database state, and
leave the candidate artifacts intact for diagnosis. Do not delete or overwrite
either source archive.

## Final Verification Matrix

| Area | Command or action | Required result |
| --- | --- | --- |
| Go process role | `go test ./internal/container -run TestProcessRoleFromEnv` | Active/passive defaults and invalid rejection pass |
| Compose contracts | `docker compose ... config --quiet` | Production and passive overrides resolve correctly |
| Lifecycle tests | `node --test scripts/*.test.mjs scripts/data-plane/*.test.mjs` | Required safety gates remain present |
| Workspace share | `scripts/verify-shared-workspace.ps1` on both hosts | Same logical root supports hash-verified read/write |
| Source snapshots | `SHA256SUMS.txt` | Every archive hash matches before transfer/restore |
| Merge preview | `schema-compare.json`, `conflicts.json` | Same schema, all conflicts audited, local wins |
| Candidate data | `verification.json` | FK, row, object, and workspace checks pass |
| Production | `/health`, login, retrieval, agent, file, Skill smoke | `192.168.9.177` is healthy and Knavis branded |
| Rollback | restore exercise | Prior verified stack can be restored without raw volume operations |
