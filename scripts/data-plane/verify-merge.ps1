[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string]$PostgresContainer,
    [Parameter(Mandatory = $true)]
    [string]$DatabaseName,
    [Parameter(Mandatory = $true)]
    [string]$DatabaseUser,
    [Parameter(Mandatory = $true)]
    [string]$ObjectManifest,
    [Parameter(Mandatory = $true)]
    [string]$MinioCredentialEnvFile,
    [Parameter(Mandatory = $true)]
    [string]$MinioEndpoint,
    [Parameter(Mandatory = $true)]
    [ValidatePattern('^[a-z0-9][a-z0-9.-]{1,61}[a-z0-9]$')]
    [string]$TargetBucket,
    [Parameter(Mandatory = $true)]
    [string]$OutputRoot,
    [string]$MergePlanDir,
    [string]$ExpectedWorkspaceManifest,
    [string]$TargetWorkspaceRoot,
    [string]$ApplicationSmokeReport,
    [string]$MinioDockerNetwork,
    [string]$MinioMcImage = 'minio/minio:RELEASE.2025-09-07T16-13-09Z',
    [switch]$RequireApplicationSmoke
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

function Get-ObjectKeyFromMinioPath {
    param([Parameter(Mandatory = $true)][string]$Path, [Parameter(Mandatory = $true)][string]$BucketName)
    $prefix = "minio://$BucketName/"
    if (-not $Path.StartsWith($prefix)) {
        throw "Knowledge file path is outside the verified MinIO bucket: $Path"
    }
    return Assert-RelativeStorageKey -Key $Path.Substring($prefix.Length)
}

function Get-ExpectedWorkspaceFiles {
    param([Parameter(Mandatory = $true)][string]$ManifestPath)
    $document = Read-JsonArtifact -Path $ManifestPath
    $filesProperty = $document.PSObject.Properties['files']
    if ($null -eq $filesProperty) {
        throw "Workspace manifest is missing files: $ManifestPath"
    }
    $map = @{}
    foreach ($file in @($filesProperty.Value)) {
        $key = Assert-RelativeStorageKey -Key ([string]$file.relative_path)
        $map[$key] = [pscustomobject]@{
            bytes  = [int64]$file.bytes
            sha256 = ([string]$file.sha256).ToLowerInvariant()
        }
    }
    return $map
}

Assert-ContainerRunning -Name $PostgresContainer
$objectManifestPath = Assert-ExistingFile -Path $ObjectManifest -Label 'object-manifest.json'
$credentialEnvPath = Assert-ExistingFile -Path $MinioCredentialEnvFile -Label 'MinIO credential environment file'
if (($ExpectedWorkspaceManifest -and -not $TargetWorkspaceRoot) -or ($TargetWorkspaceRoot -and -not $ExpectedWorkspaceManifest)) {
    throw 'Workspace verification requires both ExpectedWorkspaceManifest and TargetWorkspaceRoot'
}
$endpoint = Get-MinIOEndpoint -Endpoint $MinioEndpoint
$artifactRoot = New-ArtifactDirectory -OutputRoot $OutputRoot -Prefix 'merge-verification'
$failures = New-Object System.Collections.Generic.List[string]

