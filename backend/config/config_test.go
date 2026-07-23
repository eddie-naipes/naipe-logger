package config

import (
	"logTime-go/backend/api"
	"logTime-go/backend/security"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// fakeKeyring troca o cofre do sistema por um em memória durante o teste, para
// não depender de Keychain/Secret Service nem sujar as credenciais reais.
func fakeKeyring(t *testing.T) func() (string, bool) {
	t.Helper()

	var stored *string
	origStore, origLoad, origDelete := storeToken, loadToken, deleteToken

	storeToken = func(token string) error {
		stored = &token
		return nil
	}
	loadToken = func() (string, error) {
		if stored == nil {
			return "", security.ErrNoToken
		}
		return *stored, nil
	}
	deleteToken = func() error {
		stored = nil
		return nil
	}

	t.Cleanup(func() {
		storeToken, loadToken, deleteToken = origStore, origLoad, origDelete
	})

	return func() (string, bool) {
		if stored == nil {
			return "", false
		}
		return *stored, true
	}
}

// config.json das versões antigas: guardava email:senha no campo authToken,
// protegido por uma chave derivável do código-fonte.
const legacyConfigJSON = `{
  "teamworkConfig": {
    "authToken": "dXN1YXJpb0BlbXByZXNhLmNvbTpzZW5oYTEyMw==",
    "userId": 42,
    "apiHost": "teamwork.empresa.com.br",
    "minutosPorDia": 480
  },
  "savedTasks": [],
  "appSettings": { "darkMode": true, "language": "pt-BR" }
}`

func TestLoadExpurgaCredencialAntiga(t *testing.T) {
	fakeKeyring(t)
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.json")

	if err := os.WriteFile(configPath, []byte(legacyConfigJSON), 0600); err != nil {
		t.Fatalf("erro ao preparar config legado: %v", err)
	}

	m, err := newManagerAt(dir)
	if err != nil {
		t.Fatalf("newManagerAt falhou: %v", err)
	}

	if !m.LegacyCredentialPurged() {
		t.Error("LegacyCredentialPurged() = false, esperava true para config legado")
	}

	rewritten, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("erro ao reler config: %v", err)
	}

	if strings.Contains(string(rewritten), "authToken") {
		t.Errorf("campo authToken sobreviveu no disco:\n%s", rewritten)
	}
	if strings.Contains(string(rewritten), "dXN1YXJpb0BlbXByZXNhLmNvbTpzZW5oYTEyMw==") {
		t.Errorf("credencial antiga sobreviveu no disco:\n%s", rewritten)
	}

	// O resto da configuração precisa ser preservado — só a credencial sai.
	cfg := m.GetTeamworkConfig()
	if cfg.UserID != 42 {
		t.Errorf("UserID = %d, esperava 42", cfg.UserID)
	}
	if cfg.ApiHost != "teamwork.empresa.com.br" {
		t.Errorf("ApiHost = %q, esperava preservado", cfg.ApiHost)
	}
	if !m.GetAppSettings().DarkMode {
		t.Error("AppSettings.DarkMode perdido durante o expurgo")
	}
}

func TestLoadSemCredencialAntigaNaoAlertaUsuario(t *testing.T) {
	fakeKeyring(t)
	dir := t.TempDir()
	current := `{"teamworkConfig":{"userId":7,"apiHost":"https://teamwork.empresa.com.br","minutosPorDia":480}}`

	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(current), 0600); err != nil {
		t.Fatalf("erro ao preparar config: %v", err)
	}

	m, err := newManagerAt(dir)
	if err != nil {
		t.Fatalf("newManagerAt falhou: %v", err)
	}

	if m.LegacyCredentialPurged() {
		t.Error("LegacyCredentialPurged() = true para config já no formato novo")
	}
}

