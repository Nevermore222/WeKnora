[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [ValidatePattern('^[A-Za-z0-9_-]+$')]
    [string]$SourceName,
    [Parameter(Mandatory = $true)]
    [string]$PostgresContainer,
    [Parameter(Mandatory = $true)]
    [string]$DatabaseName,
    [Parameter(Mandatory = $true)]
    [string]$DatabaseUser,
    [Parameter(Mandatory = $true)]
    [string]$DataFilesVolume,
    [Parameter(Mandatory = $true)]
    [string]$WorkspaceRoot,
    [Parameter(Mandatory = $true)]
    [string]$OutputRoot,
    [string[]]$ExpectedExtension = @('pg_trgm', 'pg_search', 'uuid-ossp', 'vector')
)

$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'common.ps1')

function ConvertTo-PgLiteral {
    param([Parameter(Mandatory = $true)][string]$Value)
    return "'" + $Value.Replace("'", "''") + "'"
}

function Get-PrimaryKeyExpression {
    param(
        [Parameter(Mandatory = $true)][string]$TableName,
        [Parameter(Mandatory = $true)][hashtable]$PrimaryKeyMap
    )

    $columns = @($PrimaryKeyMap[$TableName])
    if ($columns.Count -eq 0) {
        return "'{}'::jsonb"
    }
    $parts = foreach ($column in $columns) {
        "$(ConvertTo-PgLiteral -Value $column), t.$(ConvertTo-PgIdentifier -Name $column)"
    }
    return "jsonb_build_object($($parts -join ', '))"
}

$artifactRoot = New-ArtifactDirectory -OutputRoot $OutputRoot -Prefix "preflight-$SourceName"
Assert-ExistingDirectory -Path $artifactRoot -Label 'Preflight artifact' | Out-Null
Invoke-CheckedOutput -FilePath 'docker' -Arguments @('version', '--format', '{{.Server.Version}}') | Out-Null
Assert-ContainerRunning -Name $PostgresContainer
Assert-DockerVolume -Name $DataFilesVolume
$resolvedWorkspaceRoot = Assert-ExistingDirectory -Path $WorkspaceRoot -Label 'Workspace'

