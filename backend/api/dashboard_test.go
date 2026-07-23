package api

import (
	"testing"
	"time"
)

func TestParseTeamworkDate(t *testing.T) {
	cases := []struct {
		in   string
		want string // vazio = deve falhar
	}{
		{"2025-09-30", "2025-09-30"},
		{"2025-09-30T00:00:00Z", "2025-09-30"},
		{"2025-09-30T14:35:22Z", "2025-09-30"},
		{"2025-09-30T14:35:22+00:00", "2025-09-30"},
		{"20250930", "2025-09-30"},
		{"  2025-09-30  ", "2025-09-30"},
		{"", ""},
		{"30/09/2025", ""},
		{"sem prazo", ""},
	}

	for _, tc := range cases {
		got, ok := parseTeamworkDate(tc.in)
		if tc.want == "" {
			if ok {
				t.Errorf("parseTeamworkDate(%q) = %v, esperava falha", tc.in, got)
			}
			continue
		}
		if !ok {
			t.Errorf("parseTeamworkDate(%q) falhou, esperava %s", tc.in, tc.want)
			continue
		}
		if got.Format("2006-01-02") != tc.want {
			t.Errorf("parseTeamworkDate(%q) = %s, esperava %s", tc.in, got.Format("2006-01-02"), tc.want)
		}
	}
}

func TestPrazosUsamDataRealEOmitemTarefasSemPrazo(t *testing.T) {
	hoje := time.Date(2025, 9, 15, 0, 0, 0, 0, time.Local)

	tasks := []TeamworkTask{
		{ID: 1, Content: "Vence depois", DueDate: "2025-09-30", ProjectName: "P", Priority: "high"},
		{ID: 2, Content: "Sem prazo", DueDate: "", ProjectName: "P"},
		{ID: 3, Content: "Vence antes", DueDate: "2025-09-20", ProjectName: "P", Priority: "low"},
		{ID: 4, Content: "Ja venceu", DueDate: "2025-09-01", ProjectName: "P"},
		{ID: 5, Content: "Vence hoje", DueDate: "2025-09-15", ProjectName: "P"},
		{ID: 6, Content: "Prazo ilegivel", DueDate: "15/09/2025", ProjectName: "P"},
	}

	got := filtrarEOrdenarPrazos(tasks, hoje, upcomingDeadlinesLimit)

	// Devem sobrar apenas 5, 3 e 1 — nessa ordem.
	wantIDs := []int{5, 3, 1}
	if len(got) != len(wantIDs) {
		t.Fatalf("devolveu %d tarefas (%v), esperava %d", len(got), got, len(wantIDs))
	}
	for i, wantID := range wantIDs {
		if got[i]["id"] != wantID {
			t.Errorf("posição %d: id = %v, esperava %d", i, got[i]["id"], wantID)
		}
	}

	// A data precisa ser a da tarefa, não uma gerada a partir de hoje.
	if got[0]["dueDate"] != "2025-09-15" {
		t.Errorf("dueDate = %v, esperava 2025-09-15 (o prazo real da tarefa)", got[0]["dueDate"])
	}
	if got[2]["dueDate"] != "2025-09-30" {
		t.Errorf("dueDate = %v, esperava 2025-09-30", got[2]["dueDate"])
	}

	// A prioridade precisa vir da API, não ser fixada em "Normal".
	if got[1]["priority"] != "low" {
		t.Errorf("priority = %v, esperava \"low\" vindo da API", got[1]["priority"])
	}
}

// Regressão do bug: as datas eram geradas como hoje+1, hoje+2, hoje+3...
// independentemente do prazo real de cada tarefa.
func TestPrazosNaoSaoSequenciaAPartirDeHoje(t *testing.T) {
	hoje := time.Date(2025, 9, 15, 0, 0, 0, 0, time.Local)

	tasks := []TeamworkTask{
		{ID: 1, Content: "A", DueDate: "2025-12-01"},
		{ID: 2, Content: "B", DueDate: "2025-12-02"},
	}

	got := filtrarEOrdenarPrazos(tasks, hoje, upcomingDeadlinesLimit)
	if len(got) != 2 {
		t.Fatalf("devolveu %d tarefas, esperava 2", len(got))
	}

	for i, item := range got {
		gerada := hoje.AddDate(0, 0, i+1).Format("2006-01-02")
		if item["dueDate"] == gerada {
			t.Errorf("posição %d: dueDate = %v, que coincide com a data gerada pelo bug antigo", i, item["dueDate"])
		}
	}
}

func TestPrazosRespeitamLimite(t *testing.T) {
	hoje := time.Date(2025, 9, 15, 0, 0, 0, 0, time.Local)

	var tasks []TeamworkTask
	for i := 1; i <= 12; i++ {
		tasks = append(tasks, TeamworkTask{
			ID:      i,
			Content: "T",
			DueDate: hoje.AddDate(0, 0, i).Format("2006-01-02"),
		})
	}

	if got := filtrarEOrdenarPrazos(tasks, hoje, upcomingDeadlinesLimit); len(got) != upcomingDeadlinesLimit {
		t.Errorf("devolveu %d tarefas, esperava o limite de %d", len(got), upcomingDeadlinesLimit)
	}
}
