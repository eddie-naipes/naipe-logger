package api

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Payload no formato que a BrasilAPI devolve hoje: datas ISO, não DD/MM. Com o
// conversor antigo todas as linhas eram descartadas e só o fallback valia.
const brasilAPIPayload2026 = `[
{"date":"2026-01-01","name":"Confraternização mundial","type":"national"},
{"date":"2026-02-16","name":"Carnaval","type":"national"},
{"date":"2026-02-17","name":"Carnaval","type":"national"},
{"date":"2026-04-03","name":"Sexta-feira Santa","type":"national"},
{"date":"2026-04-05","name":"Páscoa","type":"national"},
{"date":"2026-04-21","name":"Tiradentes","type":"national"},
{"date":"2026-05-01","name":"Dia do trabalho","type":"national"},
{"date":"2026-06-04","name":"Corpus Christi","type":"national"},
{"date":"2026-09-07","name":"Independência do Brasil","type":"national"},
{"date":"2026-10-12","name":"Nossa Senhora Aparecida","type":"national"},
{"date":"2026-11-02","name":"Finados","type":"national"},
{"date":"2026-11-15","name":"Proclamação da República","type":"national"},
{"date":"2026-11-20","name":"Dia da consciência negra","type":"national"},
{"date":"2026-12-25","name":"Natal","type":"national"}
]`

// apontaBrasilAPIPara redireciona a BrasilAPI para um servidor de teste e
// limpa o cache global do ano ao final.
func apontaBrasilAPIPara(t *testing.T, handler http.HandlerFunc, year int) {
	t.Helper()

	server := httptest.NewServer(handler)
	original := brasilAPIBaseURL
	brasilAPIBaseURL = server.URL + "/api/feriados/v1"

	limpar := func() {
		holidayCacheLock.Lock()
		delete(holidayCache, year)
		holidayCacheLock.Unlock()
	}
	limpar()

	t.Cleanup(func() {
		// Uma revalidação em segundo plano ainda pode estar lendo a URL.
		holidayBackground.Wait()
		brasilAPIBaseURL = original
		server.Close()
		limpar()
	})
}

func TestBrasilAPIAceitaDatasISO(t *testing.T) {
	apontaBrasilAPIPara(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/feriados/v1/2026" {
			t.Errorf("caminho = %q, esperava /api/feriados/v1/2026", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(brasilAPIPayload2026))
	}, 2026)

	api := &TeamworkAPI{cache: NewCache()}
	holidays, err := api.GetBrazilianHolidays(2026)
	if err != nil {
		t.Fatalf("GetBrazilianHolidays devolveu erro: %v", err)
	}

	for _, data := range []string{"2026-02-16", "2026-02-17", "2026-06-04", "2026-11-20"} {
		h, ok := holidays[data]
		if !ok {
			t.Errorf("%s ausente dos feriados", data)
			continue
		}
		if h.Source != "BrasilAPI" {
			t.Errorf("%s veio de %q, esperava BrasilAPI (a fonte primária precisa vencer o fallback)", data, h.Source)
		}
	}

	holidayCacheLock.RLock()
	cache := holidayCache[2026]
	holidayCacheLock.RUnlock()
	if cache == nil || !containsString(cache.Sources, "BrasilAPI") {
		t.Fatalf("cache deveria registrar a BrasilAPI como fonte, ficou %+v", cache)
	}
}

func TestBrasilAPIFormatoDesconhecidoEhErro(t *testing.T) {
	apontaBrasilAPIPara(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`[{"date":"16 de fevereiro","name":"Carnaval"}]`))
	}, 2026)

	provider := &BrasilAPIProvider{}
	if _, err := provider.GetHolidays(t.Context(), 2026); err == nil {
		t.Fatal("payload sem nenhuma data reconhecível deveria devolver erro")
	}

	// O fallback ainda responde, mas com cache curto para tentar de novo logo.
	api := &TeamworkAPI{cache: NewCache()}
	if _, err := api.GetBrazilianHolidays(2026); err != nil {
		t.Fatalf("o fallback deveria cobrir a falha da BrasilAPI: %v", err)
	}
	holidayCacheLock.RLock()
	cache := holidayCache[2026]
	holidayCacheLock.RUnlock()
	if cache == nil {
		t.Fatal("cache não foi gravado")
	}
	if limite := time.Now().Add(fallbackOnlyHolidayTTL + time.Minute); cache.ExpiresAt.After(limite) {
		t.Errorf("cache só com fallback expira em %v; deveria ser curto (%v)", cache.ExpiresAt, fallbackOnlyHolidayTTL)
	}
}

func TestGetBrazilianHolidaysFazUmaSoBuscaConcorrente(t *testing.T) {
	var chamadas int32
	apontaBrasilAPIPara(t, func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&chamadas, 1)
		// Lento de propósito: todas as goroutines chegam com a busca em voo.
		time.Sleep(100 * time.Millisecond)
		_, _ = w.Write([]byte(brasilAPIPayload2026))
	}, 2026)

	api := &TeamworkAPI{cache: NewCache()}

	var wg sync.WaitGroup
	erros := make(chan error, 10)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			holidays, err := api.GetBrazilianHolidays(2026)
			if err == nil && len(holidays) == 0 {
				t.Error("busca compartilhada devolveu mapa vazio")
			}
			erros <- err
		}()
	}
	wg.Wait()
	close(erros)

	for err := range erros {
		if err != nil {
			t.Errorf("GetBrazilianHolidays devolveu erro: %v", err)
		}
	}
	if got := atomic.LoadInt32(&chamadas); got != 1 {
		t.Errorf("BrasilAPI recebeu %d chamadas, esperava 1 (as demais devem esperar a busca em voo)", got)
	}
}

func TestFallbackIncluiConscienciaNegraDesde2024(t *testing.T) {
	provider := &FixedHolidaysProvider{}

	temData := func(year int, data string) bool {
		holidays, err := provider.GetHolidays(t.Context(), year)
		if err != nil {
			t.Fatalf("fallback devolveu erro: %v", err)
		}
		for _, h := range holidays {
			if h.Date == data {
				return true
			}
		}
		return false
	}

	if !temData(2024, "2024-11-20") {
		t.Error("20/11/2024 deveria ser feriado nacional (Lei 14.759/2023)")
	}
	if !temData(2026, "2026-11-20") {
		t.Error("20/11/2026 deveria ser feriado nacional")
	}
	if temData(2023, "2023-11-20") {
		t.Error("20/11/2023 não era feriado nacional")
	}

	// Carnaval (segunda e terça) e Corpus Christi vêm dos móveis.
	for _, data := range []string{"2026-02-16", "2026-02-17", "2026-06-04"} {
		if !temData(2026, data) {
			t.Errorf("%s ausente do fallback", data)
		}
	}
}
