#!/bin/bash

echo "==================================================="
echo "   Build do Teamwork Logger para Linux"
echo "   Autor: Edmilson Dias"
echo "==================================================="
echo ""

# Criar pasta dist se não existir
mkdir -p dist

# Executar o build do Wails
echo "Executando wails build para Linux..."
wails build -platform linux/amd64

# Verificar se o build foi bem-sucedido
if [ $? -ne 0 ]; then
    echo "Erro ao compilar a aplicação!"
    exit 1
fi

# Copiar para pasta dist
echo "Copiando executável para pasta dist..."
cp build/bin/teamwork-logger dist/

echo ""
echo "==================================================="
echo "Build finalizado!"
echo ""
echo "O executável foi gerado com sucesso."
echo "Distribua o arquivo 'dist/teamwork-logger' para seus colegas Linux."
echo "==================================================="
echo ""