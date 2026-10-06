package holidays

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"logTime-go/backend/api"
	"logTime-go/backend/config"
)

const (
	dateLayout = "2006-01-02"
	// maxAbsenceDays limita um período de ausência (dois anos), para que uma
	// data digitada errada não gere um intervalo de séculos.
	maxAbsenceDays = 731
	maxNameLength  = 120
)

// Validate confere e normaliza a configuração antes de gravá-la: UF
// conhecida, datas AAAA-MM-DD reais, tipos aceitos, fim >= início. Devolve a
// versão normalizada (UF em maiúsculas, textos aparados, listas ordenadas).
func Validate(settings config.CalendarSettings) (config.CalendarSettings, error) {
	out := settings.Normalized()

	out.UF = strings.ToUpper(strings.TrimSpace(out.UF))
	var state State
	if out.UF != "" {
		var ok bool
		state, ok = StateByUF(out.UF)
		if !ok {
			return config.CalendarSettings{}, fmt.Errorf("UF desconhecida: %q", settings.UF)
		}
	}

	// Desligar só faz sentido para feriados que a UF escolhida tem; o resto
	// é descartado em silêncio (ex.: o usuário trocou de UF).
	disabled := make([]string, 0, len(out.DisabledStateHolidays))
	seen := map[string]bool{}
	for _, md := range out.DisabledStateHolidays {
		md = strings.TrimSpace(md)
		if seen[md] || !stateHasMonthDay(state, md) {
			continue
		}
		seen[md] = true
		disabled = append(disabled, md)
	}
	sort.Strings(disabled)
	out.DisabledStateHolidays = disabled

	for i := range out.CustomHolidays {
		h := &out.CustomHolidays[i]
		h.Name = strings.TrimSpace(h.Name)
		h.Type = strings.TrimSpace(h.Type)
		h.Date = strings.TrimSpace(h.Date)
		if _, err := time.Parse(dateLayout, h.Date); err != nil {
			return config.CalendarSettings{}, fmt.Errorf("feriado %q: data inválida (use AAAA-MM-DD): %q", h.Name, h.Date)
		}
		if h.Name == "" {
			return config.CalendarSettings{}, fmt.Errorf("feriado em %s sem nome", h.Date)
		}
		if len([]rune(h.Name)) > maxNameLength {
			return config.CalendarSettings{}, fmt.Errorf("nome do feriado muito longo (máx. %d caracteres)", maxNameLength)
		}
		switch h.Type {
		case config.CustomHolidayMunicipal, config.CustomHolidayBridge, config.CustomHolidayOther:
		default:
			return config.CalendarSettings{}, fmt.Errorf("feriado %q: tipo inválido %q (use municipal, ponte ou outro)", h.Name, h.Type)
		}
	}
	sort.SliceStable(out.CustomHolidays, func(i, j int) bool {
		return out.CustomHolidays[i].Date < out.CustomHolidays[j].Date
	})

	for i := range out.Absences {
		a := &out.Absences[i]
		a.Description = strings.TrimSpace(a.Description)
		a.Start = strings.TrimSpace(a.Start)
		a.End = strings.TrimSpace(a.End)
		start, err := time.Parse(dateLayout, a.Start)
		if err != nil {
			return config.CalendarSettings{}, fmt.Errorf("ausência: data inicial inválida (use AAAA-MM-DD): %q", a.Start)
		}
		end, err := time.Parse(dateLayout, a.End)
		if err != nil {
			return config.CalendarSettings{}, fmt.Errorf("ausência: data final inválida (use AAAA-MM-DD): %q", a.End)
		}
		if end.Before(start) {
			return config.CalendarSettings{}, fmt.Errorf("ausência: a data final (%s) é anterior à inicial (%s)", a.End, a.Start)
		}
		if end.Sub(start) > maxAbsenceDays*24*time.Hour {
			return config.CalendarSettings{}, fmt.Errorf("ausência de %s a %s é longa demais (máx. 2 anos)", a.Start, a.End)
		}
		if len([]rune(a.Description)) > maxNameLength {
			return config.CalendarSettings{}, fmt.Errorf("descrição da ausência muito longa (máx. %d caracteres)", maxNameLength)
		}
	}
	sort.SliceStable(out.Absences, func(i, j int) bool { return out.Absences[i].Start < out.Absences[j].Start })

	return out, nil
}

func stateHasMonthDay(state State, md string) bool {
	for _, h := range state.Holidays {
		if h.MonthDay == md {
			return true
		}
	}
	return false
}

type absenceRange struct {
	start, end  string
	description string
}

// Calendar é a configuração já compilada para consultas rápidas por dia.
type Calendar struct {
	state     map[string]StateHoliday // "MM-DD"
	fixed     map[string]config.CustomHoliday
	recurring map[string]config.CustomHoliday // "MM-DD"
	absences  []absenceRange
}

