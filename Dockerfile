# Dockerfile para build Wails Linux
FROM golang:1.24-bookworm

# Instalar dependências necessárias para Wails Linux
RUN apt-get update && apt-get install -y \
    build-essential \
    pkg-config \
    libgtk-3-dev \
    libwebkit2gtk-4.0-dev \
    libwebkit2gtk-4.1-dev \
    nodejs \
    npm \
    && rm -rf /var/lib/apt/lists/*

# Instalar Wails
RUN go install github.com/wailsapp/wails/v2/cmd/wails@latest

# Definir diretório de trabalho
WORKDIR /app

# Copiar arquivos do projeto
COPY . .

# Instalar dependências frontend
RUN cd frontend && npm install

# Build do projeto
RUN wails build -platform linux/amd64

# Comando padrão
CMD ["echo", "Build concluído! Verifique /app/build/bin/"]