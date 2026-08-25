@echo off
setlocal
powershell.exe -NoProfile -ExecutionPolicy Bypass -File "%~dp0verify.ps1" %*
set "exitCode=%ERRORLEVEL%"
endlocal & exit /b %exitCode%
