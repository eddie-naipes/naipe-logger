package api

import (
	"strings"
	"testing"
	"time"
)

// fakeExtraDays é um provedor de dias extras em memória, indexado por data.
type fakeExtraDays map[string]ExtraNonWorkingDay

func (f fakeExtraDays) ExtraNonWorkingDay(date time.Time) (ExtraNonWorkingDay, bool) {
	d, ok := f[date.Format("2006-01-02")]
	return d, ok
}

func TestGetWorkingDaysExcluiDiasExtras(t *testing.T) {
	// Julho/2026: 08 qua, 09 estadual, 10 ponte, 11-12 fim de semana,
	// 13-14 férias, 15 qua. 15/07 também é marcado como feriado nacional.
	seedHolidayCache(t, 2026, "2026-07-15")

	client := NewTeamworkAPI(Config{})
	client.SetExtraNonWorkingDays(fakeExtraDays{
		"2026-07-09": {Type: NonWorkingDayStateHoliday, Name: "Revolução Constitucionalista"},
		"2026-07-10": {Type: NonWorkingDayBridge, Name: "Ponte"},
		"2026-07-11": {Type: NonWorkingDayVacation, Name: "Férias"}, // sábado
		"2026-07-13": {Type: NonWorkingDayVacation, Name: "Férias"},
		"2026-07-14": {Type: NonWorkingDayVacation, Name: "Férias"},
		"2026-07-15": {Type: NonWorkingDayMunicipal, Name: "Coincide com nacional"},
	})

	dias, err := client.GetWorkingDays("2026-07-08", "2026-07-16")
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(dias, ","); got != "2026-07-08,2026-07-16" {
		t.Errorf("dias úteis = %s", got)
	}
	if client.IsWorkDay(time.Date(2026, 7, 9, 0, 0, 0, 0, time.Local)) {
		t.Error("feriado estadual não é dia útil")
	}

	nwd, err := client.ListNonWorkingDays(2026, 7)
	if err != nil {
		t.Fatal(err)
	}
	porData := map[string][]NonWorkingDay{}
	for _, d := range nwd {
		porData[d.Date] = append(porData[d.Date], d)
	}
	esperado := map[string]string{
		"2026-07-09": NonWorkingDayStateHoliday,
		"2026-07-10": NonWorkingDayBridge,
		"2026-07-11": nonWorkingDayWeekend, // fim de semana vence as férias
		"2026-07-13": NonWorkingDayVacation,
		"2026-07-15": nonWorkingDayHoliday, // nacional vence o municipal
	}
	for d, tipo := range esperado {
		lista := porData[d]
		if len(lista) != 1 || lista[0].Type != tipo {
			t.Errorf("%s: %+v, esperava um único dia do tipo %q", d, lista, tipo)
		}
	}
}

func TestSemProvedorComportamentoAntigo(t *testing.T) {
	seedHolidayCache(t, 2026)
	client := NewTeamworkAPI(Config{})
	client.SetExtraNonWorkingDays(fakeExtraDays{"2026-07-09": {Type: NonWorkingDayStateHoliday}})
	client.SetExtraNonWorkingDays(nil)

	if !client.IsWorkDay(time.Date(2026, 7, 9, 0, 0, 0, 0, time.Local)) {
		t.Error("sem provedor, 09/07 volta a ser dia útil")
	}
}

func TestMapaDeDiaExtraTemDescricao(t *testing.T) {
	m := NonWorkingDay{Date: "2026-07-13", Type: NonWorkingDayVacation, Name: "Férias"}.toMap()
	if _, ok := m["description"]; !ok {
		t.Errorf("dias não úteis que não são fim de semana mantêm description/isOptional: %v", m)
	}
}

func TestSetExtraNonWorkingDaysInvalidaDashboard(t *testing.T) {
	client := NewTeamworkAPI(Config{UserID: 1})
	client.cache.Set(cacheKeyDashboardStatsPrefix+"1", DashboardStats{DiasUteisMes: 22}, time.Hour)
	client.SetExtraNonWorkingDays(fakeExtraDays{})
	if _, found := client.cache.Get(cacheKeyDashboardStatsPrefix + "1"); found {
		t.Error("mudar o calendário deveria descartar a contagem de dias úteis em cache")
	}
}
