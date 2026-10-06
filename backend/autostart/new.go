package autostart

import (
	"log/slog"
	"os"
	"path/filepath"
)

// New monta o Manager do SO atual. Sem implementação (ou sem conseguir
// descobrir as pastas), Status informa "não suportado".
func New(version string, dev bool) Manager {
	launcher, err := defaultLauncher()
	if err != nil {
		slog.Warn("Iniciar com o sistema indisponível", "err", err)
		launcher = nil
	}
	return Manager{Launcher: launcher, Executable: currentExecutable, Version: version, Dev: dev}
}

// currentExecutable resolve links simbólicos, para o item de inicialização
// apontar para o binário real.
func currentExecutable() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	return filepath.Abs(exe)
}
