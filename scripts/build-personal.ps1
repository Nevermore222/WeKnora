# build-personal.ps1 — Build Xelora Personal desktop EXE for Windows amd64
# Usage: .\scripts\build-personal.ps1 [-BrandProfile public|internal] [-SkipFrontend] [-DefaultServerURL <url>] [-DefaultServerName <name>] [-OutputDir <path>]
#
# Prerequisites:
#   - Go 1.26+ with CGO enabled (gcc / MSVC build tools)
#   - Wails v2 CLI matching go.mod (`go install github.com/wailsapp/wails/v2/cmd/wails@v2.12.0`)
#   - Node.js 18+ and npm (for frontend build)
#   - sqlite3 header on the CGO include path (for sqlite-vec CGO bindings):
#       copy mattn/go-sqlite3's sqlite3-binding.h to <dir>/sqlite3.h, then
#       `go env -w CGO_CFLAGS=-I<dir>`
#
# Notes:
#   - Bindings generation is skipped (-skipbindings). The Wails "new Go
#     WebView2Loader" crashes during bindings generation on some Windows
#     environments (exit 0xc0000139, see wailsapp/wails#2004). The generated
#     bindings are already committed under frontend/src/wailsjs, and the
#     desktop-bound methods are invoked dynamically via window.go.main.App,
#     so regeneration is not required for a working build. If you add new
#     Wails-bound methods and need fresh TypeScript bindings, build once with
#     `-tags "sqlite_fts5,native_webview2loader"` and without -skipbindings.
#
# The script:
#   1. Builds the Vue frontend (unless -SkipFrontend)
#   2. Runs `wails build` targeting windows/amd64 with edition=personal
#   3. Copies .env.personal.example alongside the output binary

param(
    [ValidateSet("public", "internal")]
    [string]$BrandProfile = "public",
    [string]$ReleaseVersion = "",
    [switch]$SkipFrontend,
    [string]$DefaultServerURL = "",
    [string]$DefaultServerName = "",
    [string]$DefaultUpdateFeedURL = "",
    [string]$OutputDir = ""
)

$ErrorActionPreference = "Stop"
$RepoRoot = Split-Path -Parent $PSScriptRoot
$NormalizedDefaultServerURL = ""
if (-not [string]::IsNullOrWhiteSpace($DefaultServerURL)) {
    $NormalizedDefaultServerURL = $DefaultServerURL.Trim().TrimEnd('/')
    $parsedDefaultServerURL = $null
    $isValidDefaultServerURL = [System.Uri]::TryCreate(
        $NormalizedDefaultServerURL,
        [System.UriKind]::Absolute,
        [ref]$parsedDefaultServerURL
    )
    if (-not $isValidDefaultServerURL -or
        $parsedDefaultServerURL.Scheme -notin @('http', 'https') -or
        [string]::IsNullOrWhiteSpace($parsedDefaultServerURL.Host)) {
        throw "-DefaultServerURL must be an absolute HTTP or HTTPS URL with a host: $DefaultServerURL"
    }
}

$NormalizedDefaultUpdateFeedURL = ""
if (-not [string]::IsNullOrWhiteSpace($DefaultUpdateFeedURL)) {
    $NormalizedDefaultUpdateFeedURL = $DefaultUpdateFeedURL.Trim().TrimEnd('/')
    $parsedDefaultUpdateFeedURL = $null
    $isValidDefaultUpdateFeedURL = [System.Uri]::TryCreate(
        $NormalizedDefaultUpdateFeedURL,
        [System.UriKind]::Absolute,
        [ref]$parsedDefaultUpdateFeedURL
    )
    if (-not $isValidDefaultUpdateFeedURL -or
        $parsedDefaultUpdateFeedURL.Scheme -notin @('http', 'https') -or
        [string]::IsNullOrWhiteSpace($parsedDefaultUpdateFeedURL.Host)) {
        throw "-DefaultUpdateFeedURL must be an absolute HTTP or HTTPS URL with a host: $DefaultUpdateFeedURL"
    }
}

$Version = $ReleaseVersion.Trim()
if ($Version -eq "") {
    if ($BrandProfile -eq "internal") {
        # Knavis has its own release train; the public source VERSION remains
        # independent so public Xelora builds are not renumbered.
        $Version = "1.0.0"
    } elseif (Test-Path "VERSION") {
        $Version = (Get-Content "VERSION" -Raw).Trim()
    } else {
        $Version = "unknown"
    }
}
$Version = $Version.TrimStart('v')
if ($Version -notmatch '^[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$') {
    throw "Release version must be semantic version 1.2.3 (optionally with prerelease/build metadata): $Version"
}