// NewCalendar compila a configuração. Entradas inválidas são ignoradas (a
// validação acontece ao salvar; um config.json editado à mão não derruba o
// cálculo de dias úteis).
func NewCalendar(settings config.CalendarSettings) *Calendar {
	c := &Calendar{
		state:     map[string]StateHoliday{},
		fixed:     map[string]config.CustomHoliday{},
		recurring: map[string]config.CustomHoliday{},
	}

	if state, ok := StateByUF(strings.ToUpper(strings.TrimSpace(settings.UF))); ok {
		disabled := map[string]bool{}
		for _, md := range settings.DisabledStateHolidays {
			disabled[md] = true
		}
		for _, h := range state.Holidays {
			if !disabled[h.MonthDay] {
				c.state[h.MonthDay] = h
			}
		}
	}

	for _, h := range settings.CustomHolidays {
		d, err := time.Parse(dateLayout, h.Date)
		if err != nil {
			continue
		}
		if h.Recurring {
			c.recurring[d.Format("01-02")] = h
		} else {
			c.fixed[d.Format(dateLayout)] = h
		}
	}

	for _, a := range settings.Absences {
		start, errS := time.Parse(dateLayout, a.Start)
		end, errE := time.Parse(dateLayout, a.End)
		if errS != nil || errE != nil || end.Before(start) {
			continue
		}
		c.absences = append(c.absences, absenceRange{
			start: start.Format(dateLayout), end: end.Format(dateLayout), description: a.Description,
		})
	}

	return c
}

// customTypes traduz o tipo gravado no config para o tipo de NonWorkingDay.
var customTypes = map[string]string{
	config.CustomHolidayMunicipal: api.NonWorkingDayMunicipal,
	config.CustomHolidayBridge:    api.NonWorkingDayBridge,
	config.CustomHolidayOther:     api.NonWorkingDayCustom,
}

// ExtraNonWorkingDay implementa api.ExtraNonWorkingDays. Num dia que
// acumula motivos vence, nesta ordem: feriado estadual, feriado
// personalizado com data exata, personalizado recorrente, férias.
func (c *Calendar) ExtraNonWorkingDay(date time.Time) (api.ExtraNonWorkingDay, bool) {
	if c == nil {
		return api.ExtraNonWorkingDay{}, false
	}
	ymd := date.Format(dateLayout)
	md := date.Format("01-02")

	if h, ok := c.state[md]; ok {
		return api.ExtraNonWorkingDay{Type: api.NonWorkingDayStateHoliday, Name: h.Name, Description: h.Source}, true
	}
	if h, ok := c.fixed[ymd]; ok {
		return customDay(h), true
	}
	// 29/02 recorrente só existe nos anos bissextos: time.Format nunca gera
	// "02-29" num ano comum, então não há o que tratar.
	if h, ok := c.recurring[md]; ok {
		return customDay(h), true
	}
	for _, a := range c.absences {
		if ymd >= a.start && ymd <= a.end {
			name := a.description
			if name == "" {
				name = "Férias/ausência"
			}
			return api.ExtraNonWorkingDay{Type: api.NonWorkingDayVacation, Name: name}, true
		}
	}
	return api.ExtraNonWorkingDay{}, false
}

func customDay(h config.CustomHoliday) api.ExtraNonWorkingDay {
	t, ok := customTypes[h.Type]
	if !ok {
		t = api.NonWorkingDayCustom
	}
	return api.ExtraNonWorkingDay{Type: t, Name: h.Name}
}

// Provider é o api.ExtraNonWorkingDays compartilhado pela App: o mesmo
// objeto é ligado a cada cliente novo e só o calendário interno é trocado
// quando a configuração muda.
type Provider struct {
	mutex    sync.RWMutex
	calendar *Calendar
}

// NewProvider cria o provedor já com a configuração informada.
func NewProvider(settings config.CalendarSettings) *Provider {
	return &Provider{calendar: NewCalendar(settings)}
}

// Update recompila o calendário com a configuração nova.
func (p *Provider) Update(settings config.CalendarSettings) {
	c := NewCalendar(settings)
	p.mutex.Lock()
	p.calendar = c
	p.mutex.Unlock()
}

// ExtraNonWorkingDay implementa api.ExtraNonWorkingDays.
func (p *Provider) ExtraNonWorkingDay(date time.Time) (api.ExtraNonWorkingDay, bool) {
	p.mutex.RLock()
	c := p.calendar
	p.mutex.RUnlock()
	return c.ExtraNonWorkingDay(date)
}

// DayInfo é um dia não útil extra de um ano, para a lista da tela de feriados.
type DayInfo struct {
	Date        string `json:"date"`
	Type        string `json:"type"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// DaysInYear lista os dias extras do ano, incluindo os que caem em fim de
// semana (a tela mostra tudo o que foi configurado), ordenados por data.
func (p *Provider) DaysInYear(year int) []DayInfo {
	days := make([]DayInfo, 0)
	start := time.Date(year, 1, 1, 0, 0, 0, 0, time.UTC)
	for d := start; d.Year() == year; d = d.AddDate(0, 0, 1) {
		if extra, ok := p.ExtraNonWorkingDay(d); ok {
			days = append(days, DayInfo{
				Date: d.Format(dateLayout), Type: extra.Type, Name: extra.Name, Description: extra.Description,
			})
		}
	}
	return days
}
