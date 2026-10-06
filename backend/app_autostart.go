package backend

import (
	"log/slog"

	"logTime-go/backend/autostart"
	"logTime-go/backend/logging"
)

// Bindings de "Iniciar com o sistema (minimizado)". O estado vem sempre do SO
// (registro, LaunchAgent ou .desktop), não de config.json: o usuário pode
// removê-lo por fora (Gerenciador de Tarefas, Itens de Início...).

// newAutostart é variável para os testes usarem um registro falso.
var newAutostart = autostart.New

func (a *App) autostart() autostart.Manager {
	return newAutostart(a.GetAppVersion(), logging.DevBuild())
}

// GetAutostart informa se o app abre no login, se isso é suportado aqui e,
// se não for, por quê (ex.: versão de desenvolvimento).
func (a *App) GetAutostart() autostart.Status {
	return a.autostart().Status()
}

// SetAutostart liga ou desliga a abertura no login (minimizado).
func (a *App) SetAutostart(enabled bool) (autostart.Status, error) {
	st, err := a.autostart().SetEnabled(enabled)
	if err != nil {
		slog.Warn("Não foi possível alterar o início com o sistema", "ligar", enabled, "err", err)
		return st, err
	}
	slog.Info("Início com o sistema alterado", "ligado", st.Enabled)
	return st, nil
}
