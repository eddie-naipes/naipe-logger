package logging

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// restauraPadrao devolve o slog padrão ao estado anterior ao teste, já que
// Setup o substitui globalmente.
func restauraPadrao(t *testing.T) {
	t.Helper()
	original := slog.Default()
	t.Cleanup(func() {
		slog.SetDefault(original)
		ClearSecrets()
	})
}

func TestIsTruthyLigaDebugSoComValorExplicito(t *testing.T) {
	for _, v := range []string{"1", "true", "TRUE", " yes ", "on", "sim"} {
		if !IsTruthy(v) {
			t.Errorf("IsTruthy(%q) = false, esperava true", v)
		}
	}
	for _, v := range []string{"", "0", "false", "não", "talvez"} {
		if IsTruthy(v) {
			t.Errorf("IsTruthy(%q) = true, esperava false", v)
		}
	}
}

func TestSetupGravaArquivoComNivelInfoPorPadrao(t *testing.T) {
	restauraPadrao(t)
	dir := filepath.Join(t.TempDir(), "logs")

	logger, err := Setup(Options{Dir: dir})
	if err != nil {
		t.Fatalf("Setup: %v", err)
	}
	slog.Debug("detalhe invisivel")
	slog.Info("evento visivel", "ano", 2026)
	_ = logger.Close()

	data, err := os.ReadFile(filepath.Join(dir, LogFileName))
	if err != nil {
		t.Fatalf("lendo log: %v", err)
	}
	if strings.Contains(string(data), "detalhe invisivel") {
		t.Error("Debug não deveria ser gravado sem TEAMWORK_LOGGER_DEBUG")
	}
	if !strings.Contains(string(data), "evento visivel") || !strings.Contains(string(data), "ano=2026") {
		t.Errorf("log sem a entrada Info: %q", data)
	}
	if logger.Dir() != dir {
		t.Errorf("Dir() = %q, esperava %q", logger.Dir(), dir)
	}

	if runtime.GOOS != "windows" {
		info, _ := os.Stat(filepath.Join(dir, LogFileName))
		if info.Mode().Perm() != 0600 {
			t.Errorf("permissão do log = %v, esperava 0600", info.Mode().Perm())
		}
		dinfo, _ := os.Stat(dir)
		if dinfo.Mode().Perm() != 0700 {
			t.Errorf("permissão da pasta = %v, esperava 0700", dinfo.Mode().Perm())
		}
	}
}

func TestSetupDebugEEspelhoNoStderr(t *testing.T) {
	restauraPadrao(t)
	var stderr bytes.Buffer

	logger, err := Setup(Options{Dir: t.TempDir(), Debug: true, Stderr: true, StderrOut: &stderr})
	if err != nil {
		t.Fatalf("Setup: %v", err)
	}
	defer logger.Close()

	slog.Debug("diagnostico ligado")
	if !strings.Contains(stderr.String(), "diagnostico ligado") {
		t.Errorf("Debug deveria aparecer no stderr com Debug+Stderr, saída: %q", stderr.String())
	}
}

