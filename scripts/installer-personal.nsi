; Xelora Personal — NSIS Installer Script
; Build with: makensis /DXELORA_VERSION=1.0.0 scripts/installer-personal.nsi
;
; Prerequisites:
;   - Run scripts/build-personal.ps1 first to produce dist/personal/
;   - NSIS 3.x installed (https://nsis.sourceforge.io)
;
; Produces: dist/Xelora-Personal-Setup-<version>.exe

!ifndef XELORA_VERSION
  !define XELORA_VERSION "1.0.0"
!endif

!define PRODUCT_NAME "Xelora Personal"
!define PRODUCT_PUBLISHER "Xelora"
!define PRODUCT_WEB_SITE "https://github.com/Tencent/Xelora"
!define PRODUCT_DIR_REGKEY "Software\Microsoft\Windows\CurrentVersion\App Paths\Xelora Personal.exe"
!define PRODUCT_UNINST_KEY "Software\Microsoft\Windows\CurrentVersion\Uninstall\${PRODUCT_NAME}"

Name "${PRODUCT_NAME} ${XELORA_VERSION}"
OutFile "..\dist\Xelora-Personal-Setup-${XELORA_VERSION}.exe"
InstallDir "$PROGRAMFILES64\Xelora Personal"
InstallDirRegKey HKLM "${PRODUCT_DIR_REGKEY}" ""
RequestExecutionLevel admin
Unicode true
SetCompressor /SOLID lzma

; ── Pages ──
Page directory
Page instfiles
UninstPage uninstConfirm
UninstPage instfiles

; ── Install Section ──
Section "MainSection" SEC01
  SetOutPath "$INSTDIR"
  SetOverwrite ifnewer

  ; Core binary
  File "..\dist\personal\Xelora Personal.exe"

  ; Environment template
  File "..\dist\personal\.env.personal"

  ; Config directory
  SetOutPath "$INSTDIR\config"
  File /r "..\dist\personal\config\*.*"

  ; SQLite migrations
  SetOutPath "$INSTDIR\migrations\sqlite"
  File /r "..\dist\personal\migrations\sqlite\*.*"

  ; Preloaded skills
  SetOutPath "$INSTDIR\skills"
  File /r "..\dist\personal\skills\*.*"

  ; Data directory (created empty; SQLite DB and files land here)
  SetOutPath "$INSTDIR\data"
  FileOpen $0 "$INSTDIR\data\.gitkeep" w
  FileClose $0

  ; App registry key for "App Paths" (Run dialog / shell)
  WriteRegStr HKLM "${PRODUCT_DIR_REGKEY}" "" "$INSTDIR\Xelora Personal.exe"

  ; Start Menu shortcut
  CreateDirectory "$SMPROGRAMS\Xelora Personal"
  CreateShortcut "$SMPROGRAMS\Xelora Personal\Xelora Personal.lnk" "$INSTDIR\Xelora Personal.exe"
  CreateShortcut "$SMPROGRAMS\Xelora Personal\Uninstall.lnk" "$INSTDIR\uninst.exe"

  ; Desktop shortcut
  CreateShortcut "$DESKTOP\Xelora Personal.lnk" "$INSTDIR\Xelora Personal.exe"

  ; Uninstaller
  WriteUninstaller "$INSTDIR\uninst.exe"
  WriteRegStr HKLM "${PRODUCT_UNINST_KEY}" "DisplayName" "$(^Name)"
  WriteRegStr HKLM "${PRODUCT_UNINST_KEY}" "UninstallString" "$INSTDIR\uninst.exe"
  WriteRegStr HKLM "${PRODUCT_UNINST_KEY}" "DisplayIcon" "$INSTDIR\Xelora Personal.exe"
  WriteRegStr HKLM "${PRODUCT_UNINST_KEY}" "DisplayVersion" "${XELORA_VERSION}"
  WriteRegStr HKLM "${PRODUCT_UNINST_KEY}" "URLInfoAbout" "${PRODUCT_WEB_SITE}"
  WriteRegStr HKLM "${PRODUCT_UNINST_KEY}" "Publisher" "${PRODUCT_PUBLISHER}"
SectionEnd

; ── Uninstall Section ──
Section Uninstall
  ; Remove shortcuts
  Delete "$DESKTOP\Xelora Personal.lnk"
  RMDir /r "$SMPROGRAMS\Xelora Personal"

  ; Remove installed files (but preserve user data by default)
  Delete "$INSTDIR\Xelora Personal.exe"
  Delete "$INSTDIR\.env.personal"
  Delete "$INSTDIR\uninst.exe"
  RMDir /r "$INSTDIR\config"
  RMDir /r "$INSTDIR\migrations"
  RMDir /r "$INSTDIR\skills"

  ; Note: $INSTDIR\data is intentionally NOT deleted to preserve the user's
  ; knowledge bases, chat history, and generated files. Users can delete it
  ; manually for a full clean-up.

  RMDir "$INSTDIR"

  DeleteRegKey HKLM "${PRODUCT_UNINST_KEY}"
  DeleteRegKey HKLM "${PRODUCT_DIR_REGKEY}"
  SetAutoClose true
SectionEnd
