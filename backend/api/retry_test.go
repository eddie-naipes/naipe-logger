package api

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestIsIdempotent(t *testing.T) {
	idempotentes := []string{
		http.MethodGet, http.MethodHead, http.MethodPut,
		http.MethodDelete, http.MethodOptions,
	}
	for _, m := range idempotentes {
		if !isIdempotent(m) {
			t.Errorf("isIdempotent(%s) = false, esperava true", m)
		}
	}

	// POST cria um lançamento novo a cada chamada.
	if isIdempotent(http.MethodPost) {
		t.Error("isIdempotent(POST) = true; repetir POST pode duplicar horas")
	}
	if isIdempotent(http.MethodPatch) {
		t.Error("isIdempotent(PATCH) = true, esperava false")
	}
}

func TestShouldRetryStatus(t *testing.T) {
	cases := []struct {
		status     int
		idempotent bool
		want       bool
		motivo     string
	}{
		{429, true, true, "rate limit em método idempotente"},
		{429, false, true, "rate limit em POST: rejeitado sem processar, seguro repetir"},
		{500, true, true, "5xx em método idempotente"},
		{503, true, true, "5xx em método idempotente"},
		{500, false, false, "5xx em POST: pode ter sido processado, não repetir"},
		{200, true, false, "sucesso"},
		{201, false, false, "criado"},
		{401, true, false, "credencial inválida não melhora com repetição"},
		{404, true, false, "não encontrado"},
		{400, true, false, "requisição inválida"},
	}

	for _, tc := range cases {
		if got := shouldRetryStatus(tc.status, tc.idempotent); got != tc.want {
			t.Errorf("shouldRetryStatus(%d, idempotent=%v) = %v, esperava %v (%s)",
				tc.status, tc.idempotent, got, tc.want, tc.motivo)
		}
	}
}

func TestBackoffDuration(t *testing.T) {
	// Progressão exponencial a partir de baseBackoff, limitada por maxBackoff.
	if got := backoffDuration(1, 0); got != baseBackoff {
		t.Errorf("backoffDuration(1) = %v, esperava %v", got, baseBackoff)
	}
	if got := backoffDuration(2, 0); got != 2*baseBackoff {
		t.Errorf("backoffDuration(2) = %v, esperava %v", got, 2*baseBackoff)
	}
	if got := backoffDuration(3, 0); got != 4*baseBackoff {
		t.Errorf("backoffDuration(3) = %v, esperava %v", got, 4*baseBackoff)
	}
	if got := backoffDuration(20, 0); got != maxBackoff {
		t.Errorf("backoffDuration(20) = %v, esperava o teto %v", got, maxBackoff)
	}

	// Retry-After do servidor tem precedência, mas continua limitado pelo teto.
	if got := backoffDuration(1, 3*time.Second); got != 3*time.Second {
		t.Errorf("backoffDuration com Retry-After = %v, esperava 3s", got)
	}
	if got := backoffDuration(1, time.Hour); got != maxBackoff {
		t.Errorf("backoffDuration com Retry-After enorme = %v, esperava o teto %v", got, maxBackoff)
	}
}

func TestParseRetryAfter(t *testing.T) {
	if got := parseRetryAfter("5"); got != 5*time.Second {
		t.Errorf("parseRetryAfter(\"5\") = %v, esperava 5s", got)
	}
	if got := parseRetryAfter(""); got != 0 {
		t.Errorf("parseRetryAfter(\"\") = %v, esperava 0", got)
	}
	if got := parseRetryAfter("0"); got != 0 {
		t.Errorf("parseRetryAfter(\"0\") = %v, esperava 0", got)
	}
	if got := parseRetryAfter("abacaxi"); got != 0 {
		t.Errorf("parseRetryAfter(inválido) = %v, esperava 0", got)
	}

	// Data HTTP no futuro.
	futuro := time.Now().Add(10 * time.Second).UTC().Format(http.TimeFormat)
	if got := parseRetryAfter(futuro); got <= 0 || got > 11*time.Second {
		t.Errorf("parseRetryAfter(data futura) = %v, esperava algo perto de 10s", got)
	}

	// Data no passado não deve gerar espera.
	passado := time.Now().Add(-time.Hour).UTC().Format(http.TimeFormat)
	if got := parseRetryAfter(passado); got != 0 {
		t.Errorf("parseRetryAfter(data passada) = %v, esperava 0", got)
	}
}

