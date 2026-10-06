package api

import (
	"sync"
	"time"
)

// Tipos de NonWorkingDay além de "weekend" e "holiday" (feriado nacional).
// Vêm do calendário configurado pelo usuário (backend/holidays): feriados do
// estado escolhido, feriados municipais/pontes/outros cadastrados à mão e
// períodos de férias ou ausência.
const (
	NonWorkingDayStateHoliday = "state_holiday"
	NonWorkingDayMunicipal    = "municipal"
	NonWorkingDayBridge       = "bridge"
	NonWorkingDayCustom       = "custom"
	NonWorkingDayVacation     = "vacation"
)

// ExtraNonWorkingDay é um dia sem expediente que não é fim de semana nem
// feriado nacional.
type ExtraNonWorkingDay struct {
	Type        string
	Name        string
	Description string
}

// ExtraNonWorkingDays informa os dias não úteis vindos da configuração do
// usuário. Precisa ser barato e não tocar a rede: é consultado uma vez por dia
// em GetWorkingDays.
type ExtraNonWorkingDays interface {
	ExtraNonWorkingDay(date time.Time) (ExtraNonWorkingDay, bool)
}

// extraDaysHolder guarda o provedor do cliente sob lock próprio: ele é
// trocado pela tela de configuração enquanto outros bindings calculam dias
// úteis em paralelo.
type extraDaysHolder struct {
	mutex    sync.RWMutex
	provider ExtraNonWorkingDays
}

// SetExtraNonWorkingDays associa ao cliente o provedor de dias não úteis
// extras (nil remove). Sem provedor só valem fins de semana e feriados
// nacionais, como antes. Os números do dashboard dependem dos dias úteis,
// então o cache deles é descartado.
func (t *TeamworkAPI) SetExtraNonWorkingDays(provider ExtraNonWorkingDays) {
	t.extraDays.mutex.Lock()
	t.extraDays.provider = provider
	t.extraDays.mutex.Unlock()

	t.InvalidateCalendarCaches()
}

// InvalidateCalendarCaches descarta o que foi calculado a partir dos dias
// úteis. Chamado quando a configuração de feriados/férias muda.
func (t *TeamworkAPI) InvalidateCalendarCaches() {
	if t.cache != nil {
		t.cache.DeletePrefix(cacheKeyDashboardStatsPrefix)
	}
}

// extraNonWorkingDay consulta o provedor do cliente, se houver.
func (t *TeamworkAPI) extraNonWorkingDay(date time.Time) (ExtraNonWorkingDay, bool) {
	t.extraDays.mutex.RLock()
	provider := t.extraDays.provider
	t.extraDays.mutex.RUnlock()
	if provider == nil {
		return ExtraNonWorkingDay{}, false
	}
	return provider.ExtraNonWorkingDay(date)
}

// appendExtraNonWorkingDays acrescenta os dias extras do período que caem em
// dia de semana e não são feriado nacional: esses já estão na lista com o tipo
// "holiday" e o calendário mostra um rótulo por dia.
func (t *TeamworkAPI) appendExtraNonWorkingDays(days []NonWorkingDay, start, end time.Time) []NonWorkingDay {
	listed := make(map[string]bool, len(days))
	for _, d := range days {
		listed[d.Date] = true
	}

	for current := start; !current.After(end); current = current.AddDate(0, 0, 1) {
		date := formatDate(current)
		if listed[date] {
			continue
		}
		extra, ok := t.extraNonWorkingDay(current)
		if !ok {
			continue
		}
		days = append(days, NonWorkingDay{
			Date:        date,
			Type:        extra.Type,
			Name:        extra.Name,
			Description: extra.Description,
		})
	}
	return days
}
