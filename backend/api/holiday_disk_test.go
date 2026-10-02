package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"testing"
	"time"
)

// usaDiscoTemporario liga a persistência de feriados numa pasta temporária e
// a desliga ao final, para que nenhum teste toque ~/.teamwork-logger.
func usaDiscoTemporario(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := SetHolidayCacheDir(dir); err != nil {
		t.Fatalf("SetHolidayCacheDir: %v", err)
	}
	t.Cleanup(func() {
		holidayBackground.Wait()
		_ = SetHolidayCacheDir("")
	})
	return dir
}

// servidorContado responde com o payload da BrasilAPI (ou com status, se
// diferente de 200) e conta as chamadas.
func servidorContado(t *testing.T, status int) *int32 {
	t.Helper()
	var chamadas int32
	apontaBrasilAPIPara(t, func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&chamadas, 1)
		if status != http.StatusOK {
			w.WriteHeader(status)
			return
		}
		_, _ = w.Write([]byte(brasilAPIPayload2026))
	}, 2026)
	return &chamadas
}

// gravaCacheEmDisco escreve um holidays-2026.json da BrasilAPI com a
// expiração dada.
func gravaCacheEmDisco(t *testing.T, dir string, expiresAt time.Time) {
	t.Helper()
	cache := HolidayCache{
		Year: 2026,
		Holidays: map[string]Holiday{
			"2026-03-19": {Date: "2026-03-19", Name: "Feriado avulso do disco", Type: "nacional", Source: brasilAPIName},
		},
		CachedAt:  expiresAt.Add(-holidayFreshTTL),
		Sources:   []string{brasilAPIName},
		ExpiresAt: expiresAt,
	}
	data, _ := json.Marshal(cache)
	if err := os.WriteFile(holidayCachePath(dir, 2026), data, 0600); err != nil {
		t.Fatal(err)
	}
}

func cacheEmMemoria(year int) *HolidayCache {
	holidayCacheLock.RLock()
	defer holidayCacheLock.RUnlock()
	return holidayCache[year]
}

func TestFeriadosDaBrasilAPISaoPersistidosEmDisco(t *testing.T) {
	dir := usaDiscoTemporario(t)
	servidorContado(t, http.StatusOK)

	api := &TeamworkAPI{cache: NewCache()}
	if _, err := api.GetBrazilianHolidays(2026); err != nil {
		t.Fatal(err)
	}

	path := holidayCachePath(dir, 2026)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("arquivo de cache não foi gravado: %v", err)
	}
	var salvo HolidayCache
	if err := json.Unmarshal(data, &salvo); err != nil {
		t.Fatalf("cache em disco ilegível: %v", err)
	}
	if salvo.Year != 2026 || !salvo.authoritative() || len(salvo.Holidays) == 0 {
		t.Errorf("cache em disco inesperado: %+v", salvo)
	}
	if prazo := time.Until(salvo.ExpiresAt); prazo < holidayFreshTTL-time.Hour {
		t.Errorf("cache da BrasilAPI deveria valer ~30 dias, vale %v", prazo)
	}
	if runtime.GOOS != "windows" {
		info, _ := os.Stat(path)
		if info.Mode().Perm() != 0600 {
			t.Errorf("permissão = %v, esperava 0600", info.Mode().Perm())
		}
	}
}

func TestCacheEmDiscoValidoEvitaARede(t *testing.T) {
	dir := usaDiscoTemporario(t)
	chamadas := servidorContado(t, http.StatusOK)
	gravaCacheEmDisco(t, dir, time.Now().Add(24*time.Hour))

	if n := LoadHolidayCacheFromDisk(); n != 1 {
		t.Fatalf("LoadHolidayCacheFromDisk carregou %d anos, esperava 1", n)
	}

	api := &TeamworkAPI{cache: NewCache()}
	holidays, err := api.GetBrazilianHolidays(2026)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := holidays["2026-03-19"]; !ok {
		t.Error("deveria servir o feriado gravado em disco")
	}
	if n := atomic.LoadInt32(chamadas); n != 0 {
		t.Errorf("BrasilAPI chamada %d vezes com cache válido em disco", n)
	}
}

func TestCacheEmDiscoCarregadoSobDemanda(t *testing.T) {
	dir := usaDiscoTemporario(t)
	chamadas := servidorContado(t, http.StatusOK)
	gravaCacheEmDisco(t, dir, time.Now().Add(24*time.Hour))

	// Sem LoadHolidayCacheFromDisk: a primeira consulta lê o disco sozinha.
	api := &TeamworkAPI{cache: NewCache()}
	holidays, err := api.GetBrazilianHolidays(2026)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := holidays["2026-03-19"]; !ok || atomic.LoadInt32(chamadas) != 0 {
		t.Errorf("deveria usar o disco sem ir à rede (chamadas=%d)", atomic.LoadInt32(chamadas))
	}
}

