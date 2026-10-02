package backend

import (
	"fmt"
	"sort"
	"time"

	"logTime-go/backend/api"
)

// Bindings de feriados e dias não úteis.

// GetBrazilianHolidays devolve os feriados do ano ordenados por data. Era um
// mapa data->feriado: o gerador do Wails não emitia api.Holiday em models.ts
// para valores de mapa (App.d.ts referenciava um tipo inexistente). O
// frontend já fazia Object.values(...).sort, que funciona igual com a lista.
func (a *App) GetBrazilianHolidays(year int) ([]api.Holiday, error) {
	client, err := a.client()
	if err != nil {
		return nil, err
	}
	holidays, err := client.GetBrazilianHolidays(year)
	if err != nil {
		return nil, err
	}
	return sortedHolidays(holidays), nil
}

func sortedHolidays(holidays map[string]api.Holiday) []api.Holiday {
	list := make([]api.Holiday, 0, len(holidays))
	for _, h := range holidays {
		list = append(list, h)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Date < list[j].Date })
	return list
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

// GetHolidayCacheStats devolve o estado do cache de feriados. Tipado (mesmo
// JSON do antigo map[string]interface{}) para gerar o modelo em models.ts.
func (a *App) GetHolidayCacheStats() (api.HolidayCacheStats, error) {
	client, err := a.client()
	if err != nil {
		return api.HolidayCacheStats{}, err
	}
	return client.HolidayCacheSummary(), nil
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

// RefreshHolidaysForYear descarta o ano (memória e disco) e o busca de novo.
func (a *App) RefreshHolidaysForYear(year int) ([]api.Holiday, error) {
	client, err := a.client()
	if err != nil {
		return nil, err
	}

	client.ClearHolidaysCacheForYear(year)
	holidays, err := client.GetBrazilianHolidays(year)
	if err != nil {
		return nil, err
	}
	return sortedHolidays(holidays), nil
}
