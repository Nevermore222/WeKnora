[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string]$ServerSnapshotDir,
    [Parameter(Mandatory = $true)]
    [string]$LocalSnapshotDir,
    [Parameter(Mandatory = $true)]
    [string]$OutputRoot,
    [string]$PostgresImage = 'paradedb/paradedb:v0.22.2-pg17'
)

$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'common.ps1')

function Read-JsonArtifact {
    param([Parameter(Mandatory = $true)][string]$Path)
    if (-not (Test-Path -LiteralPath $Path -PathType Leaf)) {
        throw "Required artifact is missing: $Path"
    }
    return Get-Content -LiteralPath $Path -Raw | ConvertFrom-Json
}

function Get-CanonicalJson {
    param([AllowNull()][object]$Value)
    if ($null -eq $Value) {
        return 'null'
    }
    return ($Value | ConvertTo-Json -Depth 60 -Compress)
}

function Assert-SnapshotDirectory {
    param([Parameter(Mandatory = $true)][string]$Path, [Parameter(Mandatory = $true)][string]$Label)
    $resolved = Assert-ExistingDirectory -Path $Path -Label $Label
    $dumpFiles = @(Get-ChildItem -LiteralPath $resolved -Filter '*.dump' -File)
    if ($dumpFiles.Count -ne 1) {
        throw "$Label must contain exactly one PostgreSQL custom dump: $resolved"
    }
    $snapshot = Read-JsonArtifact -Path (Join-Path $resolved 'snapshot.json')
    $schema = Read-JsonArtifact -Path (Join-Path $resolved 'database-schema.json')
    if (-not (Test-Path -LiteralPath (Join-Path $resolved 'preflight.json') -PathType Leaf)) {
        throw "$Label is missing the copied preflight report: $resolved"
    }
    $expectedDump = @($snapshot.archives | Where-Object { [string]$_.path -match '\.dump$' })
    if ($expectedDump.Count -ne 1) {
        throw "$Label snapshot does not contain one dump hash: $resolved"
    }
    $actualHash = (Get-FileHash -LiteralPath $dumpFiles[0].FullName -Algorithm SHA256).Hash
    if ($actualHash -ine [string]$expectedDump[0].sha256) {
        throw "$Label dump hash does not match snapshot.json: $resolved"
    }
    [pscustomobject]@{
        Root      = $resolved
        DumpPath  = $dumpFiles[0].FullName
        DumpName  = $dumpFiles[0].Name
        Snapshot  = $snapshot
        Schema    = $schema
    }
}

function Invoke-MergeDbCommand {
    param(
        [Parameter(Mandatory = $true)][string]$Container,
        [Parameter(Mandatory = $true)][string]$Database,
        [Parameter(Mandatory = $true)][string]$Sql
    )
    $args = @(
        'exec', $Container, 'psql', '--username=postgres', "--dbname=$Database",
        '--no-password', '--quiet', '--command=' + $Sql
    )
    Invoke-Checked -FilePath 'docker' -Arguments $args
}

function Wait-TemporaryPostgres {
    param([Parameter(Mandatory = $true)][string]$Container)
    for ($attempt = 0; $attempt -lt 90; $attempt++) {
        & docker exec $Container pg_isready --username=postgres --dbname=postgres 1>$null 2>$null
        if ($LASTEXITCODE -eq 0) {
            return
        }
        Start-Sleep -Seconds 1
    }
    throw "Temporary PostgreSQL did not become ready: $Container"
}

function Get-SchemaProperty {
    param([Parameter(Mandatory = $true)][object]$Object, [Parameter(Mandatory = $true)][string]$Name)
    $property = $Object.PSObject.Properties[$Name]
    if ($null -eq $property) {
        return ''
    }
    return $property.Value
}

function ConvertTo-PgLiteral {
    param([Parameter(Mandatory = $true)][string]$Value)
    return "'" + $Value.Replace("'", "''") + "'"
}

function Get-PrimaryKeyMap {
    param([Parameter(Mandatory = $true)][object]$Schema)
    $map = @{}
    foreach ($row in @($Schema.primary_keys)) {
        $table = [string]$row.table_name
        if (-not $map.ContainsKey($table)) {
            $map[$table] = New-Object System.Collections.Generic.List[string]
        }
        [void]$map[$table].Add([string]$row.column_name)
    }
    return $map
}

