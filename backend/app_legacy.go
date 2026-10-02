package backend

import (
	"log/slog"

	"logTime-go/backend/legacy"
)

// Bindings da instalação antiga (Windows, Program Files/HKLM). Ver o pacote
// legacy para as chaves de registro dos instaladores anteriores.

// Indireções para os testes (o registro real só existe no Windows).
var (
	detectLegacy       = legacy.Detect
	runLegacyUninstall = legacy.RunUninstaller
)

// GetLegacyInstall informa se há uma instalação antiga (como administrador,
// em Program Files) além da atual. Found=false fora do Windows ou se nada for
// encontrado; falhas de leitura do registro só vão para o log.
func (a *App) GetLegacyInstall() legacy.Install {
	install, err := detectLegacy()
	if err != nil {
		slog.Warn("Não foi possível procurar instalação antiga", "err", err)
		return legacy.Install{}
	}
	return install
}

// RunLegacyUninstaller executa o desinstalador da instalação antiga. O
// Windows pede confirmação de administrador (UAC). A instalação é detectada de
// novo aqui: o comando executado nunca vem do frontend.
func (a *App) RunLegacyUninstaller() error {
	install, err := detectLegacy()
	if err != nil {
		return err
	}
	if !install.Found {
		return legacy.ErrNotFound
	}
	slog.Info("Executando desinstalador da instalação antiga", "local", install.InstallLocation)
	return runLegacyUninstall(install)
}
