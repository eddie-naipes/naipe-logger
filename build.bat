@echo off
echo ==========================================================
echo     Build do Teamwork Logger com Instalador NSIS
echo     Autor: Edmilson Dias
echo ==========================================================
echo.

REM Definir caminho completo para o NSIS
set NSIS_PATH="C:\Program Files (x86)\NSIS"

REM Verificar se a pasta do ícone existe
if not exist "build\windows" (
    echo Criando pasta para o icone...
    mkdir "build\windows"
)

REM Verificar se o ícone existe
if not exist "build\windows\icon.ico" (
    echo [AVISO] Icone nao encontrado em build\windows\icon.ico
    echo Certifique-se de adicionar seu icone antes de distribuir.
    echo O build continuará, mas sem o icone personalizado.
    echo.
)

REM Verificar se NSIS está instalado
if not exist %NSIS_PATH% (
    echo [ERRO] NSIS não encontrado em %NSIS_PATH%
    echo 1. Baixe NSIS de: https://nsis.sourceforge.io/Download
    echo 2. Instale e adicione ao PATH do sistema
    echo O build continuará, mas sem criar o instalador.
    echo.
) else (
    echo NSIS encontrado em: %NSIS_PATH%

    REM Adicionar temporariamente o diretório do NSIS ao PATH
    set "PATH=%NSIS_PATH%;%PATH%"
    echo PATH temporariamente ajustado para incluir o diretório do NSIS
)

echo Iniciando o processo de build...
echo.

REM Criar pasta dist se não existir
if not exist "dist" mkdir dist

REM Primeiro, vamos criar apenas o executável
echo Executando wails build para gerar o executável...
wails build -platform windows/amd64 -webview2 embed -skipbindings

if %errorlevel% neq 0 (
    echo.
    echo [ERRO] Falha ao compilar a aplicacao!
    echo Verifique os erros acima e tente novamente.
    goto :end
)

REM Verificar se o executável foi gerado
if exist "build\bin\teamwork-logger.exe" (
    echo Executável gerado: build\bin\teamwork-logger.exe

    REM Copiar para pasta dist
    echo Copiando executável para pasta dist...
    copy "build\bin\teamwork-logger.exe" dist\

    REM Agora vamos criar manualmente o instalador NSIS se o makensis existir
    if exist "%NSIS_PATH%\makensis.exe" (
        echo Criando arquivo de configuração NSIS temporário...

        echo OutFile "build\bin\teamwork-logger-setup.exe" > installer.nsi
        echo InstallDir "$PROGRAMFILES\Teamwork Logger" >> installer.nsi
        echo Name "Teamwork Logger" >> installer.nsi
        echo !include "MUI2.nsh" >> installer.nsi
        echo !define MUI_ABORTWARNING >> installer.nsi
        echo !insertmacro MUI_PAGE_WELCOME >> installer.nsi
        echo !insertmacro MUI_PAGE_DIRECTORY >> installer.nsi
        echo !insertmacro MUI_PAGE_INSTFILES >> installer.nsi
        echo !insertmacro MUI_PAGE_FINISH >> installer.nsi
        echo !insertmacro MUI_UNPAGE_CONFIRM >> installer.nsi
        echo !insertmacro MUI_UNPAGE_INSTFILES >> installer.nsi
        echo !insertmacro MUI_LANGUAGE "Portuguese" >> installer.nsi
        echo Section "Instalar" >> installer.nsi
        echo   SetOutPath "$INSTDIR" >> installer.nsi
        echo   File "build\bin\teamwork-logger.exe" >> installer.nsi
        echo   WriteUninstaller "$INSTDIR\uninstall.exe" >> installer.nsi
        echo   CreateDirectory "$SMPROGRAMS\Teamwork Logger" >> installer.nsi
        echo   CreateShortcut "$SMPROGRAMS\Teamwork Logger\Teamwork Logger.lnk" "$INSTDIR\teamwork-logger.exe" >> installer.nsi
        echo   CreateShortcut "$DESKTOP\Teamwork Logger.lnk" "$INSTDIR\teamwork-logger.exe" >> installer.nsi
        echo SectionEnd >> installer.nsi
        echo Section "Uninstall" >> installer.nsi
        echo   Delete "$INSTDIR\teamwork-logger.exe" >> installer.nsi
        echo   Delete "$INSTDIR\uninstall.exe" >> installer.nsi
        echo   Delete "$SMPROGRAMS\Teamwork Logger\Teamwork Logger.lnk" >> installer.nsi
        echo   Delete "$DESKTOP\Teamwork Logger.lnk" >> installer.nsi
        echo   RMDir "$SMPROGRAMS\Teamwork Logger" >> installer.nsi
        echo   RMDir "$INSTDIR" >> installer.nsi
        echo SectionEnd >> installer.nsi

        echo Executando NSIS manualmente...
        "%NSIS_PATH%\makensis.exe" installer.nsi

        if exist "build\bin\teamwork-logger-setup.exe" (
            echo Instalador NSIS gerado: build\bin\teamwork-logger-setup.exe
            echo Copiando instalador para pasta dist...
            copy "build\bin\teamwork-logger-setup.exe" dist\
            del installer.nsi
        ) else (
            echo [ERRO] Falha ao criar o instalador NSIS.
        )
    ) else (
        echo [AVISO] makensis.exe não encontrado em %NSIS_PATH%. Não foi possível criar o instalador.
    )
) else (
    echo [AVISO] Executável não encontrado em build\bin\teamwork-logger.exe
)

echo.
echo ==========================================================
echo Build finalizado!
if exist "build\bin\teamwork-logger-setup.exe" (
    echo O instalador foi gerado com sucesso.
    echo Distribua o arquivo 'dist\teamwork-logger-setup.exe' para seus colegas.
    echo O instalador criará automaticamente um atalho na área de trabalho.
) else (
    echo O executável foi gerado, mas o instalador NSIS não foi criado.
    echo Você pode distribuir o executável 'dist\teamwork-logger.exe' diretamente,
    echo mas os usuários precisarão criar o atalho manualmente.
    echo.
    echo Para gerar o instalador, instale o NSIS e execute este script novamente.
)
echo ==========================================================
echo.

:end
pause