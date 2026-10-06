package planning

import (
	"reflect"
	"testing"

	"logTime-go/backend/api"
)

func lanc(id int, data string, taskID, minutos int, descricao string) api.TimeEntryReport {
	return api.TimeEntryReport{
		ID: id, Date: data, TaskID: taskID, TaskName: "Tarefa", ProjectName: "Projeto",
		Minutes: minutos, Description: descricao, IsBillable: true,
	}
}

func TestWeekDays(t *testing.T) {
	casos := map[string]string{
		"2025-09-15": "2025-09-15", // segunda
		"2025-09-17": "2025-09-15", // quarta
		"2025-09-21": "2025-09-15", // domingo pertence à semana que começou na segunda
		"2025-09-22": "2025-09-22",
	}
	for data, segunda := range casos {
		got, dias, err := WeekDays(data)
		if err != nil {
			t.Fatalf("%s: erro %v", data, err)
		}
		if got != segunda || len(dias) != 7 || dias[0] != segunda {
			t.Errorf("WeekDays(%s) = %s %v, esperava início %s", data, got, dias, segunda)
		}
	}
	if _, _, err := WeekDays("15/09/2025"); err == nil {
		t.Error("esperava erro para data inválida")
	}
}

func TestBuildWeekGridAgregaPorTarefaEDia(t *testing.T) {
	entries := []api.TimeEntryReport{
		lanc(1, "2025-09-15", 10, 120, "a"),
		lanc(2, "2025-09-15", 10, 60, "b"),
		lanc(3, "2025-09-16", 10, 30, "c"),
		lanc(4, "2025-09-16", 20, 240, "d"),
		lanc(5, "2025-09-08", 10, 999, "semana anterior: ignorado"),
	}
	entries[3].ProjectName = "Alfa"
	salvas := []api.Task{
		{TaskID: 30, TaskName: "Salva sem lançamento", ProjectName: "Zeta"},
		{TaskID: 10, TaskName: "Tarefa salva", ProjectName: "Projeto"},
	}

	grid, err := BuildWeekGrid("2025-09-17", entries, salvas)
	if err != nil {
		t.Fatal(err)
	}

	if grid.WeekStart != "2025-09-15" || len(grid.Days) != 7 {
		t.Fatalf("semana = %s %v", grid.WeekStart, grid.Days)
	}

	ids := make([]int, 0)
	for _, r := range grid.Rows {
		ids = append(ids, r.TaskID)
	}
	// Com lançamento ordenadas por projeto (Alfa < Projeto), depois as salvas vazias.
	if !reflect.DeepEqual(ids, []int{20, 10, 30}) {
		t.Fatalf("ordem das linhas = %v", ids)
	}

	t10 := grid.Rows[1]
	if !t10.Saved || t10.Total != 210 || t10.Cells[0].Minutes != 180 || t10.Cells[1].Minutes != 30 {
		t.Errorf("linha da tarefa 10 = %+v", t10)
	}
	if len(t10.Cells[0].Entries) != 2 || t10.Cells[0].Entries[0].ID != 1 {
		t.Errorf("a célula deve carregar os IDs dos lançamentos: %+v", t10.Cells[0].Entries)
	}
	if grid.Rows[0].Saved {
		t.Error("tarefa 20 não está salva")
	}
	vazia := grid.Rows[2]
	if !vazia.Saved || vazia.Total != 0 || vazia.TaskName != "Salva sem lançamento" || len(vazia.Cells) != 7 {
		t.Errorf("linha da tarefa salva vazia = %+v", vazia)
	}

	if !reflect.DeepEqual(grid.DayTotals, []int{180, 270, 0, 0, 0, 0, 0}) || grid.Total != 450 {
		t.Errorf("totais = %v / %d", grid.DayTotals, grid.Total)
	}
}

func TestBuildCopyPlan(t *testing.T) {
	anteriores := []api.TimeEntryReport{
		lanc(3, "2025-09-09", 20, 60, "Reunião"), // terça
		lanc(1, "2025-09-08", 10, 240, "Dev"),    // segunda
		lanc(2, "2025-09-08", 20, 120, ""),       // segunda, sem descrição
		lanc(4, "2025-09-10", 10, 480, "Dev"),    // quarta -> feriado na semana atual
		lanc(5, "2025-09-11", 0, 60, "Sem tarefa"),
		lanc(6, "2025-09-13", 10, 60, "Sábado"),
	}
	anteriores[1].IsBillable = false
	anteriores[0].StartTime = "15:30"

	feriado := "2025-09-17"
	util := func(d string) bool {
		return d != feriado && d != "2025-09-20" && d != "2025-09-21"
	}

	got, err := BuildCopyPlan(anteriores, "2025-09-15", util)
	if err != nil {
		t.Fatal(err)
	}

	if len(got.Plan) != 2 {
		t.Fatalf("plano = %+v, esperava segunda e terça", got.Plan)
	}

	seg := got.Plan[0]
	if seg.Date != "2025-09-15" || seg.TotalMin != 360 || len(seg.Entries) != 2 {
		t.Fatalf("segunda = %+v", seg)
	}
	if e := seg.Entries[0]; e.TaskID != 10 || e.Entry.Minutes != 240 || e.Entry.Time != "09:00:00" ||
		e.Entry.IsBillable || e.Entry.Description != "Dev" {
		t.Errorf("1ª entrada da segunda = %+v", e)
	}
	if e := seg.Entries[1]; e.TaskID != 20 || e.Entry.Time != "13:00:00" || e.Entry.Description != "Tarefa" {
		t.Errorf("2ª entrada deve vir em sequência e usar o nome da tarefa sem descrição: %+v", e)
	}

	ter := got.Plan[1]
	if ter.Date != "2025-09-16" || ter.Entries[0].Entry.Time != "15:30:00" {
		t.Errorf("terça deve manter o horário informado: %+v", ter)
	}

	if !reflect.DeepEqual(got.SkippedDays, []string{"2025-09-17", "2025-09-20"}) {
		t.Errorf("dias pulados = %v", got.SkippedDays)
	}
	if got.SkippedEntries != 3 {
		t.Errorf("entradas puladas = %d, esperava 3 (feriado, sem tarefa, sábado)", got.SkippedEntries)
	}
}

func TestBuildCopyPlanSemanaAnteriorVazia(t *testing.T) {
	got, err := BuildCopyPlan(nil, "2025-09-15", func(string) bool { return true })
	if err != nil {
		t.Fatal(err)
	}
	if got.Plan == nil || len(got.Plan) != 0 || got.SkippedDays == nil {
		t.Errorf("esperava plano vazio não nil: %+v", got)
	}
}
