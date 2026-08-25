[CmdletBinding()]
param(
    [string]$PackageDir = "",
    [switch]$CheckImages,
    [switch]$VerifyChecksums
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
        [string]$Name
    )

    $line = Get-Content -LiteralPath $Path | Where-Object {
        $_ -match "^$([regex]::Escape($Name))="
    } | Select-Object -First 1
    if ($null -eq $line) { return "" }
    return (($line -split "=", 2)[1]).Trim()
}

function Assert-PathInside {
    param(
        [string]$Root,
        [string]$Candidate
    )

    $rootFull = [System.IO.Path]::GetFullPath($Root).TrimEnd([IO.Path]::DirectorySeparatorChar, [IO.Path]::AltDirectorySeparatorChar) + [IO.Path]::DirectorySeparatorChar
    $candidateFull = [System.IO.Path]::GetFullPath($Candidate)
    if (-not $candidateFull.StartsWith($rootFull, [System.StringComparison]::OrdinalIgnoreCase)) {
        throw "Checksum entry escapes the package root: $Candidate"
    }
}

Assert-Command docker
if ([string]::IsNullOrWhiteSpace($PackageDir)) {
    $PackageDir = $PSScriptRoot
}
$packageRoot = (Resolve-Path -LiteralPath $PackageDir).Path
$composeFile = Join-Path $packageRoot "docker-compose.yml"
$envFile = Join-Path $packageRoot ".env"
$offlineDir = Join-Path $packageRoot "offline-images"
$manifestFile = Join-Path $offlineDir "manifest.txt"

foreach ($requiredPath in @(
    $composeFile,
    $envFile,
    $manifestFile,
    (Join-Path $packageRoot "config\config.yaml"),
    (Join-Path $packageRoot "skills\preloaded"),
    (Join-Path $packageRoot "data\workspaces")
)) {
    if (-not (Test-Path -LiteralPath $requiredPath)) {
        throw "Required package path not found: $requiredPath"
    }
}

Invoke-Checked "docker" @("info", "--format", "{{.OSType}} {{.Architecture}} {{.ServerVersion}}")

Push-Location $packageRoot
try {
    Write-Host "==> Validating Compose configuration" -ForegroundColor Cyan
    Invoke-Checked "docker" @("compose", "--env-file", $envFile, "-f", $composeFile, "config", "--quiet")

    $imageTag = Get-EnvValue -Path $envFile -Name "XELORA_VERSION"
    if ([string]::IsNullOrWhiteSpace($imageTag)) {
        throw "XELORA_VERSION is missing from $envFile"
    }

    $manifestLines = @(Get-Content -LiteralPath $manifestFile | Where-Object {
        -not [string]::IsNullOrWhiteSpace($_) -and $_ -ne "IMAGE_EXPORT_SKIPPED"
    })
    if ($manifestLines.Count -eq 0) {
        throw "No offline image records found in $manifestFile"
    }

    foreach ($line in $manifestLines) {
        $image = ($line -split "`t", 2)[0]
        $safeName = $image -replace "[:/]", "_"
        $archive = Join-Path $offlineDir "$safeName.tar"
        if (-not (Test-Path -LiteralPath $archive)) {
            throw "Image archive is missing for $image`: $archive"
        }
        if ($image -like "*:$imageTag" -or $image -notlike "*:*") {
            Write-Host "    archive: $image" -ForegroundColor DarkGray
        }
        if ($CheckImages) {
            $inspect = @(docker image inspect $image --format "{{.Id}}")
            if ($LASTEXITCODE -ne 0 -or $inspect.Count -eq 0) {
                throw "Docker image is not loaded locally: $image"
            }
        }
    }

    if ($VerifyChecksums) {
        $releaseRoot = Split-Path -Parent $packageRoot
        $checksumFile = Join-Path $releaseRoot "SHA256SUMS.txt"
        if (-not (Test-Path -LiteralPath $checksumFile)) {
            throw "Checksum file not found: $checksumFile"
        }
        $verified = 0
        foreach ($line in @(Get-Content -LiteralPath $checksumFile)) {
            if ([string]::IsNullOrWhiteSpace($line)) { continue }
            if ($line -notmatch "^(?<expected>[A-Fa-f0-9]{64})\s{2}(?<relative>.+)$") {
                throw "Malformed checksum entry: $line"
            }
            $relative = $Matches.relative.Replace("/", "\")
            $candidate = Join-Path $releaseRoot $relative
            Assert-PathInside -Root $releaseRoot -Candidate $candidate
            if (-not (Test-Path -LiteralPath $candidate -PathType Leaf)) {
                throw "Checksum target is missing: $candidate"
            }
            $actual = (Get-FileHash -LiteralPath $candidate -Algorithm SHA256).Hash
            if ($actual -ine $Matches.expected) {
                throw "Checksum mismatch: $relative"
            }
            $verified++
        }
        Write-Host "    SHA256 files verified: $verified" -ForegroundColor Green
    }

    Write-Host "Knavis internal package verification passed." -ForegroundColor Green
} finally {
    Pop-Location
}