function Get-TableOrder {
    param([Parameter(Mandatory = $true)][object]$Schema)
    $tables = @($Schema.tables | ForEach-Object { [string]$_.table_name })
    $indegree = @{}
    $children = @{}
    foreach ($table in $tables) {
        $indegree[$table] = 0
        $children[$table] = New-Object System.Collections.Generic.List[string]
    }
    $seenEdges = @{}
    foreach ($foreignKey in @($Schema.foreign_keys)) {
        $child = [string]$foreignKey.table_name
        $parent = [string]$foreignKey.foreign_table_name
        if (-not $indegree.ContainsKey($child) -or -not $indegree.ContainsKey($parent)) {
            throw "Foreign key references a table absent from schema report: $parent -> $child"
        }
        $edge = "$parent`n$child"
        if ($seenEdges.ContainsKey($edge)) {
            continue
        }
        $seenEdges[$edge] = $true
        [void]$children[$parent].Add($child)
        $indegree[$child]++
    }

    $queue = New-Object System.Collections.Generic.Queue[string]
    foreach ($table in $tables | Sort-Object) {
        if ($indegree[$table] -eq 0) {
            $queue.Enqueue($table)
        }
    }
    $ordered = New-Object System.Collections.Generic.List[string]
    while ($queue.Count -gt 0) {
        $table = $queue.Dequeue()
        [void]$ordered.Add($table)
        foreach ($child in @($children[$table] | Sort-Object)) {
            $indegree[$child]--
            if ($indegree[$child] -eq 0) {
                $queue.Enqueue($child)
            }
        }
    }
    if ($ordered.Count -ne $tables.Count) {
        throw 'Foreign-key graph contains a cycle; automatic merge is not safe'
    }
    return @($ordered)
}

