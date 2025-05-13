@echo off
echo ==========================================================
echo     Build do Teamwork Logger com Instalador NSIS
echo     Autor: Edmilson Dias
echo ==========================================================
echo.

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
where makensis >nul 2>nul
if %errorlevel% neq 0 (
    echo [ERRO] NSIS não encontrado! Você precisa instalar o NSIS:
    echo 1. Baixe NSIS de: https://nsis.sourceforge.io/Download
    echo 2. Instale e adicione ao PATH do sistema
    echo O build continuará, mas sem criar o instalador.
    echo.
)

echo Iniciando o processo de build...
echo.

REM Criar pasta dist se não existir
if not exist "dist" mkdir dist

REM Executar o build do Wails com NSIS
echo Executando wails build com instalador NSIS...
wails build -platform windows/amd64 -nsis -webview2 embed

if %errorlevel% neq 0 (
    echo.
    echo [ERRO] Falha ao compilar a aplicacao!
    echo Verifique os erros acima e tente novamente.
    goto :end
)

echo.
echo Build concluido com sucesso!
echo.

REM Verificar se os arquivos foram gerados
if exist "build\bin\teamwork-logger.exe" (
    echo Executável gerado: build\bin\teamwork-logger.exe

    REM Copiar para pasta dist
    echo Copiando executável para pasta dist...
    copy "build\bin\teamwork-logger.exe" dist\
) else (
    echo [AVISO] Executável não encontrado em build\bin\teamwork-logger.exe
)

if exist "build\bin\teamwork-logger-setup.exe" (
    echo Instalador NSIS gerado: build\bin\teamwork-logger-setup.exe

    REM Copiar para pasta dist
    echo Copiando instalador para pasta dist...
    copy "build\bin\teamwork-logger-setup.exe" dist\
) else (
    echo [AVISO] Instalador NSIS não foi gerado. Verifique se o NSIS está instalado corretamente.
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