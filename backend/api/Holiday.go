package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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

func (b *BrasilAPIProvider) GetName() string  { return "BrasilAPI" }
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
			Source:     "BrasilAPI",
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

// fallbackOnlyHolidayTTL é a validade do cache quando a BrasilAPI falhou e só
// o calendário local respondeu. Sem isso, uma queda momentânea da API deixava
// o fallback valendo até o fim do ano.
const fallbackOnlyHolidayTTL = 6 * time.Hour

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

func (t *TeamworkAPI) GetBrazilianHolidays(year int) (map[string]Holiday, error) {
	holidayCacheLock.RLock()
	cache, exists := holidayCache[year]
	holidayCacheLock.RUnlock()

	// Caminho rápido. É chamado uma vez por dia verificado em GetWorkingDays,
	// por isso não loga nada aqui.
	if exists && time.Now().Before(cache.ExpiresAt) {
		return cache.Holidays, nil
	}

	ctx := t.requestContext()

	holidayCacheLock.Lock()
	if cache, exists = holidayCache[year]; exists && time.Now().Before(cache.ExpiresAt) {
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
	holidayCacheLock.Lock()
	if err == nil {
		expiresAt := calculateCacheExpiration(year)
		if !containsString(sources, "BrasilAPI") {
			if curto := time.Now().Add(fallbackOnlyHolidayTTL); curto.Before(expiresAt) {
				expiresAt = curto
			}
		}
		holidayCache[year] = &HolidayCache{
			Year:      year,
			Holidays:  holidays,
			CachedAt:  time.Now(),
			Sources:   sources,
			ExpiresAt: expiresAt,
		}
		t.logDebug("Cache de feriados atualizado para %d com %d feriados de %v (expira em %v)",
			year, len(holidays), sources, expiresAt)
	}
	delete(holidayInflight, year)
	holidayCacheLock.Unlock()

	fetch.holidays, fetch.err = holidays, err
	close(fetch.done)

	return holidays, err
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
			t.logWarn("Erro no provider de feriados %s para %d: %v", provider.GetName(), year, err)
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

func calculateCacheExpiration(year int) time.Time {
	now := time.Now()
	currentYear := now.Year()

	if year < currentYear {
		// Anos passados: cache por 1 ano
		return now.AddDate(1, 0, 0)
	} else if year == currentYear {
		// Ano atual: cache até final do ano
		return time.Date(year+1, 1, 1, 0, 0, 0, 0, time.UTC)
	} else {
		// Anos futuros: cache por 6 meses
		return now.AddDate(0, 6, 0)
	}
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
			t.logDebug("Cache de feriados removido para ano %d (expirado)", year)
		}
	}
}

// Método para pré-carregar feriados dos próximos anos
func (t *TeamworkAPI) PreloadUpcomingHolidays() error {
	currentYear := time.Now().Year()

	for year := currentYear; year <= currentYear+2; year++ {
		_, err := t.GetBrazilianHolidays(year)
		if err != nil {
			t.logDebug("Erro ao pré-carregar feriados para %d: %v", year, err)
			continue
		}
		t.logDebug("Feriados pré-carregados para %d", year)
	}

	return nil
}

// Método para obter estatísticas do cache
func (t *TeamworkAPI) GetHolidayCacheStats() map[string]interface{} {
	holidayCacheLock.RLock()
	defer holidayCacheLock.RUnlock()

	stats := map[string]interface{}{
		"cached_years":  len(holidayCache),
		"years":         make([]int, 0, len(holidayCache)),
		"cache_details": make(map[int]map[string]interface{}),
	}

	years := make([]int, 0, len(holidayCache))
	for year := range holidayCache {
		years = append(years, year)
	}
	sort.Ints(years)
	stats["years"] = years

	for year, cache := range holidayCache {
		stats["cache_details"].(map[int]map[string]interface{})[year] = map[string]interface{}{
			"holidays_count": len(cache.Holidays),
			"cached_at":      cache.CachedAt,
			"expires_at":     cache.ExpiresAt,
			"sources":        cache.Sources,
			"is_expired":     time.Now().After(cache.ExpiresAt),
		}
	}

	return stats
}

func (t *TeamworkAPI) ClearHolidaysCacheForYear(year int) {
	holidayCacheLock.Lock()
	defer holidayCacheLock.Unlock()

	delete(holidayCache, year)
	t.logDebug("Cache de feriados removido para ano %d", year)
}