function Get-DatabaseFacts {
    param([Parameter(Mandatory = $true)][string]$Container, [Parameter(Mandatory = $true)][string]$Database)
    [ordered]@{
        extensions = @(Invoke-PostgresCsv -PostgresContainer $Container -DatabaseName $Database -DatabaseUser 'postgres' -Sql @'
SELECT extname, extversion FROM pg_extension ORDER BY extname
'@ -Headers @('extname', 'extversion'))
        migrations = @(Invoke-PostgresCsv -PostgresContainer $Container -DatabaseName $Database -DatabaseUser 'postgres' -Sql @'
SELECT version::text, dirty::text FROM schema_migrations ORDER BY version
'@ -Headers @('version', 'dirty'))
        tables = @(Invoke-PostgresCsv -PostgresContainer $Container -DatabaseName $Database -DatabaseUser 'postgres' -Sql @'
SELECT table_name FROM information_schema.tables
WHERE table_schema = 'public' AND table_type = 'BASE TABLE'
ORDER BY table_name
'@ -Headers @('table_name'))
    }
}

function Get-TableColumns {
    param([Parameter(Mandatory = $true)][object]$Schema, [Parameter(Mandatory = $true)][string]$Table)
    return @($Schema.columns | Where-Object { [string]$_.table_name -eq $Table } | Sort-Object { [int]$_.ordinal_position })
}

function Get-ConflictRows {
    param(
        [Parameter(Mandatory = $true)][string]$Container,
        [Parameter(Mandatory = $true)][string]$Table,
        [Parameter(Mandatory = $true)][string[]]$PrimaryKeyColumns
    )
    $quotedTable = ConvertTo-PgIdentifier -Name $Table
    $join = ($PrimaryKeyColumns | ForEach-Object {
            "l.$(ConvertTo-PgIdentifier -Name $_) = s.$(ConvertTo-PgIdentifier -Name $_)"
        }) -join ' AND '
    $keyJson = ($PrimaryKeyColumns | ForEach-Object {
            "$(ConvertTo-PgLiteral -Value $_), l.$(ConvertTo-PgIdentifier -Name $_)"
        }) -join ', '
    $sql = @"
SELECT jsonb_build_object($keyJson)::text AS primary_key,
       encode(digest(convert_to(to_jsonb(l)::text, 'UTF8'), 'sha256'), 'hex') AS local_sha256,
       encode(digest(convert_to(to_jsonb(s)::text, 'UTF8'), 'sha256'), 'hex') AS server_sha256
FROM fdw_local.$quotedTable AS l
JOIN $quotedTable AS s ON $join
WHERE to_jsonb(l) IS DISTINCT FROM to_jsonb(s)
ORDER BY jsonb_build_object($keyJson)::text
"@
    return @(Invoke-PostgresCsv -PostgresContainer $Container -DatabaseName 'merged_preview' -DatabaseUser 'postgres' -Sql $sql -Headers @('primary_key', 'local_sha256', 'server_sha256'))
}

function Get-OnlyRowCount {
    param(
        [Parameter(Mandatory = $true)][string]$Container,
        [Parameter(Mandatory = $true)][string]$Table,
        [Parameter(Mandatory = $true)][string[]]$PrimaryKeyColumns,
        [Parameter(Mandatory = $true)][ValidateSet('local', 'server')][string]$Side
    )
    $quotedTable = ConvertTo-PgIdentifier -Name $Table
    $join = ($PrimaryKeyColumns | ForEach-Object {
            if ($Side -eq 'local') {
                "l.$(ConvertTo-PgIdentifier -Name $_) = s.$(ConvertTo-PgIdentifier -Name $_)"
            } else {
                "s.$(ConvertTo-PgIdentifier -Name $_) = l.$(ConvertTo-PgIdentifier -Name $_)"
            }
        }) -join ' AND '
    if ($Side -eq 'local') {
        $from = "fdw_local.$quotedTable AS l LEFT JOIN $quotedTable AS s ON $join"
        $missing = "s.$(ConvertTo-PgIdentifier -Name $PrimaryKeyColumns[0]) IS NULL"
    } else {
        $from = "$quotedTable AS s LEFT JOIN fdw_local.$quotedTable AS l ON $join"
        $missing = "l.$(ConvertTo-PgIdentifier -Name $PrimaryKeyColumns[0]) IS NULL"
    }
    $sql = "SELECT count(*)::bigint FROM $from WHERE $missing"
    return [int64](Invoke-PostgresScalar -PostgresContainer $Container -DatabaseName 'merged_preview' -DatabaseUser 'postgres' -Sql $sql)
}

$server = Assert-SnapshotDirectory -Path $ServerSnapshotDir -Label 'Server snapshot'
$local = Assert-SnapshotDirectory -Path $LocalSnapshotDir -Label 'Local snapshot'
$artifactRoot = New-ArtifactDirectory -OutputRoot $OutputRoot -Prefix 'merge-plan'
$containerName = "knavis_merge_$((Get-Date).ToUniversalTime().ToString('yyyyMMddHHmmss'))_$([Guid]::NewGuid().ToString('N').Substring(0, 8))"
$serverDatabase = 'server_snapshot'
$localDatabase = 'local_snapshot'
$mergedDatabase = 'merged_preview'
$containerStarted = $false

try {
    $serverSchema = $server.Schema
    $localSchema = $local.Schema
    $schemaDifferences = New-Object System.Collections.Generic.List[object]
    foreach ($propertyName in @('extensions', 'migrations', 'tables', 'columns', 'primary_keys', 'foreign_keys', 'constraints')) {
        $serverValue = Get-SchemaProperty -Object $serverSchema -Name $propertyName
        $localValue = Get-SchemaProperty -Object $localSchema -Name $propertyName
        if ((Get-CanonicalJson -Value $serverValue) -ne (Get-CanonicalJson -Value $localValue)) {
            $schemaDifferences.Add([pscustomobject]@{ property = $propertyName; server = $serverValue; local = $localValue })
        }
    }
    $schemaCompare = [ordered]@{
        generated_at = (Get-Date).ToUniversalTime().ToString('o')
        matches      = ($schemaDifferences.Count -eq 0)
        differences  = @($schemaDifferences)
    }
    Write-JsonArtifact -Value $schemaCompare -Path (Join-Path $artifactRoot 'schema-compare.json') | Out-Null
    if ($schemaDifferences.Count -gt 0) {
        throw 'Snapshot schema, extensions, migration versions, or constraints differ'
    }

    $unsupportedTables = @($serverSchema.unsupported | Where-Object { [int64]$_.row_count -gt 0 })
    if ($unsupportedTables.Count -gt 0) {
        Write-JsonArtifact -Value $unsupportedTables -Path (Join-Path $artifactRoot 'unsupported-tables.json') | Out-Null
        throw "Unsupported non-empty tables are present: $($unsupportedTables.table_name -join ', ')"
    }

    $tableOrder = @(Get-TableOrder -Schema $serverSchema)
    Write-JsonArtifact -Value ([ordered]@{ order = $tableOrder }) -Path (Join-Path $artifactRoot 'table-order.json') | Out-Null

    $serverMount = "type=bind,source=$($server.Root),target=/snapshots/server,readonly"
    $localMount = "type=bind,source=$($local.Root),target=/snapshots/local,readonly"
    $containerId = [string]((Invoke-CheckedOutput -FilePath 'docker' -Arguments @(
            'run', '-d', '--name', $containerName,
            '-e', 'POSTGRES_HOST_AUTH_METHOD=trust',
            '--mount', $serverMount,
            '--mount', $localMount,
            $PostgresImage
        ))[-1])
    $containerStarted = $true
    Wait-TemporaryPostgres -Container $containerName

    foreach ($database in @($serverDatabase, $localDatabase, $mergedDatabase)) {
        Invoke-Checked -FilePath 'docker' -Arguments @('exec', $containerName, 'createdb', '--username=postgres', $database)
    }
    Invoke-Checked -FilePath 'docker' -Arguments @(
        'exec', $containerName, 'pg_restore', '--username=postgres', '--dbname=server_snapshot',
        '--no-owner', '--no-privileges', '--exit-on-error', "/snapshots/server/$($server.DumpName)"
    )
    Invoke-Checked -FilePath 'docker' -Arguments @(
        'exec', $containerName, 'pg_restore', '--username=postgres', '--dbname=local_snapshot',
        '--no-owner', '--no-privileges', '--exit-on-error', "/snapshots/local/$($local.DumpName)"
    )

    $fdwSql = @'
CREATE EXTENSION IF NOT EXISTS postgres_fdw;
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE SCHEMA IF NOT EXISTS fdw_local;
CREATE SERVER knavis_local_snapshot FOREIGN DATA WRAPPER postgres_fdw
  OPTIONS (host '127.0.0.1', port '5432', dbname 'local_snapshot');
CREATE USER MAPPING FOR postgres SERVER knavis_local_snapshot;
IMPORT FOREIGN SCHEMA public FROM SERVER knavis_local_snapshot INTO fdw_local;
'@
    Invoke-MergeDbCommand -Container $containerName -Database $mergedDatabase -Sql $fdwSql

    $serverFacts = Get-DatabaseFacts -Container $containerName -Database $serverDatabase
    $localFacts = Get-DatabaseFacts -Container $containerName -Database $localDatabase
    Write-JsonArtifact -Value ([ordered]@{ server = $serverFacts; local = $localFacts }) -Path (Join-Path $artifactRoot 'temporary-schema-facts.json') | Out-Null
    if ((Get-CanonicalJson -Value $serverFacts.extensions) -ne (Get-CanonicalJson -Value $localFacts.extensions) -or
        (Get-CanonicalJson -Value $serverFacts.migrations) -ne (Get-CanonicalJson -Value $localFacts.migrations) -or
        (Get-CanonicalJson -Value $serverFacts.tables) -ne (Get-CanonicalJson -Value $localFacts.tables)) {
        throw 'Restored temporary database facts differ'
    }

    $primaryKeyMap = Get-PrimaryKeyMap -Schema $serverSchema
    $conflicts = New-Object System.Collections.Generic.List[object]
    $tableResults = New-Object System.Collections.Generic.List[object]
    $mergeLines = New-Object System.Collections.Generic.List[string]
    [void]$mergeLines.Add('BEGIN;')
    [void]$mergeLines.Add("SET lock_timeout = '10s';")
    [void]$mergeLines.Add("SET statement_timeout = '0';")

    foreach ($table in $tableOrder) {
        $pkColumns = @($primaryKeyMap[$table])
        $columnsForTable = @(Get-TableColumns -Schema $serverSchema -Table $table)
        $rowCount = [int64](Invoke-PostgresScalar -PostgresContainer $containerName -DatabaseName $serverDatabase -DatabaseUser 'postgres' -Sql "SELECT count(*)::bigint FROM $(ConvertTo-PgIdentifier -Name $table)")
        if ($pkColumns.Count -eq 0) {
            $tableResults.Add([pscustomobject]@{ table = $table; status = 'empty-or-unsupported'; server_rows = $rowCount })
            continue
        }

        $tableConflicts = @(Get-ConflictRows -Container $containerName -Table $table -PrimaryKeyColumns $pkColumns)
        foreach ($conflict in $tableConflicts) {
            $conflicts.Add([pscustomobject]@{
                    table            = $table
                    primary_key      = [string]$conflict.primary_key
                    local_sha256     = [string]$conflict.local_sha256
                    server_sha256    = [string]$conflict.server_sha256
                    winner           = 'local'
                    dependency_action = 'upsert in topological order'
                })
        }
        $localOnly = Get-OnlyRowCount -Container $containerName -Table $table -PrimaryKeyColumns $pkColumns -Side local
        $serverOnly = Get-OnlyRowCount -Container $containerName -Table $table -PrimaryKeyColumns $pkColumns -Side server
        $tableResults.Add([pscustomobject]@{
                table       = $table
                status      = 'supported'
                server_rows = $rowCount
                conflicts   = $tableConflicts.Count
                local_only  = $localOnly
                server_only = $serverOnly
            })

        $insertColumns = @($columnsForTable | Where-Object { (Get-SchemaProperty -Object $_ -Name 'is_generated') -ne 'ALWAYS' })
        if ($insertColumns.Count -eq 0) {
            throw "Table has no writable columns: $table"
        }
        $quotedColumns = @($insertColumns | ForEach-Object { ConvertTo-PgIdentifier -Name ([string]$_.column_name) })
        $identity = @($insertColumns | Where-Object { (Get-SchemaProperty -Object $_ -Name 'is_identity') -eq 'YES' }).Count -gt 0
        $identityClause = if ($identity) { ' OVERRIDING SYSTEM VALUE' } else { '' }
        $updateColumns = @($insertColumns | Where-Object { $pkColumns -notcontains [string]$_.column_name })
        $conflictAction = if ($updateColumns.Count -eq 0) {
            'DO NOTHING'
        } else {
            'DO UPDATE SET ' + (($updateColumns | ForEach-Object {
                        $name = ConvertTo-PgIdentifier -Name ([string]$_.column_name)
                        "$name = EXCLUDED.$name"
                    }) -join ', ')
        }
        [void]$mergeLines.Add("-- $table")
        [void]$mergeLines.Add("INSERT INTO $(ConvertTo-PgIdentifier -Name $table) ($($quotedColumns -join ', '))$identityClause")
        [void]$mergeLines.Add("SELECT $($quotedColumns -join ', ') FROM fdw_local.$(ConvertTo-PgIdentifier -Name $table)")
        [void]$mergeLines.Add("ON CONFLICT ($((@($pkColumns | ForEach-Object { ConvertTo-PgIdentifier -Name $_ })) -join ', ')) $conflictAction;")
        foreach ($sequenceColumn in @($insertColumns | Where-Object { (Get-SchemaProperty -Object $_ -Name 'column_default') -match 'nextval' })) {
            $columnName = [string]$sequenceColumn.column_name
            [void]$mergeLines.Add("SELECT setval(pg_get_serial_sequence('$(($table -replace "'", "''"))', '$(($columnName -replace "'", "''"))'), COALESCE(MAX($(ConvertTo-PgIdentifier -Name $columnName)), 1), MAX($(ConvertTo-PgIdentifier -Name $columnName)) IS NOT NULL) FROM $(ConvertTo-PgIdentifier -Name $table);")
        }
    }
    [void]$mergeLines.Add('-- COMMIT is intentionally omitted from the dry-run plan.')
    $mergeSqlPath = Join-Path $artifactRoot 'merge.sql'
    $mergeLines | Set-Content -LiteralPath $mergeSqlPath -Encoding UTF8

    Write-JsonArtifact -Value @($conflicts) -Path (Join-Path $artifactRoot 'conflicts.json') | Out-Null
    Write-JsonArtifact -Value @($unsupportedTables) -Path (Join-Path $artifactRoot 'unsupported-tables.json') | Out-Null
    $plan = [ordered]@{
        generated_at       = (Get-Date).ToUniversalTime().ToString('o')
        temporary_container = $containerName
        server_snapshot    = $server.Root
        local_snapshot     = $local.Root
        conflict_count     = $conflicts.Count
        table_count        = $tableResults.Count
        local_wins         = $true
        merge_sql_sha256   = (Get-FileHash -LiteralPath $mergeSqlPath -Algorithm SHA256).Hash
        tables             = @($tableResults)
        status             = 'dry-run'
    }
    Write-JsonArtifact -Value $plan -Path (Join-Path $artifactRoot 'merge-plan.json') | Out-Null
    Write-ArtifactSha256Sums -ArtifactRoot $artifactRoot | Out-Null
    Write-Host "Merge plan passed (dry-run only): $artifactRoot"
} finally {
    if ($containerStarted) {
        & docker rm -f $containerName 1>$null 2>$null
    }
}