$previousViteBrandProfile = [Environment]::GetEnvironmentVariable("VITE_BRAND_PROFILE", "Process")
$previousViteFrontendVersion = [Environment]::GetEnvironmentVariable("VITE_FRONTEND_VERSION", "Process")
$env:VITE_BRAND_PROFILE = $BrandProfile
$env:VITE_FRONTEND_VERSION = $Version
$EffectiveDefaultServerName = $DefaultServerName.Trim()
if ($EffectiveDefaultServerName -eq "") {
    $EffectiveDefaultServerName = if ($BrandProfile -eq "internal") { "Knavis Server" } else { "Xelora Server" }
}
$ApplicationName = if ($BrandProfile -eq "internal") { "Knavis Personal" } else { "Xelora Personal" }
$WailsConfigPath = Join-Path $RepoRoot "cmd\desktop\wails.json"
$OriginalWailsConfig = $null

Push-Location $RepoRoot
try {
    if ($BrandProfile -eq "internal" -and (Test-Path -LiteralPath $WailsConfigPath)) {
        # Wails reads product metadata from wails.json. Rewrite it only for the
        # current internal build, then restore the exact original bytes below.
        $OriginalWailsConfig = [System.IO.File]::ReadAllBytes($WailsConfigPath)
        $wailsConfig = Get-Content -LiteralPath $WailsConfigPath -Raw | ConvertFrom-Json
        $wailsConfig.name = $ApplicationName
        $wailsConfig.outputfilename = $ApplicationName
        $wailsConfig.author.name = "Knavis Team"
        $wailsConfig.info.companyName = "Knavis"
        $wailsConfig.info.productName = $ApplicationName
        $wailsConfig.info.productVersion = $Version
        $wailsConfig.info.copyright = "Copyright 2026 Knavis"
        $wailsConfig.info.comments = "$ApplicationName Desktop Edition"
        [System.IO.File]::WriteAllText(
            $WailsConfigPath,
            ($wailsConfig | ConvertTo-Json -Depth 10),
            (New-Object System.Text.UTF8Encoding($false))
        )
    }

    # ── 1. Frontend build ──
    if (-not $SkipFrontend) {
        Write-Host "==> Building frontend ($BrandProfile brand profile)..." -ForegroundColor Cyan
        Push-Location frontend
        try {
            npm ci --silent
            npm run build
        } finally {
            Pop-Location
        }
    } else {
        Write-Host "==> Skipping frontend build (-SkipFrontend)" -ForegroundColor Yellow
    }

    # ── 2. Wails build ──
    Write-Host "==> Building $ApplicationName (windows/amd64)..." -ForegroundColor Cyan

    $CommitID = git rev-parse --short HEAD 2>$null
    if (-not $CommitID) { $CommitID = "unknown" }
    $BuildTime = (Get-Date -Format "yyyy-MM-ddTHH:mm:ssZ")

    $LdFlags = "-w -s " +
        "-X 'github.com/Tencent/Xelora/internal/handler.Version=$Version' " +
        "-X 'github.com/Tencent/Xelora/internal/handler.Edition=personal' " +
        "-X 'github.com/Tencent/Xelora/internal/handler.CommitID=$CommitID' " +
        "-X 'github.com/Tencent/Xelora/internal/handler.BuildTime=$BuildTime' " +
        "-X 'github.com/Tencent/Xelora/internal/branding.PackagedProfile=$BrandProfile'"
    if ($NormalizedDefaultServerURL -ne "") {
        $LdFlags += " -X 'main.packagedEnterpriseServerURL=$NormalizedDefaultServerURL'"
    }
    if ($NormalizedDefaultUpdateFeedURL -ne "") {
        $LdFlags += " -X 'main.packagedDesktopUpdateFeedURL=$NormalizedDefaultUpdateFeedURL'"
    }

    $env:CGO_ENABLED = "1"
    $env:GOLANG_PROTOBUF_REGISTRATION_CONFLICT = "warn"

    Push-Location cmd/desktop
    try {
        $WailsBuildArguments = @(
            "-clean",
            "-skipbindings",
            "-tags", "sqlite_fts5",
            "-platform", "windows/amd64",
            "-ldflags", $LdFlags,
            "-o", "$ApplicationName.exe"
        )
        if ($SkipFrontend) {
            $WailsBuildArguments += "-s"
            $WailsBuildArguments += "-m"
        }
        wails build @WailsBuildArguments
        if ($LASTEXITCODE -ne 0) {
            throw "wails build failed with exit code $LASTEXITCODE"
        }
    } finally {
        Pop-Location
    }

    # ── 3. Package output ──
    $BinPath = Join-Path $RepoRoot ("cmd\desktop\build\bin\{0}.exe" -f $ApplicationName)
    if (-not (Test-Path $BinPath)) {
        Write-Error "Build failed: $BinPath not found"
        exit 1
    }

    if ($OutputDir -eq "") {
        $OutputDir = Join-Path $RepoRoot "dist\personal"
    }
    New-Item -ItemType Directory -Force -Path $OutputDir | Out-Null

    Copy-Item $BinPath (Join-Path $OutputDir "$ApplicationName.exe") -Force
    $PersonalEnvPath = Join-Path $OutputDir ".env.personal"
    Copy-Item ".env.personal.example" $PersonalEnvPath -Force

    # Keep the copied runtime defaults aligned with the build profile. The
    # linker-injected URL still wins at runtime, but writing it here makes the
    # portable package self-describing and editable by an administrator.
    $PersonalEnvContent = Get-Content $PersonalEnvPath -Raw
    if ($NormalizedDefaultServerURL -ne "") {
        $PersonalEnvContent = [regex]::Replace(
            $PersonalEnvContent,
            '(?m)^XELORA_PERSONAL_SERVER_URL=.*$',
            "XELORA_PERSONAL_SERVER_URL=$NormalizedDefaultServerURL"
        )
    }
    $PersonalEnvContent = [regex]::Replace(
        $PersonalEnvContent,
        '(?m)^XELORA_PERSONAL_SERVER_NAME=.*$',
        "XELORA_PERSONAL_SERVER_NAME=$EffectiveDefaultServerName"
    )
    $PersonalEnvContent = [regex]::Replace(
        $PersonalEnvContent,
        '(?m)^XELORA_UPDATE_FEED_URL=.*$',
        "XELORA_UPDATE_FEED_URL=$NormalizedDefaultUpdateFeedURL"
    )
    [System.IO.File]::WriteAllText(
        $PersonalEnvPath,
        $PersonalEnvContent,
        (New-Object System.Text.UTF8Encoding($false))
    )

    # Copy the built frontend as web/ — the embedded backend serves the SPA from
    # ./web (serveFrontendStatic). Without it, GET / falls through to the auth
    # middleware and the window shows {"error":"Unauthorized: missing authentication"}.
    if (Test-Path "frontend\dist\index.html") {
        if (Test-Path (Join-Path $OutputDir "web")) {
            Remove-Item (Join-Path $OutputDir "web") -Recurse -Force
        }
        Copy-Item "frontend\dist" (Join-Path $OutputDir "web") -Recurse -Force
    } else {
        Write-Warning "frontend/dist/index.html not found — run without -SkipFrontend first, or build the frontend manually."
    }

    # Copy config and migrations for standalone operation
    if (Test-Path "config") {
        Copy-Item "config" (Join-Path $OutputDir "config") -Recurse -Force
    }
    if (Test-Path "migrations\sqlite") {
        New-Item -ItemType Directory -Force -Path (Join-Path $OutputDir "migrations\sqlite") | Out-Null
        Copy-Item "migrations\sqlite\*" (Join-Path $OutputDir "migrations\sqlite\") -Force
    }
    # Copy preloaded skills
    if (Test-Path "skills\preloaded") {
        Copy-Item "skills" (Join-Path $OutputDir "skills") -Recurse -Force
    }

    # Build and copy Pi workspace sidecar (Personal cwd file/shell runtime)
    $sidecarDir = Join-Path $RepoRoot "sidecars\pi-workspace"
    if (Test-Path (Join-Path $sidecarDir "package.json")) {
        Write-Host "==> Building pi-workspace sidecar..."
        Push-Location $sidecarDir
        try {
            if (-not (Test-Path "node_modules")) {
                npm ci 2>$null
                if ($LASTEXITCODE -ne 0) { npm install }
            }
            npm run build
            if ($LASTEXITCODE -ne 0) {
                Write-Warning "pi-workspace sidecar build failed — Personal file/shell tools need Node sidecar"
            }
        } finally {
            Pop-Location
        }
        $sidecarOut = Join-Path $OutputDir "pi-workspace"
        if (Test-Path $sidecarOut) {
            Remove-Item $sidecarOut -Recurse -Force
        }
        New-Item -ItemType Directory -Force -Path (Join-Path $sidecarOut "dist") | Out-Null
        if (Test-Path (Join-Path $sidecarDir "dist")) {
            Copy-Item (Join-Path $sidecarDir "dist\*") (Join-Path $sidecarOut "dist\") -Recurse -Force
        }
        Copy-Item (Join-Path $sidecarDir "package.json") $sidecarOut -Force
        Write-Host "    Sidecar: $(Join-Path $sidecarOut 'dist\main.js')"
        Write-Host "    Requires system Node >= 22 on PATH (or set XELORA_NODE)" -ForegroundColor Gray
    }

    Write-Host ""
    Write-Host "==> Build complete!" -ForegroundColor Green
    Write-Host "    Output: $OutputDir"
    Write-Host "    Binary: $(Join-Path $OutputDir "$ApplicationName.exe")"
    Write-Host ""
    Write-Host "    To run: cd '$OutputDir'; .\'$ApplicationName.exe'" -ForegroundColor Gray
} finally {
    if ($null -ne $OriginalWailsConfig) {
        [System.IO.File]::WriteAllBytes($WailsConfigPath, $OriginalWailsConfig)
    }
    if ($null -eq $previousViteBrandProfile) {
        Remove-Item Env:VITE_BRAND_PROFILE -ErrorAction SilentlyContinue
    } else {
        $env:VITE_BRAND_PROFILE = $previousViteBrandProfile
    }
    if ($null -eq $previousViteFrontendVersion) {
        Remove-Item Env:VITE_FRONTEND_VERSION -ErrorAction SilentlyContinue
    } else {
        $env:VITE_FRONTEND_VERSION = $previousViteFrontendVersion
    }
    Pop-Location
}
