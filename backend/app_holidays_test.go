package backend

import (
	"testing"

	"logTime-go/backend/api"
)

func TestSortedHolidaysOrdenaPorData(t *testing.T) {
	got := sortedHolidays(map[string]api.Holiday{
		"2026-12-25": {Date: "2026-12-25", Name: "Natal"},
		"2026-01-01": {Date: "2026-01-01", Name: "Ano Novo"},
		"2026-04-21": {Date: "2026-04-21", Name: "Tiradentes"},
	})
	if len(got) != 3 || got[0].Date != "2026-01-01" || got[1].Date != "2026-04-21" || got[2].Date != "2026-12-25" {
		t.Errorf("ordem inesperada: %+v", got)
	}
	if vazio := sortedHolidays(nil); vazio == nil || len(vazio) != 0 {
		t.Errorf("mapa nil deveria virar lista vazia (JSON [] e não null): %#v", vazio)
	}
}
