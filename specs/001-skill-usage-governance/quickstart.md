# Quickstart: Executor Runtime Planning Validation

This quickstart validates the planned runtime shape before implementation tasks are generated.

## 1. Confirm Xelora Services

```powershell
cd F:\Docker\WeKnora\Xelora
docker compose ps
```

Expected:

- `app`, `frontend`, `postgres`, and `redis` are running for the current local deployment.
- Existing dirty worktree changes are understood before implementing this plan.

## 2. Confirm WSL/Linux Readiness For CubeSandbox

From Windows:

```powershell
wsl --status
wsl -l -v
```

From the target WSL distro:

```bash
uname -a
ls -l /dev/kvm || true
```

Expected:

- A Linux environment is available for CubeSandbox setup.
- KVM/PVM capability is verified or explicitly documented as missing.

## 3. Validate Gateway Contract Without CubeSandbox

Before integrating CubeSandbox, implement a local provider stub and verify:

```powershell
# Example future flow
Invoke-RestMethod -Method Post http://localhost:8080/api/executor/workspaces -Body '{"session_id":"demo"}' -ContentType 'application/json'
Invoke-RestMethod -Method Post http://localhost:8080/api/executor/jobs -Body '{"workspace_id":"demo","provider":"local","command":"python","args":["-c","open(\"report.md\",\"w\").write(\"hello\")"]}' -ContentType 'application/json'
```

Expected:

- A job is created.
- Logs are captured.
- `report.md` is registered as an artifact.
- The artifact is downloadable from Xelora.

## 4. Validate CubeSandbox Provider Adapter

After CubeSandbox is running in WSL/Linux, configure Gateway with:

```text
EXECUTOR_PROVIDER=cubesandbox
CUBESANDBOX_ENDPOINT=<local-or-remote-endpoint>
EXECUTOR_WORKSPACE_ROOT=<gateway-managed-workspace-root>
```

Run the same job contract used by the local provider.

Expected:

- The same API request runs through CubeSandbox.
- Workspace files remain available to Xelora after job completion.
- Provider failures are reported as structured job errors.

## 5. Validate Web Agent Flow

Use a web chat session and ask the agent to create a small Markdown file in the session workspace.

Expected:

- The agent invokes Executor Gateway.
- The UI shows job status and logs.
- A real Markdown artifact appears in the conversation.
- Download returns the generated file, not just model-written text.
