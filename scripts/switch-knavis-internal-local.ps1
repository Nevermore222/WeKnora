[CmdletBinding()]
param(
    [string]$LanIP = "",
    [ValidateRange(1, 65535)]
    [int]$AppPort = 8080,
    [ValidateRange(1, 65535)]
    [int]$FrontendPort = 80,
    [string]$ImageTag = "",
    [switch]$SkipImageBuild,
    [switch]$SkipClientBuild,
    [switch]$SkipHealthCheck
)

$ErrorActionPreference = "Stop"
$repoRoot = Split-Path -Parent $PSScriptRoot
$composeFile = Join-Path $repoRoot "docker-compose.yml"
$overlayFile = Join-Path $repoRoot "docker-compose.knavis-internal.yml"
$sourceEnvFile = Join-Path $repoRoot ".env"
$internalEnvFile = Join-Path $repoRoot ".env.knavis-internal"

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

function Set-EnvValue {
    param(
        [string]$Path,
        [string]$Name,
        [string]$Value
    )

    $content = [System.IO.File]::ReadAllText($Path)
    $pattern = "(?m)^$([regex]::Escape($Name))=.*$"
    $replacement = "$Name=$Value"
    if ([regex]::IsMatch($content, $pattern)) {
        $content = [regex]::Replace($content, $pattern, [System.Text.RegularExpressions.MatchEvaluator]{ param($match) $replacement })
    } else {
        if (-not $content.EndsWith("`n")) { $content += "`n" }
        $content += "$replacement`n"
    }
    [System.IO.File]::WriteAllText($Path, $content, (New-Object System.Text.UTF8Encoding($false)))
}

function Resolve-LanIP {
    $addresses = @(Get-NetIPAddress -AddressFamily IPv4 -ErrorAction Stop | Where-Object {
        $_.AddressState -eq "Preferred" -and
        $_.IPAddress -notmatch "^(127|169\.254)" -and
        $_.InterfaceAlias -notmatch "(?i)(loopback|vEthernet|default switch|docker)"
    })
    $preferred = $addresses | Where-Object { $_.PrefixLength -ge 16 -and $_.PrefixLength -le 30 } | Select-Object -First 1
    if ($null -eq $preferred) { $preferred = $addresses | Select-Object -First 1 }
    if ($null -eq $preferred) { throw "Could not determine a LAN IPv4 address. Pass -LanIP explicitly." }
    return $preferred.IPAddress
}

function Get-ComposeArguments {
    param([string]$EnvFile)
    return @(
        "compose",
        "--env-file", $EnvFile,
        "-f", $composeFile,
        "-f", $overlayFile
    )
}

function Wait-HttpHealthy {
    param(
        [string]$Url,
        [ValidateRange(1, 600)]
        [int]$TimeoutSeconds = 60
    )

    $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
    $lastFailure = "No response received"
    do {
        try {
            $response = Invoke-WebRequest -UseBasicParsing -Uri $Url -TimeoutSec 10
            if ($response.StatusCode -ge 200 -and $response.StatusCode -lt 400) {
                return $response
            }
            $lastFailure = "HTTP $($response.StatusCode)"
        } catch {
            $lastFailure = $_.Exception.Message
        }
        Start-Sleep -Seconds 2
    } while ((Get-Date) -lt $deadline)

    throw "Health check did not become ready within $TimeoutSeconds seconds: $Url. Last failure: $lastFailure"
}

foreach ($requiredPath in @($composeFile, $overlayFile, $sourceEnvFile)) {
    if (-not (Test-Path -LiteralPath $requiredPath)) {
        throw "Required file not found: $requiredPath"
    }
}

if ([string]::IsNullOrWhiteSpace($LanIP)) { $LanIP = Resolve-LanIP }
if ([string]::IsNullOrWhiteSpace($ImageTag)) {
    $commitID = (git -C $repoRoot rev-parse --short HEAD 2>$null).Trim()
    if ([string]::IsNullOrWhiteSpace($commitID)) { $commitID = "unknown" }
    $ImageTag = "knavis-internal-local-$commitID-$(Get-Date -Format yyyyMMdd-HHmmss)"
}

