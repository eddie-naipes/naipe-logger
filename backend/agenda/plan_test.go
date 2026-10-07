package agenda

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func occ(t *testing.T, key, titulo, data, ini, fim string) Occurrence {
	t.Helper()
	loc := mustLoc(t, "America/Sao_Paulo")
	s, err := time.ParseInLocation("2006-01-02 15:04", data+" "+ini, loc)
	if err != nil {
		t.Fatal(err)
	}
	e, err := time.ParseInLocation("2006-01-02 15:04", data+" "+fim, loc)
	if err != nil {
		t.Fatal(err)
	}
	if e.Before(s) {
		e = e.AddDate(0, 0, 1)
	}
	return Occurrence{Key: key, UID: key, Summary: titulo, Start: s, End: e, Source: "Trabalho"}
}

func resumo(items []PlanItem) []string {
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = fmt.Sprintf("%s %s %s-%s %d %s task=%d", it.Key, it.Date, it.StartTime, it.EndTime, it.Minutes, it.Status, it.Task.TaskID)
	}
	return out
}

var tarefaDaily = TaskRef{TaskID: 10, TaskName: "Cerimônias", ProjectID: 1, ProjectName: "Projeto X"}

func TestBuildPlanMapeiaPorRegraEmOrdemEUsaTituloComoDescricao(t *testing.T) {
	items, err := BuildPlan([]Occurrence{
		occ(t, "a", "Daily do time", "2026-10-05", "09:00", "09:15"),
		occ(t, "b", "Planning Sprint 12", "2026-10-05", "10:00", "11:00"),
		occ(t, "c", "Almoço com cliente", "2026-10-05", "12:00", "13:00"),
	}, PlanOptions{
		Rules: []Rule{
			{Match: "daily", Task: tarefaDaily},
			{Match: `^planning\b`, Regex: true, Task: TaskRef{TaskID: 20}, Description: "Planejamento"},
			{Match: "sprint", Task: TaskRef{TaskID: 30}},
		},
		Billable: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	iguais(t, resumo(items), []string{
		"a 2026-10-05 09:00-09:15 15 mapped task=10",
		"b 2026-10-05 10:00-11:00 60 mapped task=20",
		"c 2026-10-05 12:00-13:00 60 unmapped task=0",
	})
	if items[0].Description != "Daily do time" || items[1].Description != "Planejamento" || !items[0].Billable {
		t.Errorf("descrições/billable = %+v", items)
	}
	if items[1].RuleIndex != 1 {
		t.Errorf("RuleIndex = %d", items[1].RuleIndex)
	}
}

func TestBuildPlanTarefaPadraoParaEventosSemRegra(t *testing.T) {
	items, _ := BuildPlan([]Occurrence{occ(t, "a", "Alinhamento", "2026-10-05", "09:00", "10:00")},
		PlanOptions{DefaultTask: TaskRef{TaskID: 99}})
	if items[0].Status != StatusMapped || items[0].Task.TaskID != 99 || items[0].Reason != "Tarefa padrão" {
		t.Errorf("item = %+v", items[0])
	}
}

func TestBuildPlanIgnoraComMotivo(t *testing.T) {
	diaInteiro := occ(t, "dia", "Feriado interno", "2026-10-05", "00:00", "00:00")
	diaInteiro.AllDay = true
	diaInteiro.End = diaInteiro.Start.AddDate(0, 0, 1)
	cancelado := occ(t, "cancel", "Review", "2026-10-05", "08:00", "08:30")
	cancelado.Cancelled = true
	livre := occ(t, "livre", "Foco", "2026-10-05", "13:00", "15:00")
	livre.Transparent = true
	recusado := occ(t, "recusa", "Comitê", "2026-10-05", "16:00", "17:00")
	recusado.Attendees = []Attendee{{Email: "outra@exemplo.com", PartStat: "ACCEPTED"}, {Email: "eu@exemplo.com", PartStat: "DECLINED"}}

	items, err := BuildPlan([]Occurrence{
		diaInteiro, cancelado, livre, recusado,
		occ(t, "almoco", "Almoço", "2026-10-05", "12:00", "13:00"),
		occ(t, "curto", "Café rápido", "2026-10-05", "17:00", "17:05"),
	}, PlanOptions{UserEmail: "EU@exemplo.com", IgnoreWords: []string{"almoço"}, MinMinutes: 10})
	if err != nil {
		t.Fatal(err)
	}
	motivos := map[string]string{}
	for _, it := range items {
		if it.Status != StatusIgnored {
			t.Errorf("%s deveria ser ignorado: %+v", it.Key, it)
		}
		motivos[it.Key] = it.Reason
	}
	esperado := map[string]string{
		"dia":    "Evento de dia inteiro",
		"cancel": "Evento cancelado",
		"livre":  "Marcado como livre",
		"recusa": "Convite recusado",
		"almoco": "palavra ignorada",
		"curto":  "Duração menor que 10 min",
	}
	for k, trecho := range esperado {
		if !strings.Contains(motivos[k], trecho) {
			t.Errorf("motivo de %s = %q; esperava conter %q", k, motivos[k], trecho)
		}
	}
}

func TestBuildPlanIncluiTransparenteQuandoConfigurado(t *testing.T) {
	livre := occ(t, "livre", "Foco", "2026-10-05", "13:00", "15:00")
	livre.Transparent = true
	items, _ := BuildPlan([]Occurrence{livre}, PlanOptions{IncludeTransparent: true})
	if items[0].Status != StatusUnmapped {
		t.Errorf("status = %s", items[0].Status)
	}
}

func TestBuildPlanCortaSobreposicaoELimitaAoDia(t *testing.T) {
	items, err := BuildPlan([]Occurrence{
		occ(t, "a", "A", "2026-10-05", "09:00", "10:00"),
		occ(t, "b", "B", "2026-10-05", "09:30", "10:30"), // vira 10:00-10:30
		occ(t, "c", "C", "2026-10-05", "09:45", "10:15"), // inteiro dentro de A+B
		occ(t, "d", "D", "2026-10-05", "23:00", "01:00"), // passa da meia-noite
	}, PlanOptions{DefaultTask: TaskRef{TaskID: 1}})
	if err != nil {
		t.Fatal(err)
	}
	porChave := map[string]PlanItem{}
	for _, it := range items {
		porChave[it.Key] = it
	}
	if b := porChave["b"]; b.StartTime != "10:00" || b.Minutes != 30 || b.Status != StatusMapped {
		t.Errorf("b = %+v", b)
	}
	if c := porChave["c"]; c.Status != StatusIgnored || c.Reason != "Sobreposto a outro evento" {
		t.Errorf("c = %+v", c)
	}
	if d := porChave["d"]; d.Minutes != 60 || d.EndTime != "24:00" {
		t.Errorf("d = %+v", d)
	}
}

func TestBuildPlanArredondaPara15Minutos(t *testing.T) {
	items, _ := BuildPlan([]Occurrence{
		occ(t, "a", "A", "2026-10-05", "09:00", "09:50"), // 50 -> 45
		occ(t, "b", "B", "2026-10-05", "10:00", "10:53"), // 53 -> 60
		occ(t, "c", "C", "2026-10-05", "11:00", "11:05"), // 5 -> 15 (não zera)
	}, PlanOptions{RoundTo: 15})
	got := []int{items[0].Minutes, items[1].Minutes, items[2].Minutes}
	if got[0] != 45 || got[1] != 60 || got[2] != 15 {
		t.Errorf("minutos = %v", got)
	}
}

func TestBuildPlanMarcaJaLancados(t *testing.T) {
	items, _ := BuildPlan([]Occurrence{
		occ(t, "a", "Daily", "2026-10-05", "09:00", "09:15"),
		occ(t, "b", "Daily", "2026-10-06", "09:00", "09:15"),
	}, PlanOptions{Rules: []Rule{{Match: "daily", Task: tarefaDaily}}, Imported: func(k string) bool { return k == "a" }})
	if items[0].Status != StatusImported || items[0].Task.TaskID != 10 || items[1].Status != StatusMapped {
		t.Errorf("items = %+v", items)
	}
}

func TestCompileRulesRecusaRegexInvalidaERegraVazia(t *testing.T) {
	if _, err := CompileRules([]Rule{{Match: "(", Regex: true}}); err == nil {
		t.Error("regex inválida aceita")
	}
	if _, err := CompileRules([]Rule{{Match: "  "}}); err == nil {
		t.Error("regra vazia aceita")
	}
	// Palavra-chave com caracteres especiais é literal.
	res, err := CompileRules([]Rule{{Match: "1:1 (gestor)"}})
	if err != nil || !res[0].MatchString("Meu 1:1 (GESTOR) semanal") {
		t.Errorf("palavra-chave literal: %v", err)
	}
}
