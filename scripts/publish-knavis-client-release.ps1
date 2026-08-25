[CmdletBinding()]
param(
    [Parameter(Mandatory)]
    [string]$PackagePath,
    [Parameter(Mandatory)]
    [string]$Version,
    [ValidateSet("stable", "beta")]
    [string]$Channel = "stable",
    [string]$Build = "",
    [string]$Notes = "",
    [string]$ReleaseRoot = ""
)

$ErrorActionPreference = "Stop"
$repoRoot = Split-Path -Parent $PSScriptRoot

function Write-Utf8NoBom {
    param(
        [string]$Path,
        [string]$Content
    )

    [System.IO.File]::WriteAllText(
        $Path,
        $Content,
        (New-Object System.Text.UTF8Encoding($false))
    )
}

function Write-JsonAtomically {
    param(
        [string]$Path,
        [object]$Value
    )

    $directory = Split-Path -Parent $Path
    New-Item -ItemType Directory -Force -Path $directory | Out-Null
    $temporaryPath = Join-Path $directory (".{0}.{1}.tmp" -f ([System.IO.Path]::GetFileName($Path)), $PID)
    try {
        Write-Utf8NoBom -Path $temporaryPath -Content (($Value | ConvertTo-Json -Depth 12) + [Environment]::NewLine)
        Move-Item -LiteralPath $temporaryPath -Destination $Path -Force
    } finally {
        Remove-Item -LiteralPath $temporaryPath -Force -ErrorAction SilentlyContinue
    }
}

function Get-SemanticVersion {
    param([string]$Value)

    $normalized = $Value.Trim()
    if ($normalized.StartsWith("v", [System.StringComparison]::OrdinalIgnoreCase)) {
        $normalized = $normalized.Substring(1)
    }
    $pattern = '^(?<major>0|[1-9]\d*)\.(?<minor>0|[1-9]\d*)\.(?<patch>0|[1-9]\d*)(?:-(?<prerelease>[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?(?:\+(?<build>[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?$'
    if ($normalized -notmatch $pattern) {
        throw "Release version must be semantic version 1.2.3: $Value"
    }
    return [pscustomobject]@{
        Normalized = $normalized
        Major = [Int64]$Matches.major
        Minor = [Int64]$Matches.minor
        Patch = [Int64]$Matches.patch
        PreRelease = $Matches.prerelease
    }
}

function Compare-SemanticVersion {
    param(
        [string]$Left,
        [string]$Right
    )

    $leftVersion = Get-SemanticVersion $Left
    $rightVersion = Get-SemanticVersion $Right
    foreach ($part in @("Major", "Minor", "Patch")) {
        if ($leftVersion.$part -gt $rightVersion.$part) { return 1 }
        if ($leftVersion.$part -lt $rightVersion.$part) { return -1 }
    }

    $leftPre = [string]$leftVersion.PreRelease
    $rightPre = [string]$rightVersion.PreRelease
    if ($leftPre -eq "" -and $rightPre -eq "") { return 0 }
    if ($leftPre -eq "") { return 1 }
    if ($rightPre -eq "") { return -1 }

    $leftParts = $leftPre -split '\.'
    $rightParts = $rightPre -split '\.'
    $length = [Math]::Min($leftParts.Count, $rightParts.Count)
    for ($index = 0; $index -lt $length; $index++) {
        $leftPart = $leftParts[$index]
        $rightPart = $rightParts[$index]
        $leftNumeric = $leftPart -match '^\d+$'
        $rightNumeric = $rightPart -match '^\d+$'
        if ($leftNumeric -and $rightNumeric) {
            if ([Int64]$leftPart -gt [Int64]$rightPart) { return 1 }
            if ([Int64]$leftPart -lt [Int64]$rightPart) { return -1 }
            continue
        }
        if ($leftNumeric -and -not $rightNumeric) { return -1 }
        if (-not $leftNumeric -and $rightNumeric) { return 1 }
        $comparison = [string]::CompareOrdinal($leftPart, $rightPart)
        if ($comparison -gt 0) { return 1 }
        if ($comparison -lt 0) { return -1 }
    }
    if ($leftParts.Count -gt $rightParts.Count) { return 1 }
    if ($leftParts.Count -lt $rightParts.Count) { return -1 }
    return 0
}

function Get-ReleaseEntries {
    param([object]$Manifest)

    $entries = @()
    if ($null -ne $Manifest.releases) {
        $entries += @($Manifest.releases)
    }
    if ($null -ne $Manifest.latest) {
        $latestVersion = (Get-SemanticVersion ([string]$Manifest.latest.version)).Normalized
        if (-not (@($entries | Where-Object {
            (Get-SemanticVersion ([string]$_.version)).Normalized -eq $latestVersion
        }).Count)) {
            $entries += $Manifest.latest
        }
    }
    return @($entries)
}

function Assert-ClientPackage {
    param([string]$Path)

    if ([System.IO.Path]::GetExtension($Path) -ine ".zip") {
        throw "Client package must be a .zip file: $Path"
    }
    Add-Type -AssemblyName System.IO.Compression.FileSystem
    $archive = [System.IO.Compression.ZipFile]::OpenRead($Path)
    try {
        $desktopExecutable = @($archive.Entries | Where-Object {
            $_.FullName -ieq "Knavis Personal.exe"
        }) | Select-Object -First 1
        if ($null -eq $desktopExecutable) {
            throw "Client package must contain Knavis Personal.exe: $Path"
        }
    } finally {
        $archive.Dispose()
    }
}

