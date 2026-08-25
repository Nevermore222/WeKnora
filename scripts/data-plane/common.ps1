Set-StrictMode -Version Latest

function Invoke-Checked {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory = $true)]
        [string]$FilePath,
        [string[]]$Arguments = @()
    )

    & $FilePath @Arguments
    if ($LASTEXITCODE -ne 0) {
        throw "$FilePath failed with exit code $LASTEXITCODE"
    }
}

function Invoke-CheckedOutput {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory = $true)]
        [string]$FilePath,
        [string[]]$Arguments = @()
    )

    $output = & $FilePath @Arguments
    if ($LASTEXITCODE -ne 0) {
        throw "$FilePath failed with exit code $LASTEXITCODE"
    }
    return @($output)
}

function Assert-ExistingDirectory {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory = $true)]
        [string]$Path,
        [Parameter(Mandatory = $true)]
        [string]$Label
    )

    if (-not (Test-Path -LiteralPath $Path -PathType Container)) {
        throw "$Label directory does not exist: $Path"
    }
    return (Resolve-Path -LiteralPath $Path).Path
}

function New-ArtifactDirectory {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory = $true)]
        [string]$OutputRoot,
        [Parameter(Mandatory = $true)]
        [string]$Prefix
    )

    $root = if (Test-Path -LiteralPath $OutputRoot -PathType Container) {
        (Resolve-Path -LiteralPath $OutputRoot).Path
    } else {
        (New-Item -ItemType Directory -Path $OutputRoot -Force).FullName
    }
    if ($Prefix -notmatch '^[A-Za-z0-9_-]+$') {
        throw "Artifact prefix contains unsupported characters: $Prefix"
    }

    $stamp = Get-Date -Format 'yyyyMMdd-HHmmss'
    $path = Join-Path $root "$Prefix-$stamp-$([Guid]::NewGuid().ToString('N'))"
    if (Test-Path -LiteralPath $path) {
        throw "Refusing to reuse an existing artifact directory: $path"
    }
    return (New-Item -ItemType Directory -Path $path).FullName
}

function Write-ArtifactHash {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory = $true)]
        [string]$Path
    )

    $item = Get-Item -LiteralPath $Path -ErrorAction Stop
    if ($item.PSIsContainer) {
        throw "Hash target must be a file: $Path"
    }
    $hash = Get-FileHash -LiteralPath $Path -Algorithm SHA256
    [pscustomobject]@{
        path   = $Path
        sha256 = $hash.Hash
        bytes  = $item.Length
    }
}

function Write-JsonArtifact {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory = $true)]
        [object]$Value,
        [Parameter(Mandatory = $true)]
        [string]$Path
    )

    $Value | ConvertTo-Json -Depth 30 | Set-Content -LiteralPath $Path -Encoding UTF8
    return Write-ArtifactHash -Path $Path
}

function Write-ArtifactSha256Sums {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory = $true)]
        [string]$ArtifactRoot
    )

    $root = (Resolve-Path -LiteralPath $ArtifactRoot).Path
    $manifestPath = Join-Path $root 'SHA256SUMS.txt'
    $lines = foreach ($file in Get-ChildItem -LiteralPath $root -File -Recurse -Force | Sort-Object FullName) {
        if ($file.FullName -eq $manifestPath) {
            continue
        }
        $relative = $file.FullName.Substring($root.Length).TrimStart('\', '/') -replace '\\', '/'
        $hash = (Get-FileHash -LiteralPath $file.FullName -Algorithm SHA256).Hash.ToLowerInvariant()
        "$hash  $relative"
    }
    $lines | Set-Content -LiteralPath $manifestPath -Encoding ASCII
    return Write-ArtifactHash -Path $manifestPath
}

function Assert-DockerObject {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory = $true)]
        [string]$Name,
        [Parameter(Mandatory = $true)]
        [string]$Label
    )

    & docker inspect $Name 1>$null 2>$null
    if ($LASTEXITCODE -ne 0) {
        throw "$Label was not found: $Name"
    }
}

function Assert-ContainerRunning {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory = $true)]
        [string]$Name
    )

    Assert-DockerObject -Name $Name -Label 'PostgreSQL container'
    $running = (& docker inspect --format '{{.State.Running}}' $Name 2>$null) -join ''
    if ($LASTEXITCODE -ne 0 -or $running -ne 'true') {
        throw "PostgreSQL container is not running: $Name"
    }
}

function Assert-ContainersStopped {
    [CmdletBinding()]
    param(
        [string[]]$Names = @()
    )

    foreach ($name in $Names) {
        $exists = & docker inspect $name 1>$null 2>$null
        if ($LASTEXITCODE -ne 0) {
            continue
        }
        $running = (& docker inspect --format '{{.State.Running}}' $name 2>$null) -join ''
        if ($running -eq 'true') {
            throw "Application container must be stopped before snapshot: $name"
        }
    }
}

function Assert-DockerVolume {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory = $true)]
        [string]$Name
    )

    Assert-DockerObject -Name $Name -Label 'Docker data-files volume'
}

