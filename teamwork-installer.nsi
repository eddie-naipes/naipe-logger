!define PRODUCT_NAME "Naipe Logger"
!define PRODUCT_VERSION "1.0.0"
!define PRODUCT_PUBLISHER "Naipe Sync Solutions"
!define PRODUCT_WEB_SITE "https://naipe-logger.com"
!define PRODUCT_DIR_REGKEY "Software\Microsoft\Windows\CurrentVersion\App Paths\teamwork-logger.exe"
!define PRODUCT_UNINST_KEY "Software\Microsoft\Windows\CurrentVersion\Uninstall\${PRODUCT_NAME}"
!define PRODUCT_UNINST_ROOT_KEY "HKLM"

SetCompressor lzma
RequestExecutionLevel admin

!include "MUI2.nsh"
!include "FileFunc.nsh"
!include "LogicLib.nsh"
!include "WinMessages.nsh"
!include "x64.nsh"

Name "${PRODUCT_NAME}"
OutFile "NaipeLogger-Setup-${PRODUCT_VERSION}.exe"
InstallDir "$PROGRAMFILES64\${PRODUCT_NAME}"
InstallDirRegKey HKLM "${PRODUCT_DIR_REGKEY}" ""
ShowInstDetails show
ShowUnInstDetails show

!define MUI_ABORTWARNING
!define MUI_FINISHPAGE_RUN "$INSTDIR\teamwork-logger.exe"
!define MUI_FINISHPAGE_RUN_TEXT "Executar ${PRODUCT_NAME}"
!define MUI_FINISHPAGE_LINK "Visite nosso site"
!define MUI_FINISHPAGE_LINK_LOCATION "${PRODUCT_WEB_SITE}"

!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_LICENSE "LICENSE.txt"
!insertmacro MUI_PAGE_COMPONENTS
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH

!insertmacro MUI_UNPAGE_WELCOME
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_UNPAGE_FINISH

!insertmacro MUI_LANGUAGE "Portuguese"

VIProductVersion "1.0.0.0"
VIAddVersionKey /LANG=${LANG_PORTUGUESE} "ProductName" "${PRODUCT_NAME}"
VIAddVersionKey /LANG=${LANG_PORTUGUESE} "CompanyName" "${PRODUCT_PUBLISHER}"
VIAddVersionKey /LANG=${LANG_PORTUGUESE} "LegalCopyright" "© 2025 ${PRODUCT_PUBLISHER}"
VIAddVersionKey /LANG=${LANG_PORTUGUESE} "FileDescription" "Instalador do ${PRODUCT_NAME}"
VIAddVersionKey /LANG=${LANG_PORTUGUESE} "FileVersion" "${PRODUCT_VERSION}"

Function .onInit
  UserInfo::GetAccountType
  pop $0
  ${If} $0 != "admin"
    MessageBox mb_iconstop "Este instalador requer privilégios de administrador."
    SetErrorLevel 740
    Quit
  ${EndIf}

  ${If} ${RunningX64}
    SetRegView 64
    StrCpy $INSTDIR "$PROGRAMFILES64\${PRODUCT_NAME}"
  ${Else}
    MessageBox MB_OK|MB_ICONSTOP "Este aplicativo requer Windows 64-bit."
    Quit
  ${EndIf}
FunctionEnd

