package api

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

// mesmoJSON confere que o tipo e o mapa equivalente serializam igual: é o que
// garante que trocar o retorno dos bindings não muda nada para o frontend.
func mesmoJSON(t *testing.T, nome string, tipado interface{}, mapa interface{}) {
	t.Helper()

	var a, b interface{}
	bytesTipado, _ := json.Marshal(tipado)
	bytesMapa, _ := json.Marshal(mapa)
	_ = json.Unmarshal(bytesTipado, &a)
	_ = json.Unmarshal(bytesMapa, &b)

	if !reflect.DeepEqual(a, b) {
		t.Errorf("%s: JSON difere\n tipado: %s\n mapa:   %s", nome, bytesTipado, bytesMapa)
	}
}

func TestTiposDoDashboardMantemOJSONDosMapas(t *testing.T) {
	stats := DashboardStats{TarefasPendentes: 3, Projetos: 2, HorasLogadas: 12.5, HorasLogadasChange: -10,
		DiasUteisMes: 22, DiasUteisRestantes: 5, DiasUteisPassados: 17}
	mesmoJSON(t, "DashboardStats", stats, stats.toMap())

	atividade := RecentActivity{ID: 1, Type: "timelog", Description: "d", Minutes: 30, Date: "2025-09-01",
		ProjectID: 2, ProjectName: "P", TaskID: 3, TaskName: "T"}
	mesmoJSON(t, "RecentActivity", atividade, atividade.toMap())

	prazo := UpcomingDeadline{ID: 1, Name: "n", DueDate: "2025-09-30", Priority: "high", ProjectID: 2, ProjectName: "P"}
	mesmoJSON(t, "UpcomingDeadline", prazo, prazo.toMap())

	feriado := NonWorkingDay{Date: "2025-11-20", Type: nonWorkingDayHoliday, Name: "Consciência Negra", IsOptional: true, Description: "x"}
	mesmoJSON(t, "NonWorkingDay feriado", feriado, feriado.toMap())

	fimDeSemana := NonWorkingDay{Date: "2025-11-22", Type: nonWorkingDayWeekend, Name: "Saturday"}
	mesmoJSON(t, "NonWorkingDay fim de semana", fimDeSemana, fimDeSemana.toMap())
}

func TestNonWorkingDayMapaPreservaFormatoAntigo(t *testing.T) {
	// Feriado sempre trazia description e isOptional, mesmo vazios.
	feriado := NonWorkingDay{Date: "2025-12-25", Type: nonWorkingDayHoliday, Name: "Natal"}.toMap()
	if _, ok := feriado["description"]; !ok {
		t.Error("feriado sem a chave description")
	}
	if v, ok := feriado["isOptional"]; !ok || v != false {
		t.Errorf("feriado com isOptional = %v, %v", v, ok)
	}

	// Fim de semana só tinha date, type e name.
	fimDeSemana := NonWorkingDay{Date: "2025-12-27", Type: nonWorkingDayWeekend, Name: "Saturday"}.toMap()
	if len(fimDeSemana) != 3 {
		t.Errorf("fim de semana com chaves extras: %v", fimDeSemana)
	}
}

func TestGetDashboardStatsDevolveMapaNovoACadaChamada(t *testing.T) {
	api := NewTeamworkAPI(Config{AuthToken: "x", ApiHost: "empresa.teamwork.com", UserID: 1})
	api.cache.Set("dashboard_stats_1", DashboardStats{HorasLogadas: 8}, time.Hour)

	primeiro, err := api.GetDashboardStats()
	if err != nil {
		t.Fatalf("GetDashboardStats: %v", err)
	}
	// O binding ajustava "horasLogadas" no mapa devolvido; isso não pode
	// vazar para o cache.
	primeiro["horasLogadas"] = 99.0

	segundo, _ := api.GetDashboardStats()
	if segundo["horasLogadas"] != 8.0 {
		t.Errorf("horasLogadas = %v; mutar o mapa devolvido contaminou o cache", segundo["horasLogadas"])
	}
}

func TestHolidayCacheStatsMantemOJSON(t *testing.T) {
	seedHolidayCache(t, 2031, "2031-01-01")

	api := &TeamworkAPI{}
	mesmoJSON(t, "HolidayCacheStats", api.HolidayCacheSummary(), api.GetHolidayCacheStats())

	stats := api.HolidayCacheSummary()
	detalhe, ok := stats.CacheDetails[2031]
	if !ok || detalhe.HolidaysCount != 1 {
		t.Errorf("detalhe de 2031 = %+v, %v", detalhe, ok)
	}
}
