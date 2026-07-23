package api

import (
	"testing"
	"time"
)

// seedHolidayCache popula o cache global de feriados para que os testes não
// dependam da BrasilAPI nem da rede.
func seedHolidayCache(t *testing.T, year int, dates ...string) {
	t.Helper()

	holidays := make(map[string]Holiday, len(dates))
	for _, d := range dates {
		holidays[d] = Holiday{Date: d, Name: "Feriado de teste", Type: "nacional"}
	}

	holidayCacheLock.Lock()
	holidayCache[year] = &HolidayCache{
		Year:      year,
		Holidays:  holidays,
		CachedAt:  time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
	}
	holidayCacheLock.Unlock()

	t.Cleanup(func() {
		holidayCacheLock.Lock()
		delete(holidayCache, year)
		holidayCacheLock.Unlock()
	})
}

func TestGetWorkingDaysExcluiFimDeSemanaEFeriado(t *testing.T) {
	// Setembro/2025: 01=segunda ... 07=domingo. 07/09 é feriado mas cai num
	// domingo, então o dia útil realmente perdido é 08/09 (segunda), que
	// marcamos como feriado de teste.
	seedHolidayCache(t, 2025, "2025-09-08")

	api := &TeamworkAPI{Config: Config{AuthToken: "tok", ApiHost: "https://x.com"}, cache: NewCache()}

	got, err := api.GetWorkingDays("2025-09-01", "2025-09-14")
	if err != nil {
		t.Fatalf("GetWorkingDays devolveu erro: %v", err)
	}

	want := []string{
		"2025-09-01", "2025-09-02", "2025-09-03", "2025-09-04", "2025-09-05",
		"2025-09-09", "2025-09-10", "2025-09-11", "2025-09-12",
	}

	if len(got) != len(want) {
		t.Fatalf("GetWorkingDays devolveu %d dias (%v), esperava %d (%v)", len(got), got, len(want), want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("dia[%d] = %s, esperava %s", i, got[i], want[i])
		}
	}
}

func TestGetWorkingDaysValidaIntervalo(t *testing.T) {
	api := &TeamworkAPI{Config: Config{AuthToken: "tok", ApiHost: "https://x.com"}, cache: NewCache()}

	if _, err := api.GetWorkingDays("2025-09-10", "2025-09-01"); err == nil {
		t.Error("GetWorkingDays aceitou fim anterior ao início")
	}
	if _, err := api.GetWorkingDays("10/09/2025", "2025-09-11"); err == nil {
		t.Error("GetWorkingDays aceitou data em formato inválido")
	}
}

func TestCreateDistributionPlanRespeitaWorkingDays(t *testing.T) {
	api := &TeamworkAPI{Config: Config{AuthToken: "tok", ApiHost: "https://x.com"}, cache: NewCache()}

	tasks := []Task{
		{
			TaskID:      1,
			TaskName:    "Só às segundas",
			WorkingDays: []int{1}, // 1 = segunda-feira
			Entries:     []TimeEntry{{Minutes: 60, Time: "09:00"}},
		},
		{
			TaskID:   2,
			TaskName: "Todo dia útil",
			Entries:  []TimeEntry{{Minutes: 120, Time: "14:00"}},
		},
	}

	// 08/09/2025 = segunda, 09/09 = terça.
	plan := api.CreateDistributionPlan([]string{"2025-09-08", "2025-09-09"}, tasks)

	if len(plan) != 2 {
		t.Fatalf("plano tem %d dias, esperava 2", len(plan))
	}

	segunda, terca := plan[0], plan[1]

	if len(segunda.Entries) != 2 {
		t.Errorf("segunda tem %d entradas, esperava 2 (ambas as tarefas)", len(segunda.Entries))
	}
	if segunda.TotalMin != 180 {
		t.Errorf("segunda.TotalMin = %d, esperava 180", segunda.TotalMin)
	}

	if len(terca.Entries) != 1 {
		t.Fatalf("terça tem %d entradas, esperava 1 (tarefa restrita a segunda deve sair)", len(terca.Entries))
	}
	if terca.Entries[0].TaskID != 2 {
		t.Errorf("terça manteve a tarefa errada: TaskID = %d, esperava 2", terca.Entries[0].TaskID)
	}
	if terca.TotalMin != 120 {
		t.Errorf("terça.TotalMin = %d, esperava 120", terca.TotalMin)
	}
}

func TestCreateDistributionPlanOmiteDiasVazios(t *testing.T) {
	api := &TeamworkAPI{Config: Config{AuthToken: "tok", ApiHost: "https://x.com"}, cache: NewCache()}

	tasks := []Task{{
		TaskID:      1,
		WorkingDays: []int{1}, // só segunda
		Entries:     []TimeEntry{{Minutes: 60}},
	}}

	// 09/09 e 10/09 são terça e quarta: nenhuma entrada se aplica.
	plan := api.CreateDistributionPlan([]string{"2025-09-09", "2025-09-10"}, tasks)

	if len(plan) != 0 {
		t.Errorf("plano tem %d dias, esperava 0 (dias sem entrada não entram no plano)", len(plan))
	}
}

func TestCalculateTotalMinutes(t *testing.T) {
	api := &TeamworkAPI{cache: NewCache()}

	tasks := []Task{
		{Entries: []TimeEntry{{Minutes: 120}, {Minutes: 90}, {Minutes: 240}}},
		{Entries: []TimeEntry{{Minutes: 30}}},
	}

	if got := api.CalculateTotalMinutes(tasks); got != 480 {
		t.Errorf("CalculateTotalMinutes = %d, esperava 480", got)
	}
	if got := api.CalculateTotalMinutes(nil); got != 0 {
		t.Errorf("CalculateTotalMinutes(nil) = %d, esperava 0", got)
	}
}

func TestIsWorkDay(t *testing.T) {
	seedHolidayCache(t, 2025, "2025-09-08")

	api := &TeamworkAPI{Config: Config{AuthToken: "tok", ApiHost: "https://x.com"}, cache: NewCache()}

	cases := []struct {
		date string
		want bool
	}{
		{"2025-09-05", true},  // sexta
		{"2025-09-06", false}, // sábado
		{"2025-09-07", false}, // domingo
		{"2025-09-08", false}, // segunda, mas feriado
		{"2025-09-09", true},  // terça
	}

	for _, tc := range cases {
		d, err := time.Parse("2006-01-02", tc.date)
		if err != nil {
			t.Fatalf("data de teste inválida %s: %v", tc.date, err)
		}
		if got := api.IsWorkDay(d); got != tc.want {
			t.Errorf("IsWorkDay(%s) = %v, esperava %v", tc.date, got, tc.want)
		}
	}
}
