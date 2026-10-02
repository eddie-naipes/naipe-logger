package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"logTime-go/backend/api"
)

// Testes de robustez da persistência: token aparado, rollback do cofre,
// arquivos corrompidos, permissões, escrita atômica e migração do diretório do
// executável.

func lerArquivo(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("erro ao ler %s: %v", path, err)
	}
	return string(data)
}

func escreverArquivo(t *testing.T, path, conteudo string, perm os.FileMode) {
	t.Helper()
	if err := os.WriteFile(path, []byte(conteudo), perm); err != nil {
		t.Fatalf("erro ao escrever %s: %v", path, err)
	}
}

func TestSetConnectionAparaTokenNaMemoriaENoCofre(t *testing.T) {
	peek := fakeKeyring(t)

	m, err := newManagerAt(t.TempDir())
	if err != nil {
		t.Fatalf("newManagerAt falhou: %v", err)
	}

	if err := m.SetConnection("teamwork.empresa.com.br", 1, "  tok-com-espacos \n"); err != nil {
		t.Fatalf("SetConnection falhou: %v", err)
	}

	noCofre, _ := peek()
	naMemoria := m.GetTeamworkConfig().AuthToken
	if noCofre != "tok-com-espacos" || naMemoria != noCofre {
		t.Errorf("cofre = %q, memória = %q; esperava ambos \"tok-com-espacos\"", noCofre, naMemoria)
	}
}

func TestSetConnectionRecusaTokenVazio(t *testing.T) {
	peek := fakeKeyring(t)

	m, err := newManagerAt(t.TempDir())
	if err != nil {
		t.Fatalf("newManagerAt falhou: %v", err)
	}

	if err := m.SetConnection("teamwork.empresa.com.br", 1, "   "); err == nil {
		t.Error("SetConnection aceitou token só com espaços")
	}
	if _, ok := peek(); ok {
		t.Error("token vazio chegou ao cofre")
	}
}

// impedirGravacao troca config.json por um diretório, o que faz o rename da
// escrita atômica falhar em qualquer sistema operacional.
func impedirGravacao(t *testing.T, dir string) {
	t.Helper()
	path := filepath.Join(dir, "config.json")
	if err := os.RemoveAll(path); err != nil {
		t.Fatalf("erro ao remover config.json: %v", err)
	}
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatalf("erro ao criar diretório no lugar de config.json: %v", err)
	}
}

func TestSetConnectionRestauraTokenAnteriorQuandoSaveFalha(t *testing.T) {
	peek := fakeKeyring(t)
	dir := t.TempDir()

	m, err := newManagerAt(dir)
	if err != nil {
		t.Fatalf("newManagerAt falhou: %v", err)
	}
	if err := m.SetConnection("teamwork.empresa.com.br", 1, "tok-antigo"); err != nil {
		t.Fatalf("SetConnection inicial falhou: %v", err)
	}

	impedirGravacao(t, dir)

	if err := m.SetConnection("outra.empresa.com.br", 2, "tok-novo"); err == nil {
		t.Fatal("SetConnection devolveu nil apesar da falha ao gravar config.json")
	}

	if got, _ := peek(); got != "tok-antigo" {
		t.Errorf("cofre = %q após falha, esperava o token anterior restaurado", got)
	}
	cfg := m.GetTeamworkConfig()
	if cfg.AuthToken != "tok-antigo" || cfg.UserID != 1 || cfg.ApiHost != "https://teamwork.empresa.com.br" {
		t.Errorf("memória alterada apesar da falha: %+v", cfg)
	}
}

func TestSetConnectionLimpaCofreQuandoSaveFalhaSemTokenAnterior(t *testing.T) {
	peek := fakeKeyring(t)
	dir := t.TempDir()

	m, err := newManagerAt(dir)
	if err != nil {
		t.Fatalf("newManagerAt falhou: %v", err)
	}

	impedirGravacao(t, dir)

	if err := m.SetConnection("teamwork.empresa.com.br", 1, "tok-novo"); err == nil {
		t.Fatal("SetConnection devolveu nil apesar da falha ao gravar config.json")
	}
	if got, ok := peek(); ok {
		t.Errorf("cofre ficou com %q após falha, esperava vazio", got)
	}
	if m.GetTeamworkConfig().AuthToken != "" {
		t.Error("token ficou na memória apesar da falha")
	}
}

func backupsCom(t *testing.T, dir, prefixo string) []string {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(dir, prefixo+".corrompido-*"))
	if err != nil {
		t.Fatalf("glob falhou: %v", err)
	}
	return matches
}

