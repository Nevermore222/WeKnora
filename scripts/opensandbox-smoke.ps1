Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent $PSScriptRoot
$containerRequestPath = "/tmp/opensandbox-smoke-request.json"
$helperPath = "/app/internal/executor/scripts/opensandbox_exec.py"

Write-Host "[1/5] Checking app container..."
$containerId = docker compose ps -q app
if (-not $containerId) {
    throw "app service is not running. Start it with 'docker compose up -d app' first."
}

Write-Host "[2/5] Verifying OpenSandbox environment inside app container..."
$missing = docker compose exec -T app sh -lc @'
missing=""
for name in XELORA_OPENSANDBOX_BASE_URL XELORA_OPENSANDBOX_API_KEY XELORA_OPENSANDBOX_TEMPLATE_ID; do
  if [ -z "$(printenv "$name")" ]; then
    missing="$missing $name"
  fi
done
printf "%s" "$missing"
'@
if ($missing -and $missing.Trim()) {
    throw "Missing required env vars in app container:$missing`nUpdate .env, rebuild app, then retry."
}

Write-Host "[3/5] Preparing smoke request payload..."
$request = @{
    base_path   = "/app/skills/preloaded/data-processor"
    script_path = "/app/skills/preloaded/data-processor/scripts/analyze.py"
    args        = @("--type", "numeric")
    stdin       = '{"items":[1,2,3,4,5]}'
    timeout_sec = 60
} | ConvertTo-Json -Compress

$request | docker compose exec -T app sh -lc "cat > $containerRequestPath"

Write-Host "[4/5] Running OpenSandbox helper..."
$raw = docker compose exec -T app python3 $helperPath $containerRequestPath

Write-Host "[5/5] Result"
$raw

try {
    $result = $raw | ConvertFrom-Json
    if ($result.exit_code -eq 0 -and -not $result.error) {
        Write-Host ""
        Write-Host "Smoke test succeeded." -ForegroundColor Green
    } else {
        Write-Host ""
        Write-Warning "Smoke test reached helper but the remote command did not succeed."
    }
} catch {
    Write-Warning "Helper output was not valid JSON. Inspect the raw output above."
}
