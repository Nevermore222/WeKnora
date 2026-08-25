# Knavis Internal Release Packaging Plan

## Goal

Produce a reproducible internal release bundle for deployment on the Windows host at `192.168.9.177` using Docker Desktop, plus a Windows desktop client built with the internal Knavis brand and a packaged default enterprise server origin of `http://192.168.9.177:8080`.

The release keeps Xelora technical identifiers, Docker image repositories, API contracts, knowledge-base retrieval, agent orchestration, permissions, and workspace execution behavior unchanged. Only the internal frontend brand profile and release-facing client material use Knavis, consistent with the approved branding scope.

## Delivery Layout

Generate artifacts outside the Git worktree under a timestamped directory such as `E:\Xelora\deliverables\knavis-internal-<version>-<commit>-<timestamp>`:

- `server/`: Compose deployment files, runtime configuration, preloaded skills, offline image archives, import/deploy scripts, and a manifest.
- `client/`: Knavis internal Windows client payload and, when NSIS is available, the installer.
- `manifest/`: source revision, dirty-worktree summary, build tool versions, image digests, and verification results.

No generated delivery artifact is staged or committed automatically.

## Execution Tasks

### 1. Freeze inputs and preflight the target

- Record Git branch, revision, dirty paths, `VERSION`, Windows architecture, Docker Desktop availability, Compose version, Node/npm, Go, Wails, and NSIS availability.
- Check the current `.env` without copying its secrets into the release and enumerate Compose services/images, profiles, bind mounts, and published ports.
- Test reachability of `192.168.9.177` and the intended web/API ports without restarting or replacing existing containers.
- Confirm that no n8n, CL, or COBOL material is included in the internal release source list.

Verification commands:

```powershell
git status --short --branch
docker version
docker compose version
docker compose config --services
Test-NetConnection 192.168.9.177 -Port 8080
```

### 2. Build and verify the internal Knavis client

- Build the frontend with `VITE_BRAND_PROFILE=internal`.
- Build the Windows amd64 Personal client with `scripts/build-personal.ps1 -BrandProfile internal -DefaultServerURL 'http://192.168.9.177:8080'`.
- Preserve the executable's technical compatibility name where required by the existing Wails/runtime and installer wiring; expose Knavis through the frontend main logo and package metadata.
- Verify the built web assets contain Knavis and do not expose the hidden Claw Skill or Chrome Plugin navigation links.
- Run the focused desktop environment tests and frontend tests/build.

Verification commands:

```powershell
go test ./cmd/desktop
Set-Location frontend
npm run build
Set-Location ..
```

### 3. Build or reuse versioned Docker images

- Reuse the existing `wechatopenai/xelora-app`, `wechatopenai/xelora-docreader`, and `wechatopenai/xelora-sandbox` image IDs by default; internal and public releases share these service images.
- Build only `wechatopenai/xelora-ui` after the internal Knavis frontend has populated `frontend/dist`, because the static frontend contains the selected main logo.
- Rebuild `app`, `docreader`, or `sandbox` only with an explicit source-release request when core code changes require it.
- Keep infrastructure image versions from the existing Compose contract, and do not add n8n/CL/COBOL services to the package.
- Record image IDs and digests after the build.

Verification commands:

```powershell
$env:XELORA_VERSION = '<release-tag>'
$env:XELORA_BRAND_PROFILE = 'internal'
docker build --pull=false -t wechatopenai/xelora-ui:<release-tag> frontend
docker image inspect wechatopenai/xelora-app:<release-tag> wechatopenai/xelora-ui:<release-tag> wechatopenai/xelora-docreader:<release-tag> wechatopenai/xelora-sandbox:<release-tag>
```

### 4. Assemble the Windows Docker Desktop server bundle

- Copy only the runtime files required by Compose: `docker-compose.yml`, a generated deployment environment template, `config/config.yaml`, `skills/preloaded`, the offline import script, and a target deployment script.
- Export the application, frontend, docreader, sandbox, OpenSandbox, PostgreSQL, Redis, and any other enabled Compose images into image archives.
- Generate target-specific `.env` values for `192.168.9.177`; never copy the developer `.env` with its existing secrets into the release.
- Configure the client-facing ports explicitly and create the workspace host directory expected by `XELORA_WORKSPACE_HOST_ROOT`.
- Use `docker compose up -d --no-build` after image import so the target does not silently rebuild or pull a different image.

Verification commands:

```powershell
docker compose --env-file .env config
docker compose --env-file .env up -d --no-build
docker compose ps
Invoke-WebRequest http://192.168.9.177:8080/health
Invoke-WebRequest http://192.168.9.177/
```

### 5. Verify deployment and client handoff

- Confirm all required services are healthy and the frontend proxies to the app service.
- Confirm the client can reach the packaged origin and that the login page is Knavis-branded.
- Confirm existing agent/RAG behavior is not changed by the packaging operation; packaging verification is limited to build/runtime health and does not claim a new knowledge-base regression suite.
- Write a manifest with exact artifact paths, source revision, image tags/digests, target address, and any unverified external dependency.

## Stop Conditions

- Do not overwrite existing target containers or persistent volumes until their ownership and current status are known.
- Do not package `.env` secrets, local databases, developer workspaces, n8n data, or CL/COBOL artifacts.
- If Docker Desktop, Wails/Go, or NSIS is unavailable, produce the verified subset and record the missing prerequisite instead of claiming a complete package.
