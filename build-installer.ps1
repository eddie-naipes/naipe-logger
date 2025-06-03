param(
    [string]$Version = "1.0.0"
)

$nsisPath = "C:\Program Files (x86)\NSIS\makensis.exe"
if (-not (Test-Path $nsisPath)) {
    Write-Host "ERRO: NSIS não encontrado" -ForegroundColor Red
    exit 1
}

$exePath = "build\bin\teamwork-logger.exe"
if (-not (Test-Path $exePath)) {
    Write-Host "Executando build..." -ForegroundColor Yellow
    wails build -platform windows/amd64 -ldflags "-H windowsgui -s -w"
}

if (-not (Test-Path $exePath)) {
    Write-Host "ERRO: Executável não encontrado após build!" -ForegroundColor Red
    exit 1
}

$licenseContent = @"
MIT License

Copyright (c) 2025 Naipe Sync Solutions

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
"@

$readmeContent = @"
NAIPE LOGGER - GUIA RÁPIDO
============================

Obrigado por instalar o Naipe Logger!

PRIMEIROS PASSOS:
1. Execute o aplicativo
2. Faça login com suas credenciais do Teamwork
3. Configure suas tarefas favoritas
4. Use templates para apontamento automático

RECURSOS:
✓ Lançamento automático de horas
✓ Calendário inteligente
✓ Templates reutilizáveis
✓ Relatórios em PDF
✓ Interface moderna

SUPORTE:
- GitHub: https://github.com/eddie-naipes/naipe-logger
- Site: https://naipe-logger.com

VERSÃO: $Version
DATA: $(Get-Date -Format 'dd/MM/yyyy')
"@

$licenseContent | Out-File -FilePath "LICENSE.txt" -Encoding UTF8
$readmeContent | Out-File -FilePath "README.txt" -Encoding UTF8

Write-Host "Verificando arquivo do executável..." -ForegroundColor Yellow
Write-Host "Executável encontrado: $exePath" -ForegroundColor Green

$nsiContent = Get-Content "installer.nsi" -Raw
$nsiContent = $nsiContent -replace '\$\{PRODUCT_VERSION\}', $Version
$nsiContent | Out-File -FilePath "installer-build.nsi" -Encoding UTF8

Write-Host "Gerando instalador..." -ForegroundColor Yellow
$process = Start-Process -FilePath $nsisPath -ArgumentList "/V4", "installer-build.nsi" -Wait -PassThru -RedirectStandardOutput "nsis-output.txt" -RedirectStandardError "nsis-error.txt"

if ($process.ExitCode -eq 0) {
    $installerFile = "NaipeLogger-Setup-$Version.exe"
    if (Test-Path $installerFile) {
        $finalName = "NaipeLogger-Setup.exe"
        if (Test-Path $finalName) { Remove-Item $finalName -Force }
        Rename-Item $installerFile $finalName
        $fileSize = (Get-Item $finalName).Length / 1MB
        Write-Host "Instalador criado: $finalName ($([math]::Round($fileSize, 2)) MB)" -ForegroundColor Green
    } else {
        Write-Host "ERRO: Instalador não foi criado!" -ForegroundColor Red
    }
} else {
    Write-Host "ERRO no NSIS:" -ForegroundColor Red
    if (Test-Path "nsis-error.txt") {
        Get-Content "nsis-error.txt" | Write-Host -ForegroundColor Red
    }
    if (Test-Path "nsis-output.txt") {
        Get-Content "nsis-output.txt" | Write-Host -ForegroundColor Yellow
    }
    exit 1
}

Remove-Item "installer-build.nsi" -ErrorAction SilentlyContinue
Remove-Item "nsis-output.txt" -ErrorAction SilentlyContinue
Remove-Item "nsis-error.txt" -ErrorAction SilentlyContinue

Write-Host "Build concluído!" -ForegroundColor Green