$foreignKeyCheck = @(Invoke-PostgresCsv -PostgresContainer $PostgresContainer -DatabaseName $DatabaseName -DatabaseUser $DatabaseUser -Sql @'
SELECT conname
FROM pg_constraint
WHERE contype = 'f' AND convalidated = false
ORDER BY conname
'@ -Headers @('conname'))
if ($foreignKeyCheck.Count -gt 0) {
    $failures.Add("Unvalidated foreign keys: $($foreignKeyCheck.conname -join ', ')")
}
$migrationDirty = @(Invoke-PostgresCsv -PostgresContainer $PostgresContainer -DatabaseName $DatabaseName -DatabaseUser $DatabaseUser -Sql @'
SELECT version::text, dirty::text
FROM schema_migrations
WHERE dirty = true
ORDER BY version
'@ -Headers @('version', 'dirty'))
if ($migrationDirty.Count -gt 0) {
    $failures.Add("Dirty migration versions: $($migrationDirty.version -join ', ')")
}
$unresolvedLocal = @(Invoke-PostgresCsv -PostgresContainer $PostgresContainer -DatabaseName $DatabaseName -DatabaseUser $DatabaseUser -Sql @'
SELECT id, file_path
FROM knowledges
WHERE deleted_at IS NULL
  AND (file_path LIKE 'local://%' OR file_path LIKE '/data/files/%')
ORDER BY id
'@ -Headers @('id', 'file_path'))
if ($unresolvedLocal.Count -gt 0) {
    $failures.Add("Unresolved local knowledge paths: $($unresolvedLocal.id -join ', ')")
}

$objectManifestDocument = Read-JsonArtifact -Path $objectManifestPath
$objectsProperty = $objectManifestDocument.PSObject.Properties['objects']
if ($null -eq $objectsProperty) {
    throw 'object-manifest.json is missing its objects array'
}
$expectedObjects = @{}
foreach ($object in @($objectsProperty.Value)) {
    $key = Assert-RelativeStorageKey -Key ([string]$object.object_key)
    $hash = ([string]$object.sha256).ToLowerInvariant()
    if ($hash -notmatch '^[a-f0-9]{64}$') {
        throw "object-manifest.json has an invalid SHA-256 for: $key"
    }
    $expectedObjects[$key] = [pscustomobject]@{
        bytes  = [int64]$object.bytes
        sha256 = $hash
        source = 'object-manifest'
    }
}

$retainedPaths = @(Invoke-PostgresCsv -PostgresContainer $PostgresContainer -DatabaseName $DatabaseName -DatabaseUser $DatabaseUser -Sql @'
SELECT id, file_path, COALESCE(file_hash, '') AS file_hash
FROM knowledges
WHERE deleted_at IS NULL
  AND file_path LIKE 'minio://%'
ORDER BY id
'@ -Headers @('id', 'file_path', 'file_hash'))
foreach ($row in $retainedPaths) {
    try {
        $key = Get-ObjectKeyFromMinioPath -Path ([string]$row.file_path) -BucketName $TargetBucket
        $rowHash = ([string]$row.file_hash).ToLowerInvariant()
        if ($rowHash -notmatch '^[a-f0-9]{64}$') {
            if (-not $expectedObjects.ContainsKey($key)) {
                throw "Knowledge record has no SHA-256 and is absent from object-manifest.json: $($row.id)"
            }
            $rowHash = [string]$expectedObjects[$key].sha256
        }
        if ($expectedObjects.ContainsKey($key) -and [string]$expectedObjects[$key].sha256 -ne $rowHash) {
            throw "Knowledge file_hash differs from object-manifest.json: $($row.id)"
        }
        $expectedBytes = -1
        if ($expectedObjects.ContainsKey($key)) {
            $expectedBytes = [int64]$expectedObjects[$key].bytes
        }
        $expectedObjects[$key] = [pscustomobject]@{
            bytes  = $expectedBytes
            sha256 = $rowHash
            source = 'database'
        }
    } catch {
        $failures.Add($_.Exception.Message)
    }
}

$verifiedRoot = Join-Path $artifactRoot 'objects'
New-Item -ItemType Directory -Path $verifiedRoot | Out-Null
$objectListPath = Join-Path $artifactRoot 'object-list.tsv'
$tab = [string][char]9
$objectLines = @(
    foreach ($key in $expectedObjects.Keys | Sort-Object) {
        $key + $tab + [string]$expectedObjects[$key].sha256
    }
)
[IO.File]::WriteAllText($objectListPath, (($objectLines -join [Environment]::NewLine) + [Environment]::NewLine), [Text.UTF8Encoding]::new($false))

$mcScript = @'
set -eu
[ -n "$MINIO_ACCESS_KEY_ID" ] || { echo "MINIO_ACCESS_KEY_ID is required" >&2; exit 41; }
[ -n "$MINIO_SECRET_ACCESS_KEY" ] || { echo "MINIO_SECRET_ACCESS_KEY is required" >&2; exit 42; }
mc alias set knavis "$KNAVIS_MINIO_ENDPOINT" "$MINIO_ACCESS_KEY_ID" "$MINIO_SECRET_ACCESS_KEY" >/dev/null
while IFS="$(printf '\t')" read -r key sha; do
  [ -n "$key" ] || continue
  mkdir -p "$(dirname "/verified/$key")"
  mc cat "knavis/$KNAVIS_TARGET_BUCKET/$key" > "/verified/$key"
  actual="$(sha256sum "/verified/$key" | awk '{print $1}')"
  [ "$actual" = "$sha" ] || { echo "object hash mismatch: $key" >&2; exit 47; }
done < /input/object-list.tsv
'@
$mcArgs = @(
    'run', '--rm', '--env-file', $credentialEnvPath,
    '--entrypoint', '/bin/sh',
    '--mount', "type=bind,source=$artifactRoot,target=/input,readonly",
    '--mount', "type=bind,source=$verifiedRoot,target=/verified",
    '-e', "KNAVIS_MINIO_ENDPOINT=$endpoint", '-e', "KNAVIS_TARGET_BUCKET=$TargetBucket"
)
if ($MinioDockerNetwork) {
    $mcArgs += @('--network', $MinioDockerNetwork)
}
$mcArgs += @($MinioMcImage, 'sh', '-c', $mcScript)
try {
    Invoke-Checked -FilePath 'docker' -Arguments $mcArgs
} catch {
    $failures.Add("MinIO object verification failed: $($_.Exception.Message)")
}

$objectResults = New-Object System.Collections.Generic.List[object]
foreach ($key in $expectedObjects.Keys | Sort-Object) {
    $path = Join-Path $verifiedRoot ($key -replace '/', '\')
    if (-not (Test-Path -LiteralPath $path -PathType Leaf)) {
        $failures.Add("Verified MinIO object is missing locally: $key")
        continue
    }
    $actualHash = (Get-FileHash -LiteralPath $path -Algorithm SHA256).Hash.ToLowerInvariant()
    $expected = $expectedObjects[$key]
    if ($actualHash -ne [string]$expected.sha256) {
        $failures.Add("Verified MinIO object SHA-256 differs: $key")
    }
    $actualBytes = [int64](Get-Item -LiteralPath $path).Length
    if ([int64]$expected.bytes -ge 0 -and $actualBytes -ne [int64]$expected.bytes) {
        $failures.Add("Verified MinIO object byte count differs: $key")
    }
    $objectResults.Add([pscustomobject]@{
            object_key = $key
            sha256     = $actualHash
            bytes      = $actualBytes
            source     = [string]$expected.source
        })
}

$rowCounts = New-Object System.Collections.Generic.List[object]
if ($MergePlanDir) {
    $mergePlanPath = Assert-ExistingFile -Path (Join-Path $MergePlanDir 'merge-plan.json') -Label 'merge-plan.json'
    $mergePlan = Read-JsonArtifact -Path $mergePlanPath
    foreach ($table in @($mergePlan.tables | Where-Object { [string]$_.status -eq 'supported' })) {
        $name = [string]$table.table
        $expectedCount = [int64]$table.server_rows + [int64]$table.local_only
        $actualCount = [int64](Invoke-PostgresScalar -PostgresContainer $PostgresContainer -DatabaseName $DatabaseName -DatabaseUser $DatabaseUser -Sql "SELECT count(*)::bigint FROM $(ConvertTo-PgIdentifier -Name $name)")
        $rowCounts.Add([pscustomobject]@{ table = $name; expected = $expectedCount; actual = $actualCount })
        if ($actualCount -ne $expectedCount) {
            $failures.Add("Merged row count differs for $($name): expected=$expectedCount actual=$actualCount")
        }
    }
}

$workspaceResult = $null
if ($ExpectedWorkspaceManifest) {
    $expectedWorkspace = Get-ExpectedWorkspaceFiles -ManifestPath (Assert-ExistingFile -Path $ExpectedWorkspaceManifest -Label 'Workspace manifest')
    $actualWorkspace = @{}
    foreach ($file in @(Get-WorkspaceManifest -Root (Assert-ExistingDirectory -Path $TargetWorkspaceRoot -Label 'Target workspace'))) {
        $actualWorkspace[[string]$file.relative_path] = $file
    }
    $missing = @($expectedWorkspace.Keys | Where-Object { -not $actualWorkspace.ContainsKey($_) })
    $changed = @($expectedWorkspace.Keys | Where-Object {
            $actualWorkspace.ContainsKey($_) -and (([string]$actualWorkspace[$_].sha256).ToLowerInvariant() -ne [string]$expectedWorkspace[$_].sha256 -or [int64]$actualWorkspace[$_].bytes -ne [int64]$expectedWorkspace[$_].bytes)
        })
    $workspaceResult = [ordered]@{
        expected_count = $expectedWorkspace.Count
        actual_count   = $actualWorkspace.Count
        missing        = $missing
        changed        = $changed
    }
    if ($missing.Count -gt 0 -or $changed.Count -gt 0) {
        $failures.Add('Workspace manifest does not match the expected snapshot')
    }
}

$applicationSmoke = [ordered]@{ status = 'not-supplied' }
if ($ApplicationSmokeReport) {
    $smokePath = Assert-ExistingFile -Path $ApplicationSmokeReport -Label 'Application smoke report'
    $applicationSmoke = Read-JsonArtifact -Path $smokePath
    if ([string]$applicationSmoke.status -notin @('success', 'passed')) {
        $failures.Add('Application smoke report is not successful')
    }
} elseif ($RequireApplicationSmoke) {
    $failures.Add('Application smoke report is required')
}

$verification = [ordered]@{
    generated_at           = (Get-Date).ToUniversalTime().ToString('o')
    postgres_container     = $PostgresContainer
    database_name          = $DatabaseName
    target_bucket          = $TargetBucket
    foreign_key_check      = @($foreignKeyCheck)
    dirty_migrations       = @($migrationDirty)
    unresolved_local_paths = @($unresolvedLocal)
    object_manifest        = $objectManifestPath
    objects                = @($objectResults)
    row_counts             = @($rowCounts)
    workspace              = $workspaceResult
    application_smoke      = $applicationSmoke
    failures               = @($failures)
    status                 = if ($failures.Count -eq 0) { 'success' } else { 'failed' }
}
Write-JsonArtifact -Value $verification -Path (Join-Path $artifactRoot 'verification.json') | Out-Null
Copy-Item -LiteralPath $objectManifestPath -Destination (Join-Path $artifactRoot 'object-manifest.json')
$summary = @(
    "status=$($verification.status)",
    "unvalidated_foreign_keys=$($foreignKeyCheck.Count)",
    "dirty_migrations=$($migrationDirty.Count)",
    "unresolved_local_paths=$($unresolvedLocal.Count)",
    "verified_objects=$($objectResults.Count)",
    "failures=$($failures.Count)"
)
[IO.File]::WriteAllText((Join-Path $artifactRoot 'verification-summary.txt'), (($summary -join [Environment]::NewLine) + [Environment]::NewLine), [Text.UTF8Encoding]::new($false))
Write-ArtifactSha256Sums -ArtifactRoot $artifactRoot | Out-Null
if ($failures.Count -gt 0) {
    throw "Merge verification failed. See: $artifactRoot"
}
Write-Host "Merge verification passed: $artifactRoot"
