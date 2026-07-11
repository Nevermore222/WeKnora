# OpenSandbox Smoke Test

This is the shortest repo-local path for validating that the OpenSandbox helper
can reach a real OpenSandbox service and execute one simple skill script.

## Preconditions

1. Fill these variables in `.env`:
   - `XELORA_OPENSANDBOX_BASE_URL`
   - `XELORA_OPENSANDBOX_API_KEY`
   - `XELORA_OPENSANDBOX_TEMPLATE_ID`
   - optional: `XELORA_OPENSANDBOX_PYTHON`
2. Start the OpenSandbox server and recreate the app so the latest env is loaded:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\opensandbox-service.ps1 up
```

3. Rebuild the app image if the helper dependency layer changed:

```powershell
docker compose up -d --build app
```

4. Ensure the `opensandbox-server` and `app` containers are healthy:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\opensandbox-service.ps1 status
docker compose logs --tail 100 opensandbox-server
docker compose logs --tail 100 app
```

## Smoke Command

Run this from the repo root:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\opensandbox-service.ps1 smoke
```

The script:

- verifies the `app` container is running
- verifies the required OpenSandbox env vars exist inside the container
- writes a temporary request payload to the container
- calls `/app/internal/executor/scripts/opensandbox_exec.py`
- runs `skills/preloaded/data-processor/scripts/analyze.py` remotely with
  `stdin='{"items":[1,2,3,4,5]}'`

## Expected Success Shape

The helper returns JSON similar to:

```json
{
  "stdout": "...analysis output...",
  "stderr": "",
  "exit_code": 0,
  "error": "",
  "duration_ms": 1234,
  "sandbox_id": "..."
}
```

## Common Failure Patterns

- Missing env vars: the script stops before running the helper
- Import errors: rebuild `app` so `opensandbox`, `httpx`, and related packages
  are installed
- Authentication or template errors: the helper returns JSON with `exit_code: 1`
  and an error string from the OpenSandbox SDK or API
- Remote runtime errors: `stdout` or `stderr` comes back, but `exit_code` is
  non-zero because the target script failed inside the sandbox

## Notes

- This smoke test validates the helper path, not the full Xelora UI workflow.
- It uses a low-risk built-in skill script as a remote execution sample.
- Once this passes, the next step is wiring one real `provider=opensandbox`
  execution through the Xelora gateway path.
