param(
    [ValidateSet("up", "down", "status", "logs", "smoke", "restart-app")]
    [string]$Action = "status"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent $PSScriptRoot

function Invoke-Compose {
    param(
        [Parameter(ValueFromRemainingArguments = $true)]
        [string[]]$Args
    )

    Push-Location $repoRoot
    try {
        & docker compose @Args
    } finally {
        Pop-Location
    }
}

function Get-EnvSetting {
    param([string]$Name)

    $envFile = Join-Path $repoRoot ".env"
    if (-not (Test-Path $envFile)) {
        return $null
    }

    $line = Get-Content $envFile | Where-Object { $_ -match "^$Name=" } | Select-Object -First 1
    if (-not $line) {
        return $null
    }
    return ($line -split "=", 2)[1].Trim()
}

switch ($Action) {
    "up" {
        $images = @(
            "opensandbox/server:latest",
            (Get-EnvSetting "XELORA_OPENSANDBOX_EXECD_IMAGE"),
            (Get-EnvSetting "XELORA_OPENSANDBOX_EGRESS_IMAGE"),
            (Get-EnvSetting "XELORA_OPENSANDBOX_TEMPLATE_ID")
        ) | Where-Object { $_ }

        Write-Host "[1/3] Pre-pulling OpenSandbox images..."
        $images | Select-Object -Unique | ForEach-Object {
            Write-Host "Pulling $_"
            docker pull $_ | Out-Host
        }

        Write-Host "[2/3] Starting OpenSandbox server..."
        Invoke-Compose up -d opensandbox-server

        Write-Host "[3/3] Recreating app so OpenSandbox env is loaded..."
        Invoke-Compose up -d app
        break
    }
    "down" {
        Invoke-Compose stop opensandbox-server
        break
    }
    "restart-app" {
        Invoke-Compose up -d app
        break
    }
    "status" {
        Invoke-Compose ps opensandbox-server app
        Write-Host ""
        try {
            $health = Invoke-RestMethod -Uri "http://127.0.0.1:8090/health" -TimeoutSec 10
            Write-Host "OpenSandbox health: $($health.status)"
        } catch {
            Write-Warning "OpenSandbox health endpoint is not reachable on http://127.0.0.1:8090/health"
        }
        break
    }
    "logs" {
        Invoke-Compose logs --tail 120 opensandbox-server
        break
    }
    "smoke" {
        & (Join-Path $PSScriptRoot "opensandbox-smoke.ps1")
        break
    }
}