func TestSetupSemArquivoCaiParaStderr(t *testing.T) {
	restauraPadrao(t)
	// Um arquivo no lugar da pasta impede criar o diretório de logs.
	bloqueio := filepath.Join(t.TempDir(), "arquivo")
	if err := os.WriteFile(bloqueio, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer

	logger, err := Setup(Options{Dir: filepath.Join(bloqueio, "logs"), StderrOut: &stderr})
	if err == nil {
		t.Fatal("esperava erro ao abrir o log em pasta impossível")
	}
	slog.Info("ainda registra")
	if !strings.Contains(stderr.String(), "ainda registra") {
		t.Errorf("sem arquivo, o log deveria ir para o stderr: %q", stderr.String())
	}
	if logger.Dir() != "" {
		t.Errorf("Dir() deveria ser vazio sem arquivo, veio %q", logger.Dir())
	}
}

type textoComToken struct{ valor string }

func (x textoComToken) String() string { return "conteudo " + x.valor }

// O token nunca pode chegar ao disco, venha ele na mensagem, num atributo
// string, num erro, num Stringer, num slice ou já codificado em Basic auth.
func TestTokenNuncaApareceNoLog(t *testing.T) {
	restauraPadrao(t)
	dir := t.TempDir()
	const token = "twp_SEGREDO_abc123XYZ"
	basic := base64.StdEncoding.EncodeToString([]byte(token + ":X"))

	logger, err := Setup(Options{Dir: dir, Debug: true})
	if err != nil {
		t.Fatalf("Setup: %v", err)
	}
	AddSecret(token)
	AddSecret(basic)

	slog.Info("mensagem com "+token, "valor", token)
	slog.Warn("falha", "err", errors.New("requisição falhou: Basic "+basic))
	slog.Debug("stringer", "obj", textoComToken{token})
	slog.Debug("lista", "itens", []string{"a", token})
	slog.Info("grupo", slog.Group("req", slog.String("auth", token)))
	slog.With("fixo", token).Info("com atributo fixo")
	slog.Info("rotulado", "corpo", `{"password":"hunter2"}`)
	_ = logger.Close()

	data, err := os.ReadFile(filepath.Join(dir, LogFileName))
	if err != nil {
		t.Fatal(err)
	}
	log := string(data)
	for _, proibido := range []string{token, basic, "hunter2"} {
		if strings.Contains(log, proibido) {
			t.Errorf("log contém segredo %q:\n%s", proibido, log)
		}
	}
	if !strings.Contains(log, Redacted) {
		t.Errorf("esperava marcas %s no log:\n%s", Redacted, log)
	}
	if !strings.Contains(log, "com atributo fixo") {
		t.Error("as mensagens em si deveriam continuar no log")
	}
}

func TestAddSecretIgnoraValoresCurtos(t *testing.T) {
	t.Cleanup(ClearSecrets)
	AddSecret("abc")
	if got := RedactSecrets("abc def"); got != "abc def" {
		t.Errorf("valor curto não deveria ser mascarado: %q", got)
	}
}

func TestSanitizeMantemRegraDePalavraChave(t *testing.T) {
	for _, s := range []string{"meu token", "Password=1", "TOKEN"} {
		if Sanitize(s) != Redacted {
			t.Errorf("Sanitize(%q) deveria mascarar", s)
		}
	}
	if Sanitize("feriado de 2026") != "feriado de 2026" {
		t.Error("texto comum não deveria mudar")
	}
}

func TestRotatingFileGiraEMantemSoOsBackupsConfigurados(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "app.log")

	r, err := OpenRotatingFile(path, 100, 2)
	if err != nil {
		t.Fatal(err)
	}
	linha := strings.Repeat("x", 39) + "\n" // 40 bytes: 2 linhas por arquivo
	for i := 0; i < 10; i++ {
		if _, err := fmt.Fprint(r, linha); err != nil {
			t.Fatalf("escrita %d: %v", i, err)
		}
	}
	if err := r.Close(); err != nil {
		t.Fatal(err)
	}

	for _, nome := range []string{"app.log", "app.log.1", "app.log.2"} {
		info, err := os.Stat(filepath.Join(dir, nome))
		if err != nil {
			t.Errorf("%s deveria existir: %v", nome, err)
			continue
		}
		if info.Size() > 100 {
			t.Errorf("%s passou do limite: %d bytes", nome, info.Size())
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "app.log.3")); !os.IsNotExist(err) {
		t.Error("app.log.3 não deveria existir com maxBackups=2")
	}
}

func TestRotatingFileGravaEntradaMaiorQueOLimite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.log")
	r, err := OpenRotatingFile(path, 10, 1)
	if err != nil {
		t.Fatal(err)
	}
	grande := strings.Repeat("y", 50)
	if n, err := r.Write([]byte(grande)); err != nil || n != 50 {
		t.Fatalf("Write = %d, %v", n, err)
	}
	_ = r.Close()
	data, _ := os.ReadFile(path)
	if string(data) != grande {
		t.Errorf("entrada grande perdida: %q", data)
	}
}

func TestRotatingFileContinuaDeOndeParou(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app.log")
	if err := os.WriteFile(path, []byte(strings.Repeat("z", 90)), 0644); err != nil {
		t.Fatal(err)
	}
	r, err := OpenRotatingFile(path, 100, 1)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = r.Write([]byte(strings.Repeat("w", 20)))
	_ = r.Close()

	// O arquivo pré-existente já tinha 90 bytes: os 20 novos forçam a rotação.
	if _, err := os.Stat(path + ".1"); err != nil {
		t.Errorf("esperava rotação considerando o tamanho anterior: %v", err)
	}
}
