param(
    [string]$HostRoot = (Join-Path (Split-Path -Parent $PSScriptRoot) "data\workspaces-e2e"),
    [string]$AppContainer = "Xelora-app"
)

$ErrorActionPreference = "Stop"

$resolvedRoot = (New-Item -ItemType Directory -Force -Path $HostRoot).FullName
$env:XELORA_WORKSPACE_HOST_ROOT = $resolvedRoot
$env:XELORA_WORKSPACE_CONTAINER_ROOT = "/workspaces"

docker compose up -d --build app frontend
docker compose exec -T app sh -lc "test -w /workspaces && printf workspace-mount-ok > /workspaces/.mount-smoke"

$hostProbe = Join-Path $resolvedRoot ".mount-smoke"
if (-not (Test-Path -LiteralPath $hostProbe)) {
    throw "workspace probe was not visible on the Windows host: $hostProbe"
}

$content = Get-Content -Raw -LiteralPath $hostProbe
if ($content -ne "workspace-mount-ok") {
    throw "workspace probe content mismatch: $content"
}

Remove-Item -LiteralPath $hostProbe -Force
Write-Host "Host workspace mount passed: $resolvedRoot"
