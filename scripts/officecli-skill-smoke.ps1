param(
    [string]$AppContainer = "Xelora-app",
    [string]$SandboxImage = "wechatopenai/xelora-sandbox:latest"
)

$ErrorActionPreference = "Stop"

$repoRoot = Split-Path -Parent $PSScriptRoot
$hostSkillDir = Join-Path $repoRoot "skills\preloaded\officecli-document-editing"
$hostOutputDir = Join-Path $hostSkillDir ".smoke-output"

if (Test-Path $hostOutputDir) {
    Remove-Item $hostOutputDir -Recurse -Force
}
New-Item -ItemType Directory -Path $hostOutputDir | Out-Null

$requests = @(
    @{ name = "create-docx"; payload = '{"action":"create","file":".smoke-output/smoke.docx","force":true}' },
    @{ name = "create-xlsx"; payload = '{"action":"create","file":".smoke-output/smoke.xlsx","force":true}' },
    @{ name = "create-pptx"; payload = '{"action":"create","file":".smoke-output/smoke.pptx","force":true}' },
    @{ name = "add-docx-paragraph"; payload = '{"action":"add","file":".smoke-output/smoke.docx","parent":"/body","type":"paragraph","props":{"text":"Hello Xelora","style":"Heading1"}}' },
    @{ name = "add-xlsx-cell"; payload = '{"action":"add","file":".smoke-output/smoke.xlsx","parent":"/Sheet1","type":"cell","props":{"ref":"A1","value":"Revenue","bold":"true"}}' },
    @{ name = "batch-pptx-slide-title"; payload = '{"action":"batch","file":".smoke-output/smoke.pptx","commands":[{"command":"add","parent":"/","type":"slide"},{"command":"add","parent":"/slide[1]","type":"shape","props":{"text":"Hello Xelora","x":"1cm","y":"1cm","width":"8cm","height":"2cm","size":"24pt"}}]}' },
    @{ name = "view-docx-text"; payload = '{"action":"view","file":".smoke-output/smoke.docx","mode":"text","max_lines":20}' },
    @{ name = "view-xlsx-text"; payload = '{"action":"view","file":".smoke-output/smoke.xlsx","mode":"text","max_lines":20}' },
    @{ name = "view-pptx-text"; payload = '{"action":"view","file":".smoke-output/smoke.pptx","mode":"text","max_lines":20}' },
    @{ name = "validate-docx"; payload = '{"action":"validate","file":".smoke-output/smoke.docx"}' },
    @{ name = "validate-xlsx"; payload = '{"action":"validate","file":".smoke-output/smoke.xlsx"}' },
    @{ name = "validate-pptx"; payload = '{"action":"validate","file":".smoke-output/smoke.pptx"}' }
)

foreach ($request in $requests) {
    $requestPath = ".smoke-output/$($request.name).json"
    $hostRequestPath = Join-Path $hostSkillDir ($requestPath -replace '/', '\')
    [System.IO.File]::WriteAllText(
        $hostRequestPath,
        $request.payload,
        [System.Text.UTF8Encoding]::new($false)
    )
    docker exec $AppContainer docker run --rm `
        --user 1000:1000 `
        --cap-drop ALL `
        --network none `
        --pids-limit 100 `
        --security-opt no-new-privileges `
        --volumes-from $AppContainer `
        -e OFFICECLI_SKIP_UPDATE=1 `
        -e OFFICECLI_RESIDENT_FLUSH=each `
        -w /app/skills/preloaded/officecli-document-editing `
        $SandboxImage `
        sh -c "python3 scripts/officecli_bridge.py $requestPath"
}

$expectedFiles = @("smoke.docx", "smoke.xlsx", "smoke.pptx")
foreach ($file in $expectedFiles) {
    $path = Join-Path $hostOutputDir $file
    if (!(Test-Path $path)) {
        throw "OfficeCLI skill smoke file was not created on host: $path"
    }
}

Get-ChildItem $hostOutputDir | Select-Object Name,Length
Remove-Item $hostOutputDir -Recurse -Force
Write-Host "officecli skill smoke passed"
