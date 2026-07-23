package api

import "testing"

func plano(dias ...WorkDay) []WorkDay { return dias }

func dia(data string, entradas ...EntryTask) WorkDay {
	total := 0
	for _, e := range entradas {
		total += e.Entry.Minutes
	}
	return WorkDay{Date: data, Entries: entradas, TotalMin: total}
}

func entrada(taskID, minutos int) EntryTask {
	return EntryTask{TaskID: taskID, Entry: TimeEntry{Minutes: minutos}}
}

func lancamento(data string, taskID, minutos int, nome string) TimeEntryReport {
	return TimeEntryReport{Date: data, TaskID: taskID, Minutes: minutos, TaskName: nome}
}

func TestDetectConflictsSemLancamentoExistente(t *testing.T) {
	p := plano(dia("2025-09-15", entrada(1, 480)))

	if got := detectConflicts(p, nil); len(got) != 0 {
		t.Errorf("detectConflicts = %v, esperava nenhum conflito", got)
	}
}

func TestDetectConflictsDiaJaTemLancamento(t *testing.T) {
	p := plano(dia("2025-09-15", entrada(1, 480)))
	existentes := []TimeEntryReport{
		lancamento("2025-09-15", 1, 120, "Tarefa A"),
		lancamento("2025-09-15", 2, 60, "Tarefa B"),
	}

	got := detectConflicts(p, existentes)
	if len(got) != 1 {
		t.Fatalf("detectConflicts devolveu %d conflitos, esperava 1", len(got))
	}

	c := got[0]
	if c.Date != "2025-09-15" {
		t.Errorf("Date = %q, esperava 2025-09-15", c.Date)
	}
	if c.ExistingMinutes != 180 {
		t.Errorf("ExistingMinutes = %d, esperava 180", c.ExistingMinutes)
	}
	if c.ExistingEntries != 2 {
		t.Errorf("ExistingEntries = %d, esperava 2", c.ExistingEntries)
	}
	if c.PlannedMinutes != 480 {
		t.Errorf("PlannedMinutes = %d, esperava 480", c.PlannedMinutes)
	}

	// A tarefa 1 colide diretamente; a 2 só ocupa o mesmo dia.
	if len(c.SameTask) != 1 {
		t.Fatalf("SameTask = %v, esperava 1 colisão direta", c.SameTask)
	}
	if c.SameTask[0].TaskID != 1 {
		t.Errorf("SameTask[0].TaskID = %d, esperava 1", c.SameTask[0].TaskID)
	}
	if c.SameTask[0].ExistingMinutes != 120 || c.SameTask[0].PlannedMinutes != 480 {
		t.Errorf("SameTask[0] = %+v, esperava existente=120 planejado=480", c.SameTask[0])
	}
	if c.SameTask[0].TaskName != "Tarefa A" {
		t.Errorf("SameTask[0].TaskName = %q, esperava \"Tarefa A\"", c.SameTask[0].TaskName)
	}
}

func TestDetectConflictsIgnoraDiasLivres(t *testing.T) {
	p := plano(
		dia("2025-09-15", entrada(1, 480)),
		dia("2025-09-16", entrada(1, 480)),
		dia("2025-09-17", entrada(1, 480)),
	)
	existentes := []TimeEntryReport{
		lancamento("2025-09-16", 1, 60, "Tarefa A"),
	}

	got := detectConflicts(p, existentes)
	if len(got) != 1 {
		t.Fatalf("detectConflicts devolveu %d conflitos, esperava 1", len(got))
	}
	if got[0].Date != "2025-09-16" {
		t.Errorf("conflito em %q, esperava 2025-09-16", got[0].Date)
	}
}

func TestDetectConflictsIgnoraDiasSemEntradaNoPlano(t *testing.T) {
	// Dia presente no plano mas sem entradas nao deve gerar aviso, mesmo que
	// exista lancamento naquela data.
	p := plano(WorkDay{Date: "2025-09-15", Entries: nil})
	existentes := []TimeEntryReport{lancamento("2025-09-15", 1, 60, "Tarefa A")}

	if got := detectConflicts(p, existentes); len(got) != 0 {
		t.Errorf("detectConflicts = %v, esperava nenhum conflito", got)
	}
}

func TestDetectConflictsAgregaMultiplasEntradasDaMesmaTarefa(t *testing.T) {
	p := plano(dia("2025-09-15", entrada(1, 120), entrada(1, 240)))
	existentes := []TimeEntryReport{
		lancamento("2025-09-15", 1, 30, "Tarefa A"),
		lancamento("2025-09-15", 1, 90, "Tarefa A"),
	}

	got := detectConflicts(p, existentes)
	if len(got) != 1 {
		t.Fatalf("devolveu %d conflitos, esperava 1", len(got))
	}
	if len(got[0].SameTask) != 1 {
		t.Fatalf("SameTask = %v, esperava 1 entrada agregada", got[0].SameTask)
	}
	if got[0].SameTask[0].ExistingMinutes != 120 {
		t.Errorf("ExistingMinutes = %d, esperava 120 (30+90)", got[0].SameTask[0].ExistingMinutes)
	}
	if got[0].SameTask[0].PlannedMinutes != 360 {
		t.Errorf("PlannedMinutes = %d, esperava 360 (120+240)", got[0].SameTask[0].PlannedMinutes)
	}
}

func TestDetectConflictsOrdenadosPorData(t *testing.T) {
	p := plano(
		dia("2025-09-17", entrada(1, 60)),
		dia("2025-09-15", entrada(1, 60)),
		dia("2025-09-16", entrada(1, 60)),
	)
	existentes := []TimeEntryReport{
		lancamento("2025-09-17", 1, 60, "A"),
		lancamento("2025-09-15", 1, 60, "A"),
		lancamento("2025-09-16", 1, 60, "A"),
	}

	got := detectConflicts(p, existentes)
	want := []string{"2025-09-15", "2025-09-16", "2025-09-17"}
	if len(got) != len(want) {
		t.Fatalf("devolveu %d conflitos, esperava %d", len(got), len(want))
	}
	for i, data := range want {
		if got[i].Date != data {
			t.Errorf("conflito[%d].Date = %q, esperava %q", i, got[i].Date, data)
		}
	}
}

func TestPlanDateRange(t *testing.T) {
	p := plano(
		dia("2025-09-17", entrada(1, 60)),
		dia("2025-09-15", entrada(1, 60)),
		dia("2025-09-16", entrada(1, 60)),
	)

	inicio, fim, ok := planDateRange(p)
	if !ok {
		t.Fatal("planDateRange devolveu ok=false para plano válido")
	}
	if inicio != "2025-09-15" || fim != "2025-09-17" {
		t.Errorf("planDateRange = (%q, %q), esperava (2025-09-15, 2025-09-17)", inicio, fim)
	}

	if _, _, ok := planDateRange(nil); ok {
		t.Error("planDateRange(nil) devolveu ok=true")
	}
	if _, _, ok := planDateRange(plano(WorkDay{Date: "2025-09-15"})); ok {
		t.Error("planDateRange devolveu ok=true para plano sem entradas")
	}
}
