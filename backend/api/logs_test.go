package api

import (
	"bytes"
	"encoding/base64"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"logTime-go/backend/logging"
)

// capturaLog troca o slog padrão por um handler em memória (com o mesmo
// mascaramento da produção) enquanto fn roda.
func capturaLog(t *testing.T, level slog.Level, fn func()) string {
	t.Helper()
	var buf bytes.Buffer
	original := slog.Default()
	slog.SetDefault(slog.New(logging.NewRedactingHandler(
		slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: level}))))
	defer slog.SetDefault(original)
	fn()
	return buf.String()
}

func TestSanitizeForLogMascaraValoresSensiveis(t *testing.T) {
	if got := sanitizeForLog("meu token secreto"); got != logging.Redacted {
		t.Errorf("sanitizeForLog = %v, esperava %s", got, logging.Redacted)
	}
	if got := sanitizeForLog(42); got != 42 {
		t.Errorf("valores não string não deveriam mudar: %v", got)
	}
}

// O token de um cliente criado com NewTeamworkAPI nunca pode sair no log,
// mesmo quando aparece dentro de um erro ou já em Basic auth.
func TestTokenDoClienteNuncaApareceNoLog(t *testing.T) {
	t.Cleanup(logging.ClearSecrets)
	const token = "tkn_cliente_9f8e7d6c5b"
	basic := base64.StdEncoding.EncodeToString([]byte(token + ":X"))

	_ = NewTeamworkAPI(Config{AuthToken: token, ApiHost: "empresa.teamwork.com"})

	saida := capturaLog(t, slog.LevelDebug, func() {
		slog.Warn("falha "+token, "err", errors.New("Authorization: Basic "+basic), "valor", token)
	})
	if strings.Contains(saida, token) || strings.Contains(saida, basic) {
		t.Errorf("token vazou no log: %q", saida)
	}
}

func TestDebugSilenciosoNoNivelInfo(t *testing.T) {
	saida := capturaLog(t, slog.LevelInfo, func() {
		slog.Debug("detalhe", "x", 1)
	})
	if saida != "" {
		t.Errorf("Debug não deveria sair no nível Info: %q", saida)
	}
}
