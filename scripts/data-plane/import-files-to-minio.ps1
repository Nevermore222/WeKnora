[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string]$DataFilesArchive,
    [Parameter(Mandatory = $true)]
    [string]$DataFilesManifest,
    [Parameter(Mandatory = $true)]
    [string]$FileReferencesReport,
    [Parameter(Mandatory = $true)]
    [ValidatePattern('^[A-Fa-f0-9]{64}$')]
    [string]$ReviewedReferencesHash,
    [Parameter(Mandatory = $true)]
    [string]$MinioCredentialEnvFile,
    [Parameter(Mandatory = $true)]
    [string]$MinioEndpoint,
    [Parameter(Mandatory = $true)]
    [ValidatePattern('^[a-z0-9][a-z0-9.-]{1,61}[a-z0-9]$')]
    [string]$TargetBucket,
    [Parameter(Mandatory = $true)]
    [string]$OutputRoot,
    [string]$MinioDockerNetwork,
    [string]$PostgresContainer,
    [string]$DatabaseName,
    [string]$DatabaseUser,
    [string]$MinioMcImage = 'minio/minio:RELEASE.2025-09-07T16-13-09Z',
    [switch]$ApproveFileImport,
    [switch]$ApplyReviewedReferences
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

function Read-JsonArtifact {
    param([Parameter(Mandatory = $true)][string]$Path)
    return Get-Content -LiteralPath $Path -Raw | ConvertFrom-Json
}

function ConvertTo-PgLiteral {
    param([Parameter(Mandatory = $true)][string]$Value)
    return "'" + $Value.Replace("'", "''") + "'"
}

function Get-MinIOEndpoint {
    param([Parameter(Mandatory = $true)][string]$Endpoint)
    $normalized = $Endpoint.Trim()
    if ($normalized -notmatch '^https?://') {
        $normalized = "http://$normalized"
    }
    $uri = $null
    if (-not [Uri]::TryCreate($normalized, [UriKind]::Absolute, [ref]$uri) -or $uri.Scheme -notin @('http', 'https')) {
        throw "MinIO endpoint must be an HTTP(S) endpoint: $Endpoint"
    }
    return $normalized.TrimEnd('/')
}

function Get-RelativeKeyFromLocalPath {
    param([Parameter(Mandatory = $true)][string]$Path)
    if ($Path.StartsWith('local://')) {
        return Assert-RelativeStorageKey -Key $Path.Substring('local://'.Length)
    }
    if ($Path.StartsWith('/data/files/')) {
        return Assert-RelativeStorageKey -Key $Path.Substring('/data/files/'.Length)
    }
    throw "Reviewed file reference is not a local storage path: $Path"
}

function Get-ManifestMap {
    param([Parameter(Mandatory = $true)][object]$Manifest)
    $filesProperty = $Manifest.PSObject.Properties['files']
    if ($null -eq $filesProperty) {
        throw 'Data-files manifest is missing its files array'
    }
    $map = @{}
    foreach ($file in @($filesProperty.Value)) {
        $key = Assert-RelativeStorageKey -Key ([string]$file.relative_path)
        if ($map.ContainsKey($key)) {
            throw "Data-files manifest contains the same key more than once: $key"
        }
        $hash = ([string]$file.sha256).ToLowerInvariant()
        if ($hash -notmatch '^[a-f0-9]{64}$') {
            throw "Data-files manifest contains an invalid SHA-256 for: $key"
        }
        $map[$key] = [pscustomobject]@{
            relative_path = $key
            bytes         = [int64]$file.bytes
            sha256        = $hash
        }
    }
    return $map
}

