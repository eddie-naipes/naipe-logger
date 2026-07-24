package backend

import (
	"fmt"
	"time"

	"logTime-go/backend/api"
)

// Bindings de feriados e dias não úteis.

func (a *App) GetBrazilianHolidays(year int) (map[string]api.Holiday, error) {
	if !a.api().IsConfigured() {
		return nil, fmt.Errorf("API não configurada")
	}

	return a.api().GetBrazilianHolidays(year)
}

func (a *App) GetHolidaysForMonth(year, month int) ([]api.Holiday, error) {
	if !a.api().IsConfigured() {
		return nil, fmt.Errorf("API não configurada")
	}

	return a.api().GetHolidaysForMonth(year, month)
}

func (a *App) GetAllNonWorkingDays(year, month int) ([]map[string]interface{}, error) {
	if !a.api().IsConfigured() {
		return nil, fmt.Errorf("API não configurada")
	}

	return a.api().GetAllNonWorkingDays(year, month)
}

func (a *App) IsWorkDay(date string) (bool, error) {
	if !a.api().IsConfigured() {
		return false, fmt.Errorf("API não configurada")
	}

	dateObj, err := time.Parse("2006-01-02", date)
	if err != nil {
		return false, fmt.Errorf("formato de data inválido: %v", err)
	}

	return a.api().IsWorkDay(dateObj), nil
}

func (a *App) GetHolidayCacheStats() (map[string]interface{}, error) {
	if !a.api().IsConfigured() {
		return nil, fmt.Errorf("API não configurada")
	}

	return a.api().GetHolidayCacheStats(), nil
}

func (a *App) ClearHolidayCache() error {
	if !a.api().IsConfigured() {
		return fmt.Errorf("API não configurada")
	}

	a.api().ClearExpiredHolidayCache()
	return nil
}

func (a *App) PreloadHolidays() error {
	if !a.api().IsConfigured() {
		return fmt.Errorf("API não configurada")
	}

	return a.api().PreloadUpcomingHolidays()
}

func (a *App) RefreshHolidaysForYear(year int) (map[string]api.Holiday, error) {
	if !a.api().IsConfigured() {
		return nil, fmt.Errorf("API não configurada")
	}

	a.api().ClearHolidaysCacheForYear(year)

	return a.api().GetBrazilianHolidays(year)
}
