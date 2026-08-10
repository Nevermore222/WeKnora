[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string]$ServerPackageDir,
    [string]$DestinationName = 'shared-data-plane'
)

$ErrorActionPreference = 'Stop'
$repoRoot = Split-Path -Parent $PSScriptRoot

function Assert-ExistingFile {
    param([Parameter(Mandatory = $true)][string]$Path, [Parameter(Mandatory = $true)][string]$Label)
    if (-not (Test-Path -LiteralPath $Path -PathType Leaf)) {
        throw "$Label is missing: $Path"
    }
    return (Resolve-Path -LiteralPath $Path).Path
}

function Write-Utf8NoBom {
    param([Parameter(Mandatory = $true)][string]$Path, [Parameter(Mandatory = $true)][string]$Content)
    [IO.File]::WriteAllText($Path, $Content, [Text.UTF8Encoding]::new($false))
}

$serverRoot = (Resolve-Path -LiteralPath $ServerPackageDir).Path
Assert-ExistingFile -Path (Join-Path $serverRoot 'docker-compose.yml') -Label 'Base server package Compose file' | Out-Null
if ($DestinationName -notmatch '^[A-Za-z0-9_-]+$') {
    throw "DestinationName is invalid: $DestinationName"
}
$destinationRoot = Join-Path $serverRoot $DestinationName
if (Test-Path -LiteralPath $destinationRoot) {
    throw "Shared data-plane package destination already exists: $destinationRoot"
}

$requiredFiles = @(
    [pscustomobject]@{ source = 'docker-compose.knavis-production.yml'; destination = 'docker-compose.knavis-production.yml' },
    [pscustomobject]@{ source = '.env.knavis-production.example'; destination = '.env.knavis-production.example' },
    [pscustomobject]@{ source = '.env.knavis-passive.example'; destination = '.env.knavis-passive.example' },
    [pscustomobject]@{ source = 'scripts\run-knavis-production-migration.ps1'; destination = 'scripts\run-knavis-production-migration.ps1' },
    [pscustomobject]@{ source = 'scripts\start-knavis-passive.ps1'; destination = 'scripts\start-knavis-passive.ps1' },
    [pscustomobject]@{ source = 'scripts\verify-shared-workspace.ps1'; destination = 'scripts\verify-shared-workspace.ps1' },
    [pscustomobject]@{ source = 'docs\deployment\knavis-shared-production-data-plane.md'; destination = 'docs\knavis-shared-production-data-plane.md' }
)
foreach ($entry in $requiredFiles) {
    Assert-ExistingFile -Path (Join-Path $repoRoot $entry.source) -Label "Release material $($entry.source)" | Out-Null
}
$dataPlaneScripts = @(Get-ChildItem -LiteralPath (Join-Path $repoRoot 'scripts\data-plane') -Filter '*.ps1' -File)
if ($dataPlaneScripts.Count -eq 0) {
    throw 'No data-plane PowerShell scripts are available for packaging'
}

New-Item -ItemType Directory -Path (Join-Path $destinationRoot 'scripts\data-plane'), (Join-Path $destinationRoot 'docs') | Out-Null
foreach ($entry in $requiredFiles) {
    $destinationPath = Join-Path $destinationRoot $entry.destination
    $destinationDirectory = Split-Path -Parent $destinationPath
    if (-not (Test-Path -LiteralPath $destinationDirectory)) {
        New-Item -ItemType Directory -Path $destinationDirectory | Out-Null
    }
    Copy-Item -LiteralPath (Join-Path $repoRoot $entry.source) -Destination $destinationPath
}
foreach ($script in $dataPlaneScripts) {
    Copy-Item -LiteralPath $script.FullName -Destination (Join-Path (Join-Path $destinationRoot 'scripts\data-plane') $script.Name)
}

$forbidden = @(
    '(^|[\\/])\.env$',
    '\.dump$',
    '\.tar\.gz$',
    '\.sqlite3?$',
    '\.db(-wal|-shm)?$',
    '(^|[\\/])data([\\/]|$)'
)
$packagedFiles = @(Get-ChildItem -LiteralPath $destinationRoot -Recurse -File)
foreach ($file in $packagedFiles) {
    $relative = $file.FullName.Substring($destinationRoot.Length).TrimStart('\', '/').Replace('\', '/')
    foreach ($pattern in $forbidden) {
        if ($relative -match $pattern) {
            throw "Forbidden release artifact entered the package: $relative"
        }
    }
}

$commitID = (git -C $repoRoot rev-parse --short HEAD 2>$null).Trim()
if ([string]::IsNullOrWhiteSpace($commitID)) {
    $commitID = 'unknown'
}
$records = @(
    foreach ($file in @(Get-ChildItem -LiteralPath $destinationRoot -Recurse -File | Sort-Object FullName)) {
        $relative = $file.FullName.Substring($destinationRoot.Length).TrimStart('\', '/').Replace('\', '/')
        [pscustomobject]@{
            path   = $relative
            sha256 = (Get-FileHash -LiteralPath $file.FullName -Algorithm SHA256).Hash
            bytes  = $file.Length
        }
    }
)
$manifest = [ordered]@{
    generated_at = (Get-Date).ToUniversalTime().ToString('o')
    git_commit   = $commitID
    package_root = $destinationRoot
    files        = $records
}
Write-Utf8NoBom -Path (Join-Path $destinationRoot 'materials-manifest.json') -Content ($manifest | ConvertTo-Json -Depth 8)
$checksums = @(
    foreach ($record in $records) {
        "$($record.sha256)  $($record.path)"
    }
)
Write-Utf8NoBom -Path (Join-Path $destinationRoot 'SHA256SUMS.txt') -Content (($checksums -join [Environment]::NewLine) + [Environment]::NewLine)
Write-Host "Shared Knavis data-plane release materials added: $destinationRoot"
