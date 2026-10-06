package autostart

import (
	"strings"
)

// RunValueName é o nome do valor em HKCU\...\CurrentVersion\Run.
const RunValueName = "TeamworkLogger"

// ValueStore abstrai a chave Run do registro (no Windows, registry.Key; nos
// testes, um mapa).
type ValueStore interface {
	// GetString devolve o valor e se ele existe.
	GetString(name string) (string, bool, error)
	SetString(name, value string) error
	// Delete remove o valor; remover um valor inexistente não é erro.
	Delete(name string) error
}

// RunKey é o Launcher do Windows: um valor na chave Run do usuário (HKCU, sem
// precisar de administrador) com a linha de comando do app.
type RunKey struct {
	Store ValueStore
	Name  string
}

func (r RunKey) name() string {
	if r.Name == "" {
		return RunValueName
	}
	return r.Name
}

// WindowsCommandLine monta "exe" args, com o executável sempre entre aspas
// (caminhos como C:\Program Files\... têm espaços).
func WindowsCommandLine(exe string, args []string) string {
	parts := []string{`"` + exe + `"`}
	for _, a := range args {
		if strings.ContainsAny(a, " \t\"") {
			a = `"` + strings.ReplaceAll(a, `"`, `\"`) + `"`
		}
		parts = append(parts, a)
	}
	return strings.Join(parts, " ")
}

func (r RunKey) Enable(exe string, args []string) error {
	cmd := WindowsCommandLine(exe, args)
	if atual, ok, err := r.Store.GetString(r.name()); err == nil && ok && atual == cmd {
		return nil
	}
	return r.Store.SetString(r.name(), cmd)
}

func (r RunKey) Disable() error {
	return r.Store.Delete(r.name())
}

func (r RunKey) Enabled() (bool, error) {
	_, ok, err := r.Store.GetString(r.name())
	return ok, err
}