$extensions = @(Invoke-PostgresCsv -PostgresContainer $PostgresContainer -DatabaseName $DatabaseName -DatabaseUser $DatabaseUser -Sql @'
SELECT extname, extversion
FROM pg_extension
ORDER BY extname
'@ -Headers @('extname', 'extversion'))
$extensionNames = @($extensions | ForEach-Object { [string]$_.extname })
$missingExtensions = @($ExpectedExtension | Where-Object { $extensionNames -notcontains $_ })
if ($missingExtensions.Count -gt 0) {
    throw "Required PostgreSQL extensions are missing: $($missingExtensions -join ', ')"
}

$migrationTableExists = Invoke-PostgresScalar -PostgresContainer $PostgresContainer -DatabaseName $DatabaseName -DatabaseUser $DatabaseUser -Sql "SELECT to_regclass('public.schema_migrations') IS NOT NULL"
if ($migrationTableExists -ne 't') {
    throw 'schema_migrations table is missing'
}
$migrationRows = @(Invoke-PostgresCsv -PostgresContainer $PostgresContainer -DatabaseName $DatabaseName -DatabaseUser $DatabaseUser -Sql @'
SELECT version::text, dirty::text
FROM schema_migrations
ORDER BY version
'@ -Headers @('version', 'dirty'))
if ($migrationRows.Count -eq 0) {
    throw 'schema_migrations is empty'
}
$dirtyRows = @($migrationRows | Where-Object { ([string]$_.dirty).ToLowerInvariant() -eq 'true' })
if ($dirtyRows.Count -gt 0) {
    throw "Database migration state is dirty at version(s): $($dirtyRows.version -join ', ')"
}

$tables = @(Invoke-PostgresCsv -PostgresContainer $PostgresContainer -DatabaseName $DatabaseName -DatabaseUser $DatabaseUser -Sql @'
SELECT table_name
FROM information_schema.tables
WHERE table_schema = 'public'
  AND table_type = 'BASE TABLE'
ORDER BY table_name
'@ -Headers @('table_name'))
$columns = @(Invoke-PostgresCsv -PostgresContainer $PostgresContainer -DatabaseName $DatabaseName -DatabaseUser $DatabaseUser -Sql @'
SELECT table_name, column_name, ordinal_position, data_type, udt_name,
       is_nullable, COALESCE(column_default, '') AS column_default
FROM information_schema.columns
WHERE table_schema = 'public'
ORDER BY table_name, ordinal_position
'@ -Headers @('table_name', 'column_name', 'ordinal_position', 'data_type', 'udt_name', 'is_nullable', 'column_default'))
$primaryKeys = @(Invoke-PostgresCsv -PostgresContainer $PostgresContainer -DatabaseName $DatabaseName -DatabaseUser $DatabaseUser -Sql @'
SELECT tc.table_name, kcu.column_name, kcu.ordinal_position
FROM information_schema.table_constraints tc
JOIN information_schema.key_column_usage kcu
  ON tc.constraint_name = kcu.constraint_name
 AND tc.table_schema = kcu.table_schema
 AND tc.table_name = kcu.table_name
WHERE tc.table_schema = 'public'
  AND tc.constraint_type = 'PRIMARY KEY'
ORDER BY tc.table_name, kcu.ordinal_position
'@ -Headers @('table_name', 'column_name', 'ordinal_position'))
$foreignKeys = @(Invoke-PostgresCsv -PostgresContainer $PostgresContainer -DatabaseName $DatabaseName -DatabaseUser $DatabaseUser -Sql @'
SELECT tc.constraint_name, tc.table_name, kcu.column_name,
       ccu.table_name AS foreign_table_name,
       ccu.column_name AS foreign_column_name
FROM information_schema.table_constraints tc
JOIN information_schema.key_column_usage kcu
  ON tc.constraint_name = kcu.constraint_name
 AND tc.table_schema = kcu.table_schema
 AND tc.table_name = kcu.table_name
JOIN information_schema.constraint_column_usage ccu
  ON ccu.constraint_name = tc.constraint_name
 AND ccu.table_schema = tc.table_schema
WHERE tc.table_schema = 'public'
  AND tc.constraint_type = 'FOREIGN KEY'
ORDER BY tc.table_name, tc.constraint_name, kcu.ordinal_position
'@ -Headers @('constraint_name', 'table_name', 'column_name', 'foreign_table_name', 'foreign_column_name'))
$constraintValidation = @(Invoke-PostgresCsv -PostgresContainer $PostgresContainer -DatabaseName $DatabaseName -DatabaseUser $DatabaseUser -Sql @'
SELECT con.conname, cls.relname AS table_name, con.contype, con.convalidated::text
FROM pg_constraint con
JOIN pg_class cls ON cls.oid = con.conrelid
JOIN pg_namespace nsp ON nsp.oid = cls.relnamespace
WHERE nsp.nspname = 'public'
ORDER BY cls.relname, con.conname
'@ -Headers @('constraint_name', 'table_name', 'constraint_type', 'convalidated'))
$unvalidatedConstraints = @($constraintValidation | Where-Object { ([string]$_.convalidated).ToLowerInvariant() -eq 'false' })
if ($unvalidatedConstraints.Count -gt 0) {
    throw "Database contains unvalidated constraints: $($unvalidatedConstraints.constraint_name -join ', ')"
}

$primaryKeyMap = @{}
foreach ($row in $primaryKeys) {
    $tableName = [string]$row.table_name
    if (-not $primaryKeyMap.ContainsKey($tableName)) {
        $primaryKeyMap[$tableName] = New-Object System.Collections.Generic.List[string]
    }
    [void]$primaryKeyMap[$tableName].Add([string]$row.column_name)
}

$rowCounts = @(
    foreach ($table in $tables) {
        $tableName = [string]$table.table_name
        $quotedTable = ConvertTo-PgIdentifier -Name $tableName
        $count = Invoke-PostgresScalar -PostgresContainer $PostgresContainer -DatabaseName $DatabaseName -DatabaseUser $DatabaseUser -Sql "SELECT count(*)::bigint FROM $quotedTable"
        [pscustomobject]@{
            table_name       = $tableName
            row_count        = [int64]$count
            has_primary_key  = $primaryKeyMap.ContainsKey($tableName)
        }
    }
)
$unsupportedTables = @($rowCounts | Where-Object { $_.row_count -gt 0 -and -not $_.has_primary_key })
if ($unsupportedTables.Count -gt 0) {
    throw "Non-empty tables without primary keys: $($unsupportedTables.table_name -join ', ')"
}

$dataFilesManifest = @(Get-DataFilesVolumeManifest -VolumeName $DataFilesVolume)
$dataFilesByKey = @{}
foreach ($file in $dataFilesManifest) {
    $key = Assert-RelativeStorageKey -Key ([string]$file.relative_path)
    $dataFilesByKey[$key] = $file
}

$fileReferences = New-Object System.Collections.Generic.List[object]
$pathPattern = '(?:local://|minio://|/data/files/)[^"\s<>\]\},]+'
foreach ($column in @($columns | Where-Object { $_.data_type -in @('text', 'character varying', 'jsonb') })) {
    $tableName = [string]$column.table_name
    $columnName = [string]$column.column_name
    $quotedTable = ConvertTo-PgIdentifier -Name $tableName
    $quotedColumn = ConvertTo-PgIdentifier -Name $columnName
    $pkExpression = Get-PrimaryKeyExpression -TableName $tableName -PrimaryKeyMap $primaryKeyMap
    $sql = @"
SELECT ($pkExpression)::text AS primary_key_json,
       ($quotedColumn)::text AS value
FROM $quotedTable AS t
WHERE ($quotedColumn)::text LIKE '%local://%'
   OR ($quotedColumn)::text LIKE '%minio://%'
   OR ($quotedColumn)::text LIKE '%/data/files/%'
"@
    $matches = @(Invoke-PostgresCsv -PostgresContainer $PostgresContainer -DatabaseName $DatabaseName -DatabaseUser $DatabaseUser -Sql $sql -Headers @('primary_key_json', 'value'))
    foreach ($match in $matches) {
        $value = [string]$match.value
        foreach ($regexMatch in [regex]::Matches($value, $pathPattern)) {
            $path = $regexMatch.Value.TrimEnd('.', ',', ';', ')')
            if ($path.StartsWith('local://')) {
                $key = Assert-RelativeStorageKey -Key $path.Substring('local://'.Length)
                if (-not $dataFilesByKey.ContainsKey($key)) {
                    throw "Local file reference is absent from the declared data-files volume: $path"
                }
                $scheme = 'local'
            } elseif ($path.StartsWith('/data/files/')) {
                $key = Assert-RelativeStorageKey -Key $path.Substring('/data/files/'.Length)
                if (-not $dataFilesByKey.ContainsKey($key)) {
                    throw "Legacy file reference is absent from the declared data-files volume: $path"
                }
                $scheme = 'local'
            } else {
                $scheme = 'minio'
            }
            $fileReferences.Add([pscustomobject]@{
                    table          = $tableName
                    column         = $columnName
                    primary_key    = [string]$match.primary_key_json
                    path           = $path
                    scheme         = $scheme
                    value_sha256   = Get-Sha256Hex -Bytes ([Text.Encoding]::UTF8.GetBytes($value))
                })
        }
    }
}

$workspaceManifest = @(Get-WorkspaceManifest -Root $resolvedWorkspaceRoot)
foreach ($workspaceFile in $workspaceManifest) {
    $workspacePath = Join-Path $resolvedWorkspaceRoot ([string]$workspaceFile.relative_path -replace '/', '\')
    $workspaceHash = (Get-FileHash -LiteralPath $workspacePath -Algorithm SHA256).Hash.ToLowerInvariant()
    if ($workspaceHash -ne [string]$workspaceFile.sha256) {
        throw "Workspace file changed while preflight was running: $($workspaceFile.relative_path)"
    }
}
$schemaReport = [ordered]@{
    source_name     = $SourceName
    generated_at    = (Get-Date).ToUniversalTime().ToString('o')
    extensions      = $extensions
    migrations      = $migrationRows
    tables          = $tables
    columns         = $columns
    primary_keys    = $primaryKeys
    foreign_keys    = $foreignKeys
    constraints     = $constraintValidation
    row_counts      = $rowCounts
    unsupported     = $unsupportedTables
}
$fileReferenceReport = [ordered]@{
    source_name  = $SourceName
    generated_at = (Get-Date).ToUniversalTime().ToString('o')
    references   = @($fileReferences)
}

Write-JsonArtifact -Value $schemaReport -Path (Join-Path $artifactRoot 'database-schema.json') | Out-Null
Write-JsonArtifact -Value $fileReferenceReport -Path (Join-Path $artifactRoot 'file-references.json') | Out-Null
Write-JsonArtifact -Value ([ordered]@{ source_name = $SourceName; files = $dataFilesManifest }) -Path (Join-Path $artifactRoot 'data-files-manifest.json') | Out-Null
Write-JsonArtifact -Value ([ordered]@{ source_name = $SourceName; root = $resolvedWorkspaceRoot; files = $workspaceManifest }) -Path (Join-Path $artifactRoot 'workspace-manifest.json') | Out-Null

$preflight = [ordered]@{
    source_name          = $SourceName
    generated_at         = (Get-Date).ToUniversalTime().ToString('o')
    postgres_container   = $PostgresContainer
    database_name        = $DatabaseName
    data_files_volume    = $DataFilesVolume
    workspace_root       = $resolvedWorkspaceRoot
    extension_count      = $extensions.Count
    table_count          = $tables.Count
    file_reference_count = $fileReferences.Count
    data_file_count      = $dataFilesManifest.Count
    workspace_file_count = $workspaceManifest.Count
    status                = 'success'
}
Write-JsonArtifact -Value $preflight -Path (Join-Path $artifactRoot 'preflight.json') | Out-Null
New-Item -ItemType File -Path (Join-Path $artifactRoot 'preflight.ok') | Out-Null
Write-ArtifactSha256Sums -ArtifactRoot $artifactRoot | Out-Null
Write-Host "Preflight passed: $artifactRoot"