func TestCacheVencidoServeJaERevalidaEmSegundoPlano(t *testing.T) {
	dir := usaDiscoTemporario(t)
	chamadas := servidorContado(t, http.StatusOK)
	gravaCacheEmDisco(t, dir, time.Now().Add(-time.Hour))

	api := &TeamworkAPI{cache: NewCache()}
	holidays, err := api.GetBrazilianHolidays(2026)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := holidays["2026-03-19"]; !ok {
		t.Error("o dado vencido deveria ser servido imediatamente")
	}

	holidayBackground.Wait()
	if n := atomic.LoadInt32(chamadas); n != 1 {
		t.Errorf("esperava 1 revalidação, houve %d", n)
	}
	atual := cacheEmMemoria(2026)
	if atual == nil || !time.Now().Before(atual.ExpiresAt) {
		t.Fatalf("revalidação deveria renovar a validade: %+v", atual)
	}
	if _, ok := atual.Holidays["2026-02-16"]; !ok {
		t.Error("a revalidação deveria trazer os feriados da BrasilAPI")
	}

	var salvo HolidayCache
	data, _ := os.ReadFile(holidayCachePath(dir, 2026))
	_ = json.Unmarshal(data, &salvo)
	if _, ok := salvo.Holidays["2026-02-16"]; !ok {
		t.Error("o disco deveria ter sido atualizado pela revalidação")
	}
}

func TestBrasilAPIForaUsaDiscoAntesDoFallback(t *testing.T) {
	dir := usaDiscoTemporario(t)
	servidorContado(t, http.StatusInternalServerError)
	gravaCacheEmDisco(t, dir, time.Now().Add(-time.Hour))
	LoadHolidayCacheFromDisk()

	api := &TeamworkAPI{cache: NewCache()}
	if _, err := api.GetBrazilianHolidays(2026); err != nil {
		t.Fatal(err)
	}
	holidayBackground.Wait()

	atual := cacheEmMemoria(2026)
	if atual == nil || !atual.authoritative() {
		t.Fatalf("com a BrasilAPI fora, o dado dela em disco deveria continuar valendo: %+v", atual)
	}
	if _, ok := atual.Holidays["2026-03-19"]; !ok {
		t.Error("o feriado do disco sumiu ao trocar pelo fallback")
	}
	if prazo := time.Until(atual.ExpiresAt); prazo <= 0 || prazo > fallbackOnlyHolidayTTL+time.Minute {
		t.Errorf("próxima tentativa deveria ser em ~%v, ficou em %v", fallbackOnlyHolidayTTL, prazo)
	}

	// Chamadas seguintes não disparam nova revalidação (agora está "válido").
	holidays, _ := api.GetBrazilianHolidays(2026)
	if _, ok := holidays["2026-03-19"]; !ok {
		t.Error("deveria continuar servindo o dado do disco")
	}
}

func TestBrasilAPIForaSemDiscoUsaFallbackENaoPersiste(t *testing.T) {
	dir := usaDiscoTemporario(t)
	servidorContado(t, http.StatusInternalServerError)

	api := &TeamworkAPI{cache: NewCache()}
	holidays, err := api.GetBrazilianHolidays(2026)
	if err != nil {
		t.Fatal(err)
	}
	if h, ok := holidays["2026-12-25"]; !ok || h.Source != "Fixed" {
		t.Errorf("esperava o calendário local, veio %+v", h)
	}
	if _, err := os.Stat(holidayCachePath(dir, 2026)); !os.IsNotExist(err) {
		t.Error("o calendário local não deveria ser gravado em disco")
	}
}

func TestLimparCacheDeFeriadosApagaODisco(t *testing.T) {
	dir := usaDiscoTemporario(t)
	gravaCacheEmDisco(t, dir, time.Now().Add(time.Hour))
	cache2027 := HolidayCache{Year: 2027, Sources: []string{brasilAPIName},
		Holidays: map[string]Holiday{"2027-01-01": {Date: "2027-01-01", Source: brasilAPIName}}}
	data, _ := json.Marshal(cache2027)
	_ = os.WriteFile(holidayCachePath(dir, 2027), data, 0600)
	LoadHolidayCacheFromDisk()
	t.Cleanup(func() {
		holidayCacheLock.Lock()
		delete(holidayCache, 2026)
		delete(holidayCache, 2027)
		holidayCacheLock.Unlock()
	})

	api := &TeamworkAPI{}
	api.ClearHolidaysCacheForYear(2026)
	if _, err := os.Stat(holidayCachePath(dir, 2026)); !os.IsNotExist(err) {
		t.Error("ClearHolidaysCacheForYear deveria apagar o arquivo do ano")
	}
	if _, err := os.Stat(holidayCachePath(dir, 2027)); err != nil {
		t.Error("ClearHolidaysCacheForYear não deveria apagar outros anos")
	}

	api.ClearAllHolidayCache()
	if anos := persistedHolidayYears(dir); len(anos) != 0 {
		t.Errorf("ClearAllHolidayCache deixou arquivos: %v", anos)
	}
	if cacheEmMemoria(2027) != nil {
		t.Error("ClearAllHolidayCache deveria limpar a memória")
	}
}

func TestCacheEmDiscoCorrompidoEhDescartado(t *testing.T) {
	dir := usaDiscoTemporario(t)
	path := filepath.Join(dir, "holidays-2026.json")
	if err := os.WriteFile(path, []byte("{nao é json"), 0600); err != nil {
		t.Fatal(err)
	}
	holidayCacheLock.Lock()
	delete(holidayCache, 2026)
	holidayCacheLock.Unlock()

	if n := LoadHolidayCacheFromDisk(); n != 0 {
		t.Errorf("arquivo corrompido não deveria ser carregado (n=%d)", n)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("arquivo corrompido deveria ser apagado")
	}
}
