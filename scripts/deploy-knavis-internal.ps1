[CmdletBinding()]
param(
    [string]$PackageDir = "",
    [switch]$AllowReplaceExisting,
    [switch]$SkipImageImport,
    [switch]$SkipHealthCheck
)

$ErrorActionPreference = "Stop"

function Assert-Command {
    param([string]$Name)
    if (-not (Get-Command $Name -ErrorAction SilentlyContinue)) {
        throw "Missing required command: $Name"
    }
}

function Invoke-Checked {
    param(
        [string]$FilePath,
        [string[]]$Arguments
    )

    & $FilePath @Arguments
    if ($LASTEXITCODE -ne 0) {
        throw "$FilePath failed with exit code $LASTEXITCODE"
    }
}

function Get-EnvValue {
    param(
        [string]$Path,
        [string]$Name,
        [string]$Fallback
    )

    $line = Get-Content -LiteralPath $Path | Where-Object {
        $_ -match "^$([regex]::Escape($Name))="
    } | Select-Object -First 1
    if ($null -eq $line) { return $Fallback }
    return (($line -split "=", 2)[1]).Trim()
}

Assert-Command docker
if ([string]::IsNullOrWhiteSpace($PackageDir)) {
    $scriptCompose = Join-Path $PSScriptRoot "docker-compose.yml"
    if (Test-Path -LiteralPath $scriptCompose) {
        $PackageDir = $PSScriptRoot
    } else {
        $PackageDir = Split-Path -Parent $PSScriptRoot
    }
}
$packageRoot = (Resolve-Path -LiteralPath $PackageDir).Path
$composeFile = Join-Path $packageRoot "docker-compose.yml"
$envFile = Join-Path $packageRoot ".env"
$offlineDir = Join-Path $packageRoot "offline-images"

foreach ($requiredPath in @($composeFile, $envFile)) {
    if (-not (Test-Path -LiteralPath $requiredPath)) {
        throw "Required deployment file not found: $requiredPath"
    }
}

Invoke-Checked "docker" @("info", "--format", "{{.OSType}} {{.Architecture}} {{.ServerVersion}}")

$serviceNames = @(
    "Xelora-frontend",
    "Xelora-app",
    "Xelora-docreader",
    "Xelora-postgres",
    "Xelora-redis",
    "Xelora-opensandbox"
)
$existingNames = @(docker ps -a --format "{{.Names}}" | Where-Object { $serviceNames -contains $_ })
if ($existingNames.Count -gt 0 -and -not $AllowReplaceExisting) {
    throw "Existing Xelora containers detected: $($existingNames -join ', '). Re-run with -AllowReplaceExisting only after confirming their data and ownership."
}

Push-Location $packageRoot
try {
    if (-not $SkipImageImport) {
        if (-not (Test-Path -LiteralPath $offlineDir)) {
            throw "Offline image directory not found: $offlineDir"
        }
        $tarFiles = @(Get-ChildItem -LiteralPath $offlineDir -Filter *.tar | Sort-Object Name)
        if ($tarFiles.Count -eq 0) {
            throw "No offline image archives found in: $offlineDir"
        }
        foreach ($tar in $tarFiles) {
            Write-Host "==> Loading $($tar.Name)" -ForegroundColor Cyan
            Invoke-Checked "docker" @("load", "-i", $tar.FullName)
        }
    }

    Write-Host "==> Validating Compose configuration" -ForegroundColor Cyan
    Invoke-Checked "docker" @("compose", "--env-file", $envFile, "-f", $composeFile, "config", "--quiet")

    Write-Host "==> Starting Knavis internal services" -ForegroundColor Cyan
    Invoke-Checked "docker" @("compose", "--env-file", $envFile, "-f", $composeFile, "up", "-d", "--no-build")
    docker compose --env-file $envFile -f $composeFile ps

    if (-not $SkipHealthCheck) {
        $appPort = Get-EnvValue -Path $envFile -Name "APP_PORT" -Fallback "8080"
        $frontendPort = Get-EnvValue -Path $envFile -Name "FRONTEND_PORT" -Fallback "80"
        foreach ($url in @("http://127.0.0.1:$appPort/health", "http://127.0.0.1:$frontendPort/")) {
            Write-Host "==> Checking $url" -ForegroundColor Cyan
            $response = Invoke-WebRequest -Uri $url -UseBasicParsing -TimeoutSec 30
            if ($response.StatusCode -lt 200 -or $response.StatusCode -ge 400) {
                throw "Health check returned HTTP $($response.StatusCode): $url"
            }
            Write-Host "    HTTP $($response.StatusCode)" -ForegroundColor Green
        }
    }

    Write-Host "Knavis internal deployment is running." -ForegroundColor Green
} finally {
    Pop-Location
}