function Get-ReviewedKnowledgeReferences {
    param(
        [Parameter(Mandatory = $true)][object]$Report,
        [Parameter(Mandatory = $true)][hashtable]$ManifestByKey,
        [Parameter(Mandatory = $true)][string]$TargetBucketName
    )

    $referencesProperty = $Report.PSObject.Properties['references']
    if ($null -eq $referencesProperty) {
        throw 'file-references.json is missing its references array'
    }

    $result = New-Object System.Collections.Generic.List[object]
    $seenIDs = @{}
    foreach ($reference in @($referencesProperty.Value)) {
        $path = [string]$reference.path
        $isLocal = ([string]$reference.scheme -eq 'local') -or $path.StartsWith('local://') -or $path.StartsWith('/data/files/')
        if (-not $isLocal) {
            continue
        }
        if ([string]$reference.table -ne 'knowledges' -or [string]$reference.column -ne 'file_path') {
            throw "A local file reference is outside the reviewed knowledges.file_path list: $($reference.table).$($reference.column)"
        }
        $primaryKey = ([string]$reference.primary_key | ConvertFrom-Json)
        $knowledgeID = [string]$primaryKey.id
        if ([string]::IsNullOrWhiteSpace($knowledgeID)) {
            throw "Reviewed knowledge reference has no id primary key: $path"
        }
        if ($seenIDs.ContainsKey($knowledgeID)) {
            throw "Reviewed knowledge reference has a duplicate id: $knowledgeID"
        }
        $key = Get-RelativeKeyFromLocalPath -Path $path
        if (-not $ManifestByKey.ContainsKey($key)) {
            throw "Reviewed local path is absent from the data-files manifest: $path"
        }
        $seenIDs[$knowledgeID] = $true
        $result.Add([pscustomobject]@{
                id                  = $knowledgeID
                old_path            = $path
                object_key          = $key
                new_path            = "minio://$TargetBucketName/$key"
                source_value_sha256 = [string]$reference.value_sha256
                bytes               = [int64]$ManifestByKey[$key].bytes
                sha256              = [string]$ManifestByKey[$key].sha256
            })
    }
    return @($result)
}

function Assert-ArchiveEntries {
    param([Parameter(Mandatory = $true)][string[]]$Entries)
    foreach ($entryLine in $Entries) {
        $entry = ([string]$entryLine).Trim()
        if ([string]::IsNullOrWhiteSpace($entry) -or $entry -eq '.') {
            continue
        }
        if ($entry -match '(^/|^[A-Za-z]:|(^|/)\.\.(/|$))') {
            throw "Data-files archive contains an unsafe entry: $entry"
        }
        $key = $entry.TrimStart('.', '/')
        if ($key.EndsWith('/')) {
            $key = $key.TrimEnd('/')
        }
        if ($key) {
            Assert-RelativeStorageKey -Key $key | Out-Null
        }
    }
}

