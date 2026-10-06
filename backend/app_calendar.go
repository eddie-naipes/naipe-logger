package backend

import (
	"fmt"

	"logTime-go/backend/config"
	"logTime-go/backend/holidays"
)

// Bindings do calendário de trabalho: UF dos feriados estaduais, feriados
// municipais/pontes e férias. Nada aqui fala com o Teamwork; os dias entram
// em IsWorkDay/GetWorkingDays/GetAllNonWorkingDays via holidays.Provider.

// GetWorkCalendarSettings devolve a configuração atual do calendário.
func (a *App) GetWorkCalendarSettings() config.CalendarSettings {
	return a.configManager.GetCalendarSettings()
}

// SaveWorkCalendarSettings valida, grava e passa a aplicar a configuração.
// Devolve a versão normalizada (UF em maiúsculas, listas ordenadas).
func (a *App) SaveWorkCalendarSettings(settings config.CalendarSettings) (config.CalendarSettings, error) {
	normalized, err := holidays.Validate(settings)
	if err != nil {
		return config.CalendarSettings{}, err
	}
	if err := a.configManager.SetCalendarSettings(normalized); err != nil {
		return config.CalendarSettings{}, fmt.Errorf("erro ao salvar calendário: %v", err)
	}
	if a.calendar != nil {
		a.calendar.Update(normalized)
	}
	// O dashboard guarda em cache a contagem de dias úteis do mês.
	if client := a.api(); client != nil {
		client.InvalidateCalendarCaches()
	}
	return normalized, nil
}

// GetBrazilianStates lista as UFs e os feriados estaduais de data fixa
// embutidos no aplicativo.
func (a *App) GetBrazilianStates() []holidays.State {
	return holidays.States()
}

// GetExtraNonWorkingDays lista os dias não úteis extras do ano (estaduais,
// personalizados e férias) segundo a configuração atual.
func (a *App) GetExtraNonWorkingDays(year int) ([]holidays.DayInfo, error) {
	if year < 1900 || year > 2200 {
		return nil, fmt.Errorf("ano inválido: %d", year)
	}
	if a.calendar == nil {
		return []holidays.DayInfo{}, nil
	}
	return a.calendar.DaysInYear(year), nil
}
