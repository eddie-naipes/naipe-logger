package api

import (
	"context"
	"net/http"
	"strconv"
	"time"
)

const (
	// maxRetries é quantas tentativas ADICIONAIS são feitas após a primeira.
	maxRetries = 3
	// baseBackoff é a espera após a primeira falha; dobra a cada tentativa.
	baseBackoff = 500 * time.Millisecond
	// maxBackoff limita a espera para que um lote não trave por minutos.
	maxBackoff = 8 * time.Second
)

// isIdempotent decide se repetir a requisição é seguro.
//
// POST cria um lançamento novo a cada chamada: se o servidor devolveu 5xx ou a
// conexão caiu, não há como saber se a entrada foi criada, e repetir poderia
// duplicar horas. Os demais métodos usados aqui são idempotentes por definição.
func isIdempotent(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodPut, http.MethodDelete, http.MethodOptions:
		return true
	default:
		return false
	}
}

// shouldRetryStatus decide se vale repetir com base no status devolvido.
//
// 429 é sempre seguro repetir, inclusive em POST: o servidor rejeitou a
// requisição sem processá-la. 5xx só é repetido em métodos idempotentes, já que
// a requisição pode ter sido parcialmente processada.
func shouldRetryStatus(status int, idempotent bool) bool {
	if status == http.StatusTooManyRequests {
		return true
	}
	if status >= 500 && status <= 599 && idempotent {
		return true
	}
	return false
}

// backoffDuration calcula a espera antes da tentativa `attempt` (1 = primeira
// repetição). Um Retry-After informado pelo servidor tem precedência.
func backoffDuration(attempt int, retryAfter time.Duration) time.Duration {
	if retryAfter > 0 {
		if retryAfter > maxBackoff {
			return maxBackoff
		}
		return retryAfter
	}

	if attempt < 1 {
		attempt = 1
	}

	wait := baseBackoff
	for i := 1; i < attempt; i++ {
		wait *= 2
		if wait >= maxBackoff {
			return maxBackoff
		}
	}
	return wait
}

// sleepContext espera a duração pedida, mas desiste assim que o contexto for
// cancelado. Um time.Sleep puro deixaria o aplicativo preso por até maxBackoff
// depois de o usuário já ter fechado a janela no meio de um lote.
func sleepContext(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}

	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// parseRetryAfter interpreta o cabeçalho Retry-After, que pode vir em segundos
// ou como data HTTP. Valores ausentes ou inválidos devolvem 0.
func parseRetryAfter(header string) time.Duration {
	if header == "" {
		return 0
	}

	if seconds, err := strconv.Atoi(header); err == nil {
		if seconds <= 0 {
			return 0
		}
		return time.Duration(seconds) * time.Second
	}

	if deadline, err := http.ParseTime(header); err == nil {
		if wait := time.Until(deadline); wait > 0 {
			return wait
		}
	}

	return 0
}
