[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string]$HostRoot,
    [string]$AppContainer = "Knavis-passive-app"
)

$ErrorActionPreference = "Stop"
$probeName = ".shared-mount-smoke"
$resolvedRoot = (New-Item -ItemType Directory -Force -Path $HostRoot).FullName
$hostProbe = Join-Path $resolvedRoot $probeName

if (Test-Path -LiteralPath $hostProbe) {
    throw "Refusing to overwrite an existing workspace probe: $hostProbe"
}

$payload = "knavis-shared-workspace-$([Guid]::NewGuid().ToString('N'))"
$payloadBytes = [Text.Encoding]::UTF8.GetBytes($payload)
$sha256 = [Security.Cryptography.SHA256]::Create()
try {
    $expectedHash = ([BitConverter]::ToString($sha256.ComputeHash($payloadBytes))).Replace("-", "")
} finally {
    $sha256.Dispose()
}

$probeCreated = $false
try {
    $containerCommand = "set -eu; test -d /workspaces; test -w /workspaces; printf '%s' '$payload' > /workspaces/$probeName"
    & docker exec $AppContainer sh -lc $containerCommand
    if ($LASTEXITCODE -ne 0) {
        throw "Container workspace probe failed: $AppContainer"
    }
    $probeCreated = $true

    if (-not (Test-Path -LiteralPath $hostProbe)) {
        throw "Workspace probe was not visible on the Windows host: $hostProbe"
    }

    $actualHash = (Get-FileHash -LiteralPath $hostProbe -Algorithm SHA256).Hash
    if ($actualHash -ne $expectedHash) {
        throw "Workspace probe SHA-256 mismatch: expected $expectedHash, got $actualHash"
    }

    Write-Host "Shared workspace mount passed: $resolvedRoot"
} finally {
    if ($probeCreated -and (Test-Path -LiteralPath $hostProbe)) {
        Remove-Item -LiteralPath $hostProbe -Force
    }
}
