package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckUpdatesOnStartupLigadoPorPadrao(t *testing.T) {
	fakeKeyring(t)
	m, err := newManagerAt(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if !m.GetAppSettings().CheckUpdatesOnStartup {
		t.Error("instalação nova deveria checar atualizações ao iniciar")
	}
}

// Um config.json gravado por versão anterior não tem o campo: ele precisa
// continuar ligado, não virar false.
func TestCheckUpdatesOnStartupAusenteNoJSONFicaLigado(t *testing.T) {
	fakeKeyring(t)
	dir := t.TempDir()
	antigo := `{"appSettings":{"darkMode":true,"autoUpdate":false,"language":"pt-BR"}}`
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(antigo), 0600); err != nil {
		t.Fatal(err)
	}

	m, err := newManagerAt(dir)
	if err != nil {
		t.Fatal(err)
	}
	s := m.GetAppSettings()
	if !s.CheckUpdatesOnStartup || !s.DarkMode {
		t.Errorf("configurações carregadas: %+v", s)
	}
}

func TestCheckUpdatesOnStartupDesligadoPersiste(t *testing.T) {
	fakeKeyring(t)
	dir := t.TempDir()
	m, err := newManagerAt(dir)
	if err != nil {
		t.Fatal(err)
	}
	s := m.GetAppSettings()
	s.CheckUpdatesOnStartup = false
	if err := m.SetAppSettings(s); err != nil {
		t.Fatal(err)
	}

	recarregado, err := newManagerAt(dir)
	if err != nil {
		t.Fatal(err)
	}
	if recarregado.GetAppSettings().CheckUpdatesOnStartup {
		t.Error("a escolha de desligar deveria sobreviver a um reinício")
	}
}
