[CmdletBinding()]
param(
    [string]$EnvFile = ".env.knavis-passive",
    [string]$DeploymentRoot = (Split-Path -Parent $PSScriptRoot)
)

$ErrorActionPreference = "Stop"
$passiveRoot = Split-Path -Parent $PSScriptRoot
if (-not (Test-Path -LiteralPath $DeploymentRoot -PathType Container)) {
    throw "Deployment root is missing: $DeploymentRoot"
}
$deploymentRoot = (Resolve-Path -LiteralPath $DeploymentRoot).Path
$base = Join-Path $deploymentRoot "docker-compose.yml"
$passive = Join-Path $passiveRoot "docker-compose.knavis-passive.yml"
$envPath = if ([System.IO.Path]::IsPathRooted($EnvFile)) {
    $EnvFile
} else {
    Join-Path $deploymentRoot $EnvFile
}

foreach ($path in @($base, $passive, $envPath)) {
    if (-not (Test-Path -LiteralPath $path)) {
        throw "Missing required file: $path"
    }
}

$envPath = (Resolve-Path -LiteralPath $envPath).Path
$projectName = "knavis-passive"
$hadServiceEnvFile = Test-Path Env:KNAVIS_PASSIVE_SERVICE_ENV_FILE
$previousServiceEnvFile = $env:KNAVIS_PASSIVE_SERVICE_ENV_FILE
$env:KNAVIS_PASSIVE_SERVICE_ENV_FILE = $envPath

try {
    & docker compose -p $projectName --env-file $envPath -f $base -f $passive config --quiet
    if ($LASTEXITCODE -ne 0) {
        throw "Passive Compose validation failed"
    }

    & docker compose -p $projectName --env-file $envPath -f $base -f $passive up -d --no-deps app frontend
    if ($LASTEXITCODE -ne 0) {
        throw "Passive services failed to start"
    }
} finally {
    if ($hadServiceEnvFile) {
        $env:KNAVIS_PASSIVE_SERVICE_ENV_FILE = $previousServiceEnvFile
    } else {
        Remove-Item Env:KNAVIS_PASSIVE_SERVICE_ENV_FILE -ErrorAction SilentlyContinue
    }
}
