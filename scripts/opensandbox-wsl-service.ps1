param(
    [ValidateSet("up", "down", "status", "logs", "sync-config", "restart-app", "smoke")]
    [string]$Action = "status",
    [string]$Distro = "Ubuntu-22.04"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent $PSScriptRoot
$envFile = Join-Path $repoRoot ".env"
$wslRoot = "/root/opensandbox-wsl"
$wslConfigPath = "$wslRoot/config.toml"
$wslPidPath = "$wslRoot/run/opensandbox-server.pid"
$wslLogPath = "$wslRoot/logs/opensandbox-server.log"
$wslPort = 18090

function Get-EnvSetting {
    param([string]$Name)
    $line = Get-Content $envFile | Where-Object { $_ -match "^$Name=" } | Select-Object -First 1
    if (-not $line) { return $null }
    return ($line -split "=", 2)[1].Trim()
}

function Invoke-Wsl {
    param([string]$Command)
    & wsl -d $Distro -- bash -lc $Command
}

function Convert-ToWslPath {
    param([string]$WindowsPath)
    $normalized = $WindowsPath -replace '\\', '/'
    if ($normalized -match '^([A-Za-z]):/(.*)$') {
        return "/mnt/$($matches[1].ToLower())/$($matches[2])"
    }
    throw "Unsupported Windows path for WSL conversion: $WindowsPath"
}

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

function Sync-WslConfig {
    $apiKey = Get-EnvSetting "XELORA_OPENSANDBOX_API_KEY"
    $execdImage = Get-EnvSetting "XELORA_OPENSANDBOX_EXECD_IMAGE"
    $egressImage = Get-EnvSetting "XELORA_OPENSANDBOX_EGRESS_IMAGE"
    $networkMode = Get-EnvSetting "XELORA_OPENSANDBOX_DOCKER_NETWORK_MODE"
    $hostIp = Get-EnvSetting "XELORA_OPENSANDBOX_SERVER_HOST_IP"
    $portMin = Get-EnvSetting "XELORA_OPENSANDBOX_PORT_RANGE_MIN"
    $portMax = Get-EnvSetting "XELORA_OPENSANDBOX_PORT_RANGE_MAX"
    $pidsLimit = Get-EnvSetting "XELORA_OPENSANDBOX_PIDS_LIMIT"
    $logLevel = Get-EnvSetting "XELORA_OPENSANDBOX_LOG_LEVEL"

    if (-not $networkMode) { $networkMode = "bridge" }
    if (-not $hostIp) { $hostIp = "host.docker.internal" }

    $config = @"
[server]
host = "0.0.0.0"
port = $wslPort
api_key = "$apiKey"

[log]
level = "${logLevel}"

[runtime]
type = "docker"
execd_image = "$execdImage"

[egress]
image = "$egressImage"

[docker]
network_mode = "$networkMode"
host_ip = "$hostIp"
port_range_min = $portMin
port_range_max = $portMax
drop_capabilities = ["AUDIT_WRITE", "MKNOD", "NET_ADMIN", "NET_RAW", "SYS_ADMIN", "SYS_MODULE", "SYS_PTRACE", "SYS_TIME", "SYS_TTY_CONFIG"]
no_new_privileges = true
pids_limit = $pidsLimit

[store]
type = "sqlite"
path = "$wslRoot/data/opensandbox.db"

[ingress]
mode = "direct"
"@

    $tempPath = Join-Path $repoRoot ".opensandbox-wsl-config.toml"
    $utf8NoBom = New-Object System.Text.UTF8Encoding($false)
    [System.IO.File]::WriteAllText($tempPath, $config, $utf8NoBom)
    $tempWslPath = Convert-ToWslPath $tempPath
    Invoke-Wsl ('mkdir -p ' + $wslRoot + '/run ' + $wslRoot + '/logs ' + $wslRoot + '/data; cp "' + $tempWslPath + '" "' + $wslConfigPath + '"')
}

switch ($Action) {
    "sync-config" {
        Sync-WslConfig
    }
    "up" {
        Sync-WslConfig
        Invoke-Wsl ('mkdir -p ' + $wslRoot + '/run ' + $wslRoot + '/logs ' + $wslRoot + '/data')
        Invoke-Wsl ('if [ -f ' + $wslPidPath + ' ] && kill -0 $(cat ' + $wslPidPath + ') 2>/dev/null; then echo already-running; exit 0; fi; nohup /root/.local/bin/opensandbox-server --config ' + $wslConfigPath + ' > ' + $wslLogPath + ' 2>&1 & echo $! > ' + $wslPidPath + '; sleep 3')
        Invoke-Compose up -d app
        try {
            Invoke-RestMethod -Uri "http://127.0.0.1:$wslPort/health" -TimeoutSec 10 | Out-Null
            Write-Host "OpenSandbox WSL service is healthy on port $wslPort"
        } catch {
            Write-Warning "WSL OpenSandbox process started, but health endpoint is not yet ready."
        }
    }
    "down" {
        Invoke-Wsl ('if [ -f ' + $wslPidPath + ' ]; then kill $(cat ' + $wslPidPath + ') 2>/dev/null || true; rm -f ' + $wslPidPath + '; fi')
    }
    "status" {
        Invoke-Wsl ('if [ -f ' + $wslPidPath + ' ] && kill -0 $(cat ' + $wslPidPath + ') 2>/dev/null; then echo PID:$(cat ' + $wslPidPath + '); else echo PID:stopped; fi')
        try {
            $health = Invoke-RestMethod -Uri "http://127.0.0.1:$wslPort/health" -TimeoutSec 10
            Write-Host "Health: $($health.status)"
        } catch {
            Write-Warning "Health endpoint not reachable on http://127.0.0.1:$wslPort/health"
        }
    }
    "logs" {
        Invoke-Wsl ('tail -n 120 ' + $wslLogPath + ' 2>/dev/null || true')
    }
    "restart-app" {
        Invoke-Compose up -d app
    }
    "smoke" {
        & (Join-Path $PSScriptRoot "opensandbox-smoke.ps1")
    }
}
