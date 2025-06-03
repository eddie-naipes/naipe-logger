package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
}

var (
	cachedHolidays     = make(map[int]map[string]Holiday)
	cachedHolidaysLock sync.RWMutex
)

func (t *TeamworkAPI) GetBrazilianHolidays(year int) (map[string]Holiday, error) {
	cachedHolidaysLock.RLock()
	holidays, exists := cachedHolidays[year]
	cachedHolidaysLock.RUnlock()

	if exists {
		return holidays, nil
	}

	cachedHolidaysLock.Lock()
	defer cachedHolidaysLock.Unlock()

	if holidays, exists = cachedHolidays[year]; exists {
		return holidays, nil
	}

	holidays = make(map[string]Holiday)

	fixedHolidays := getFixedHolidays(year)
	for dateStr, holiday := range fixedHolidays {
		holidays[dateStr] = holiday
	}

	apiHolidays, err := fetchHolidaysFromAPI(year)
	if err == nil && len(apiHolidays) > 0 {
		holidays = apiHolidays
	}

	cachedHolidays[year] = holidays

	return holidays, nil
}

func fetchHolidaysFromAPI(year int) (map[string]Holiday, error) {
	url := fmt.Sprintf("https://brasilapi.com.br/api/feriados/v1/%d", year)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("erro ao buscar feriados: HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var apiHolidays []struct {
		Date string `json:"date"`
		Name string `json:"name"`
		Type string `json:"type"`
	}

	if err := json.Unmarshal(body, &apiHolidays); err != nil {
		return nil, err
	}

	holidays := make(map[string]Holiday)
	for _, h := range apiHolidays {
		dateParts := strings.Split(h.Date, "/")
		if len(dateParts) != 3 {
			continue
		}

		day, _ := strconv.Atoi(dateParts[0])
		month, _ := strconv.Atoi(dateParts[1])

		dateStr := fmt.Sprintf("%d-%02d-%02d", year, month, day)

		holiday := Holiday{
			Date:       dateStr,
			Name:       h.Name,
			Type:       "nacional",
			IsOptional: false,
		}

		holidays[dateStr] = holiday
	}

	return holidays, nil
}

func getFixedHolidays(year int) map[string]Holiday {
	holidays := make(map[string]Holiday)

	holidays[fmt.Sprintf("%d-01-01", year)] = Holiday{
		Date:       fmt.Sprintf("%d-01-01", year),
		Name:       "Confraternização Universal",
		Type:       "nacional",
		IsOptional: false,
	}

	holidays[fmt.Sprintf("%d-04-21", year)] = Holiday{
		Date:       fmt.Sprintf("%d-04-21", year),
		Name:       "Tiradentes",
		Type:       "nacional",
		IsOptional: false,
	}

	holidays[fmt.Sprintf("%d-05-01", year)] = Holiday{
		Date:       fmt.Sprintf("%d-05-01", year),
		Name:       "Dia do Trabalho",
		Type:       "nacional",
		IsOptional: false,
	}

	holidays[fmt.Sprintf("%d-09-07", year)] = Holiday{
		Date:       fmt.Sprintf("%d-09-07", year),
		Name:       "Independência do Brasil",
		Type:       "nacional",
		IsOptional: false,
	}

	holidays[fmt.Sprintf("%d-10-12", year)] = Holiday{
		Date:       fmt.Sprintf("%d-10-12", year),
		Name:       "Nossa Senhora Aparecida",
		Type:       "nacional",
		IsOptional: false,
	}

	holidays[fmt.Sprintf("%d-11-02", year)] = Holiday{
		Date:       fmt.Sprintf("%d-11-02", year),
		Name:       "Finados",
		Type:       "nacional",
		IsOptional: false,
	}

	holidays[fmt.Sprintf("%d-11-15", year)] = Holiday{
		Date:       fmt.Sprintf("%d-11-15", year),
		Name:       "Proclamação da República",
		Type:       "nacional",
		IsOptional: false,
	}

	holidays[fmt.Sprintf("%d-12-25", year)] = Holiday{
		Date:       fmt.Sprintf("%d-12-25", year),
		Name:       "Natal",
		Type:       "nacional",
		IsOptional: false,
	}

	return holidays
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