func TestLoadComConfigCorrompidoSegueComPadrao(t *testing.T) {
	fakeKeyring(t)
	dir := t.TempDir()
	escreverArquivo(t, filepath.Join(dir, "config.json"), `{"teamworkConfig": {"userId": 4`, 0600)

	m, err := newManagerAt(dir)
	if err != nil {
		t.Fatalf("newManagerAt não deveria falhar com JSON corrompido: %v", err)
	}

	if got := m.GetTeamworkConfig().MinutosPorDia; got != 480 {
		t.Errorf("MinutosPorDia = %d, esperava o padrão 480", got)
	}

	backups := m.CorruptedConfigBackups()
	if len(backups) != 1 {
		t.Fatalf("CorruptedConfigBackups = %v, esperava 1 backup", backups)
	}
	if got := backupsCom(t, dir, "config.json"); len(got) != 1 || got[0] != backups[0] {
		t.Errorf("backup no disco = %v, esperava %q", got, backups[0])
	}
	if lerArquivo(t, backups[0]) != `{"teamworkConfig": {"userId": 4` {
		t.Error("conteúdo corrompido não foi preservado no backup")
	}
	if _, err := os.Stat(filepath.Join(dir, "config.json")); !os.IsNotExist(err) {
		t.Errorf("config.json corrompido continua no lugar (err=%v)", err)
	}

	// O app continua utilizável: a próxima gravação cria um config.json válido.
	if err := m.SetMinutosPorDia(360); err != nil {
		t.Fatalf("SetMinutosPorDia falhou após recuperação: %v", err)
	}
}

func TestLoadComTemplatesCorrompidoSegueSemTemplates(t *testing.T) {
	fakeKeyring(t)
	dir := t.TempDir()
	escreverArquivo(t, filepath.Join(dir, "templates.json"), `[1, 2`, 0600)

	m, err := newManagerAt(dir)
	if err != nil {
		t.Fatalf("newManagerAt não deveria falhar com templates corrompidos: %v", err)
	}

	if len(m.GetTemplates()) != 0 {
		t.Errorf("GetTemplates = %v, esperava vazio", m.GetTemplates())
	}
	if len(m.CorruptedConfigBackups()) != 1 || len(backupsCom(t, dir, "templates.json")) != 1 {
		t.Errorf("backup de templates.json não registrado: %v", m.CorruptedConfigBackups())
	}
	if err := m.SaveTemplate(api.Template{Name: "novo"}); err != nil {
		t.Fatalf("SaveTemplate falhou após recuperação: %v", err)
	}
}

func TestCorruptedConfigBackupsNuncaENil(t *testing.T) {
	fakeKeyring(t)

	m, err := newManagerAt(t.TempDir())
	if err != nil {
		t.Fatalf("newManagerAt falhou: %v", err)
	}
	if got := m.CorruptedConfigBackups(); got == nil || len(got) != 0 {
		t.Errorf("CorruptedConfigBackups = %#v, esperava slice vazio não-nil", got)
	}
}

func TestTemplatesNullNaoCausaPanicAoSalvar(t *testing.T) {
	fakeKeyring(t)
	dir := t.TempDir()
	escreverArquivo(t, filepath.Join(dir, "templates.json"), `null`, 0600)

	m, err := newManagerAt(dir)
	if err != nil {
		t.Fatalf("newManagerAt falhou: %v", err)
	}

	if err := m.SaveTemplate(api.Template{Name: "tpl"}); err != nil {
		t.Fatalf("SaveTemplate falhou: %v", err)
	}
	if _, ok := m.GetTemplate("tpl"); !ok {
		t.Error("template salvo não encontrado")
	}
	if len(m.CorruptedConfigBackups()) != 0 {
		t.Error("\"null\" não é corrupção e não deveria gerar backup")
	}
}

func TestConfigNullUsaPadroes(t *testing.T) {
	fakeKeyring(t)
	dir := t.TempDir()
	escreverArquivo(t, filepath.Join(dir, "config.json"), `null`, 0600)

	m, err := newManagerAt(dir)
	if err != nil {
		t.Fatalf("newManagerAt falhou: %v", err)
	}
	if m.GetTeamworkConfig().MinutosPorDia != 480 || m.GetAppSettings().Language != "pt-BR" {
		t.Errorf("padrões perdidos: %+v / %+v", m.GetTeamworkConfig(), m.GetAppSettings())
	}
	if m.GetSavedTasks() == nil {
		t.Error("GetSavedTasks devolveu nil")
	}
}