$releaseVersion = Get-SemanticVersion $Version
if ($Channel -eq "stable" -and $releaseVersion.PreRelease) {
    throw "Stable channel cannot publish prerelease version: $($releaseVersion.Normalized)"
}
if ([string]::IsNullOrWhiteSpace($ReleaseRoot)) {
    $ReleaseRoot = Join-Path $repoRoot "dist\client-releases"
} elseif (-not [System.IO.Path]::IsPathRooted($ReleaseRoot)) {
    $ReleaseRoot = Join-Path $repoRoot $ReleaseRoot
}
$ReleaseRoot = [System.IO.Path]::GetFullPath($ReleaseRoot)

$sourcePath = (Resolve-Path -LiteralPath $PackagePath).Path
Assert-ClientPackage -Path $sourcePath
$fileName = [System.IO.Path]::GetFileName($sourcePath)
if ($fileName -notmatch '^[A-Za-z0-9][A-Za-z0-9._-]*\.zip$') {
    throw "Client package filename is unsafe: $fileName"
}
if ($fileName -notmatch ("-{0}\.zip$" -f [regex]::Escape($releaseVersion.Normalized))) {
    throw "Client package filename must end with -$($releaseVersion.Normalized).zip: $fileName"
}

$channelsDirectory = Join-Path $ReleaseRoot "channels"
$channelManifestPath = Join-Path $channelsDirectory "$Channel.json"
$existingManifest = $null
$existingReleases = @()
if (Test-Path -LiteralPath $channelManifestPath) {
    $existingManifest = Get-Content -LiteralPath $channelManifestPath -Raw | ConvertFrom-Json
    if ($existingManifest.schemaVersion -ne 1 -or [string]$existingManifest.channel -ne $Channel) {
        throw "Existing channel manifest is invalid: $channelManifestPath"
    }
    $existingReleases = Get-ReleaseEntries $existingManifest
    foreach ($existing in $existingReleases) {
        $existingVersion = (Get-SemanticVersion ([string]$existing.version)).Normalized
        if ($existingVersion -eq $releaseVersion.Normalized) {
            throw "Release version already exists in $Channel channel: $($releaseVersion.Normalized)"
        }
    }
    if ($null -ne $existingManifest.latest) {
        $latestVersion = (Get-SemanticVersion ([string]$existingManifest.latest.version)).Normalized
        if ((Compare-SemanticVersion $releaseVersion.Normalized $latestVersion) -le 0) {
            throw "New release version must be newer than $Channel latest version $latestVersion"
        }
    }
}

New-Item -ItemType Directory -Force -Path $ReleaseRoot, $channelsDirectory | Out-Null
$destinationPath = Join-Path $ReleaseRoot $fileName
$sourceHash = (Get-FileHash -LiteralPath $sourcePath -Algorithm SHA256).Hash.ToLowerInvariant()
$sourceSize = (Get-Item -LiteralPath $sourcePath).Length
if (Test-Path -LiteralPath $destinationPath) {
    $destinationHash = (Get-FileHash -LiteralPath $destinationPath -Algorithm SHA256).Hash.ToLowerInvariant()
    if ($destinationHash -ne $sourceHash) {
        throw "Refusing to replace an existing release artifact with different bytes: $destinationPath"
    }
} else {
    Copy-Item -LiteralPath $sourcePath -Destination $destinationPath -Force
}

$publishedAt = [DateTime]::UtcNow.ToString("o")
$releaseEntry = [ordered]@{
    version = $releaseVersion.Normalized
    build = if ([string]::IsNullOrWhiteSpace($Build)) { $publishedAt } else { $Build.Trim() }
    file = $fileName
    sha256 = $sourceHash
    sizeBytes = [Int64]$sourceSize
    publishedAt = $publishedAt
    notes = $Notes.Trim()
}
$channelManifest = [ordered]@{
    schemaVersion = 1
    product = "Knavis Personal"
    channel = $Channel
    generatedAt = $publishedAt
    latest = $releaseEntry
    releases = @($releaseEntry) + @($existingReleases)
}
Write-JsonAtomically -Path $channelManifestPath -Value $channelManifest

$indexPath = Join-Path $ReleaseRoot "release-index.json"
$existingIndex = $null
if (Test-Path -LiteralPath $indexPath) {
    $existingIndex = Get-Content -LiteralPath $indexPath -Raw | ConvertFrom-Json
}
$otherChannels = @()
if ($null -ne $existingIndex -and $null -ne $existingIndex.channels) {
    $otherChannels = @($existingIndex.channels | Where-Object { [string]$_.id -ne $Channel })
}
$channelRecord = [ordered]@{
    id = $Channel
    label = if ($Channel -eq "stable") { "Stable" } else { "Beta" }
    manifest = "channels/$Channel.json"
    latestVersion = $releaseVersion.Normalized
    publishedAt = $publishedAt
}
$index = [ordered]@{
    schemaVersion = 1
    product = "Knavis Personal"
    generatedAt = $publishedAt
    channels = @($channelRecord) + @($otherChannels)
}
Write-JsonAtomically -Path $indexPath -Value $index

if ($Channel -eq "stable") {
    $latestCompat = [ordered]@{
        product = "Knavis Personal"
        version = $releaseEntry.version
        build = $releaseEntry.build
        file = $releaseEntry.file
        sha256 = $releaseEntry.sha256
        sizeBytes = $releaseEntry.sizeBytes
        updatedAt = $publishedAt
    }
    Write-JsonAtomically -Path (Join-Path $ReleaseRoot "latest.json") -Value $latestCompat
}

Write-Host "Published Knavis $Channel release v$($releaseVersion.Normalized)" -ForegroundColor Green
Write-Host "  Package: $destinationPath"
Write-Host "  SHA-256: $sourceHash"
Write-Host "  Channel manifest: $channelManifestPath"