function Invoke-PostgresScalar {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory = $true)]
        [string]$PostgresContainer,
        [Parameter(Mandatory = $true)]
        [string]$DatabaseName,
        [Parameter(Mandatory = $true)]
        [string]$DatabaseUser,
        [Parameter(Mandatory = $true)]
        [string]$Sql
    )

    $args = @(
        'exec', $PostgresContainer, 'psql',
        "--username=$DatabaseUser", "--dbname=$DatabaseName", '--no-password',
        '--tuples-only', '--no-align', '--quiet', "--command=$Sql"
    )
    $output = Invoke-CheckedOutput -FilePath 'docker' -Arguments $args
    return (($output -join "`n").Trim())
}

function Invoke-PostgresCsv {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory = $true)]
        [string]$PostgresContainer,
        [Parameter(Mandatory = $true)]
        [string]$DatabaseName,
        [Parameter(Mandatory = $true)]
        [string]$DatabaseUser,
        [Parameter(Mandatory = $true)]
        [string]$Sql,
        [Parameter(Mandatory = $true)]
        [string[]]$Headers
    )

    $args = @(
        'exec', $PostgresContainer, 'psql',
        "--username=$DatabaseUser", "--dbname=$DatabaseName", '--no-password',
        '--tuples-only', '--csv', '--quiet', "--command=$Sql"
    )
    $output = Invoke-CheckedOutput -FilePath 'docker' -Arguments $args
    $text = ($output -join "`n").Trim()
    if ([string]::IsNullOrWhiteSpace($text)) {
        return @()
    }
    return @(ConvertFrom-Csv -InputObject $text -Header $Headers)
}

function ConvertTo-PgIdentifier {
    param([Parameter(Mandatory = $true)][string]$Name)
    return '"' + $Name.Replace('"', '""') + '"'
}

function Get-Sha256Hex {
    param([Parameter(Mandatory = $true)][byte[]]$Bytes)
    $sha256 = [Security.Cryptography.SHA256]::Create()
    try {
        return ([BitConverter]::ToString($sha256.ComputeHash($Bytes))).Replace('-', '').ToLowerInvariant()
    } finally {
        $sha256.Dispose()
    }
}

function Get-WorkspaceManifest {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory = $true)]
        [string]$Root
    )

    $resolvedRoot = Assert-ExistingDirectory -Path $Root -Label 'Workspace'
    $allItems = @(Get-ChildItem -LiteralPath $resolvedRoot -Recurse -Force -ErrorAction Stop)
    $reparse = @($allItems | Where-Object { $_.Attributes -band [IO.FileAttributes]::ReparsePoint })
    if ($reparse.Count -gt 0) {
        throw "Workspace contains reparse points; refusing to archive outside-root content: $($reparse[0].FullName)"
    }

    $files = @($allItems | Where-Object { -not $_.PSIsContainer })
    return @(
        foreach ($file in $files) {
            $relative = $file.FullName.Substring($resolvedRoot.Length).TrimStart('\', '/') -replace '\\', '/'
            [pscustomobject]@{
                relative_path = $relative
                bytes         = $file.Length
                sha256        = (Get-FileHash -LiteralPath $file.FullName -Algorithm SHA256).Hash.ToLowerInvariant()
            }
        }
    )
}

function Get-DataFilesVolumeManifest {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory = $true)]
        [string]$VolumeName
    )

    $script = @'
set -eu
find /scan -type f -exec sh -c '
for file do
  relative="${file#/scan/}"
  if printf "%s" "$relative" | LC_ALL=C grep -q "[[:cntrl:]]"; then
    exit 17
  fi
  bytes="$(stat -c %s "$file")"
  hash="$(sha256sum "$file" | cut -d " " -f 1)"
  printf "%s\t%s\t%s\n" "$relative" "$bytes" "$hash"
done
' sh {} +
'@
    $encodedScript = [Convert]::ToBase64String([Text.Encoding]::UTF8.GetBytes($script))
    $command = "printf %s $encodedScript | base64 -d | sh"
    $args = @(
        'run', '--rm',
        '--mount', "type=volume,source=$VolumeName,target=/scan,readonly",
        'alpine:3.20', 'sh', '-c', $command
    )
    $lines = Invoke-CheckedOutput -FilePath 'docker' -Arguments $args
    $manifest = foreach ($line in $lines) {
        if ([string]::IsNullOrWhiteSpace($line)) {
            continue
        }
        $parts = $line -split "`t", 3
        if ($parts.Count -ne 3) {
            throw 'Data-files manifest returned an invalid record'
        }
        [pscustomobject]@{
            relative_path = $parts[0]
            bytes         = [int64]$parts[1]
            sha256        = $parts[2].ToLowerInvariant()
        }
    }
    return @($manifest)
}

function Assert-RelativeStorageKey {
    param([Parameter(Mandatory = $true)][string]$Key)
    if ([string]::IsNullOrWhiteSpace($Key) -or $Key.StartsWith('/') -or $Key -match '^[A-Za-z]:') {
        throw "Storage key is not relative: $Key"
    }
    if ($Key -match '[\r\n\t\x00]') {
        throw "Storage key contains a control character: $Key"
    }
    $normalized = $Key.Replace('\', '/')
    if ($normalized.Split('/') -contains '..') {
        throw "Storage key escapes its root: $Key"
    }
    return $normalized.TrimStart('./')
}
