[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string]$MergeExecutionDir,
    [Parameter(Mandatory = $true)]
    [string]$OutputRoot,
    [string]$CandidateContainerName,
    [string]$CandidateDatabase = 'knavis_candidate',
    [string]$PostgresImage = 'paradedb/paradedb:v0.22.2-pg17',
    [string[]]$ExpectedExtension = @('pg_trgm', 'pg_search', 'uuid-ossp', 'vector')
)

$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'common.ps1')

function Assert-ExistingFile {
    param([Parameter(Mandatory = $true)][string]$Path, [Parameter(Mandatory = $true)][string]$Label)
    if (-not (Test-Path -LiteralPath $Path -PathType Leaf)) {
        throw "$Label is missing: $Path"
    }
    return (Resolve-Path -LiteralPath $Path).Path
}

function Assert-ArchiveHash {
    param([Parameter(Mandatory = $true)][string]$Root, [Parameter(Mandatory = $true)][string]$Archive)
    $manifestPath = Assert-ExistingFile -Path (Join-Path $Root 'SHA256SUMS.txt') -Label 'Merge execution SHA256SUMS.txt'
    $archiveName = Split-Path -Leaf $Archive
    $expected = $null
    foreach ($line in @(Get-Content -LiteralPath $manifestPath)) {
        if ($line -match '^([A-Fa-f0-9]{64})\s+\*?(.+)$' -and $matches[2].Replace('\', '/') -eq $archiveName) {
            $expected = $matches[1]
            break
        }
    }
    if ([string]::IsNullOrWhiteSpace($expected)) {
        throw "SHA256SUMS.txt has no record for merged archive: $archiveName"
    }
    $actual = (Get-FileHash -LiteralPath $Archive -Algorithm SHA256).Hash
    if ($actual -ine $expected) {
        throw 'Merged archive hash does not match SHA256SUMS.txt'
    }
    return $actual
}

function Wait-CandidatePostgres {
    param([Parameter(Mandatory = $true)][string]$Container)
    for ($attempt = 0; $attempt -lt 90; $attempt++) {
        & docker exec $Container pg_isready --username=postgres --dbname=postgres 1>$null 2>$null
        if ($LASTEXITCODE -eq 0) {
            return
        }
        Start-Sleep -Seconds 1
    }
    throw "Candidate PostgreSQL did not become ready: $Container"
}

function Invoke-CandidateSql {
    param([Parameter(Mandatory = $true)][string]$Container, [Parameter(Mandatory = $true)][string]$Database, [Parameter(Mandatory = $true)][string]$Sql)
    Invoke-Checked -FilePath 'docker' -Arguments @(
        'exec', $Container, 'psql', '--username=postgres', "--dbname=$Database",
        '--no-password', '--set=ON_ERROR_STOP=1', '--command=' + $Sql
    )
}

$executionRoot = Assert-ExistingDirectory -Path $MergeExecutionDir -Label 'Merge execution'
$executionPath = Assert-ExistingFile -Path (Join-Path $executionRoot 'merge-execution.json') -Label 'merge-execution.json'
$execution = Get-Content -LiteralPath $executionPath -Raw | ConvertFrom-Json
if ([string]$execution.status -ne 'temporary-candidate' -or [bool]$execution.production_restore) {
    throw 'Merge execution artifact is not an approved non-production candidate'
}
$archivePath = Assert-ExistingFile -Path (Join-Path $executionRoot 'merged-local-wins.dump') -Label 'Merged PostgreSQL archive'
$archiveHash = Assert-ArchiveHash -Root $executionRoot -Archive $archivePath
if ([string]::IsNullOrWhiteSpace($CandidateContainerName)) {
    $CandidateContainerName = 'knavis_merge_candidate_' + (Get-Date).ToUniversalTime().ToString('yyyyMMddHHmmss') + '_' + [Guid]::NewGuid().ToString('N').Substring(0, 8)
}
if ($CandidateContainerName -notmatch '^knavis_merge_candidate_[a-zA-Z0-9_-]+$') {
    throw "Candidate container name must start with knavis_merge_candidate_: $CandidateContainerName"
}
& docker inspect $CandidateContainerName 1>$null 2>$null
if ($LASTEXITCODE -eq 0) {
    throw "Candidate container already exists: $CandidateContainerName"
}

$artifactRoot = New-ArtifactDirectory -OutputRoot $OutputRoot -Prefix 'merge-candidate'
$archiveRoot = Split-Path -Parent $archivePath
$archiveName = Split-Path -Leaf $archivePath
$containerStarted = $false
try {
    Invoke-Checked -FilePath 'docker' -Arguments @(
        'run', '-d', '--name', $CandidateContainerName,
        '-e', 'POSTGRES_HOST_AUTH_METHOD=trust',
        '--tmpfs', '/var/lib/postgresql/data:rw',
        '--mount', "type=bind,source=$archiveRoot,target=/input,readonly",
        $PostgresImage
    )
    $containerStarted = $true
    Wait-CandidatePostgres -Container $CandidateContainerName
    Invoke-Checked -FilePath 'docker' -Arguments @(
        'exec', $CandidateContainerName, 'createdb', '--username=postgres', $CandidateDatabase
    )
    foreach ($extension in $ExpectedExtension) {
        if ($extension -notmatch '^[A-Za-z0-9_-]+$') {
            throw "Unsupported PostgreSQL extension name: $extension"
        }
        $quotedExtension = '"' + $extension.Replace('"', '""') + '"'
        Invoke-CandidateSql -Container $CandidateContainerName -Database $CandidateDatabase -Sql ('CREATE EXTENSION IF NOT EXISTS ' + $quotedExtension + ';')
    }
    Invoke-Checked -FilePath 'docker' -Arguments @(
        'exec', $CandidateContainerName, 'pg_restore', '--username=postgres', "--dbname=$CandidateDatabase",
        '--no-owner', '--no-privileges', '--exit-on-error', "/input/$archiveName"
    )
    $dirty = @(Invoke-PostgresCsv -PostgresContainer $CandidateContainerName -DatabaseName $CandidateDatabase -DatabaseUser 'postgres' -Sql @'
SELECT version::text, dirty::text
FROM schema_migrations
WHERE dirty = true
ORDER BY version
'@ -Headers @('version', 'dirty'))
    if ($dirty.Count -gt 0) {
        throw "Candidate restore contains dirty migrations: $($dirty.version -join ', ')"
    }
    $candidate = [ordered]@{
        generated_at       = (Get-Date).ToUniversalTime().ToString('o')
        container_name     = $CandidateContainerName
        database_name      = $CandidateDatabase
        archive            = $archivePath
        archive_sha256     = $archiveHash
        published_ports    = @()
        production_restore = $false
        status             = 'running'
    }
    Write-JsonArtifact -Value $candidate -Path (Join-Path $artifactRoot 'candidate.json') | Out-Null
    Write-ArtifactSha256Sums -ArtifactRoot $artifactRoot | Out-Null
    Write-Host "Isolated merge candidate is running: $CandidateContainerName/$CandidateDatabase"
} catch {
    if ($containerStarted) {
        & docker rm -f $CandidateContainerName 1>$null 2>$null
    }
    throw
}
