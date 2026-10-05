package config

import (
	"os"
	"path/filepath"
	"testing"
)

// Um config.json de versão anterior não tem "calendar": tudo vazio (nenhuma
// UF, feriado ou ausência), com listas [] e não nil.
func TestCalendarioAusenteNoConfigAntigoFicaVazio(t *testing.T) {
	fakeKeyring(t)
	dir := t.TempDir()
	antigo := `{"teamworkConfig":{"apiHost":"https://x.teamwork.com","userId":7,"minutosPorDia":480},"savedTasks":[],"appSettings":{"language":"pt-BR"}}`
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(antigo), 0600); err != nil {
		t.Fatal(err)
	}

	m, err := newManagerAt(dir)
	if err != nil {
		t.Fatal(err)
	}
	c := m.GetCalendarSettings()
	if c.UF != "" || c.CustomHolidays == nil || c.Absences == nil || c.DisabledStateHolidays == nil {
		t.Fatalf("calendário de config antigo: %+v", c)
	}
	if len(c.CustomHolidays) != 0 || len(c.Absences) != 0 {
		t.Errorf("não deveria inventar dias: %+v", c)
	}
	if m.GetTeamworkConfig().UserID != 7 {
		t.Error("o resto do config antigo deveria continuar lido")
	}
}

func TestCalendarioPersisteEntreReinicios(t *testing.T) {
	fakeKeyring(t)
	dir := t.TempDir()
	m, err := newManagerAt(dir)
	if err != nil {
		t.Fatal(err)
	}

	err = m.SetCalendarSettings(CalendarSettings{
		UF:             "SP",
		CustomHolidays: []CustomHoliday{{Date: "2026-01-25", Recurring: true, Name: "Aniversário de SP", Type: CustomHolidayMunicipal}},
		Absences:       []Absence{{Start: "2026-12-21", End: "2027-01-08", Description: "Férias"}},
	})
	if err != nil {
		t.Fatal(err)
	}

	recarregado, err := newManagerAt(dir)
	if err != nil {
		t.Fatal(err)
	}
	c := recarregado.GetCalendarSettings()
	if c.UF != "SP" || len(c.CustomHolidays) != 1 || len(c.Absences) != 1 || !c.CustomHolidays[0].Recurring {
		t.Errorf("calendário recarregado: %+v", c)
	}
}

// GetCalendarSettings devolve cópia: mexer nela não altera o Manager.
func TestCalendarioDevolveCopia(t *testing.T) {
	fakeKeyring(t)
	m, err := newManagerAt(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := m.SetCalendarSettings(CalendarSettings{Absences: []Absence{{Start: "2026-01-01", End: "2026-01-02"}}}); err != nil {
		t.Fatal(err)
	}
	c := m.GetCalendarSettings()
	c.Absences[0].Start = "1999-01-01"
	if m.GetCalendarSettings().Absences[0].Start != "2026-01-01" {
		t.Error("a cópia devolvida compartilha memória com o Manager")
	}
}
