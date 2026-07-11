param(
    [string]$SandboxImage = "wechatopenai/xelora-sandbox:latest"
)

$ErrorActionPreference = "Stop"

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$hostSkillDir = Join-Path $repoRoot "skills\preloaded\workspace-file-writer"
$hostOutputDir = Join-Path $hostSkillDir ".smoke-output"
$containerWorkDir = "/app/skills/preloaded/workspace-file-writer"

if (Test-Path $hostOutputDir) {
    Remove-Item -Recurse -Force $hostOutputDir
}
New-Item -ItemType Directory -Path $hostOutputDir | Out-Null

function Write-Utf8NoBom {
    param(
        [string]$Path,
        [string]$Content
    )

    [System.IO.File]::WriteAllText($Path, $Content, [System.Text.UTF8Encoding]::new($false))
}

function Invoke-Writer {
    param(
        [string]$RequestName,
        [string]$RequestJson
    )

    $requestHostPath = Join-Path $hostSkillDir $RequestName
    Write-Utf8NoBom -Path $requestHostPath -Content $RequestJson

    try {
        docker exec Xelora-app docker run --rm `
            --user 1000:1000 `
            --cap-drop ALL `
            --network none `
            --pids-limit 100 `
            --security-opt no-new-privileges `
            --volumes-from Xelora-app `
            -w $containerWorkDir `
            $SandboxImage `
            sh -c "python3 scripts/workspace_file_writer.py $RequestName"
    }
    finally {
        if (Test-Path $requestHostPath) {
            Remove-Item -Force $requestHostPath
        }
    }
}

Invoke-Writer -RequestName "smoke-write.json" -RequestJson @'
{
  "action": "write",
  "file": ".smoke-output/report.md",
  "content": "# Smoke Report\n\nCreated by workspace-file-writer.\n",
  "overwrite": true
}
'@

Invoke-Writer -RequestName "smoke-append.json" -RequestJson @'
{
  "action": "append",
  "file": ".smoke-output/report.md",
  "content": "\n## Status\n\n- ok\n"
}
'@

Invoke-Writer -RequestName "smoke-json.json" -RequestJson @'
{
  "action": "write_json",
  "file": ".smoke-output/summary.json",
  "data": {
    "project": "Xelora",
    "status": "ok"
  },
  "indent": 2
}
'@

$markdownPath = Join-Path $hostOutputDir "report.md"
$jsonPath = Join-Path $hostOutputDir "summary.json"

if (-not (Test-Path $markdownPath)) {
    throw "markdown artifact was not created"
}
if (-not (Test-Path $jsonPath)) {
    throw "json artifact was not created"
}

$markdownContent = Get-Content $markdownPath -Raw
if ($markdownContent -notmatch "Smoke Report" -or $markdownContent -notmatch "Status") {
    throw "markdown artifact content is incomplete"
}

$json = Get-Content $jsonPath -Raw | ConvertFrom-Json
if ($json.project -ne "Xelora" -or $json.status -ne "ok") {
    throw "json artifact content is invalid"
}

Write-Host "workspace-file-writer smoke passed"

Remove-Item -Recurse -Force $hostOutputDir
