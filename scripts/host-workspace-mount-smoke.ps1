param(
    [string]$HostRoot = (Join-Path (Split-Path -Parent $PSScriptRoot) "data\workspaces-e2e"),
    [string]$AppContainer = "Xelora-app"
)

$ErrorActionPreference = "Stop"

& (Join-Path $PSScriptRoot "verify-shared-workspace.ps1") -HostRoot $HostRoot -AppContainer $AppContainer
