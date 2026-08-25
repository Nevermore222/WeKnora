[CmdletBinding()]
param(
    [string]$PackageDir = "",
    [switch]$RemoveOrphans
)

$ErrorActionPreference = "Stop"

if (-not (Get-Command docker -ErrorAction SilentlyContinue)) {
    throw "Missing required command: docker"
}

if ([string]::IsNullOrWhiteSpace($PackageDir)) {
    $PackageDir = $PSScriptRoot
}
$packageRoot = (Resolve-Path -LiteralPath $PackageDir).Path
$composeFile = Join-Path $packageRoot "docker-compose.yml"
$envFile = Join-Path $packageRoot ".env"
foreach ($requiredPath in @($composeFile, $envFile)) {
    if (-not (Test-Path -LiteralPath $requiredPath)) {
        throw "Required deployment file not found: $requiredPath"
    }
}

$arguments = @("compose", "--env-file", $envFile, "-f", $composeFile, "down")
if ($RemoveOrphans) {
    $arguments += "--remove-orphans"
}

Push-Location $packageRoot
try {
    Write-Host "Stopping Knavis services without deleting volumes..." -ForegroundColor Cyan
    & docker @arguments
    if ($LASTEXITCODE -ne 0) {
        throw "docker compose down failed with exit code $LASTEXITCODE"
    }
    Write-Host "Knavis services stopped. Persistent volumes were preserved." -ForegroundColor Green
} finally {
    Pop-Location
}
