// Package autostart liga e desliga "iniciar com o sistema" (minimizado):
// Windows → valor em HKCU\Software\Microsoft\Windows\CurrentVersion\Run;
// macOS → LaunchAgent em ~/Library/LaunchAgents; Linux → entrada .desktop em
// ~/.config/autostart. Cada mecanismo é um Launcher com registro/pastas
// injetáveis, testável em qualquer SO; New escolhe o do SO atual.
package autostart

import (
	"errors"
	"strings"
)

// MinimizedFlag é o argumento que o item de inicialização passa ao app.
const MinimizedFlag = "--minimized"

// HasMinimizedFlag informa se o app foi iniciado pelo item de inicialização.
func HasMinimizedFlag(args []string) bool {
	for _, a := range args {
		if a == MinimizedFlag {
			return true
		}
	}
	return false
}

// Launcher é a implementação de um SO.
type Launcher interface {
	// Enable registra exe (caminho absoluto) com args para abrir no login.
	// É idempotente: chamar de novo reescreve o mesmo item.
	Enable(exe string, args []string) error
	// Disable remove o item; remover um item inexistente não é erro.
	Disable() error
	// Enabled informa se o item existe.
	Enabled() (bool, error)
}

// Status é o que a UI precisa para desenhar o toggle.
type Status struct {
	Enabled   bool   `json:"enabled"`
	Supported bool   `json:"supported"`
	Reason    string `json:"reason"`
}

// ErrUnsupported indica um SO sem implementação.
var ErrUnsupported = errors.New("iniciar com o sistema não é suportado neste sistema operacional")

// Manager junta o Launcher, o executável atual e a versão.
type Manager struct {
	Launcher Launcher
	// Executable devolve o caminho do executável atual (os.Executable).
	Executable func() (string, error)
	Version    string
	// Dev marca um binário de `wails dev` mesmo sem o sufixo na versão.
	Dev bool
}

// IsDevVersion informa se a versão é de desenvolvimento (`wails dev`): o
// executável fica numa pasta temporária, então não serve para o login.
func IsDevVersion(version string) bool {
	return strings.HasSuffix(version, "-dev") || strings.Contains(version, "-dev.")
}

func (m Manager) unsupportedReason() string {
	switch {
	case m.Launcher == nil:
		return ErrUnsupported.Error()
	case m.Dev || IsDevVersion(m.Version):
		return "Indisponível na versão de desenvolvimento (wails dev): o executável é temporário. Use a versão instalada."
	}
	return ""
}

// Status consulta o estado atual.
func (m Manager) Status() Status {
	if reason := m.unsupportedReason(); reason != "" {
		return Status{Supported: false, Reason: reason}
	}
	enabled, err := m.Launcher.Enabled()
	if err != nil {
		return Status{Supported: true, Reason: "Não foi possível consultar: " + err.Error()}
	}
	return Status{Enabled: enabled, Supported: true}
}

// SetEnabled liga (apontando para o executável atual com --minimized) ou desliga.
func (m Manager) SetEnabled(enabled bool) (Status, error) {
	if reason := m.unsupportedReason(); reason != "" {
		return Status{Supported: false, Reason: reason}, errors.New(reason)
	}
	if !enabled {
		if err := m.Launcher.Disable(); err != nil {
			return m.Status(), err
		}
		return m.Status(), nil
	}
	exe, err := m.Executable()
	if err != nil {
		return m.Status(), err
	}
	if err := m.Launcher.Enable(exe, []string{MinimizedFlag}); err != nil {
		return m.Status(), err
	}
	return m.Status(), nil
}
