package backend

import (
	"fmt"
	"time"

	"logTime-go/backend/api"
)

// Bindings de feriados e dias não úteis.

func (a *App) GetBrazilianHolidays(year int) (map[string]api.Holiday, error) {
	client, err := a.client()
	if err != nil {
		return nil, err
	}
	return client.GetBrazilianHolidays(year)
}

func (a *App) GetAllNonWorkingDays(year, month int) ([]map[string]interface{}, error) {
	client, err := a.client()
	if err != nil {
		return nil, err
	}
	return client.GetAllNonWorkingDays(year, month)
}

func (a *App) IsWorkDay(date string) (bool, error) {
	client, err := a.client()
	if err != nil {
		return false, err
	}

	dateObj, err := time.Parse("2006-01-02", date)
	if err != nil {
		return false, fmt.Errorf("formato de data inválido: %v", err)
	}

	return client.IsWorkDay(dateObj), nil
}

func (a *App) GetHolidayCacheStats() (map[string]interface{}, error) {
	client, err := a.client()
	if err != nil {
		return nil, err
	}
	return client.GetHolidayCacheStats(), nil
}

func (a *App) ClearHolidayCache() error {
	client, err := a.client()
	if err != nil {
		return err
	}
	// A tela pede "limpar todo o cache"; antes só os anos vencidos saíam.
	client.ClearAllHolidayCache()
	return nil
}

func (a *App) PreloadHolidays() error {
	client, err := a.client()
	if err != nil {
		return err
	}
	return client.PreloadUpcomingHolidays()
}

func (a *App) RefreshHolidaysForYear(year int) (map[string]api.Holiday, error) {
	client, err := a.client()
	if err != nil {
		return nil, err
	}

	client.ClearHolidaysCacheForYear(year)
	return client.GetBrazilianHolidays(year)
}
