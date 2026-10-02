package reports

import (
	"strings"
	"testing"

	"logTime-go/backend/api"
)

func periodo(t *testing.T, ini, fim string) Period {
	t.Helper()
	p, err := ParsePeriod(ini, fim)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func lancamentos() []api.TimeEntryReport {
	return []api.TimeEntryReport{
		{ID: 1, UserID: 42, Date: "2026-06-01", Minutes: 240, ProjectID: 10, ProjectName: "Alfa", TaskID: 100, TaskName: "Dev", IsBillable: true},
		{ID: 2, UserID: 42, Date: "2026-06-01", Minutes: 180, ProjectID: 10, ProjectName: "Alfa", TaskID: 101, TaskName: "Reunião"},
		{ID: 3, UserID: 42, Date: "2026-06-02", Minutes: 480, ProjectID: 20, ProjectName: "Beta", TaskID: 200, TaskName: "Suporte", IsBillable: true},
		{ID: 4, UserID: 42, Date: "2026-06-06", Minutes: 60, ProjectID: 10, ProjectName: "Alfa", TaskID: 100, TaskName: "Dev", IsBillable: true}, // sábado
		// Fora do relatório:
		{ID: 5, UserID: 99, Date: "2026-06-01", Minutes: 600, ProjectID: 10, ProjectName: "Alfa"},                    // outro usuário
		{ID: 6, UserID: 42, Date: "2026-06-03", Minutes: 300, ProjectID: 10, ProjectName: "Alfa", Status: "deleted"}, // excluído
		{ID: 7, UserID: 42, Date: "2026-06-03", Minutes: 300, DeletedAt: "2026-06-04"},
		{ID: 8, UserID: 42, Date: "2026-05-31", Minutes: 300}, // fora do período
	}
}

// 01/06/2026 é segunda. Dias úteis: 01 a 05 (sem feriado no teste).
var uteis = []string{"2026-06-01", "2026-06-02", "2026-06-03", "2026-06-04", "2026-06-05"}

func TestBuildTotaisECobravel(t *testing.T) {
	r := Build(lancamentos(), periodo(t, "2026-06-01", "2026-06-07"), uteis, 480, 42)

	if r.TotalMinutes != 960 || r.BillableMinutes != 780 || r.NonBillableMinutes != 180 {
		t.Errorf("totais: %+v", r)
	}
	if r.EntryCount != 4 {
		t.Errorf("lançamentos = %d", r.EntryCount)
	}
	if r.WorkingDays != 5 || r.ExpectedMinutes != 2400 || r.BalanceMinutes != 960-2400 {
		t.Errorf("jornada: dias=%d esperado=%d saldo=%d", r.WorkingDays, r.ExpectedMinutes, r.BalanceMinutes)
	}
	if r.DaysWithEntries != 3 || r.WorkingDaysWithoutEntries != 3 {
		t.Errorf("dias com lançamento=%d úteis vazios=%d", r.DaysWithEntries, r.WorkingDaysWithoutEntries)
	}
}

func TestBuildPorProjetoETarefa(t *testing.T) {
	r := Build(lancamentos(), periodo(t, "2026-06-01", "2026-06-07"), uteis, 480, 42)

	if len(r.ByProject) != 2 || r.ByProject[0].ProjectName != "Beta" && r.ByProject[0].Minutes != 480 {
		t.Fatalf("projetos: %+v", r.ByProject)
	}
	// Alfa: 240+180+60 = 480; Beta: 480. Empate desempata pelo nome.
	if r.ByProject[0].ProjectName != "Alfa" || r.ByProject[0].Minutes != 480 || r.ByProject[0].EntryCount != 3 {
		t.Errorf("projeto líder: %+v", r.ByProject[0])
	}
	if len(r.ByTask) != 3 {
		t.Fatalf("tarefas: %+v", r.ByTask)
	}
	if r.ByTask[0].TaskName != "Suporte" || r.ByTask[0].ProjectName != "Beta" {
		t.Errorf("tarefa líder: %+v", r.ByTask[0])
	}
	dev := r.ByTask[1]
	if dev.TaskName != "Dev" || dev.Minutes != 300 || dev.BillableMinutes != 300 || dev.EntryCount != 2 {
		t.Errorf("Dev: %+v", dev)
	}
}

func TestBuildPorDiaIncluiDiasSemLancamento(t *testing.T) {
	r := Build(lancamentos(), periodo(t, "2026-06-01", "2026-06-07"), uteis, 480, 42)
	if len(r.ByDay) != 7 {
		t.Fatalf("esperava 7 dias, veio %d", len(r.ByDay))
	}
	quarta := r.ByDay[2]
	if quarta.Date != "2026-06-03" || quarta.Minutes != 0 || !quarta.IsWorkingDay || quarta.ExpectedMinutes != 480 {
		t.Errorf("dia útil sem lançamento: %+v", quarta)
	}
	sabado := r.ByDay[5]
	if sabado.IsWorkingDay || sabado.Minutes != 60 || sabado.ExpectedMinutes != 0 {
		t.Errorf("sábado com lançamento: %+v", sabado)
	}
}

func TestBuildPorSemanaRecortaOPeriodo(t *testing.T) {
	// 03/06 (qua) a 10/06 (qua): semana parcial 03-07 e semana 08-10.
	r := Build(lancamentos(), periodo(t, "2026-06-03", "2026-06-10"),
		append(uteis, "2026-06-08", "2026-06-09", "2026-06-10"), 480, 42)
	if len(r.ByWeek) != 2 {
		t.Fatalf("semanas: %+v", r.ByWeek)
	}
	w := r.ByWeek[0]
	if w.WeekStart != "2026-06-03" || w.WeekEnd != "2026-06-07" || w.WorkingDays != 3 || w.Minutes != 60 || w.ExpectedMinutes != 1440 {
		t.Errorf("primeira semana: %+v", w)
	}
	if r.ByWeek[1].WeekStart != "2026-06-08" || r.ByWeek[1].WeekEnd != "2026-06-10" {
		t.Errorf("segunda semana: %+v", r.ByWeek[1])
	}
}

func TestBuildSemLancamentosDevolveListasVazias(t *testing.T) {
	r := Build(nil, periodo(t, "2026-06-06", "2026-06-07"), nil, 480, 42)
	if r.ByProject == nil || r.ByTask == nil || r.TotalMinutes != 0 || r.WorkingDays != 0 || len(r.ByDay) != 2 {
		t.Errorf("relatório vazio: %+v", r)
	}
}

func TestBuildAgrupaSemNome(t *testing.T) {
	r := Build([]api.TimeEntryReport{{Date: "2026-06-01", Minutes: 30}}, periodo(t, "2026-06-01", "2026-06-01"), nil, 480, 0)
	if len(r.ByTask) != 1 || r.ByTask[0].ProjectName != noProjectName || r.ByTask[0].TaskName != noTaskName {
		t.Errorf("sem nome: %+v", r.ByTask)
	}
}

func TestParsePeriodRecusaEntradasInvalidas(t *testing.T) {
	cases := [][2]string{
		{"../../x", "2026-01-01"},
		{"2026-01-01", "01/02/2026"},
		{"2026-02-01", "2026-01-01"},
		{"2025-01-01", "2026-01-02"}, // 367 dias
	}
	for _, c := range cases {
		if _, err := ParsePeriod(c[0], c[1]); err == nil {
			t.Errorf("ParsePeriod(%q, %q) deveria falhar", c[0], c[1])
		}
	}
	if _, err := ParsePeriod("2024-01-01", "2024-12-31"); err != nil {
		t.Errorf("ano bissexto inteiro deveria caber: %v", err)
	}
}

func TestFileNameUsaAsDatasValidadas(t *testing.T) {
	got := FileName(CSVSummary, periodo(t, "2026-06-01", "2026-06-30"))
	if got != "horas_resumido_2026-06-01_2026-06-30.csv" || strings.ContainsAny(got, `/\`) {
		t.Errorf("nome = %q", got)
	}
}
