# build-personal.ps1 — Build Xelora Personal desktop EXE for Windows amd64
# Usage: .\scripts\build-personal.ps1 [-SkipFrontend] [-OutputDir <path>]
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
    [switch]$SkipFrontend,
    [string]$OutputDir = ""
)

$ErrorActionPreference = "Stop"
$RepoRoot = Split-Path -Parent $PSScriptRoot

Push-Location $RepoRoot
try {
    # ── 1. Frontend build ──
    if (-not $SkipFrontend) {
        Write-Host "==> Building frontend..." -ForegroundColor Cyan
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
    Write-Host "==> Building Xelora Personal (windows/amd64)..." -ForegroundColor Cyan

    $Version = "1.0.0"
    if (Test-Path "VERSION") {
        $Version = (Get-Content "VERSION" -Raw).Trim()
    }
    $CommitID = git rev-parse --short HEAD 2>$null
    if (-not $CommitID) { $CommitID = "unknown" }
    $BuildTime = (Get-Date -Format "yyyy-MM-ddTHH:mm:ssZ")

    $LdFlags = "-w -s " +
        "-X 'github.com/Tencent/Xelora/internal/handler.Version=$Version' " +
        "-X 'github.com/Tencent/Xelora/internal/handler.Edition=personal' " +
        "-X 'github.com/Tencent/Xelora/internal/handler.CommitID=$CommitID' " +
        "-X 'github.com/Tencent/Xelora/internal/handler.BuildTime=$BuildTime'"

    $env:CGO_ENABLED = "1"
    $env:GOLANG_PROTOBUF_REGISTRATION_CONFLICT = "warn"

    Push-Location cmd/desktop
    try {
        wails build -clean -skipbindings `
            -tags "sqlite_fts5" `
            -platform "windows/amd64" `
            -ldflags "$LdFlags" `
            -o "Xelora Personal.exe"
    } finally {
        Pop-Location
    }

    # ── 3. Package output ──
    $BinPath = "cmd\desktop\build\bin\Xelora Personal.exe"
    if (-not (Test-Path $BinPath)) {
        Write-Error "Build failed: $BinPath not found"
        exit 1
    }

    if ($OutputDir -eq "") {
        $OutputDir = Join-Path $RepoRoot "dist\personal"
    }
    New-Item -ItemType Directory -Force -Path $OutputDir | Out-Null

    Copy-Item $BinPath (Join-Path $OutputDir "Xelora Personal.exe") -Force
    Copy-Item ".env.personal.example" (Join-Path $OutputDir ".env.personal") -Force

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

    Write-Host ""
    Write-Host "==> Build complete!" -ForegroundColor Green
    Write-Host "    Output: $OutputDir"
    Write-Host "    Binary: $(Join-Path $OutputDir 'Xelora Personal.exe')"
    Write-Host ""
    Write-Host "    To run: cd '$OutputDir'; .\'Xelora Personal.exe'" -ForegroundColor Gray
} finally {
    Pop-Location
}