func TestPermissoesSaoRestringidas(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permissões Unix não se aplicam no Windows")
	}
	fakeKeyring(t)
	dir := filepath.Join(t.TempDir(), "cfg")
	if err := os.Mkdir(dir, 0755); err != nil {
		t.Fatalf("erro ao criar diretório: %v", err)
	}
	escreverArquivo(t, filepath.Join(dir, "config.json"), `{}`, 0644)
	escreverArquivo(t, filepath.Join(dir, "templates.json"), `{}`, 0644)

	m, err := newManagerAt(dir)
	if err != nil {
		t.Fatalf("newManagerAt falhou: %v", err)
	}

	perm := func(path string) os.FileMode {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat %s: %v", path, err)
		}
		return info.Mode().Perm()
	}

	if got := perm(dir); got != 0700 {
		t.Errorf("diretório com %o, esperava 0700", got)
	}
	for _, name := range []string{"config.json", "templates.json"} {
		if got := perm(filepath.Join(dir, name)); got != 0600 {
			t.Errorf("%s existente com %o, esperava 0600", name, got)
		}
	}

	if err := m.SaveTemplate(api.Template{Name: "tpl"}); err != nil {
		t.Fatalf("SaveTemplate falhou: %v", err)
	}
	if err := m.Save(); err != nil {
		t.Fatalf("Save falhou: %v", err)
	}
	for _, name := range []string{"config.json", "templates.json"} {
		if got := perm(filepath.Join(dir, name)); got != 0600 {
			t.Errorf("%s regravado com %o, esperava 0600", name, got)
		}
	}
}

func TestEscritaAtomicaNaoDeixaTemporarios(t *testing.T) {
	fakeKeyring(t)
	dir := t.TempDir()

	m, err := newManagerAt(dir)
	if err != nil {
		t.Fatalf("newManagerAt falhou: %v", err)
	}
	for i := 0; i < 3; i++ {
		if err := m.AddSavedTask(api.Task{TaskID: i}); err != nil {
			t.Fatalf("AddSavedTask falhou: %v", err)
		}
		if err := m.SaveTemplate(api.Template{Name: "tpl"}); err != nil {
			t.Fatalf("SaveTemplate falhou: %v", err)
		}
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir falhou: %v", err)
	}
	for _, e := range entries {
		if strings.Contains(e.Name(), ".tmp-") {
			t.Errorf("arquivo temporário esquecido: %s", e.Name())
		}
	}

	// Uma falha no rename também não pode deixar lixo para trás.
	impedirGravacao(t, dir)
	if err := m.Save(); err == nil {
		t.Fatal("Save devolveu nil com config.json bloqueado")
	}
	entries, _ = os.ReadDir(dir)
	for _, e := range entries {
		if strings.Contains(e.Name(), ".tmp-") {
			t.Errorf("arquivo temporário esquecido após falha: %s", e.Name())
		}
	}
}

func TestMigracaoMoveArquivosQuandoDestinoNaoExiste(t *testing.T) {
	execDir := t.TempDir()
	configDir := filepath.Join(t.TempDir(), ".teamwork-logger")
	escreverArquivo(t, filepath.Join(execDir, "config.json"), `{"teamworkConfig":{"userId":7}}`, 0644)
	escreverArquivo(t, filepath.Join(execDir, "templates.json"), `{"a":{"name":"a"}}`, 0644)

	purged, err := migrateLegacyFiles(execDir, configDir)
	if err != nil {
		t.Fatalf("migrateLegacyFiles falhou: %v", err)
	}
	if purged {
		t.Error("purged = true sem credencial antiga")
	}

	for name, want := range map[string]string{
		"config.json":    `{"teamworkConfig":{"userId":7}}`,
		"templates.json": `{"a":{"name":"a"}}`,
	} {
		if got := lerArquivo(t, filepath.Join(configDir, name)); got != want {
			t.Errorf("%s migrado = %q, esperava %q", name, got, want)
		}
		if _, err := os.Stat(filepath.Join(execDir, name)); !os.IsNotExist(err) {
			t.Errorf("%s original não foi removido (err=%v)", name, err)
		}
	}
}

