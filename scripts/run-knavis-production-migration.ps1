[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string]$EnvFile,
    [Parameter(Mandatory = $true)]
    [string]$BackupManifest,
    [Parameter(Mandatory = $true)]
    [string]$BackupDump,
    [Parameter(Mandatory = $true)]
    [string]$PostgresContainer,
    [Parameter(Mandatory = $true)]
    [string]$DatabaseName,
    [Parameter(Mandatory = $true)]
    [string]$DatabaseUser,
    [Parameter(Mandatory = $true)]
    [string]$OutputRoot,
    [string]$BaseComposeFile = (Join-Path (Split-Path -Parent $PSScriptRoot) 'docker-compose.yml'),
    [string]$ProductionComposeFile = (Join-Path (Split-Path -Parent $PSScriptRoot) 'docker-compose.knavis-production.yml'),
    [string]$ProjectName = 'knavis-production',
    [string[]]$ApplicationContainers = @(
        'Xelora-app', 'Xelora-frontend', 'Xelora-docreader', 'Xelora-sandbox',
        'Knavis-app', 'Knavis-frontend', 'Knavis-docreader', 'Knavis-sandbox',
        'Knavis-passive-app', 'Knavis-passive-frontend'
    ),
    [switch]$MaintenanceApproved
)

$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'data-plane\common.ps1')

function Assert-ExistingFile {
    param([Parameter(Mandatory = $true)][string]$Path, [Parameter(Mandatory = $true)][string]$Label)
    if (-not (Test-Path -LiteralPath $Path -PathType Leaf)) {
        throw "$Label is missing: $Path"
    }
    return (Resolve-Path -LiteralPath $Path).Path
}

function Assert-BackupHash {
    param([Parameter(Mandatory = $true)][string]$ManifestPath, [Parameter(Mandatory = $true)][string]$DumpPath)
    $dumpName = Split-Path -Leaf $DumpPath
    $expectedHash = $null
    foreach ($line in @(Get-Content -LiteralPath $ManifestPath)) {
        if ($line -notmatch '^([A-Fa-f0-9]{64})\s+\*?(.+)$') {
            continue
        }
        $recordedPath = $matches[2].Trim().TrimStart('*').Replace('\', '/')
        if ($recordedPath -eq $dumpName -or $DumpPath.Replace('\', '/').EndsWith('/' + $recordedPath)) {
            $expectedHash = $matches[1]
            break
        }
    }
    if ([string]::IsNullOrWhiteSpace($expectedHash)) {
        throw "Backup manifest has no SHA-256 record for: $dumpName"
    }
    $actualHash = (Get-FileHash -LiteralPath $DumpPath -Algorithm SHA256).Hash
    if ($actualHash -ine $expectedHash) {
        throw 'Backup dump hash does not match BackupManifest'
    }
    return $actualHash
}

if (-not $MaintenanceApproved) {
    throw 'Pass -MaintenanceApproved only after stopping all application containers'
}

$envPath = Assert-ExistingFile -Path $EnvFile -Label 'Production environment file'
$backupManifestPath = Assert-ExistingFile -Path $BackupManifest -Label 'Backup manifest'
$backupDumpPath = Assert-ExistingFile -Path $BackupDump -Label 'Backup dump'
$baseComposePath = Assert-ExistingFile -Path $BaseComposeFile -Label 'Base Compose file'
$productionComposePath = Assert-ExistingFile -Path $ProductionComposeFile -Label 'Production Compose file'
$backupHash = Assert-BackupHash -ManifestPath $backupManifestPath -DumpPath $backupDumpPath

$envText = Get-Content -LiteralPath $envPath -Raw
if ($envText -match '(?im)^\s*AUTO_RECOVER_DIRTY\s*=\s*true\s*$') {
    throw 'AUTO_RECOVER_DIRTY must not be true during a controlled production migration'
}
Assert-ContainersStopped -Names $ApplicationContainers
Assert-ContainerRunning -Name $PostgresContainer

$composeArgs = @(
    'compose', '-p', $ProjectName,
    '--env-file', $envPath,
    '-f', $baseComposePath,
    '-f', $productionComposePath
)
Invoke-Checked -FilePath 'docker' -Arguments ($composeArgs + @('config', '--quiet'))

$beforeDirty = @(Invoke-PostgresCsv -PostgresContainer $PostgresContainer -DatabaseName $DatabaseName -DatabaseUser $DatabaseUser -Sql @'
SELECT version::text, dirty::text
FROM schema_migrations
WHERE dirty = true
ORDER BY version
'@ -Headers @('version', 'dirty'))
if ($beforeDirty.Count -gt 0) {
    throw "Refusing production migration with dirty schema_migrations rows: $($beforeDirty.version -join ', ')"
}

$artifactRoot = New-ArtifactDirectory -OutputRoot $OutputRoot -Prefix 'production-migration'
$migrationScript = @'
set -eu
encoded_password="$(python3 -c 'import os, urllib.parse; print(urllib.parse.quote(os.environ["DB_PASSWORD"], safe=""))')"
exec /usr/local/bin/migrate \
  -path /app/migrations/versioned \
  -database "postgres://$DB_USER:$encoded_password@$DB_HOST:$DB_PORT/$DB_NAME?sslmode=disable" up
'@

Invoke-Checked -FilePath 'docker' -Arguments ($composeArgs + @(
        'run', '--rm', '--no-deps', '--entrypoint', '/bin/sh', 'app', '-ec', $migrationScript
    ))

$afterDirty = @(Invoke-PostgresCsv -PostgresContainer $PostgresContainer -DatabaseName $DatabaseName -DatabaseUser $DatabaseUser -Sql @'
SELECT version::text, dirty::text
FROM schema_migrations
WHERE dirty = true
ORDER BY version
'@ -Headers @('version', 'dirty'))
if ($afterDirty.Count -gt 0) {
    throw "Production migration left dirty schema_migrations rows: $($afterDirty.version -join ', ')"
}
$versions = @(Invoke-PostgresCsv -PostgresContainer $PostgresContainer -DatabaseName $DatabaseName -DatabaseUser $DatabaseUser -Sql @'
SELECT version::text, dirty::text
FROM schema_migrations
ORDER BY version
'@ -Headers @('version', 'dirty'))
$report = [ordered]@{
    generated_at       = (Get-Date).ToUniversalTime().ToString('o')
    project_name       = $ProjectName
    postgres_container = $PostgresContainer
    database_name      = $DatabaseName
    backup_dump        = $backupDumpPath
    backup_sha256      = $backupHash
    auto_recover_dirty = $false
    migrations         = @($versions)
    status             = 'success'
}
Write-JsonArtifact -Value $report -Path (Join-Path $artifactRoot 'production-migration.json') | Out-Null
Write-ArtifactSha256Sums -ArtifactRoot $artifactRoot | Out-Null
Write-Host "Production migration completed: $artifactRoot"