// Os testes abaixo exercitam doRequest contra um servidor real. doRequest não
// valida esquema (essa barreira está em createRequest), então http basta.

func TestDoRequestRepeteEm429(t *testing.T) {
	var chamadas int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&chamadas, 1) == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	api := &TeamworkAPI{cache: NewCache()}
	req, err := http.NewRequest("GET", server.URL, nil)
	if err != nil {
		t.Fatalf("erro ao criar requisição: %v", err)
	}

	resp, body, err := api.doRequest(req)
	if err != nil {
		t.Fatalf("doRequest devolveu erro: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, esperava 200 após repetição", resp.StatusCode)
	}
	if string(body) != `{"ok":true}` {
		t.Errorf("body = %q, esperava o corpo da resposta bem-sucedida", body)
	}
	if got := atomic.LoadInt32(&chamadas); got != 2 {
		t.Errorf("servidor recebeu %d chamadas, esperava 2", got)
	}
}

func TestDoRequestNaoRepetePOSTEm500(t *testing.T) {
	var chamadas int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&chamadas, 1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	api := &TeamworkAPI{cache: NewCache()}
	req, err := http.NewRequest("POST", server.URL, bytes.NewBufferString(`{"timelog":{}}`))
	if err != nil {
		t.Fatalf("erro ao criar requisição: %v", err)
	}

	resp, _, err := api.doRequest(req)
	if err != nil {
		t.Fatalf("doRequest devolveu erro: %v", err)
	}
	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("status = %d, esperava 500", resp.StatusCode)
	}
	// Repetir aqui poderia criar o lançamento duas vezes.
	if got := atomic.LoadInt32(&chamadas); got != 1 {
		t.Errorf("servidor recebeu %d chamadas, esperava 1 (POST não deve ser repetido em 5xx)", got)
	}
}

func TestDoRequestRepetePOSTEm429ComCorpoIntacto(t *testing.T) {
	var chamadas int32
	corpos := make(chan string, 4)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := new(bytes.Buffer)
		_, _ = buf.ReadFrom(r.Body)
		corpos <- buf.String()

		if atomic.AddInt32(&chamadas, 1) == 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":42}`))
	}))
	defer server.Close()

	payload := `{"timelog":{"minutes":60}}`
	api := &TeamworkAPI{cache: NewCache()}
	req, err := http.NewRequest("POST", server.URL, bytes.NewBufferString(payload))
	if err != nil {
		t.Fatalf("erro ao criar requisição: %v", err)
	}

	resp, _, err := api.doRequest(req)
	if err != nil {
		t.Fatalf("doRequest devolveu erro: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("status = %d, esperava 201", resp.StatusCode)
	}
	if got := atomic.LoadInt32(&chamadas); got != 2 {
		t.Fatalf("servidor recebeu %d chamadas, esperava 2", got)
	}

	// O corpo precisa ser reenviado na repetição, não chegar vazio.
	close(corpos)
	for corpo := range corpos {
		if corpo != payload {
			t.Errorf("servidor recebeu corpo %q, esperava %q", corpo, payload)
		}
	}
}

func TestDoRequestDesisteAposMaxRetries(t *testing.T) {
	var chamadas int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&chamadas, 1)
		w.Header().Set("Retry-After", "0")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	api := &TeamworkAPI{cache: NewCache()}
	req, _ := http.NewRequest("GET", server.URL, nil)

	resp, _, err := api.doRequest(req)
	if err != nil {
		t.Fatalf("doRequest devolveu erro: %v", err)
	}
	if resp.StatusCode != http.StatusTooManyRequests {
		t.Errorf("status = %d, esperava 429 na desistência", resp.StatusCode)
	}
	if got := atomic.LoadInt32(&chamadas); got != maxRetries+1 {
		t.Errorf("servidor recebeu %d chamadas, esperava %d (1 + %d repetições)",
			got, maxRetries+1, maxRetries)
	}
}

func TestDoRequestNaoRepeteEmSucesso(t *testing.T) {
	var chamadas int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&chamadas, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	api := &TeamworkAPI{cache: NewCache()}
	req, _ := http.NewRequest("GET", server.URL, nil)

	if _, _, err := api.doRequest(req); err != nil {
		t.Fatalf("doRequest devolveu erro: %v", err)
	}
	if got := atomic.LoadInt32(&chamadas); got != 1 {
		t.Errorf("servidor recebeu %d chamadas, esperava 1", got)
	}
}