func TestMigracaoNaoSobrescreveDestinoExistente(t *testing.T) {
	execDir := t.TempDir()
	configDir := t.TempDir()
	escreverArquivo(t, filepath.Join(execDir, "config.json"), `{"velho":true}`, 0600)
	escreverArquivo(t, filepath.Join(execDir, "templates.json"), `{"velho":{}}`, 0600)
	escreverArquivo(t, filepath.Join(configDir, "config.json"), `{"atual":true}`, 0600)
	escreverArquivo(t, filepath.Join(configDir, "templates.json"), `{"atual":{}}`, 0600)

	if _, err := migrateLegacyFiles(execDir, configDir); err != nil {
		t.Fatalf("migrateLegacyFiles falhou: %v", err)
	}

	if got := lerArquivo(t, filepath.Join(configDir, "config.json")); got != `{"atual":true}` {
		t.Errorf("config.json atual sobrescrito: %q", got)
	}
	if got := lerArquivo(t, filepath.Join(configDir, "templates.json")); got != `{"atual":{}}` {
		t.Errorf("templates.json atual sobrescrito: %q", got)
	}
	// A cópia antiga, sem segredo, é mantida para o usuário decidir.
	if _, err := os.Stat(filepath.Join(execDir, "config.json")); err != nil {
		t.Errorf("config.json antigo deveria ter sido mantido: %v", err)
	}
}

func TestMigracaoApagaCredencialAntigaMesmoComDestinoExistente(t *testing.T) {
	execDir := t.TempDir()
	configDir := t.TempDir()
	escreverArquivo(t, filepath.Join(execDir, "config.json"), legacyConfigJSON, 0600)
	escreverArquivo(t, filepath.Join(configDir, "config.json"), `{"atual":true}`, 0600)

	purged, err := migrateLegacyFiles(execDir, configDir)
	if err != nil {
		t.Fatalf("migrateLegacyFiles falhou: %v", err)
	}
	if !purged {
		t.Error("purged = false, esperava true ao apagar credencial antiga")
	}
	if _, err := os.Stat(filepath.Join(execDir, "config.json")); !os.IsNotExist(err) {
		t.Errorf("config.json com credencial antiga continua em disco (err=%v)", err)
	}
	if got := lerArquivo(t, filepath.Join(configDir, "config.json")); got != `{"atual":true}` {
		t.Errorf("config.json atual sobrescrito: %q", got)
	}
}

func TestMigracaoNaoRepeteQuandoRemocaoFalha(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("diretório somente leitura não impede remoção no Windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("root ignora permissões de diretório")
	}

	execDir := t.TempDir()
	configDir := t.TempDir()
	escreverArquivo(t, filepath.Join(execDir, "config.json"), `{"versao":1}`, 0600)
	if err := os.Chmod(execDir, 0500); err != nil {
		t.Fatalf("chmod falhou: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(execDir, 0700) })

	if _, err := migrateLegacyFiles(execDir, configDir); err != nil {
		t.Fatalf("migrateLegacyFiles falhou: %v", err)
	}

	// O usuário altera a configuração migrada; a próxima abertura não pode
	// trazê-la de volta à cópia antiga que ficou no diretório do executável.
	escreverArquivo(t, filepath.Join(configDir, "config.json"), `{"versao":2}`, 0600)
	if _, err := migrateLegacyFiles(execDir, configDir); err != nil {
		t.Fatalf("segunda migração falhou: %v", err)
	}
	if got := lerArquivo(t, filepath.Join(configDir, "config.json")); got != `{"versao":2}` {
		t.Errorf("configuração atual sobrescrita pela cópia antiga: %q", got)
	}
}

func TestMigracaoIgnoraMesmoDiretorio(t *testing.T) {
	dir := t.TempDir()
	escreverArquivo(t, filepath.Join(dir, "config.json"), `{"atual":true}`, 0600)

	if _, err := migrateLegacyFiles(dir, dir); err != nil {
		t.Fatalf("migrateLegacyFiles falhou: %v", err)
	}
	if got := lerArquivo(t, filepath.Join(dir, "config.json")); got != `{"atual":true}` {
		t.Errorf("config.json alterado: %q", got)
	}
}

func TestMigracaoSemArquivosNaoFazNada(t *testing.T) {
	configDir := filepath.Join(t.TempDir(), "nao-criado")

	purged, err := migrateLegacyFiles(t.TempDir(), configDir)
	if err != nil || purged {
		t.Fatalf("migrateLegacyFiles = (%v, %v), esperava (false, nil)", purged, err)
	}
	if _, err := os.Stat(configDir); !os.IsNotExist(err) {
		t.Error("diretório de configuração criado sem nada a migrar")
	}
}
