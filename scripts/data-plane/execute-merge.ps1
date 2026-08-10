[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string]$MergePlanDir,
    [Parameter(Mandatory = $true)]
    [ValidatePattern('^[A-Fa-f0-9]{64}$')]
    [string]$ReviewedPlanHash,
    [Parameter(Mandatory = $true)]
    [string]$OutputRoot,
    [string]$PostgresImage = 'paradedb/paradedb:v0.22.2-pg17',
    [switch]$ApproveLocalWins
)

$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'common.ps1')

if (-not $ApproveLocalWins) {
    throw 'Pass -ApproveLocalWins only after reviewing conflicts.json'
}

$mergePlanRoot = Assert-ExistingDirectory -Path $MergePlanDir -Label 'Merge plan'
$mergeSqlPath = Join-Path $mergePlanRoot 'merge.sql'
$planHashPath = Join-Path $mergePlanRoot 'merge-plan.sha256'
$mergePlanPath = Join-Path $mergePlanRoot 'merge-plan.json'
if (-not (Test-Path -LiteralPath $mergeSqlPath -PathType Leaf)) {
    throw "Merge SQL is missing: $mergeSqlPath"
}
if (-not (Test-Path -LiteralPath $planHashPath -PathType Leaf)) {
    throw "Reviewed merge plan hash is missing: $planHashPath"
}
if (-not (Test-Path -LiteralPath $mergePlanPath -PathType Leaf)) {
    throw "Merge plan metadata is missing: $mergePlanPath"
}
$actual = (Get-FileHash -LiteralPath $mergeSqlPath -Algorithm SHA256).Hash
$recorded = (Get-Content -LiteralPath $planHashPath -Raw).Trim()
if ($recorded -notmatch '^([A-Fa-f0-9]{64})\s+merge\.sql$') {
    throw "Invalid merge-plan.sha256 format: $planHashPath"
}
if ($matches[1] -ine $actual) {
    throw 'merge-plan.sha256 does not match merge.sql'
}
if ($actual -ine $ReviewedPlanHash) {
    throw 'Reviewed merge.sql hash does not match'
}

$reviewedSql = Get-Content -LiteralPath $mergeSqlPath -Raw
if ($reviewedSql -notmatch '(?im)^\s*BEGIN\s*;') {
    throw 'Reviewed merge.sql must start a transaction'
}
if ($reviewedSql -notmatch "SET lock_timeout") {
    throw 'Reviewed merge.sql is missing lock_timeout'
}
if ($reviewedSql -match '(?im)^\s*COMMIT\s*;') {
    throw 'Dry-run merge.sql must not contain COMMIT before execution'
}
if ($reviewedSql -match '(?im)DROP\s+DATABASE|COPY\s+.*PROGRAM|\\!') {
    throw 'Reviewed merge.sql contains a prohibited database or shell operation'
}

function Assert-ReviewedSnapshot {
    param([Parameter(Mandatory = $true)][string]$Path, [Parameter(Mandatory = $true)][string]$Label)
    $root = Assert-ExistingDirectory -Path $Path -Label $Label
    $dumpFiles = @(Get-ChildItem -LiteralPath $root -Filter '*.dump' -File)
    if ($dumpFiles.Count -ne 1) {
        throw "$Label must contain exactly one PostgreSQL custom dump: $root"
    }
    $snapshotPath = Join-Path $root 'snapshot.json'
    if (-not (Test-Path -LiteralPath $snapshotPath -PathType Leaf)) {
        throw "$Label is missing snapshot.json: $root"
    }
    $snapshot = Get-Content -LiteralPath $snapshotPath -Raw | ConvertFrom-Json
    $expectedDump = @($snapshot.archives | Where-Object { [string]$_.path -match '\.dump$' })
    if ($expectedDump.Count -ne 1) {
        throw "$Label snapshot.json does not contain one dump hash: $root"
    }
    $actualHash = (Get-FileHash -LiteralPath $dumpFiles[0].FullName -Algorithm SHA256).Hash
    if ($actualHash -ine [string]$expectedDump[0].sha256) {
        throw "$Label dump hash does not match snapshot.json: $root"
    }
    return [pscustomobject]@{
        Root     = $root
        DumpPath = $dumpFiles[0].FullName
        DumpName = $dumpFiles[0].Name
    }
}

$mergePlan = Get-Content -LiteralPath $mergePlanPath -Raw | ConvertFrom-Json
$serverSnapshot = Assert-ReviewedSnapshot -Path ([string]$mergePlan.server_snapshot) -Label 'Server snapshot'
$localSnapshot = Assert-ReviewedSnapshot -Path ([string]$mergePlan.local_snapshot) -Label 'Local snapshot'
$serverRoot = $serverSnapshot.Root
$localRoot = $localSnapshot.Root

function Invoke-ExecuteDbCommand {
    param([Parameter(Mandatory = $true)][string]$Container, [Parameter(Mandatory = $true)][string]$Database, [Parameter(Mandatory = $true)][string]$Sql)
    Invoke-Checked -FilePath 'docker' -Arguments @(
        'exec', $Container, 'psql', '--username=postgres', "--dbname=$Database", '--no-password',
        '--set=ON_ERROR_STOP=1', '--command=' + $Sql
    )
}

