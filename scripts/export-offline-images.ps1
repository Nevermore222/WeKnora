[CmdletBinding()]
param(
    [string]$OutputDir = ".\dist\offline-images",
    [string]$ComposeFile = ".\docker-compose.yml",
    [string]$Version = "latest",
    [switch]$SkipBuild,
    [switch]$SkipPull
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
$outputPath = Resolve-Path -LiteralPath $repoRoot
$outputDirAbs = Join-Path $outputPath $OutputDir
New-Item -ItemType Directory -Force -Path $outputDirAbs | Out-Null

$env:XELORA_VERSION = $Version

$coreImages = @(
    "wechatopenai/xelora-app:$Version",
    "wechatopenai/xelora-ui:$Version",
    "wechatopenai/xelora-docreader:$Version",
    "wechatopenai/xelora-sandbox:$Version",
    "opensandbox/server:latest",
    "opensandbox/execd:v1.0.20",
    "opensandbox/egress:v1.1.3",
    "paradedb/paradedb:v0.22.2-pg17",
    "redis:7.0-alpine"
)

if (-not $SkipPull) {
    Write-Step "Pulling runtime dependency images"
    foreach ($image in $coreImages[4..($coreImages.Count - 1)]) {
        docker pull $image
    }
}

if (-not $SkipBuild) {
    Write-Step "Building project images from source"
    docker compose -f $ComposeFile build app frontend docreader sandbox
}

Write-Step "Exporting images to tar archives"
foreach ($image in $coreImages) {
    $safeName = ($image -replace "[:/]", "_")
    $tarPath = Join-Path $outputDirAbs "$safeName.tar"
    Write-Host "Saving $image -> $tarPath"
    docker save -o $tarPath $image
}

$manifestPath = Join-Path $outputDirAbs "manifest.txt"
$coreImages | Set-Content -LiteralPath $manifestPath -Encoding ascii

Write-Step "Done"
Write-Host "Output directory: $outputDirAbs"
Write-Host "Manifest: $manifestPath"
