[CmdletBinding()]
param(
    [string]$InputDir = ".\dist\offline-images"
)

$ErrorActionPreference = "Stop"

function Write-Step {
    param([string]$Message)
    Write-Host "==> $Message" -ForegroundColor Cyan
}

function Assert-Command {
    param([string]$Name)
    if (-not (Get-Command $Name -ErrorAction SilentlyContinue)) {
        throw "Missing required command: $Name"
    }
}

Assert-Command docker

$repoRoot = Split-Path -Parent $PSScriptRoot
$inputDirAbs = Join-Path (Resolve-Path -LiteralPath $repoRoot) $InputDir
if (-not (Test-Path -LiteralPath $inputDirAbs)) {
    throw "Input directory not found: $inputDirAbs"
}

$tarFiles = Get-ChildItem -LiteralPath $inputDirAbs -Filter *.tar | Sort-Object Name
if (-not $tarFiles) {
    throw "No tar files found in: $inputDirAbs"
}

Write-Step "Loading offline images"
foreach ($tar in $tarFiles) {
    Write-Host "Loading $($tar.FullName)"
    docker load -i $tar.FullName
}

Write-Step "Loaded images"
docker images --format "table {{.Repository}}`t{{.Tag}}`t{{.ID}}" |
    Select-String "wechatopenai/xelora-|opensandbox/|paradedb/paradedb|redis"
