package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLembretesECronometroComPadroesEmInstalacaoNova(t *testing.T) {
	fakeKeyring(t)
	m, err := newManagerAt(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if got := m.GetReminderSettings(); got != DefaultReminderSettings() {
		t.Errorf("lembretes = %+v, esperava os padrões", got)
	}
	if got := m.GetTimerSettings(); got != DefaultTimerSettings() {
		t.Errorf("cronômetro = %+v, esperava os padrões", got)
	}
}

// Um config.json anterior às seções novas fica com os padrões; um parcial só
// troca o que trouxe.
func TestLembretesConfigAntigoRecebePadroes(t *testing.T) {
	fakeKeyring(t)
	dir := t.TempDir()
	antigo := `{"appSettings":{"darkMode":true},"reminders":{"dailyTime":"17:30"},"timer":{"rounding":"esquisito"}}`
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(antigo), 0600); err != nil {
		t.Fatal(err)
	}

	m, err := newManagerAt(dir)
	if err != nil {
		t.Fatal(err)
	}
	r := m.GetReminderSettings()
	if !r.Enabled || r.DailyTime != "17:30" || !r.WorkDaysOnly || !r.MonthEndEnabled || r.MonthEndDays != 2 {
		t.Errorf("lembretes = %+v", r)
	}
	if tm := m.GetTimerSettings(); tm.Rounding != TimerRoundingExact || tm.LongRunningHours != 4 {
		t.Errorf("cronômetro = %+v", tm)
	}
}

func TestSetReminderSettingsValidaEPersiste(t *testing.T) {
	fakeKeyring(t)
	dir := t.TempDir()
	m, err := newManagerAt(dir)
	if err != nil {
		t.Fatal(err)
	}

	invalido := DefaultReminderSettings()
	invalido.DailyTime = "25:99"
	if err := m.SetReminderSettings(invalido); err == nil {
		t.Error("esperava erro para horário inválido")
	}

	novo := DefaultReminderSettings()
	novo.Enabled = false
	novo.DailyTime = "09:15"
	if err := m.SetReminderSettings(novo); err != nil {
		t.Fatal(err)
	}
	if err := m.SetTimerSettings(TimerSettings{Rounding: TimerRounding15, LongRunningHours: 2}); err != nil {
		t.Fatal(err)
	}
	if err := m.SetTimerSettings(TimerSettings{Rounding: "x"}); err == nil {
		t.Error("esperava erro para arredondamento inválido")
	}

	recarregado, err := newManagerAt(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := recarregado.GetReminderSettings(); got != novo {
		t.Errorf("lembretes recarregados = %+v, esperava %+v", got, novo)
	}
	if got := recarregado.GetTimerSettings(); got.Rounding != TimerRounding15 || got.LongRunningHours != 2 {
		t.Errorf("cronômetro recarregado = %+v", got)
	}
}
