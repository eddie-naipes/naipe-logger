package api

import (
	"io"
	"os"
	"strings"
	"testing"
)

func TestIsTruthyLigaDebugSoComValorExplicito(t *testing.T) {
	for _, v := range []string{"1", "true", "TRUE", " yes ", "on", "sim"} {
		if !isTruthy(v) {
			t.Errorf("isTruthy(%q) = false, esperava true", v)
		}
	}
	for _, v := range []string{"", "0", "false", "não", "talvez"} {
		if isTruthy(v) {
			t.Errorf("isTruthy(%q) = true, esperava false", v)
		}
	}
}

// capturaStdout devolve o que fn escreveu na saída padrão.
func capturaStdout(t *testing.T, fn func()) string {
	t.Helper()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	original := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = original }()

	fn()

	_ = w.Close()
	saida, _ := io.ReadAll(r)
	return string(saida)
}

func TestLogDebugSilenciosoSemFlagESanitizado(t *testing.T) {
	original := debugLogging
	t.Cleanup(func() { debugLogging = original })

	api := &TeamworkAPI{}

	debugLogging = false
	if saida := capturaStdout(t, func() { api.logDebug("detalhe %s", "x") }); saida != "" {
		t.Errorf("logDebug sem a flag escreveu %q", saida)
	}

	debugLogging = true
	saida := capturaStdout(t, func() { api.logDebug("valor: %s", "meu token secreto") })
	if strings.Contains(saida, "secreto") || !strings.Contains(saida, "[REDACTED]") {
		t.Errorf("logDebug deveria passar por sanitizeForLog, escreveu %q", saida)
	}
}
