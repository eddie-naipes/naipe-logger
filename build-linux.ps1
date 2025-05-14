# Script PowerShell para compilar o Teamwork Logger para Linux
Write-Host "==================================================="
Write-Host "   Build do Teamwork Logger para Linux"
Write-Host "   Autor: Edmilson Dias"
Write-Host "==================================================="
Write-Host ""

# Criar pasta dist/linux se não existir
if (-not (Test-Path "dist/linux")) {
    New-Item -ItemType Directory -Force -Path "dist/linux"
}

# Remover configurações temporariamente (para não incluir dados pessoais)
if (Test-Path "$env:USERPROFILE\.teamwork-logger") {
    Write-Host "Fazendo backup temporário das configurações..."
    Rename-Item "$env:USERPROFILE\.teamwork-logger" "$env:USERPROFILE\.teamwork-logger-backup"
}

# Executar o build do Wails para Linux
Write-Host "Executando wails build para Linux..."
wails build -platform linux/amd64 -skipbindings -v 2

# Verificar se o build foi bem-sucedido
if ($LASTEXITCODE -ne 0) {
    Write-Host "Erro ao compilar a aplicação!" -ForegroundColor Red

    # Restaurar backup de configurações se existir
    if (Test-Path "$env:USERPROFILE\.teamwork-logger-backup") {
        Rename-Item "$env:USERPROFILE\.teamwork-logger-backup" "$env:USERPROFILE\.teamwork-logger"
    }

    exit 1
}

# Copiar para pasta dist/linux
Write-Host "Copiando executável para pasta dist/linux..."
Copy-Item "build/bin/teamwork-logger" -Destination "dist/linux/"

# Criar script de instalação Linux
$installerContent = @"
#!/bin/bash

# Script de instalação do Teamwork Logger
# Autor: Edmilson Dias
# Gerado: $(Get-Date -Format "dd/MM/yyyy")

echo "Instalando Teamwork Logger..."

# Criar diretório de instalação
mkdir -p ~/TeamworkLogger

# Copiar arquivo executável
cp teamwork-logger ~/TeamworkLogger/

# Garantir permissões
chmod +x ~/TeamworkLogger/teamwork-logger

# Criar link simbólico (opcional)
mkdir -p ~/.local/bin
ln -sf ~/TeamworkLogger/teamwork-logger ~/.local/bin/teamwork-logger

# Criar atalho no desktop
cat > ~/Desktop/teamwork-logger.desktop << EOL
[Desktop Entry]
Name=Teamwork Logger
Exec=~/TeamworkLogger/teamwork-logger
Type=Application
Categories=Utility;
Terminal=false
EOL

chmod +x ~/Desktop/teamwork-logger.desktop

echo "Instalação concluída. Você pode iniciar o aplicativo pelo atalho no desktop ou executando ~/TeamworkLogger/teamwork-logger"
"@

# Salvar o script de instalação com LF (line feeds) em vez de CRLF
$installerContent = $installerContent -replace "`r`n", "`n"
[System.IO.File]::WriteAllText("dist/linux/install.sh", $installerContent, [System.Text.Encoding]::UTF8)

# Criar arquivo README.txt
$readmeContent = @"
Teamwork Logger para Linux
==========================

Instruções de Instalação:

1. Extraia este arquivo em uma pasta de sua escolha
2. Abra um terminal e navegue até essa pasta
3. Dê permissão de execução ao script de instalação:
   chmod +x install.sh
4. Execute o script de instalação:
   ./install.sh
5. Após a instalação, você pode iniciar o aplicativo pelo atalho no desktop
   ou executando o comando:
   ~/TeamworkLogger/teamwork-logger

Observações:
- O aplicativo será instalado na pasta ~/TeamworkLogger no seu diretório home
- Um atalho será criado em seu desktop
- Os arquivos de configuração serão armazenados em ~/.config/teamwork-logger

Dependências:
Este aplicativo requer o WebKit2GTK. Você pode instalá-lo com:

Ubuntu/Debian:
sudo apt-get install libwebkit2gtk-4.0-37

Fedora:
sudo dnf install webkit2gtk3

Arch Linux:
sudo pacman -S webkit2gtk
"@

# Salvar o README.txt
$readmeContent = $readmeContent -replace "`r`n", "`n"
[System.IO.File]::WriteAllText("dist/linux/README.txt", $readmeContent, [System.Text.Encoding]::UTF8)

# Criar arquivo ZIP (Windows pode criar ZIP nativamente)
Write-Host "Criando arquivo ZIP para distribuição..."
Compress-Archive -Path "dist/linux/*" -DestinationPath "dist/teamwork-logger-linux.zip" -Force

# Restaurar backup de configurações se existir
if (Test-Path "$env:USERPROFILE\.teamwork-logger-backup") {
    Write-Host "Restaurando configurações originais..."
    if (Test-Path "$env:USERPROFILE\.teamwork-logger") {
        Remove-Item -Recurse -Force "$env:USERPROFILE\.teamwork-logger"
    }
    Rename-Item "$env:USERPROFILE\.teamwork-logger-backup" "$env:USERPROFILE\.teamwork-logger"
}

Write-Host ""
Write-Host "==================================================="
Write-Host "Build finalizado!"
Write-Host ""
Write-Host "O executável foi gerado com sucesso."
Write-Host "Arquivo para distribuição: dist/teamwork-logger-linux.zip"
Write-Host "Distribua este arquivo ZIP para seus colegas Linux."
Write-Host "==================================================="