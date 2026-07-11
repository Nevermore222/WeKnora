param(
    [string]$AppContainer = "Xelora-app",
    [string]$SandboxImage = "wechatopenai/xelora-sandbox:latest"
)

$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent $PSScriptRoot
$hostSmokeFile = Join-Path $repoRoot "skills\preloaded\controlled-docker-smoke.md"

if (Test-Path $hostSmokeFile) {
    Remove-Item $hostSmokeFile -Force
}

docker exec $AppContainer docker run --rm `
    --user 1000:1000 `
    --cap-drop ALL `
    --network none `
    --pids-limit 100 `
    --security-opt no-new-privileges `
    --volumes-from $AppContainer `
    -w /app/skills/preloaded `
    $SandboxImage `
    sh -c "echo '# controlled docker smoke' > controlled-docker-smoke.md && ls -l controlled-docker-smoke.md"

if (!(Test-Path $hostSmokeFile)) {
    throw "Smoke file was not created on host: $hostSmokeFile"
}

Get-Content $hostSmokeFile
Remove-Item $hostSmokeFile -Force
Write-Host "controlled-docker smoke passed"
