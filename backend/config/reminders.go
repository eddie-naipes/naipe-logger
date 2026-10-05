package config

import (
	"fmt"
	"time"
)

// ReminderSettings configura os lembretes de lançamento de horas (seção
// "reminders" de config.json). Como Load decodifica sobre defaultAppConfig, um
// config.json antigo (sem a seção ou com campos faltando) fica com os padrões.
type ReminderSettings struct {
	// Enabled liga os lembretes. Padrão true.
	Enabled bool `json:"enabled"`
	// DailyTime é o horário local do lembrete diário, "HH:MM". Padrão "18:00".
	DailyTime string `json:"dailyTime"`
	// WorkDaysOnly só lembra em dias úteis (sem fim de semana nem feriado).
	// Padrão true.
	WorkDaysOnly bool `json:"workDaysOnly"`
	// MonthEndEnabled liga o lembrete de fim de mês, que lista os dias úteis
	// do mês abaixo da jornada. Padrão true.
	MonthEndEnabled bool `json:"monthEndEnabled"`
	// MonthEndDays é quantos dos últimos dias úteis do mês recebem o lembrete
	// de fim de mês. Padrão 2.
	MonthEndDays int `json:"monthEndDays"`
}

// DefaultReminderSettings devolve os padrões dos lembretes.
func DefaultReminderSettings() ReminderSettings {
	return ReminderSettings{
		Enabled:         true,
		DailyTime:       "18:00",
		WorkDaysOnly:    true,
		MonthEndEnabled: true,
		MonthEndDays:    2,
	}
}

// ParseDailyTime interpreta "HH:MM" e devolve hora e minuto.
func ParseDailyTime(value string) (hour, minute int, err error) {
	t, err := time.Parse("15:04", value)
	if err != nil {
		return 0, 0, fmt.Errorf("horário inválido %q: use HH:MM", value)
	}
	return t.Hour(), t.Minute(), nil
}

// normalized corrige valores inválidos vindos de um config.json editado à mão.
func (r ReminderSettings) normalized() ReminderSettings {
	def := DefaultReminderSettings()
	if _, _, err := ParseDailyTime(r.DailyTime); err != nil {
		r.DailyTime = def.DailyTime
	}
	if r.MonthEndDays <= 0 {
		r.MonthEndDays = def.MonthEndDays
	}
	return r
}

// GetReminderSettings devolve a configuração dos lembretes.
func (m *Manager) GetReminderSettings() ReminderSettings {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.appConfig.Reminders.normalized()
}

// SetReminderSettings valida e grava a configuração dos lembretes.
func (m *Manager) SetReminderSettings(settings ReminderSettings) error {
	if _, _, err := ParseDailyTime(settings.DailyTime); err != nil {
		return err
	}
	if settings.MonthEndDays < 1 || settings.MonthEndDays > 10 {
		return fmt.Errorf("dias de fim de mês inválidos: %d (use de 1 a 10)", settings.MonthEndDays)
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.appConfig.Reminders = settings
	return m.saveLocked()
}