function Wait-ExecutePostgres {
    param([Parameter(Mandatory = $true)][string]$Container)
    for ($attempt = 0; $attempt -lt 90; $attempt++) {
        & docker exec $Container pg_isready --username=postgres --dbname=postgres 1>$null 2>$null
        if ($LASTEXITCODE -eq 0) { return }
        Start-Sleep -Seconds 1
    }
    throw "Temporary PostgreSQL did not become ready: $Container"
}

$artifactRoot = New-ArtifactDirectory -OutputRoot $OutputRoot -Prefix 'merge-execution'
$executionSqlPath = Join-Path $artifactRoot 'merge-executable.sql'
($reviewedSql.TrimEnd() + "`r`nCOMMIT;`r`n") | Set-Content -LiteralPath $executionSqlPath -Encoding UTF8
$containerName = "knavis_merge_execute_$((Get-Date).ToUniversalTime().ToString('yyyyMMddHHmmss'))_$([Guid]::NewGuid().ToString('N').Substring(0, 8))"
$containerStarted = $false

try {
    $containerId = [string]((Invoke-CheckedOutput -FilePath 'docker' -Arguments @(
            'run', '-d', '--name', $containerName,
            '-e', 'POSTGRES_HOST_AUTH_METHOD=trust',
        '--mount', "type=bind,source=$serverRoot,target=/snapshots/server,readonly",
        '--mount', "type=bind,source=$localRoot,target=/snapshots/local,readonly",
            '--mount', "type=bind,source=$artifactRoot,target=/output",
            $PostgresImage
        ))[-1])
    $containerStarted = $true
    Wait-ExecutePostgres -Container $containerName
    foreach ($database in @('server_snapshot', 'local_snapshot', 'merged_preview')) {
        Invoke-Checked -FilePath 'docker' -Arguments @('exec', $containerName, 'createdb', '--username=postgres', $database)
    }
    Invoke-Checked -FilePath 'docker' -Arguments @(
        'exec', $containerName, 'pg_restore', '--username=postgres', '--dbname=server_snapshot',
        '--no-owner', '--no-privileges', '--exit-on-error', "/snapshots/server/$($serverSnapshot.DumpName)"
    )
    Invoke-Checked -FilePath 'docker' -Arguments @(
        'exec', $containerName, 'pg_restore', '--username=postgres', '--dbname=local_snapshot',
        '--no-owner', '--no-privileges', '--exit-on-error', "/snapshots/local/$($localSnapshot.DumpName)"
    )

    Invoke-ExecuteDbCommand -Container $containerName -Database 'merged_preview' -Sql @'
CREATE EXTENSION IF NOT EXISTS postgres_fdw;
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE SCHEMA IF NOT EXISTS fdw_local;
CREATE SERVER knavis_local_snapshot FOREIGN DATA WRAPPER postgres_fdw
  OPTIONS (host '127.0.0.1', port '5432', dbname 'local_snapshot');
CREATE USER MAPPING FOR postgres SERVER knavis_local_snapshot;
IMPORT FOREIGN SCHEMA public FROM SERVER knavis_local_snapshot INTO fdw_local;
'@
    Invoke-Checked -FilePath 'docker' -Arguments @(
        'exec', $containerName, 'psql', '--username=postgres', '--dbname=merged_preview',
        '--no-password', '--set=ON_ERROR_STOP=1', '--file=/output/merge-executable.sql'
    )

    $invalidForeignKeys = @(Invoke-PostgresCsv -PostgresContainer $containerName -DatabaseName 'merged_preview' -DatabaseUser 'postgres' -Sql @'
SELECT conname
FROM pg_constraint
WHERE contype = 'f' AND convalidated = false
'@ -Headers @('conname'))
    if ($invalidForeignKeys.Count -gt 0) {
        throw "Merged preview contains unvalidated foreign keys: $($invalidForeignKeys.conname -join ', ')"
    }

    $dumpInContainer = '/tmp/merged-local-wins.dump'
    Invoke-Checked -FilePath 'docker' -Arguments @(
        'exec', $containerName, 'pg_dump', '--username=postgres', '--dbname=merged_preview',
        '--format=custom', '--schema=public', '--no-owner', '--no-privileges', "--file=$dumpInContainer"
    )
    $mergedDumpPath = Join-Path $artifactRoot 'merged-local-wins.dump'
    Invoke-Checked -FilePath 'docker' -Arguments @('cp', "$containerName`:$dumpInContainer", $mergedDumpPath)

    $execution = [ordered]@{
        generated_at       = (Get-Date).ToUniversalTime().ToString('o')
        reviewed_plan_dir  = $mergePlanRoot
        reviewed_plan_hash = $actual
        temporary_container = $containerName
        archive             = $mergedDumpPath
        local_wins          = $true
        production_restore  = $false
        status              = 'temporary-candidate'
    }
    Copy-Item -LiteralPath $planHashPath -Destination (Join-Path $artifactRoot 'merge-plan.sha256')
    Write-JsonArtifact -Value $execution -Path (Join-Path $artifactRoot 'merge-execution.json') | Out-Null
    Write-ArtifactSha256Sums -ArtifactRoot $artifactRoot | Out-Null
    Write-Host "Temporary merged archive created: $artifactRoot"
} finally {
    if ($containerStarted) {
        & docker rm -f $containerName 1>$null 2>$null
    }
}
