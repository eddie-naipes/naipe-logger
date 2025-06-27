package api

import (
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
	GetHolidays(year int) ([]Holiday, error)
	GetName() string
	GetPriority() int
}

// BrasilAPI - API oficial do governo
type BrasilAPIProvider struct{}

func (b *BrasilAPIProvider) GetName() string  { return "BrasilAPI" }
func (b *BrasilAPIProvider) GetPriority() int { return 1 }
func (b *BrasilAPIProvider) GetHolidays(year int) ([]Holiday, error) {
	url := fmt.Sprintf("https://brasilapi.com.br/api/feriados/v1/%d", year)

	resp, err := http.Get(url)
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

	return holidays, nil
}

// Provider de feriados fixos como fallback
type FixedHolidaysProvider struct{}

func (f *FixedHolidaysProvider) GetName() string  { return "FixedHolidays" }
func (f *FixedHolidaysProvider) GetPriority() int { return 99 }
func (f *FixedHolidaysProvider) GetHolidays(year int) ([]Holiday, error) {
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

	// Adicionar feriados móveis (Carnaval, Corpus Christi, etc.)
	mobileHolidays := calculateMobileHolidays(year)
	holidays = append(holidays, mobileHolidays...)

	return holidays, nil
}

var (
	holidayCache     = make(map[int]*HolidayCache)
	holidayCacheLock sync.RWMutex
	holidayProviders []HolidayAPI
)

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

	// Verificar se o cache existe e não expirou
	if exists && time.Now().Before(cache.ExpiresAt) {
		t.logDebug("Usando cache de feriados para %d (expira em %v)", year, cache.ExpiresAt)
		return cache.Holidays, nil
	}

	holidayCacheLock.Lock()
	defer holidayCacheLock.Unlock()

	// Double-check após adquirir o lock
	if cache, exists = holidayCache[year]; exists && time.Now().Before(cache.ExpiresAt) {
		return cache.Holidays, nil
	}

	t.logDebug("Cache de feriados expirado ou inexistente para %d, buscando APIs...", year)

	holidays := make(map[string]Holiday)
	sources := make([]string, 0)
	var lastError error

	// Tentar cada provider em ordem de prioridade
	for _, provider := range holidayProviders {
		t.logDebug("Tentando provider %s para ano %d", provider.GetName(), year)

		providerHolidays, err := provider.GetHolidays(year)
		if err != nil {
			t.logDebug("Erro no provider %s: %v", provider.GetName(), err)
			lastError = err
			continue
		}

		if len(providerHolidays) > 0 {
			// Mesclar feriados do provider
			for _, holiday := range providerHolidays {
				// Priorizar sources com menor priority number
				existing, exists := holidays[holiday.Date]
				if !exists || provider.GetPriority() < getPriorityBySource(existing.Source) {
					holidays[holiday.Date] = holiday
				}
			}
			sources = append(sources, provider.GetName())
			t.logDebug("Provider %s retornou %d feriados", provider.GetName(), len(providerHolidays))
		}
	}

	if len(holidays) == 0 {
		return nil, fmt.Errorf("nenhum provider retornou feriados: %v", lastError)
	}

	// Calcular data de expiração
	expiresAt := calculateCacheExpiration(year)

	// Salvar no cache
	holidayCache[year] = &HolidayCache{
		Year:      year,
		Holidays:  holidays,
		CachedAt:  time.Now(),
		Sources:   sources,
		ExpiresAt: expiresAt,
	}

	t.logDebug("Cache de feriados atualizado para %d com %d feriados de %v (expira em %v)",
		year, len(holidays), sources, expiresAt)

	return holidays, nil
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

func convertBrasilAPIDate(dateStr string, year int) string {
	// BrasilAPI pode retornar formato DD/MM ou DD/MM/YYYY
	parts := strings.Split(dateStr, "/")
	if len(parts) == 2 {
		day, _ := strconv.Atoi(parts[0])
		month, _ := strconv.Atoi(parts[1])
		return fmt.Sprintf("%d-%02d-%02d", year, month, day)
	} else if len(parts) == 3 {
		day, _ := strconv.Atoi(parts[0])
		month, _ := strconv.Atoi(parts[1])
		yearPart, _ := strconv.Atoi(parts[2])
		return fmt.Sprintf("%d-%02d-%02d", yearPart, month, day)
	}
	return ""
}

func calculateMobileHolidays(year int) []Holiday {
	// Cálculo da Páscoa (algoritmo de Gauss)
	easter := calculateEaster(year)

	holidays := []Holiday{
		{
			Date:       easter.AddDate(0, 0, -47).Format("2006-01-02"), // Carnaval (47 dias antes)
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

// Método para verificar se um feriado específico existe
func (t *TeamworkAPI) IsSpecificHoliday(date time.Time, holidayName string) (bool, error) {
	year := date.Year()
	dateStr := date.Format("2006-01-02")

	holidays, err := t.GetBrazilianHolidays(year)
	if err != nil {
		return false, err
	}

	holiday, exists := holidays[dateStr]
	if !exists {
		return false, nil
	}

	return strings.Contains(strings.ToLower(holiday.Name), strings.ToLower(holidayName)), nil
}

// Método para obter feriados de múltiplos anos
func (t *TeamworkAPI) GetHolidaysForYearRange(startYear, endYear int) (map[int]map[string]Holiday, error) {
	result := make(map[int]map[string]Holiday)

	for year := startYear; year <= endYear; year++ {
		holidays, err := t.GetBrazilianHolidays(year)
		if err != nil {
			t.logDebug("Erro ao obter feriados para %d: %v", year, err)
			continue
		}
		result[year] = holidays
	}

	return result, nil
}

// Método para contar dias úteis em um período considerando feriados
func (t *TeamworkAPI) CountWorkingDaysWithHolidays(startDate, endDate string) (int, error) {
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return 0, fmt.Errorf("data inicial inválida: %v", err)
	}

	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return 0, fmt.Errorf("data final inválida: %v", err)
	}

	if end.Before(start) {
		return 0, fmt.Errorf("data final deve ser posterior à inicial")
	}

	// Obter feriados para todos os anos no intervalo
	startYear := start.Year()
	endYear := end.Year()

	allHolidays := make(map[string]Holiday)
	for year := startYear; year <= endYear; year++ {
		yearHolidays, err := t.GetBrazilianHolidays(year)
		if err != nil {
			t.logDebug("Erro ao obter feriados para %d: %v", year, err)
			continue
		}

		for date, holiday := range yearHolidays {
			allHolidays[date] = holiday
		}
	}

	workingDays := 0
	current := start

	for !current.After(end) {
		// Verificar se não é fim de semana
		if current.Weekday() != time.Saturday && current.Weekday() != time.Sunday {
			// Verificar se não é feriado
			dateStr := current.Format("2006-01-02")
			if _, isHoliday := allHolidays[dateStr]; !isHoliday {
				workingDays++
			}
		}
		current = current.AddDate(0, 0, 1)
	}

	return workingDays, nil
}
