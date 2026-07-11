param(
    [string]$AppContainer = "Xelora-app",
    [string]$SandboxImage = "wechatopenai/xelora-sandbox:latest"
)

$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent $PSScriptRoot
$hostSmokeDir = Join-Path $repoRoot "skills\preloaded\.officecli-smoke"

if (Test-Path $hostSmokeDir) {
    Remove-Item $hostSmokeDir -Recurse -Force
}

docker exec $AppContainer docker run --rm `
    --user 1000:1000 `
    --cap-drop ALL `
    --network none `
    --pids-limit 100 `
    --security-opt no-new-privileges `
    --volumes-from $AppContainer `
    -e OFFICECLI_SKIP_UPDATE=1 `
    -e OFFICECLI_RESIDENT_FLUSH=each `
    -w /app/skills/preloaded `
    $SandboxImage `
    sh -c "set -e; mkdir -p .officecli-smoke; cd .officecli-smoke; officecli --version; officecli create smoke.docx --json; officecli create smoke.xlsx --json; officecli create smoke.pptx --json; officecli validate smoke.docx --json; officecli validate smoke.xlsx --json; officecli validate smoke.pptx --json; ls -l smoke.docx smoke.xlsx smoke.pptx"

$expectedFiles = @("smoke.docx", "smoke.xlsx", "smoke.pptx")
foreach ($file in $expectedFiles) {
    $path = Join-Path $hostSmokeDir $file
    if (!(Test-Path $path)) {
        throw "OfficeCLI smoke file was not created on host: $path"
    }
}

Get-ChildItem $hostSmokeDir | Select-Object Name,Length
Remove-Item $hostSmokeDir -Recurse -Force
Write-Host "officecli smoke passed"
