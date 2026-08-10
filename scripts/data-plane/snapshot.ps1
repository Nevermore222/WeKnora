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
    [string]$PreflightDir,
    [string]$EnvFile,
    [string]$ComposeFile,
    [string[]]$ApplicationContainers = @(
        'Xelora-app', 'Xelora-frontend', 'Xelora-docreader', 'Xelora-sandbox',
        'Xelora-worker', 'Xelora-asynq-worker', 'Knavis-app', 'Knavis-frontend',
        'Knavis-passive-app', 'Knavis-passive-frontend'
    ),
    [switch]$MaintenanceApproved
)

$ErrorActionPreference = 'Stop'
. (Join-Path $PSScriptRoot 'common.ps1')

if (-not $MaintenanceApproved) {
    throw 'Pass -MaintenanceApproved only after stopping both application nodes'
}

if ($PreflightDir) {
    $preflightMarker = Join-Path $PreflightDir 'preflight.ok'
    if (-not (Test-Path -LiteralPath $preflightMarker -PathType Leaf)) {
        throw "A successful preflight marker is required: $preflightMarker"
    }
}

Assert-ContainerRunning -Name $PostgresContainer
Assert-DockerVolume -Name $DataFilesVolume
$resolvedWorkspaceRoot = Assert-ExistingDirectory -Path $WorkspaceRoot -Label 'Workspace'
Assert-ContainersStopped -Names $ApplicationContainers
$artifactRoot = New-ArtifactDirectory -OutputRoot $OutputRoot -Prefix "snapshot-$SourceName"

if ($PreflightDir) {
    foreach ($reportName in @(
            'preflight.json', 'database-schema.json', 'file-references.json',
            'data-files-manifest.json', 'workspace-manifest.json', 'SHA256SUMS.txt'
        )) {
        $reportPath = Join-Path $PreflightDir $reportName
        if (-not (Test-Path -LiteralPath $reportPath -PathType Leaf)) {
            throw "Preflight report is missing: $reportPath"
        }
        $destinationName = if ($reportName -eq 'SHA256SUMS.txt') { 'preflight-SHA256SUMS.txt' } else { $reportName }
        Copy-Item -LiteralPath $reportPath -Destination (Join-Path $artifactRoot $destinationName)
    }
}

$dumpName = "$SourceName-$([Guid]::NewGuid().ToString('N')).dump"
$dumpInContainer = "/tmp/$dumpName"
$dumpPath = Join-Path $artifactRoot "$SourceName.dump"
$dumpCreated = $false

try {
    # The archive is created through PostgreSQL's logical export interface; no
    # Docker volume or VM disk is copied.
    Invoke-Checked -FilePath 'docker' -Arguments @(
        'exec', $PostgresContainer, 'pg_dump', "--username=$DatabaseUser",
        "--dbname=$DatabaseName", '--format=custom', '--no-owner', '--no-privileges',
        "--file=$dumpInContainer"
    )
    $dumpCreated = $true
    Invoke-Checked -FilePath 'docker' -Arguments @('cp', "$PostgresContainer`:$dumpInContainer", $dumpPath)

    $dataFilesArchive = Join-Path $artifactRoot 'data-files.tar.gz'
    Invoke-Checked -FilePath 'docker' -Arguments @(
        'run', '--rm',
        '--mount', "type=volume,source=$DataFilesVolume,target=/scan,readonly",
        '--mount', "type=bind,source=$artifactRoot,target=/out",
        'alpine:3.20', 'sh', '-c',
        'set -eu; tar -C /scan -czf /out/data-files.tar.gz .'
    )

    $workspaceManifest = @(Get-WorkspaceManifest -Root $resolvedWorkspaceRoot)
    Write-JsonArtifact -Value ([ordered]@{
            source_name = $SourceName
            root        = $resolvedWorkspaceRoot
            files       = $workspaceManifest
        }) -Path (Join-Path $artifactRoot 'workspace-manifest.json') | Out-Null

    Invoke-Checked -FilePath 'docker' -Arguments @(
        'run', '--rm',
        '--mount', "type=bind,source=$resolvedWorkspaceRoot,target=/scan,readonly",
        '--mount', "type=bind,source=$artifactRoot,target=/out",
        'alpine:3.20', 'sh', '-c',
        'set -eu; tar -C /scan -czf /out/workspace.tar.gz .'
    )

    $imageNames = @()
    if ($ComposeFile) {
        $composePath = (Resolve-Path -LiteralPath $ComposeFile).Path
        $composeArgs = @('-f', $composePath, 'config', '--images')
        if ($EnvFile) {
            $composeArgs = @('--env-file', (Resolve-Path -LiteralPath $EnvFile).Path) + $composeArgs
        }
        $imageNames = @(Invoke-CheckedOutput -FilePath 'docker' -Arguments (@('compose') + $composeArgs) | ForEach-Object { [string]$_ } | Where-Object { $_ })
    }
    $imageInventory = @(
        foreach ($image in $imageNames | Sort-Object -Unique) {
            $inspect = @(& docker image inspect --format '{{.Id}}|{{join .RepoDigests ","}}' $image 2>$null)
            if ($LASTEXITCODE -ne 0) {
                [pscustomobject]@{ image = $image; status = 'not-present' }
                continue
            }
            $parts = ([string]($inspect -join '')).Split('|', 2)
            [pscustomobject]@{
                image       = $image
                image_id    = $parts[0]
                repo_digest = if ($parts.Count -gt 1) { $parts[1] } else { '' }
            }
        }
    )
    Write-JsonArtifact -Value ([ordered]@{
            source_name = $SourceName
            compose_file = $ComposeFile
            env_file = $EnvFile
            images = $imageInventory
        }) -Path (Join-Path $artifactRoot 'image-inventory.json') | Out-Null

    $workspaceArchive = Join-Path $artifactRoot 'workspace.tar.gz'
    $archiveHashes = @(
        [pscustomobject]@{
            path   = $dumpPath
            sha256 = (Get-FileHash -LiteralPath $dumpPath -Algorithm SHA256).Hash
            bytes  = (Get-Item -LiteralPath $dumpPath).Length
        }
        [pscustomobject]@{
            path   = $dataFilesArchive
            sha256 = (Get-FileHash -LiteralPath $dataFilesArchive -Algorithm SHA256).Hash
            bytes  = (Get-Item -LiteralPath $dataFilesArchive).Length
        }
        [pscustomobject]@{
            path   = $workspaceArchive
            sha256 = (Get-FileHash -LiteralPath $workspaceArchive -Algorithm SHA256).Hash
            bytes  = (Get-Item -LiteralPath $workspaceArchive).Length
        }
    )
    $snapshot = [ordered]@{
        source_name        = $SourceName
        generated_at       = (Get-Date).ToUniversalTime().ToString('o')
        postgres_container = $PostgresContainer
        database_name      = $DatabaseName
        data_files_volume  = $DataFilesVolume
        workspace_root     = $resolvedWorkspaceRoot
        maintenance_window = $true
        archives           = $archiveHashes
        status             = 'success'
    }
    Write-JsonArtifact -Value $snapshot -Path (Join-Path $artifactRoot 'snapshot.json') | Out-Null
    Write-ArtifactSha256Sums -ArtifactRoot $artifactRoot | Out-Null
    Write-Host "Snapshot passed: $artifactRoot"
} finally {
    if ($dumpCreated) {
        & docker exec $PostgresContainer rm -f $dumpInContainer 1>$null 2>$null
    }
}
