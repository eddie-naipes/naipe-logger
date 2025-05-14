; Installer script for Teamwork Logger

; Informações básicas do produto
!define PRODUCT_NAME "Naipe Timer"
!define PRODUCT_VERSION "1.0"
!define PRODUCT_PUBLISHER "Naipe sync Solutions"
!define PRODUCT_WEB_SITE "https://naipesync.com"
!define PRODUCT_UNINST_KEY "Software\Microsoft\Windows\CurrentVersion\Uninstall\${PRODUCT_NAME}"

; Configurações de interface
!include "MUI2.nsh"
!define MUI_ABORTWARNING
; Comente estas linhas se não tiver o ícone
;!define MUI_ICON "build\windows\icon.ico"
;!define MUI_UNICON "build\windows\icon.ico"

; Páginas do instalador
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH

; Páginas do desinstalador
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES

; Idioma
!insertmacro MUI_LANGUAGE "PortugueseBR"

; Nome do instalador
OutFile "dist\naipe-logger-setup.exe"

; Diretório de instalação padrão
InstallDir "$PROGRAMFILES\Naiper Logger"

; Registro para desinstalação
InstallDirRegKey HKLM "${PRODUCT_UNINST_KEY}" "UninstallString"

Name "${PRODUCT_NAME} ${PRODUCT_VERSION}"
BrandingText "${PRODUCT_NAME} Installer"

; Seção de instalação principal
Section "Instalação Principal" SEC01
  SetOutPath "$INSTDIR"
  SetOverwrite ifnewer

  ; Arquivos a serem instalados - ajuste o caminho conforme necessário
  File "build\bin\teamwork-logger.exe"

  ; Criar atalhos
  CreateDirectory "$SMPROGRAMS\${PRODUCT_NAME}"
  CreateShortcut "$SMPROGRAMS\${PRODUCT_NAME}\${PRODUCT_NAME}.lnk" "$INSTDIR\teamwork-logger.exe"
  CreateShortcut "$DESKTOP\${PRODUCT_NAME}.lnk" "$INSTDIR\teamwork-logger.exe"

  ; Criar desinstalador
  WriteUninstaller "$INSTDIR\uninst.exe"
  WriteRegStr HKLM "${PRODUCT_UNINST_KEY}" "DisplayName" "$(^Name)"
  WriteRegStr HKLM "${PRODUCT_UNINST_KEY}" "UninstallString" "$INSTDIR\uninst.exe"
  WriteRegStr HKLM "${PRODUCT_UNINST_KEY}" "DisplayIcon" "$INSTDIR\teamwork-logger.exe"
  WriteRegStr HKLM "${PRODUCT_UNINST_KEY}" "DisplayVersion" "${PRODUCT_VERSION}"
  WriteRegStr HKLM "${PRODUCT_UNINST_KEY}" "Publisher" "${PRODUCT_PUBLISHER}"
SectionEnd

; Seção de desinstalação
Section "Uninstall"
  ; Remover arquivos
  Delete "$INSTDIR\teamwork-logger.exe"
  Delete "$INSTDIR\uninst.exe"

  ; Remover atalhos
  Delete "$SMPROGRAMS\${PRODUCT_NAME}\${PRODUCT_NAME}.lnk"
  Delete "$DESKTOP\${PRODUCT_NAME}.lnk"
  RMDir "$SMPROGRAMS\${PRODUCT_NAME}"

  ; Remover diretório de instalação
  RMDir "$INSTDIR"

  ; Remover registro
  DeleteRegKey HKLM "${PRODUCT_UNINST_KEY}"
SectionEnd