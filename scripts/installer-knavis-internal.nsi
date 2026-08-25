; Knavis internal Windows client installer.
; The payload is produced by package-knavis-internal.ps1.

!ifndef KNAVIS_VERSION
  !define KNAVIS_VERSION "0.0.0"
!endif
!ifndef CLIENT_DIR
  !error "CLIENT_DIR is required"
!endif
!ifndef OUTPUT_FILE
  !define OUTPUT_FILE "Knavis-Personal-Setup-${KNAVIS_VERSION}.exe"
!endif

!define PRODUCT_NAME "Knavis Personal"
!define PRODUCT_PUBLISHER "Knavis"
!define PRODUCT_WEB_SITE "https://xelora.example.com"
!define APP_EXE "Knavis Personal.exe"
!define PRODUCT_DIR_REGKEY "Software\Microsoft\Windows\CurrentVersion\App Paths\${APP_EXE}"
!define PRODUCT_UNINST_KEY "Software\Microsoft\Windows\CurrentVersion\Uninstall\${PRODUCT_NAME}"

Name "${PRODUCT_NAME} ${KNAVIS_VERSION}"
OutFile "${OUTPUT_FILE}"
InstallDir "$PROGRAMFILES64\Knavis Personal"
InstallDirRegKey HKLM "${PRODUCT_DIR_REGKEY}" ""
RequestExecutionLevel admin
Unicode true
SetCompressor /SOLID lzma

Page directory
Page instfiles
UninstPage uninstConfirm
UninstPage instfiles

Section "MainSection" SEC01
  SetOutPath "$INSTDIR"
  SetOverwrite ifnewer
  File "${CLIENT_DIR}\${APP_EXE}"
  File "${CLIENT_DIR}\.env.personal"

  SetOutPath "$INSTDIR\web"
  File /r "${CLIENT_DIR}\web\*.*"

  SetOutPath "$INSTDIR\config"
  File /r "${CLIENT_DIR}\config\*.*"

  SetOutPath "$INSTDIR\migrations\sqlite"
  File /r "${CLIENT_DIR}\migrations\sqlite\*.*"

  SetOutPath "$INSTDIR\skills"
  File /r "${CLIENT_DIR}\skills\*.*"

  ; Pi workspace is optional and is included when the build produced it.
  !if exist "${CLIENT_DIR}\pi-workspace\package.json"
    SetOutPath "$INSTDIR\pi-workspace"
    File /r "${CLIENT_DIR}\pi-workspace\*.*"
  !endif

  WriteRegStr HKLM "${PRODUCT_DIR_REGKEY}" "" "$INSTDIR\${APP_EXE}"
  CreateDirectory "$SMPROGRAMS\Knavis Personal"
  CreateShortcut "$SMPROGRAMS\Knavis Personal\Knavis Personal.lnk" "$INSTDIR\${APP_EXE}"
  CreateShortcut "$SMPROGRAMS\Knavis Personal\Uninstall.lnk" "$INSTDIR\uninst.exe"
  CreateShortcut "$DESKTOP\Knavis Personal.lnk" "$INSTDIR\${APP_EXE}"

  WriteUninstaller "$INSTDIR\uninst.exe"
  WriteRegStr HKLM "${PRODUCT_UNINST_KEY}" "DisplayName" "$(^Name)"
  WriteRegStr HKLM "${PRODUCT_UNINST_KEY}" "UninstallString" "$INSTDIR\uninst.exe"
  WriteRegStr HKLM "${PRODUCT_UNINST_KEY}" "DisplayIcon" "$INSTDIR\${APP_EXE}"
  WriteRegStr HKLM "${PRODUCT_UNINST_KEY}" "DisplayVersion" "${KNAVIS_VERSION}"
  WriteRegStr HKLM "${PRODUCT_UNINST_KEY}" "URLInfoAbout" "${PRODUCT_WEB_SITE}"
  WriteRegStr HKLM "${PRODUCT_UNINST_KEY}" "Publisher" "${PRODUCT_PUBLISHER}"
SectionEnd

Section Uninstall
  Delete "$DESKTOP\Knavis Personal.lnk"
  RMDir /r "$SMPROGRAMS\Knavis Personal"
  Delete "$INSTDIR\${APP_EXE}"
  Delete "$INSTDIR\.env.personal"
  Delete "$INSTDIR\uninst.exe"
  RMDir /r "$INSTDIR\web"
  RMDir /r "$INSTDIR\config"
  RMDir /r "$INSTDIR\migrations"
  RMDir /r "$INSTDIR\skills"
  RMDir /r "$INSTDIR\pi-workspace"
  RMDir "$INSTDIR"
  DeleteRegKey HKLM "${PRODUCT_UNINST_KEY}"
  DeleteRegKey HKLM "${PRODUCT_DIR_REGKEY}"
  SetAutoClose true
SectionEnd
