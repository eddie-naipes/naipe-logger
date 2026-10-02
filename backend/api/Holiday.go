package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Holiday struct {
	Date        string `json:"date"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Type        string `json:"type"`
	IsOptional  bool   `json:"isOptional"`
	Source      string `json:"source,omitempty"`
}

type HolidayCache struct {
	Year      int                `json:"year"`
	Holidays  map[string]Holiday `json:"holidays"`
	CachedAt  time.Time          `json:"cachedAt"`
	Sources   []string           `json:"sources"`
	ExpiresAt time.Time          `json:"expiresAt"`
}

type HolidayAPI interface {
	GetHolidays(ctx context.Context, year int) ([]Holiday, error)
	GetName() string
	GetPriority() int
}

var (
	// brasilAPIBaseURL é variável para que os testes apontem para um
	// httptest.Server em vez da BrasilAPI real.
	brasilAPIBaseURL = "https://brasilapi.com.br/api/feriados/v1"

	// brasilAPIClient tem timeout próprio: http.Get usa o cliente padrão, sem
	// timeout, e uma BrasilAPI lenta travava a verificação de dias úteis (e,
	// antes, todas as goroutines que esperavam o lock do cache de feriados).
	brasilAPIClient = &http.Client{Timeout: 10 * time.Second}
)

// BrasilAPI - API pública de feriados nacionais
type BrasilAPIProvider struct{}

func (b *BrasilAPIProvider) GetName() string  { return brasilAPIName }
func (b *BrasilAPIProvider) GetPriority() int { return 1 }
func (b *BrasilAPIProvider) GetHolidays(ctx context.Context, year int) ([]Holiday, error) {
	url := fmt.Sprintf("%s/%d", brasilAPIBaseURL, year)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("erro ao montar requisição BrasilAPI: %v", err)
	}

	resp, err := brasilAPIClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("erro na requisição BrasilAPI: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("BrasilAPI retornou status %d", resp.StatusCode)
	}

	var apiHolidays []struct {
		Date string `json:"date"`
		Name string `json:"name"`
		Type string `json:"type"`
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(body, &apiHolidays); err != nil {
		return nil, err
	}

	holidays := make([]Holiday, 0, len(apiHolidays))
	for _, h := range apiHolidays {
		dateStr := convertBrasilAPIDate(h.Date, year)
		if dateStr == "" {
			continue
		}

		holiday := Holiday{
			Date:       dateStr,
			Name:       h.Name,
			Type:       "nacional",
			IsOptional: false,
			Source:     brasilAPIName,
		}
		holidays = append(holidays, holiday)
	}

	// Payload não vazio sem nenhuma data reconhecida indica mudança de formato
	// na API — foi assim que a troca para datas ISO passou despercebida.
	// Devolver erro faz o fallback ser tratado como tal (cache curto) em vez
	// de mascarar o problema.
	if len(apiHolidays) > 0 && len(holidays) == 0 {
		return nil, fmt.Errorf("BrasilAPI devolveu datas em formato desconhecido (ex.: %q)", apiHolidays[0].Date)
	}

	return holidays, nil
}

// Provider de feriados fixos como fallback
type FixedHolidaysProvider struct{}

func (f *FixedHolidaysProvider) GetName() string  { return "FixedHolidays" }
func (f *FixedHolidaysProvider) GetPriority() int { return 99 }
func (f *FixedHolidaysProvider) GetHolidays(_ context.Context, year int) ([]Holiday, error) {
	holidays := []Holiday{
		{
			Date:       fmt.Sprintf("%d-01-01", year),
			Name:       "Confraternização Universal",
			Type:       "nacional",
			IsOptional: false,
			Source:     "Fixed",
		},
		{
			Date:       fmt.Sprintf("%d-04-21", year),
			Name:       "Tiradentes",
			Type:       "nacional",
			IsOptional: false,
			Source:     "Fixed",
		},
		{
			Date:       fmt.Sprintf("%d-05-01", year),
			Name:       "Dia do Trabalho",
			Type:       "nacional",
			IsOptional: false,
			Source:     "Fixed",
		},
		{
			Date:       fmt.Sprintf("%d-09-07", year),
			Name:       "Independência do Brasil",
			Type:       "nacional",
			IsOptional: false,
			Source:     "Fixed",
		},
		{
			Date:       fmt.Sprintf("%d-10-12", year),
			Name:       "Nossa Senhora Aparecida",
			Type:       "nacional",
			IsOptional: false,
			Source:     "Fixed",
		},
		{
			Date:       fmt.Sprintf("%d-11-02", year),
			Name:       "Finados",
			Type:       "nacional",
			IsOptional: false,
			Source:     "Fixed",
		},
		{
			Date:       fmt.Sprintf("%d-11-15", year),
			Name:       "Proclamação da República",
			Type:       "nacional",
			IsOptional: false,
			Source:     "Fixed",
		},
		{
			Date:       fmt.Sprintf("%d-12-25", year),
			Name:       "Natal",
			Type:       "nacional",
			IsOptional: false,
			Source:     "Fixed",
		},
	}

	// Dia Nacional de Zumbi e da Consciência Negra: feriado nacional pela Lei
	// 14.759/2023, a partir de 2024. Antes disso era só estadual/municipal,
	// então não entra nos anos anteriores.
	if year >= 2024 {
		holidays = append(holidays, Holiday{
			Date:       fmt.Sprintf("%d-11-20", year),
			Name:       "Dia Nacional de Zumbi e da Consciência Negra",
			Type:       "nacional",
			IsOptional: false,
			Source:     "Fixed",
		})
	}

	// Adicionar feriados móveis (Carnaval, Corpus Christi, etc.)
	mobileHolidays := calculateMobileHolidays(year)
	holidays = append(holidays, mobileHolidays...)

	return holidays, nil
}

var (
	holidayCache     = make(map[int]*HolidayCache)
	holidayCacheLock sync.RWMutex
	holidayProviders []HolidayAPI

	// holidayInflight guarda a busca em andamento por ano (protegido por
	// holidayCacheLock). Chamadas concorrentes para o mesmo ano esperam a
	// mesma busca em vez de cada uma ir à rede.
	holidayInflight = make(map[int]*holidayFetch)
)

const (
	// fallbackOnlyHolidayTTL é a validade do cache quando a BrasilAPI falhou e
	// só o calendário local respondeu. Sem isso, uma queda momentânea da API
	// deixava o fallback valendo até o fim do ano. É também o intervalo até a
	// próxima tentativa quando a revalidação de um cache vencido falha.
	fallbackOnlyHolidayTTL = 6 * time.Hour

	// holidayFreshTTL é por quanto tempo um ano vindo da BrasilAPI é servido
	// sem consultar a rede. Feriados nacionais quase nunca mudam depois de
	// publicados; passado esse prazo o cache continua sendo usado e é
	// revalidado em segundo plano.
	holidayFreshTTL = 30 * 24 * time.Hour
)

// holidayBackground acompanha as revalidações em segundo plano; os testes
// esperam por ele.
var holidayBackground sync.WaitGroup

type holidayFetch struct {
	done     chan struct{}
	holidays map[string]Holiday
	err      error
}

func init() {
	// Registrar providers em ordem de prioridade
	holidayProviders = []HolidayAPI{
		&BrasilAPIProvider{},
		&FixedHolidaysProvider{},
	}
}

// authoritative informa se o cache veio da BrasilAPI. Só esses dados valem
// mais que o calendário local mesmo vencidos, e só eles vão para o disco.
func (c *HolidayCache) authoritative() bool {
	return c != nil && containsString(c.Sources, brasilAPIName)
}

const brasilAPIName = "BrasilAPI"

// GetBrazilianHolidays devolve os feriados do ano. Ordem de consulta: memória,
// disco (~/.teamwork-logger/cache), BrasilAPI e, se ela falhar, o último dado
// da BrasilAPI que houver ou o calendário local.
func (t *TeamworkAPI) GetBrazilianHolidays(year int) (map[string]Holiday, error) {
	holidayCacheLock.RLock()
	cache, exists := holidayCache[year]
	holidayCacheLock.RUnlock()

	if !exists {
		cache, exists = loadHolidayYearFromDisk(year)
	}

	// Caminho rápido. É chamado uma vez por dia verificado em GetWorkingDays,
	// por isso não loga nada aqui.
	if exists {
		if time.Now().Before(cache.ExpiresAt) {
			return cache.Holidays, nil
		}
		if cache.authoritative() {
			// Vencido, mas da BrasilAPI: responde já e atualiza por trás, em
			// vez de fazer o usuário esperar a rede por um dado que quase
			// certamente não mudou.
			t.revalidateHolidaysInBackground(year)
			return cache.Holidays, nil
		}
	}

	return t.fetchAndStoreHolidays(year)
}

// revalidateHolidaysInBackground dispara uma busca do ano se não houver outra
// em andamento. Uma falha não apaga o cache: storeFetchedHolidays mantém o
// dado antigo e adia a próxima tentativa.
func (t *TeamworkAPI) revalidateHolidaysInBackground(year int) {
	holidayCacheLock.RLock()
	_, inFlight := holidayInflight[year]
	holidayCacheLock.RUnlock()
	if inFlight {
		return
	}

	holidayBackground.Add(1)
	go func() {
		defer holidayBackground.Done()
		// Um panic numa goroutine derrubaria o app inteiro.
		defer func() {
			if r := recover(); r != nil {
				slog.Error("Panic ao revalidar feriados", "ano", year, "panic", r)
			}
		}()
		if _, err := t.fetchAndStoreHolidays(year); err != nil {
			slog.Warn("Erro ao revalidar feriados", "ano", year, "err", err)
		}
	}()
}

// fetchAndStoreHolidays busca o ano na rede, com uma única busca por ano em
// voo: chamadas concorrentes esperam a mesma resposta.
func (t *TeamworkAPI) fetchAndStoreHolidays(year int) (map[string]Holiday, error) {
	ctx := t.requestContext()

	holidayCacheLock.Lock()
	if cache, exists := holidayCache[year]; exists && time.Now().Before(cache.ExpiresAt) {
		holidayCacheLock.Unlock()
		return cache.Holidays, nil
	}

	// Já existe busca em andamento para o ano: espera por ela. A chamada de
	// rede acontece fora do lock, então leituras de outros anos não ficam
	// bloqueadas por uma BrasilAPI lenta.
	if fetch, inFlight := holidayInflight[year]; inFlight {
		holidayCacheLock.Unlock()
		select {
		case <-fetch.done:
			return fetch.holidays, fetch.err
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	fetch := &holidayFetch{done: make(chan struct{})}
	holidayInflight[year] = fetch
	holidayCacheLock.Unlock()

	holidays, sources, err := t.fetchHolidays(ctx, year)

	// Só trava de novo para gravar o resultado.
	var persist *HolidayCache
	holidayCacheLock.Lock()
	if err == nil {
		holidays, persist = storeFetchedHolidaysLocked(year, holidays, sources)
	}
	delete(holidayInflight, year)
	holidayCacheLock.Unlock()

	fetch.holidays, fetch.err = holidays, err
	close(fetch.done)

	// Disco fora do lock: um disco lento não pode travar as leituras.
	if persist != nil {
		persistHolidayCache(persist)
	}

	return holidays, err
}

// storeFetchedHolidaysLocked grava o resultado de uma busca e devolve os
// feriados que devem valer, mais o cache a persistir (nil se nada vai ao
// disco). Exige holidayCacheLock travado para escrita.
func storeFetchedHolidaysLocked(year int, holidays map[string]Holiday, sources []string) (map[string]Holiday, *HolidayCache) {
	now := time.Now()

	if containsString(sources, brasilAPIName) {
		cache := &HolidayCache{
			Year:      year,
			Holidays:  holidays,
			CachedAt:  now,
			Sources:   sources,
			ExpiresAt: now.Add(holidayFreshTTL),
		}
		holidayCache[year] = cache
		slog.Debug("Cache de feriados atualizado", "ano", year, "feriados", len(holidays),
			"fontes", sources, "expira", cache.ExpiresAt)
		return holidays, cache
	}

	// A BrasilAPI falhou. Um dado dela já conhecido (da memória ou do disco)
	// vale mais que o calendário local, que não tem feriados avulsos: é
	// mantido e a próxima tentativa fica para daqui a algumas horas.
	if previous, ok := holidayCache[year]; ok && previous.authoritative() {
		retry := *previous
		retry.ExpiresAt = now.Add(fallbackOnlyHolidayTTL)
		holidayCache[year] = &retry
		slog.Warn("BrasilAPI indisponível; mantendo feriados em cache", "ano", year,
			"obtidosEm", previous.CachedAt)
		return previous.Holidays, nil
	}

	holidayCache[year] = &HolidayCache{
		Year:      year,
		Holidays:  holidays,
		CachedAt:  now,
		Sources:   sources,
		ExpiresAt: now.Add(fallbackOnlyHolidayTTL),
	}
	slog.Debug("Cache de feriados com calendário local", "ano", year, "feriados", len(holidays),
		"fontes", sources)
	return holidays, nil
}

// fetchHolidays consulta os providers em ordem de prioridade e mescla os
// resultados. Não toca no cache nem em locks: roda fora da seção crítica.
func (t *TeamworkAPI) fetchHolidays(ctx context.Context, year int) (map[string]Holiday, []string, error) {
	holidays := make(map[string]Holiday)
	sources := make([]string, 0)
	var lastError error

	for _, provider := range holidayProviders {
		providerHolidays, err := provider.GetHolidays(ctx, year)
		if err != nil {
			slog.Warn("Erro no provider de feriados", "provider", provider.GetName(), "ano", year, "err", err)
			lastError = err
			continue
		}

		if len(providerHolidays) > 0 {
			for _, holiday := range providerHolidays {
				// Em datas repetidas vence o provider de menor prioridade.
				existing, exists := holidays[holiday.Date]
				if !exists || provider.GetPriority() < getPriorityBySource(existing.Source) {
					holidays[holiday.Date] = holiday
				}
			}
			sources = append(sources, provider.GetName())
		}
	}

	if len(holidays) == 0 {
		return nil, nil, fmt.Errorf("nenhum provider retornou feriados: %v", lastError)
	}

	return holidays, sources, nil
}

func containsString(values []string, target string) bool {
	for _, v := range values {
		if v == target {
			return true
		}
	}
	return false
}

func (t *TeamworkAPI) IsHoliday(date time.Time) (bool, Holiday, error) {
	year := date.Year()
	dateStr := date.Format("2006-01-02")

	holidays, err := t.GetBrazilianHolidays(year)
	if err != nil {
		return false, Holiday{}, err
	}

	holiday, isHoliday := holidays[dateStr]
	return isHoliday, holiday, nil
}

func (t *TeamworkAPI) GetHolidaysForMonth(year, month int) ([]Holiday, error) {
	allHolidays, err := t.GetBrazilianHolidays(year)
	if err != nil {
		return nil, err
	}

	monthHolidays := make([]Holiday, 0)
	monthPrefix := fmt.Sprintf("%d-%02d-", year, month)

	for _, holiday := range allHolidays {
		if strings.HasPrefix(holiday.Date, monthPrefix) {
			monthHolidays = append(monthHolidays, holiday)
		}
	}

	return monthHolidays, nil
}

func getPriorityBySource(source string) int {
	for _, provider := range holidayProviders {
		if provider.GetName() == source {
			return provider.GetPriority()
		}
	}
	return 999
}

// convertBrasilAPIDate converte a data devolvida pela BrasilAPI para
// YYYY-MM-DD. Hoje a API responde em ISO ("2026-02-16"); os formatos DD/MM e
// DD/MM/YYYY continuam aceitos. Valor irreconhecível vira "".
func convertBrasilAPIDate(dateStr string, year int) string {
	dateStr = strings.TrimSpace(dateStr)
	if dateStr == "" {
		return ""
	}

	if parsed, err := time.Parse("2006-01-02", dateStr); err == nil {
		return parsed.Format("2006-01-02")
	}

	parts := strings.Split(dateStr, "/")
	if len(parts) != 2 && len(parts) != 3 {
		return ""
	}

	day, errDay := strconv.Atoi(parts[0])
	month, errMonth := strconv.Atoi(parts[1])
	if errDay != nil || errMonth != nil {
		return ""
	}
	if len(parts) == 3 {
		explicitYear, err := strconv.Atoi(parts[2])
		if err != nil {
			return ""
		}
		year = explicitYear
	}

	// time.Date normaliza datas impossíveis (31/02 vira 03/03); comparar de
	// volta recusa esses casos em vez de inventar um feriado.
	parsed := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	if parsed.Day() != day || int(parsed.Month()) != month {
		return ""
	}
	return parsed.Format("2006-01-02")
}

func calculateMobileHolidays(year int) []Holiday {
	// Cálculo da Páscoa (algoritmo de Gauss)
	easter := calculateEaster(year)

	holidays := []Holiday{
		{
			// Carnaval é ponto facultativo nacional, mas na prática emenda
			// segunda e terça; antes só a terça entrava no fallback.
			Date:       easter.AddDate(0, 0, -48).Format("2006-01-02"),
			Name:       "Carnaval (segunda-feira)",
			Type:       "nacional",
			IsOptional: false,
			Source:     "Fixed",
		},
		{
			Date:       easter.AddDate(0, 0, -47).Format("2006-01-02"), // terça de Carnaval
			Name:       "Carnaval",
			Type:       "nacional",
			IsOptional: false,
			Source:     "Fixed",
		},
		{
			Date:       easter.AddDate(0, 0, -2).Format("2006-01-02"), // Sexta-feira Santa
			Name:       "Sexta-feira Santa",
			Type:       "nacional",
			IsOptional: false,
			Source:     "Fixed",
		},
		{
			Date:       easter.Format("2006-01-02"), // Páscoa
			Name:       "Páscoa",
			Type:       "nacional",
			IsOptional: false,
			Source:     "Fixed",
		},
		{
			Date:       easter.AddDate(0, 0, 60).Format("2006-01-02"), // Corpus Christi
			Name:       "Corpus Christi",
			Type:       "nacional",
			IsOptional: true,
			Source:     "Fixed",
		},
	}

	return holidays
}

func calculateEaster(year int) time.Time {
	// Algoritmo de Gauss para calcular a Páscoa
	a := year % 19
	b := year / 100
	c := year % 100
	d := b / 4
	e := b % 4
	f := (b + 8) / 25
	g := (b - f + 1) / 3
	h := (19*a + b - d - g + 15) % 30
	i := c / 4
	k := c % 4
	l := (32 + 2*e + 2*i - h - k) % 7
	m := (a + 11*h + 22*l) / 451
	month := (h + l - 7*m + 114) / 31
	day := ((h + l - 7*m + 114) % 31) + 1

	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
}

// Método para limpar cache antigo
func (t *TeamworkAPI) ClearExpiredHolidayCache() {
	holidayCacheLock.Lock()
	defer holidayCacheLock.Unlock()

	now := time.Now()
	for year, cache := range holidayCache {
		if now.After(cache.ExpiresAt) {
			delete(holidayCache, year)
			slog.Debug("Cache de feriados expirado removido", "ano", year)
		}
	}
}

// Método para pré-carregar feriados dos próximos anos
func (t *TeamworkAPI) PreloadUpcomingHolidays() error {
	currentYear := time.Now().Year()

	for year := currentYear; year <= currentYear+2; year++ {
		_, err := t.GetBrazilianHolidays(year)
		if err != nil {
			slog.Debug("Erro ao pré-carregar feriados", "ano", year, "err", err)
			continue
		}
		slog.Debug("Feriados pré-carregados", "ano", year)
	}

	return nil
}

// HolidayCacheStats descreve o cache de feriados, para a tela de manutenção.
type HolidayCacheStats struct {
	CachedYears  int                        `json:"cached_years"`
	Years        []int                      `json:"years"`
	CacheDetails map[int]HolidayCacheDetail `json:"cache_details"`
}

// HolidayCacheDetail é o estado do cache de um ano.
type HolidayCacheDetail struct {
	HolidaysCount int       `json:"holidays_count"`
	CachedAt      time.Time `json:"cached_at"`
	ExpiresAt     time.Time `json:"expires_at"`
	Sources       []string  `json:"sources"`
	IsExpired     bool      `json:"is_expired"`
}

// GetHolidayCacheStats devolve o estado do cache no formato de mapa que o
// binding atual expõe. O JSON é o mesmo de HolidayCacheSummary.
func (t *TeamworkAPI) GetHolidayCacheStats() map[string]interface{} {
	stats := t.HolidayCacheSummary()
	return map[string]interface{}{
		"cached_years":  stats.CachedYears,
		"years":         stats.Years,
		"cache_details": stats.CacheDetails,
	}
}

// HolidayCacheSummary é a versão tipada de GetHolidayCacheStats.
func (t *TeamworkAPI) HolidayCacheSummary() HolidayCacheStats {
	holidayCacheLock.RLock()
	defer holidayCacheLock.RUnlock()

	stats := HolidayCacheStats{
		CachedYears:  len(holidayCache),
		Years:        make([]int, 0, len(holidayCache)),
		CacheDetails: make(map[int]HolidayCacheDetail, len(holidayCache)),
	}

	now := time.Now()
	for year, cache := range holidayCache {
		stats.Years = append(stats.Years, year)
		stats.CacheDetails[year] = HolidayCacheDetail{
			HolidaysCount: len(cache.Holidays),
			CachedAt:      cache.CachedAt,
			ExpiresAt:     cache.ExpiresAt,
			Sources:       cache.Sources,
			IsExpired:     now.After(cache.ExpiresAt),
		}
	}
	sort.Ints(stats.Years)

	return stats
}

// ClearHolidaysCacheForYear esquece o ano na memória e no disco, forçando a
// próxima consulta a ir à BrasilAPI.
func (t *TeamworkAPI) ClearHolidaysCacheForYear(year int) {
	holidayCacheLock.Lock()
	delete(holidayCache, year)
	holidayCacheLock.Unlock()

	removeHolidayCacheFile(year)
	slog.Debug("Cache de feriados removido", "ano", year)
}

// ClearAllHolidayCache apaga todo o cache de feriados, em memória e em disco.
func (t *TeamworkAPI) ClearAllHolidayCache() {
	holidayCacheLock.Lock()
	holidayCache = make(map[int]*HolidayCache)
	holidayCacheLock.Unlock()

	removeAllHolidayCacheFiles()
	slog.Info("Cache de feriados limpo")
}
