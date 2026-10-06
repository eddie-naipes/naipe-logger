package config

// Calendário de trabalho do usuário: o que, além dos fins de semana e dos
// feriados nacionais, não conta como dia útil. A validação (UF conhecida,
// datas, tipos) fica em backend/holidays; aqui só se guarda e devolve cópias.

// Tipos aceitos em CustomHoliday.Type.
const (
	CustomHolidayMunicipal = "municipal"
	CustomHolidayBridge    = "ponte"
	CustomHolidayOther     = "outro"
)

// CalendarSettings é a parte de config.json com os dias não úteis extras.
// Um config.json anterior a ela decodifica com tudo vazio: nenhuma UF, nenhum
// feriado personalizado e nenhuma ausência — exatamente o comportamento antigo.
type CalendarSettings struct {
	// UF é a sigla do estado cujos feriados estaduais valem ("" = nenhum).
	UF string `json:"uf"`
	// DisabledStateHolidays lista os feriados estaduais ("MM-DD") que o
	// usuário desligou, por exemplo quando a empresa trabalha nesse dia.
	DisabledStateHolidays []string `json:"disabledStateHolidays"`
	// CustomHolidays são feriados municipais, pontes e outras folgas.
	CustomHolidays []CustomHoliday `json:"customHolidays"`
	// Absences são férias, licenças e outras ausências por período.
	Absences []Absence `json:"absences"`
}

// CustomHoliday é um dia de folga cadastrado pelo usuário. Date é sempre
// "AAAA-MM-DD"; com Recurring o ano é ignorado e o dia/mês vale todo ano.
type CustomHoliday struct {
	Date      string `json:"date"`
	Recurring bool   `json:"recurring"`
	Name      string `json:"name"`
	Type      string `json:"type"`
}

// Absence é um período de férias ou ausência, com início e fim inclusivos
// ("AAAA-MM-DD").
type Absence struct {
	Start       string `json:"start"`
	End         string `json:"end"`
	Description string `json:"description"`
}

// Normalized devolve uma cópia sem slices nil, para que o frontend receba []
// e não null, e sem compartilhar memória com quem a recebeu.
func (c CalendarSettings) Normalized() CalendarSettings {
	out := CalendarSettings{UF: c.UF}
	out.DisabledStateHolidays = append([]string{}, c.DisabledStateHolidays...)
	out.CustomHolidays = append([]CustomHoliday{}, c.CustomHolidays...)
	out.Absences = append([]Absence{}, c.Absences...)
	return out
}

// GetCalendarSettings devolve uma cópia da configuração do calendário.
func (m *Manager) GetCalendarSettings() CalendarSettings {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.appConfig.Calendar.Normalized()
}

// SetCalendarSettings grava a configuração do calendário. Se a gravação
// falhar, a memória volta ao valor anterior.
func (m *Manager) SetCalendarSettings(settings CalendarSettings) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	previous := m.appConfig.Calendar
	m.appConfig.Calendar = settings.Normalized()
	if err := m.saveLocked(); err != nil {
		m.appConfig.Calendar = previous
		return err
	}
	return nil
}
