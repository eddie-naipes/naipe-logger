package holidays

import (
	"strings"
	"testing"
	"time"

	"logTime-go/backend/api"
	"logTime-go/backend/config"
)

func dia(t *testing.T, s string) time.Time {
	t.Helper()
	d, err := time.ParseInLocation("2006-01-02", s, time.Local)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func TestFeriadoEstadualDaUFEscolhida(t *testing.T) {
	c := NewCalendar(config.CalendarSettings{UF: "SP"})
	got, ok := c.ExtraNonWorkingDay(dia(t, "2026-07-09"))
	if !ok || got.Type != api.NonWorkingDayStateHoliday || !strings.Contains(got.Name, "Constitucionalista") {
		t.Errorf("09/07 em SP: %+v %v", got, ok)
	}
	if _, ok := c.ExtraNonWorkingDay(dia(t, "2026-04-23")); ok {
		t.Error("São Jorge é do RJ, não de SP")
	}
	if _, ok := NewCalendar(config.CalendarSettings{}).ExtraNonWorkingDay(dia(t, "2026-07-09")); ok {
		t.Error("sem UF não há feriado estadual")
	}
}

func TestFeriadoEstadualDesligado(t *testing.T) {
	c := NewCalendar(config.CalendarSettings{UF: "SP", DisabledStateHolidays: []string{"07-09"}})
	if _, ok := c.ExtraNonWorkingDay(dia(t, "2026-07-09")); ok {
		t.Error("feriado desligado pelo usuário não deveria contar")
	}
}

func TestFeriadoRecorrenteValeTodoAno(t *testing.T) {
	c := NewCalendar(config.CalendarSettings{CustomHolidays: []config.CustomHoliday{
		{Date: "2020-01-25", Recurring: true, Name: "Aniversário da cidade", Type: config.CustomHolidayMunicipal},
		{Date: "2026-11-20", Name: "Ponte", Type: config.CustomHolidayBridge},
	}})
	for _, d := range []string{"2020-01-25", "2026-01-25", "2031-01-25"} {
		got, ok := c.ExtraNonWorkingDay(dia(t, d))
		if !ok || got.Type != api.NonWorkingDayMunicipal {
			t.Errorf("%s: %+v %v", d, got, ok)
		}
	}
	if got, ok := c.ExtraNonWorkingDay(dia(t, "2026-11-20")); !ok || got.Type != api.NonWorkingDayBridge {
		t.Errorf("ponte: %+v %v", got, ok)
	}
	if _, ok := c.ExtraNonWorkingDay(dia(t, "2027-11-20")); ok {
		t.Error("feriado não recorrente não deveria valer em outro ano")
	}
}

func TestFeriadoRecorrente29DeFevereiroSoEmAnoBissexto(t *testing.T) {
	c := NewCalendar(config.CalendarSettings{CustomHolidays: []config.CustomHoliday{
		{Date: "2024-02-29", Recurring: true, Name: "Bissexto", Type: config.CustomHolidayOther},
	}})
	if got, ok := c.ExtraNonWorkingDay(dia(t, "2028-02-29")); !ok || got.Type != api.NonWorkingDayCustom {
		t.Errorf("2028-02-29: %+v %v", got, ok)
	}
	for _, d := range []string{"2027-02-28", "2027-03-01"} {
		if _, ok := c.ExtraNonWorkingDay(dia(t, d)); ok {
			t.Errorf("%s não deveria virar feriado", d)
		}
	}
}

func TestFeriasAtravessandoMesesEAnos(t *testing.T) {
	c := NewCalendar(config.CalendarSettings{Absences: []config.Absence{
		{Start: "2026-12-21", End: "2027-01-08", Description: "Férias de fim de ano"},
	}})
	for _, d := range []string{"2026-12-21", "2026-12-31", "2027-01-01", "2027-01-08"} {
		got, ok := c.ExtraNonWorkingDay(dia(t, d))
		if !ok || got.Type != api.NonWorkingDayVacation || got.Name != "Férias de fim de ano" {
			t.Errorf("%s: %+v %v", d, got, ok)
		}
	}
	for _, d := range []string{"2026-12-20", "2027-01-09"} {
		if _, ok := c.ExtraNonWorkingDay(dia(t, d)); ok {
			t.Errorf("%s está fora das férias", d)
		}
	}
}

func TestPrioridadeEstadualAntesDeFerias(t *testing.T) {
	c := NewCalendar(config.CalendarSettings{
		UF:       "BA",
		Absences: []config.Absence{{Start: "2026-06-29", End: "2026-07-10"}},
	})
	got, _ := c.ExtraNonWorkingDay(dia(t, "2026-07-02"))
	if got.Type != api.NonWorkingDayStateHoliday {
		t.Errorf("02/07 na BA durante férias deveria aparecer como estadual: %+v", got)
	}
	got, _ = c.ExtraNonWorkingDay(dia(t, "2026-07-03"))
	if got.Type != api.NonWorkingDayVacation || got.Name != "Férias/ausência" {
		t.Errorf("férias sem descrição: %+v", got)
	}
}

func TestValidateNormalizaEAceitaConfigValida(t *testing.T) {
	out, err := Validate(config.CalendarSettings{
		UF:                    " rj ",
		DisabledStateHolidays: []string{"04-23", "07-09", "04-23"},
		CustomHolidays: []config.CustomHoliday{
			{Date: "2026-12-24", Name: " Véspera ", Type: "ponte"},
			{Date: "2026-01-20", Name: "São Sebastião", Type: "municipal", Recurring: true},
		},
		Absences: []config.Absence{{Start: "2026-07-01", End: "2026-07-15"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if out.UF != "RJ" {
		t.Errorf("UF = %q", out.UF)
	}
	// 07-09 é de SP: descartado; repetição removida.
	if len(out.DisabledStateHolidays) != 1 || out.DisabledStateHolidays[0] != "04-23" {
		t.Errorf("desligados = %v", out.DisabledStateHolidays)
	}
	if out.CustomHolidays[0].Date != "2026-01-20" || out.CustomHolidays[1].Name != "Véspera" {
		t.Errorf("feriados não ordenados/aparados: %+v", out.CustomHolidays)
	}
}

func TestValidateRecusaEntradasInvalidas(t *testing.T) {
	cases := map[string]config.CalendarSettings{
		"UF desconhecida":       {UF: "XX"},
		"data impossível":       {CustomHolidays: []config.CustomHoliday{{Date: "2026-02-30", Name: "x", Type: "outro"}}},
		"data em outro formato": {CustomHolidays: []config.CustomHoliday{{Date: "25/01/2026", Name: "x", Type: "outro"}}},
		"sem nome":              {CustomHolidays: []config.CustomHoliday{{Date: "2026-01-25", Type: "outro"}}},
		"tipo inválido":         {CustomHolidays: []config.CustomHoliday{{Date: "2026-01-25", Name: "x", Type: "nacional"}}},
		"fim antes do início":   {Absences: []config.Absence{{Start: "2026-02-10", End: "2026-02-01"}}},
		"ausência longa":        {Absences: []config.Absence{{Start: "2026-01-01", End: "2030-01-01"}}},
		"data de ausência":      {Absences: []config.Absence{{Start: "", End: "2026-02-01"}}},
	}
	for nome, s := range cases {
		if _, err := Validate(s); err == nil {
			t.Errorf("%s: deveria falhar", nome)
		}
	}
}

func TestProviderUpdateTrocaCalendario(t *testing.T) {
	p := NewProvider(config.CalendarSettings{})
	if _, ok := p.ExtraNonWorkingDay(dia(t, "2026-07-09")); ok {
		t.Fatal("sem configuração não há dia extra")
	}
	p.Update(config.CalendarSettings{UF: "SP"})
	if _, ok := p.ExtraNonWorkingDay(dia(t, "2026-07-09")); !ok {
		t.Error("após Update o feriado de SP deveria valer")
	}
	dias := p.DaysInYear(2026)
	if len(dias) != 1 || dias[0].Date != "2026-07-09" {
		t.Errorf("DaysInYear = %+v", dias)
	}
}