Write-Host "==> Preparing local Knavis configuration" -ForegroundColor Cyan
Copy-Item -LiteralPath $sourceEnvFile -Destination $internalEnvFile -Force
Set-EnvValue -Path $internalEnvFile -Name "XELORA_BRAND_PROFILE" -Value "internal"
Set-EnvValue -Path $internalEnvFile -Name "XELORA_VERSION" -Value $ImageTag
Set-EnvValue -Path $internalEnvFile -Name "XELORA_APP_IMAGE" -Value "wechatopenai/xelora-app:latest"
Set-EnvValue -Path $internalEnvFile -Name "XELORA_RUNTIME_VERSION" -Value "latest"
# The source .env defaults to localhost for host-side development. The app
# container must use Docker Compose service discovery after it is recreated.
Set-EnvValue -Path $internalEnvFile -Name "DB_HOST" -Value "postgres"
Set-EnvValue -Path $internalEnvFile -Name "APP_PORT" -Value ([string]$AppPort)
Set-EnvValue -Path $internalEnvFile -Name "FRONTEND_PORT" -Value ([string]$FrontendPort)
Set-EnvValue -Path $internalEnvFile -Name "APP_BACKEND_PORT" -Value "8080"
Set-EnvValue -Path $internalEnvFile -Name "XELORA_SANDBOX_DOCKER_IMAGE" -Value "wechatopenai/xelora-sandbox:latest"
Set-EnvValue -Path $internalEnvFile -Name "XELORA_WORKSPACE_HOST_ROOT" -Value "./data/workspaces"
Set-EnvValue -Path $internalEnvFile -Name "XELORA_WORKSPACE_CONTAINER_ROOT" -Value "/workspaces"

if (-not $SkipImageBuild) {
    Write-Host "==> Building internal Knavis frontend assets" -ForegroundColor Cyan
    $previousViteProfile = [Environment]::GetEnvironmentVariable("VITE_BRAND_PROFILE", "Process")
    $env:VITE_BRAND_PROFILE = "internal"
    Push-Location (Join-Path $repoRoot "frontend")
    try {
        Invoke-Checked "npm" @("ci", "--silent")
        Invoke-Checked "npm" @("run", "build")
    } finally {
        Pop-Location
        if ($null -eq $previousViteProfile) {
            Remove-Item Env:VITE_BRAND_PROFILE -ErrorAction SilentlyContinue
        } else {
            $env:VITE_BRAND_PROFILE = $previousViteProfile
        }
    }

    Write-Host "==> Building tagged internal frontend image: $ImageTag" -ForegroundColor Cyan
    Invoke-Checked "docker" @(
        "build",
        "--pull=false",
        "-t", "wechatopenai/xelora-ui:$ImageTag",
        (Join-Path $repoRoot "frontend")
    )
} else {
    Write-Host "==> Skipping image build" -ForegroundColor Yellow
}

$composeArguments = Get-ComposeArguments -EnvFile $internalEnvFile
Write-Host "==> Validating local internal Compose configuration" -ForegroundColor Cyan
Invoke-Checked "docker" ($composeArguments + @("config", "--quiet"))

Write-Host "==> Switching local Docker services to Knavis internal images" -ForegroundColor Cyan
Invoke-Checked "docker" ($composeArguments + @("up", "-d", "--no-build"))
& docker @($composeArguments + @("ps"))

if (-not $SkipClientBuild) {
    $serverOrigin = "http://$LanIP`:$AppPort"
    $clientOutput = Join-Path $repoRoot "dist\knavis-internal-client"
    Write-Host "==> Building Knavis Personal client for $serverOrigin" -ForegroundColor Cyan
    $buildScript = Join-Path $repoRoot "scripts\build-personal.ps1"
    Invoke-Checked "powershell.exe" @(
        "-NoProfile",
        "-ExecutionPolicy", "Bypass",
        "-File", $buildScript,
        "-BrandProfile", "internal",
        "-SkipFrontend",
        "-DefaultServerURL", $serverOrigin,
        "-DefaultServerName", "Knavis Server",
        "-OutputDir", $clientOutput
    )
}

if (-not $SkipHealthCheck) {
    foreach ($url in @("http://127.0.0.1:$AppPort/health", "http://127.0.0.1:$FrontendPort/")) {
        Write-Host "==> Checking $url" -ForegroundColor Cyan
        $response = Wait-HttpHealthy -Url $url
        Write-Host "    HTTP $($response.StatusCode)" -ForegroundColor Green
    }
}

Write-Host "" 
Write-Host "Knavis internal local service is running." -ForegroundColor Green
$webOrigin = "http://$LanIP"
if ($FrontendPort -ne 80) { $webOrigin += ":$FrontendPort" }
Write-Host "LAN web: $webOrigin"
Write-Host "LAN API: http://$LanIP`:$AppPort"
Write-Host "Image tag: $ImageTag"
Write-Host "Runtime env: $internalEnvFile"
