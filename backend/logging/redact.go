package logging

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
)

// Redacted substitui qualquer valor sensível nos logs.
const Redacted = "[REDACTED]"

var (
	secretsMu sync.RWMutex
	// secrets guarda os valores exatos que nunca podem chegar ao log (o token
	// de API e sua forma em Basic auth). A regra por palavra-chave de Sanitize
	// só pega valores rotulados; esta pega o segredo em qualquer posição,
	// inclusive dentro da mensagem ou de um erro.
	secrets = map[string]struct{}{}
)

// AddSecret registra um valor que deve ser mascarado em toda mensagem e
// atributo de log. Valores curtos demais são ignorados: mascarar "a" ou "1"
// destruiria os logs sem proteger nada.
func AddSecret(value string) {
	value = strings.TrimSpace(value)
	if len(value) < 6 {
		return
	}
	secretsMu.Lock()
	secrets[value] = struct{}{}
	secretsMu.Unlock()
}

// ClearSecrets esquece os segredos registrados (ex.: após o logout), para não
// manter o token antigo em memória além do necessário.
func ClearSecrets() {
	secretsMu.Lock()
	secrets = map[string]struct{}{}
	secretsMu.Unlock()
}

// RedactSecrets troca toda ocorrência de um segredo registrado por Redacted.
func RedactSecrets(s string) string {
	secretsMu.RLock()
	defer secretsMu.RUnlock()
	for secret := range secrets {
		if strings.Contains(s, secret) {
			s = strings.ReplaceAll(s, secret, Redacted)
		}
	}
	return s
}

// Sanitize aplica as regras de mascaramento a um valor de log: segredos
// registrados são trocados e, como na regra herdada do sanitizeForLog do
// pacote api, um valor que fale em "token" ou "password" é descartado inteiro
// (pode ser um corpo de resposta ou cabeçalho com o segredo em outro formato).
func Sanitize(s string) string {
	lower := strings.ToLower(s)
	if strings.Contains(lower, "password") || strings.Contains(lower, "token") {
		return Redacted
	}
	return RedactSecrets(s)
}

// redactingHandler envolve outro handler e mascara mensagem e atributos antes
// de repassá-los. Fica na base da cadeia para que nenhum caminho de log —
// arquivo ou stderr — escape do mascaramento.
type redactingHandler struct {
	next slog.Handler
}

// NewRedactingHandler devolve um handler que mascara segredos antes de next.
func NewRedactingHandler(next slog.Handler) slog.Handler {
	return &redactingHandler{next: next}
}

func (h *redactingHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.next.Enabled(ctx, level)
}

func (h *redactingHandler) Handle(ctx context.Context, r slog.Record) error {
	// A mensagem é texto fixo escrito no código; só segredos registrados são
	// trocados nela, senão "Token rejeitado" viraria [REDACTED].
	clean := slog.NewRecord(r.Time, r.Level, RedactSecrets(r.Message), r.PC)
	r.Attrs(func(a slog.Attr) bool {
		clean.AddAttrs(redactAttr(a))
		return true
	})
	return h.next.Handle(ctx, clean)
}

func (h *redactingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	clean := make([]slog.Attr, len(attrs))
	for i, a := range attrs {
		clean[i] = redactAttr(a)
	}
	return &redactingHandler{next: h.next.WithAttrs(clean)}
}

func (h *redactingHandler) WithGroup(name string) slog.Handler {
	return &redactingHandler{next: h.next.WithGroup(name)}
}

func redactAttr(a slog.Attr) slog.Attr {
	v := a.Value.Resolve()
	switch v.Kind() {
	case slog.KindString:
		return slog.String(a.Key, Sanitize(v.String()))
	case slog.KindGroup:
		group := v.Group()
		clean := make([]any, len(group))
		for i, ga := range group {
			clean[i] = redactAttr(ga)
		}
		return slog.Group(a.Key, clean...)
	case slog.KindAny:
		// Erros, Stringers, slices e structs viram texto para passar pela
		// mesma verificação; um erro de rede pode carregar a URL ou o corpo.
		var text string
		switch x := v.Any().(type) {
		case error:
			text = x.Error()
		case fmt.Stringer:
			text = x.String()
		default:
			text = fmt.Sprintf("%+v", x)
		}
		return slog.String(a.Key, Sanitize(text))
	default:
		// Números, booleanos, tempos e durações não carregam segredos.
		return slog.Attr{Key: a.Key, Value: v}
	}
}