Section "Programa Principal" SecMain
  SectionIn RO
  SetOutPath "$INSTDIR"
  SetOverwrite ifnewer

  File "build\bin\teamwork-logger.exe"
  File "LICENSE.txt"
  File "README.txt"

  SetShellVarContext all
  CreateDirectory "$APPDATA\${PRODUCT_NAME}"

  WriteRegStr HKLM "${PRODUCT_DIR_REGKEY}" "" "$INSTDIR\teamwork-logger.exe"
  WriteRegStr HKLM "${PRODUCT_UNINST_KEY}" "DisplayName" "$(^Name)"
  WriteRegStr HKLM "${PRODUCT_UNINST_KEY}" "UninstallString" "$INSTDIR\uninst.exe"
  WriteRegStr HKLM "${PRODUCT_UNINST_KEY}" "DisplayIcon" "$INSTDIR\teamwork-logger.exe"
  WriteRegStr HKLM "${PRODUCT_UNINST_KEY}" "DisplayVersion" "${PRODUCT_VERSION}"
  WriteRegStr HKLM "${PRODUCT_UNINST_KEY}" "URLInfoAbout" "${PRODUCT_WEB_SITE}"
  WriteRegStr HKLM "${PRODUCT_UNINST_KEY}" "Publisher" "${PRODUCT_PUBLISHER}"
  WriteRegDWORD HKLM "${PRODUCT_UNINST_KEY}" "NoModify" 1
  WriteRegDWORD HKLM "${PRODUCT_UNINST_KEY}" "NoRepair" 1

  ${GetSize} "$INSTDIR" "/S=0K" $0 $1 $2
  IntFmt $0 "0x%08X" $0
  WriteRegDWORD HKLM "${PRODUCT_UNINST_KEY}" "EstimatedSize" "$0"

  WriteUninstaller "$INSTDIR\uninst.exe"

  CreateDirectory "$SMPROGRAMS\${PRODUCT_NAME}"
  CreateShortCut "$SMPROGRAMS\${PRODUCT_NAME}\${PRODUCT_NAME}.lnk" "$INSTDIR\teamwork-logger.exe"
  CreateShortCut "$SMPROGRAMS\${PRODUCT_NAME}\Desinstalar ${PRODUCT_NAME}.lnk" "$INSTDIR\uninst.exe"
SectionEnd

Section "Atalho na Área de Trabalho" SecDesktop
  CreateShortCut "$DESKTOP\${PRODUCT_NAME}.lnk" "$INSTDIR\teamwork-logger.exe"
SectionEnd

Section /o "Inicialização Automática" SecStartup
  CreateShortCut "$SMSTARTUP\${PRODUCT_NAME}.lnk" "$INSTDIR\teamwork-logger.exe" "--minimized"
SectionEnd

!insertmacro MUI_FUNCTION_DESCRIPTION_BEGIN
  !insertmacro MUI_DESCRIPTION_TEXT ${SecMain} "Arquivos principais do ${PRODUCT_NAME} (obrigatório)"
  !insertmacro MUI_DESCRIPTION_TEXT ${SecDesktop} "Criar atalho na área de trabalho"
  !insertmacro MUI_DESCRIPTION_TEXT ${SecStartup} "Iniciar automaticamente com o Windows"
!insertmacro MUI_FUNCTION_DESCRIPTION_END

Section "Uninstall"
  Delete "$INSTDIR\teamwork-logger.exe"
  Delete "$INSTDIR\LICENSE.txt"
  Delete "$INSTDIR\README.txt"
  Delete "$INSTDIR\uninst.exe"
  RMDir "$INSTDIR"

  Delete "$SMPROGRAMS\${PRODUCT_NAME}\${PRODUCT_NAME}.lnk"
  Delete "$SMPROGRAMS\${PRODUCT_NAME}\Desinstalar ${PRODUCT_NAME}.lnk"
  RMDir "$SMPROGRAMS\${PRODUCT_NAME}"
  Delete "$DESKTOP\${PRODUCT_NAME}.lnk"
  Delete "$SMSTARTUP\${PRODUCT_NAME}.lnk"

  DeleteRegKey HKLM "${PRODUCT_UNINST_KEY}"
  DeleteRegKey HKLM "${PRODUCT_DIR_REGKEY}"

  MessageBox MB_YESNO|MB_ICONQUESTION "Deseja manter as configurações do usuário?" IDYES keep_data
    RMDir /r "$APPDATA\${PRODUCT_NAME}"
  keep_data:

  SetAutoClose true
SectionEnd

Function un.onInit
  MessageBox MB_ICONQUESTION|MB_YESNO|MB_DEFBUTTON2 "Tem certeza que deseja remover o $(^NameDA)?" IDYES +2
  Abort
FunctionEnd