func TestSaveNuncaGravaToken(t *testing.T) {
	fakeKeyring(t)
	dir := t.TempDir()

	m, err := newManagerAt(dir)
	if err != nil {
		t.Fatalf("newManagerAt falhou: %v", err)
	}

	// Simula uma sessão ativa: o token existe em memória, mas não pode vazar
	// para o disco quando qualquer outra configuração é salva.
	m.appConfig.TeamworkConfig.AuthToken = "tok-super-secreto"
	m.appConfig.TeamworkConfig.ApiHost = "https://teamwork.empresa.com.br"

	if err := m.SetAppSettings(AppSettings{DarkMode: true, Language: "pt-BR"}); err != nil {
		t.Fatalf("SetAppSettings falhou: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "config.json"))
	if err != nil {
		t.Fatalf("erro ao ler config: %v", err)
	}

	if strings.Contains(string(data), "tok-super-secreto") {
		t.Errorf("token vazou para config.json:\n%s", data)
	}
}

func TestManagerNaoDeadlockNasEscritas(t *testing.T) {
	fakeKeyring(t)
	dir := t.TempDir()

	m, err := newManagerAt(dir)
	if err != nil {
		t.Fatalf("newManagerAt falhou: %v", err)
	}

	// Estes caminhos chamavam Save()/SaveTemplates() já segurando o lock; com
	// eles passando a adquirir o lock, um mutex não reentrante travaria aqui.
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = m.AddSavedTask(api.Task{TaskID: 1, TaskName: "t"})
		_ = m.AddSavedTask(api.Task{TaskID: 1, TaskName: "t atualizada"})
		_ = m.RemoveSavedTask(1)
		_ = m.SaveTemplate(api.Template{Name: "tpl"})
		_ = m.DeleteTemplate("tpl")
		_ = m.SetSavedTasks(nil)
		_ = m.SaveTemplates()
		_ = m.Save()
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("deadlock: escritas no Manager não terminaram em 5s")
	}
}

func TestSetConnectionGuardaTokenNoCofreNaoNoDisco(t *testing.T) {
	peek := fakeKeyring(t)
	dir := t.TempDir()

	m, err := newManagerAt(dir)
	if err != nil {
		t.Fatalf("newManagerAt falhou: %v", err)
	}

	if err := m.SetConnection("teamwork.empresa.com.br", 99, "tok-secreto"); err != nil {
		t.Fatalf("SetConnection falhou: %v", err)
	}

	// O token vai para o cofre...
	got, ok := peek()
	if !ok || got != "tok-secreto" {
		t.Errorf("cofre contém %q (presente=%v), esperava \"tok-secreto\"", got, ok)
	}

	// ...e o host é normalizado para https no disco, sem o token.
	data, err := os.ReadFile(filepath.Join(dir, "config.json"))
	if err != nil {
		t.Fatalf("erro ao ler config: %v", err)
	}
	if strings.Contains(string(data), "tok-secreto") {
		t.Errorf("token vazou para o disco:\n%s", data)
	}
	if !strings.Contains(string(data), "https://teamwork.empresa.com.br") {
		t.Errorf("host não foi normalizado para https:\n%s", data)
	}

	// ClearConnection esvazia o cofre.
	if err := m.ClearConnection(); err != nil {
		t.Fatalf("ClearConnection falhou: %v", err)
	}
	if _, ok := peek(); ok {
		t.Error("token permaneceu no cofre após ClearConnection")
	}
}

func TestSetConnectionRecusaHostInseguro(t *testing.T) {
	peek := fakeKeyring(t)
	dir := t.TempDir()

	m, err := newManagerAt(dir)
	if err != nil {
		t.Fatalf("newManagerAt falhou: %v", err)
	}

	if err := m.SetConnection("http://teamwork.empresa.com.br", 99, "tok"); err == nil {
		t.Error("SetConnection aceitou host http://")
	}
	if _, ok := peek(); ok {
		t.Error("token foi gravado no cofre apesar do host inválido")
	}
}