function Assert-StagedFilesMatchManifest {
    param([Parameter(Mandatory = $true)][string]$StagingRoot, [Parameter(Mandatory = $true)][hashtable]$ManifestByKey)
    $items = @(Get-ChildItem -LiteralPath $StagingRoot -Recurse -Force -ErrorAction Stop)
    $reparse = @($items | Where-Object { $_.Attributes -band [IO.FileAttributes]::ReparsePoint })
    if ($reparse.Count -gt 0) {
        throw "Data-files archive extracted a reparse point: $($reparse[0].FullName)"
    }
    $actualByKey = @{}
    foreach ($item in @($items | Where-Object { -not $_.PSIsContainer })) {
        $key = $item.FullName.Substring($StagingRoot.Length).TrimStart('\', '/') -replace '\\', '/'
        $key = Assert-RelativeStorageKey -Key $key
        if ($actualByKey.ContainsKey($key)) {
            throw "Data-files archive extracted a duplicate path: $key"
        }
        $actualByKey[$key] = $item
    }
    if ($actualByKey.Count -ne $ManifestByKey.Count) {
        throw "Data-files archive count does not match its manifest: archive=$($actualByKey.Count) manifest=$($ManifestByKey.Count)"
    }
    foreach ($key in $ManifestByKey.Keys) {
        if (-not $actualByKey.ContainsKey($key)) {
            throw "Data-files archive is missing a manifest file: $key"
        }
        $item = $actualByKey[$key]
        $expected = $ManifestByKey[$key]
        if ([int64]$item.Length -ne [int64]$expected.bytes) {
            throw "Data-files archive byte count differs for: $key"
        }
        $actualHash = (Get-FileHash -LiteralPath $item.FullName -Algorithm SHA256).Hash.ToLowerInvariant()
        if ($actualHash -ne [string]$expected.sha256) {
            throw "Data-files archive SHA-256 differs for: $key"
        }
    }
}

function Get-TargetKnowledgeRows {
    param(
        [Parameter(Mandatory = $true)][string]$Container,
        [Parameter(Mandatory = $true)][string]$Database,
        [Parameter(Mandatory = $true)][string]$User,
        [Parameter(Mandatory = $true)][object[]]$References
    )
    $rows = @{}
    foreach ($reference in $References) {
        $idLiteral = ConvertTo-PgLiteral -Value ([string]$reference.id)
        $query = @"
SELECT id, tenant_id::text, knowledge_base_id, file_path
FROM knowledges
WHERE id = $idLiteral
  AND deleted_at IS NULL
"@
        $found = @(Invoke-PostgresCsv -PostgresContainer $Container -DatabaseName $Database -DatabaseUser $User -Sql $query -Headers @('id', 'tenant_id', 'knowledge_base_id', 'file_path'))
        if ($found.Count -ne 1) {
            throw "Reviewed knowledge row is absent or duplicated in the target database: $($reference.id)"
        }
        if ([string]$found[0].file_path -ne [string]$reference.old_path) {
            throw "Reviewed knowledge file path changed after review: $($reference.id)"
        }
        $rows[[string]$reference.id] = $found[0]
    }
    return $rows
}

function New-ReferenceUpdateSql {
    param(
        [Parameter(Mandatory = $true)][object[]]$References,
        [Parameter(Mandatory = $true)][hashtable]$TargetRows,
        [Parameter(Mandatory = $true)][string]$BucketName,
        [Parameter(Mandatory = $true)][bool]$UseSsl
    )
    $knowledgeBaseIDs = @($TargetRows.Values | ForEach-Object { [string]$_.knowledge_base_id } | Sort-Object -Unique)
    $tenantIDs = @($TargetRows.Values | ForEach-Object { [string]$_.tenant_id } | Sort-Object -Unique)
    $lines = New-Object System.Collections.Generic.List[string]
    [void]$lines.Add('BEGIN;')
    [void]$lines.Add("SET lock_timeout = '10s';")
    [void]$lines.Add('DO $$')
    [void]$lines.Add('DECLARE affected integer;')
    [void]$lines.Add('BEGIN')
    foreach ($reference in $References) {
        $id = ConvertTo-PgLiteral -Value ([string]$reference.id)
        $oldPath = ConvertTo-PgLiteral -Value ([string]$reference.old_path)
        $newPath = ConvertTo-PgLiteral -Value ([string]$reference.new_path)
        [void]$lines.Add("  UPDATE knowledges SET file_path = $newPath WHERE id = $id AND file_path = $oldPath AND deleted_at IS NULL;")
        [void]$lines.Add('  GET DIAGNOSTICS affected = ROW_COUNT;')
        [void]$lines.Add("  IF affected <> 1 THEN RAISE EXCEPTION 'Reviewed knowledge row changed before update'; END IF;")
    }
    foreach ($knowledgeBaseID in $knowledgeBaseIDs) {
        if ([string]::IsNullOrWhiteSpace($knowledgeBaseID)) {
            throw 'Reviewed knowledge row has no knowledge_base_id'
        }
        $literal = ConvertTo-PgLiteral -Value $knowledgeBaseID
        [void]$lines.Add("  UPDATE knowledge_bases SET storage_provider_config = jsonb_build_object('provider', 'minio') WHERE id = $literal AND deleted_at IS NULL;")
        [void]$lines.Add('  GET DIAGNOSTICS affected = ROW_COUNT;')
        [void]$lines.Add("  IF affected <> 1 THEN RAISE EXCEPTION 'Reviewed knowledge base changed before update'; END IF;")
    }
    $bucketLiteral = ConvertTo-PgLiteral -Value $BucketName
    $sslLiteral = if ($UseSsl) { 'true' } else { 'false' }
    foreach ($tenantID in $tenantIDs) {
        if ($tenantID -notmatch '^\d+$') {
            throw "Reviewed tenant id is not numeric: $tenantID"
        }
        [void]$lines.Add(@"
  UPDATE tenants
  SET storage_engine_config = jsonb_set(
      jsonb_set(COALESCE(storage_engine_config, '{}'::jsonb), '{default_provider}', to_jsonb('minio'::text), true),
      '{minio}',
      COALESCE(storage_engine_config->'minio', '{}'::jsonb) || jsonb_build_object('mode', 'docker', 'bucket_name', $bucketLiteral, 'use_ssl', $sslLiteral),
      true
  )
  WHERE id = $tenantID AND deleted_at IS NULL;
  GET DIAGNOSTICS affected = ROW_COUNT;
  IF affected <> 1 THEN RAISE EXCEPTION 'Reviewed tenant changed before update'; END IF;
"@)
    }
    [void]$lines.Add('END $$;')
    [void]$lines.Add('COMMIT;')
    return @($lines)
}

if (-not $ApproveFileImport) {
    throw 'Pass -ApproveFileImport only after reviewing the data-files manifest and file-references.json'
}
if ($ApplyReviewedReferences -and ([string]::IsNullOrWhiteSpace($PostgresContainer) -or [string]::IsNullOrWhiteSpace($DatabaseName) -or [string]::IsNullOrWhiteSpace($DatabaseUser))) {
    throw 'Applying reviewed references requires PostgresContainer, DatabaseName, and DatabaseUser'
}

$archivePath = Assert-ExistingFile -Path $DataFilesArchive -Label 'Data-files archive'
$manifestPath = Assert-ExistingFile -Path $DataFilesManifest -Label 'Data-files manifest'
$referencesPath = Assert-ExistingFile -Path $FileReferencesReport -Label 'file-references.json'
$credentialEnvPath = Assert-ExistingFile -Path $MinioCredentialEnvFile -Label 'MinIO credential environment file'
if ((Split-Path -Leaf $archivePath) -notmatch '^[A-Za-z0-9._-]+$') {
    throw "Data-files archive name contains unsupported shell characters: $archivePath"
}
$actualReferencesHash = (Get-FileHash -LiteralPath $referencesPath -Algorithm SHA256).Hash
if ($actualReferencesHash -ine $ReviewedReferencesHash) {
    throw 'Reviewed file-references.json hash does not match'
}
$endpoint = Get-MinIOEndpoint -Endpoint $MinioEndpoint
$manifestByKey = Get-ManifestMap -Manifest (Read-JsonArtifact -Path $manifestPath)
$references = @(Get-ReviewedKnowledgeReferences -Report (Read-JsonArtifact -Path $referencesPath) -ManifestByKey $manifestByKey -TargetBucketName $TargetBucket)
$artifactRoot = New-ArtifactDirectory -OutputRoot $OutputRoot -Prefix 'minio-file-import'
$stagingRoot = Join-Path $artifactRoot 'source'
$verifiedRoot = Join-Path $artifactRoot 'verified'
New-Item -ItemType Directory -Path $stagingRoot, $verifiedRoot | Out-Null

$archiveDirectory = Split-Path -Parent $archivePath
$archiveName = Split-Path -Leaf $archivePath
$archiveEntries = @(Invoke-CheckedOutput -FilePath 'docker' -Arguments @(
        'run', '--rm', '--mount', "type=bind,source=$archiveDirectory,target=/archive,readonly",
        'alpine:3.20', 'sh', '-c', "set -eu; tar -tzf /archive/$archiveName"
    ))
Assert-ArchiveEntries -Entries $archiveEntries
Invoke-Checked -FilePath 'docker' -Arguments @(
    'run', '--rm', '--mount', "type=bind,source=$archiveDirectory,target=/archive,readonly",
    '--mount', "type=bind,source=$stagingRoot,target=/staging",
    'alpine:3.20', 'sh', '-c', "set -eu; tar -xzf /archive/$archiveName -C /staging"
)
Assert-StagedFilesMatchManifest -StagingRoot $stagingRoot -ManifestByKey $manifestByKey

$transferManifestPath = Join-Path $artifactRoot 'object-transfer.tsv'
$tab = [string][char]9
$transferLines = @(
    foreach ($key in $manifestByKey.Keys | Sort-Object) {
        $file = $manifestByKey[$key]
        $key + $tab + [string]$file.sha256 + $tab + [string]$file.bytes
    }
)
[IO.File]::WriteAllText($transferManifestPath, (($transferLines -join [Environment]::NewLine) + [Environment]::NewLine), [Text.UTF8Encoding]::new($false))

$mcScript = @'
set -eu
[ -n "$MINIO_ACCESS_KEY_ID" ] || { echo "MINIO_ACCESS_KEY_ID is required" >&2; exit 41; }
[ -n "$MINIO_SECRET_ACCESS_KEY" ] || { echo "MINIO_SECRET_ACCESS_KEY is required" >&2; exit 42; }
mc alias set knavis "$KNAVIS_MINIO_ENDPOINT" "$MINIO_ACCESS_KEY_ID" "$MINIO_SECRET_ACCESS_KEY" >/dev/null
mc mb --ignore-existing "knavis/$KNAVIS_TARGET_BUCKET" >/dev/null
while IFS="$(printf '\t')" read -r key sha bytes; do
  [ -n "$key" ] || continue
  mc cp --quiet "/source/$key" "knavis/$KNAVIS_TARGET_BUCKET/$key"
  mkdir -p "$(dirname "/verified/$key")"
  mc cat "knavis/$KNAVIS_TARGET_BUCKET/$key" > "/verified/$key"
  actual="$(sha256sum "/verified/$key" | awk '{print $1}')"
  [ "$actual" = "$sha" ] || { echo "object hash mismatch: $key" >&2; exit 47; }
done < /input/object-transfer.tsv
'@
$mcArgs = @(
    'run', '--rm', '--env-file', $credentialEnvPath,
    '--entrypoint', '/bin/sh',
    '--mount', "type=bind,source=$stagingRoot,target=/source,readonly",
    '--mount', "type=bind,source=$verifiedRoot,target=/verified",
    '--mount', "type=bind,source=$artifactRoot,target=/input,readonly",
    '-e', "KNAVIS_MINIO_ENDPOINT=$endpoint", '-e', "KNAVIS_TARGET_BUCKET=$TargetBucket"
)
if ($MinioDockerNetwork) {
    $mcArgs += @('--network', $MinioDockerNetwork)
}
$mcArgs += @($MinioMcImage, 'sh', '-c', $mcScript)
Invoke-Checked -FilePath 'docker' -Arguments $mcArgs

$objects = New-Object System.Collections.Generic.List[object]
foreach ($key in $manifestByKey.Keys | Sort-Object) {
    $expected = $manifestByKey[$key]
    $verifiedPath = Join-Path $verifiedRoot ($key -replace '/', '\')
    if (-not (Test-Path -LiteralPath $verifiedPath -PathType Leaf)) {
        throw "MinIO verification download is missing: $key"
    }
    $actualHash = (Get-FileHash -LiteralPath $verifiedPath -Algorithm SHA256).Hash.ToLowerInvariant()
    if ($actualHash -ne [string]$expected.sha256) {
        throw "MinIO verification SHA-256 differs for: $key"
    }
    $objects.Add([pscustomobject]@{
            object_key = $key
            path       = "minio://$TargetBucket/$key"
            bytes      = [int64]$expected.bytes
            sha256     = [string]$expected.sha256
        })
}

$targetRows = @{}
$referenceSqlPath = Join-Path $artifactRoot 'reference-update.sql'
if ($ApplyReviewedReferences) {
    Assert-ContainerRunning -Name $PostgresContainer
    $targetRows = Get-TargetKnowledgeRows -Container $PostgresContainer -Database $DatabaseName -User $DatabaseUser -References $references
    $useSsl = ([Uri]$endpoint).Scheme -eq 'https'
    $referenceSql = New-ReferenceUpdateSql -References $references -TargetRows $targetRows -BucketName $TargetBucket -UseSsl $useSsl
    [IO.File]::WriteAllText($referenceSqlPath, (($referenceSql -join [Environment]::NewLine) + [Environment]::NewLine), [Text.UTF8Encoding]::new($false))
    $temporarySqlPath = '/tmp/knavis-reference-update-' + [Guid]::NewGuid().ToString('N') + '.sql'
    try {
        $containerSqlPath = '{0}:{1}' -f $PostgresContainer, $temporarySqlPath
        Invoke-Checked -FilePath 'docker' -Arguments @('cp', $referenceSqlPath, $containerSqlPath)
        Invoke-Checked -FilePath 'docker' -Arguments @(
            'exec', $PostgresContainer, 'psql', "--username=$DatabaseUser", "--dbname=$DatabaseName",
            '--no-password', '--set=ON_ERROR_STOP=1', "--file=$temporarySqlPath"
        )
    } finally {
        & docker exec $PostgresContainer rm -f $temporarySqlPath 1>$null 2>$null
    }
} else {
    @(
        '-- Database reference updates are intentionally disabled.',
        '-- Re-run with -ApplyReviewedReferences after reviewing reference-conversion.json.'
    ) | Set-Content -LiteralPath $referenceSqlPath -Encoding UTF8
}

$objectManifest = [ordered]@{
    generated_at                 = (Get-Date).ToUniversalTime().ToString('o')
    endpoint                     = $endpoint
    target_bucket                = $TargetBucket
    source_archive               = $archivePath
    source_archive_sha256        = (Get-FileHash -LiteralPath $archivePath -Algorithm SHA256).Hash
    reviewed_references          = $referencesPath
    reviewed_references_sha256   = $actualReferencesHash
    uploaded_object_count        = $objects.Count
    database_references_applied  = [bool]$ApplyReviewedReferences
    objects                      = @($objects)
}
$conversion = [ordered]@{
    generated_at  = (Get-Date).ToUniversalTime().ToString('o')
    source_report = $referencesPath
    conversions   = @($references)
}
Write-JsonArtifact -Value $objectManifest -Path (Join-Path $artifactRoot 'object-manifest.json') | Out-Null
Write-JsonArtifact -Value $conversion -Path (Join-Path $artifactRoot 'reference-conversion.json') | Out-Null
Copy-Item -LiteralPath $referencesPath -Destination (Join-Path $artifactRoot 'file-references.json')
Write-ArtifactSha256Sums -ArtifactRoot $artifactRoot | Out-Null
Write-Host "MinIO file import completed: $artifactRoot"
