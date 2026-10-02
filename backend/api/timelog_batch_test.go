package api

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// Os resultados precisam sair na ordem do plano, não na ordem em que as
// requisições terminaram, e um POST com 500 não pode ser repetido (poderia
// duplicar horas).
func TestLogMultipleTimesMantemOrdemENaoRepetePOSTEm500(t *testing.T) {
	var mu sync.Mutex
	chamadasPorTarefa := make(map[int]int)

	api, _ := newTestAPI(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("método = %s, esperava POST", r.Method)
		}
		// /projects/api/v3/tasks/{id}/time.json
		partes := strings.Split(r.URL.Path, "/")
		taskID, _ := strconv.Atoi(partes[len(partes)-2])

		mu.Lock()
		chamadasPorTarefa[taskID]++
		mu.Unlock()

		// Tarefas de ID menor demoram mais: sem gravar por índice, a ordem
		// dos resultados sairia invertida.
		time.Sleep(time.Duration(40-taskID*10) * time.Millisecond)

		if taskID == 3 {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprint(w, `{"errors":["falha interna"]}`)
			return
		}
		w.WriteHeader(http.StatusCreated)
		fmt.Fprintf(w, `{"timelog":{"id":%d}}`, taskID*100)
	})

	entrada := TimeEntry{Minutes: 60, Time: "09:00", Description: "x"}
	plano := []WorkDay{
		{Date: "2025-09-01", Entries: []EntryTask{{TaskID: 1, Entry: entrada}, {TaskID: 2, Entry: entrada}}},
		{Date: "2025-09-02", Entries: []EntryTask{}},
		{Date: "2025-09-03", Entries: []EntryTask{{TaskID: 3, Entry: entrada}, {TaskID: 0, Entry: entrada}}},
	}

	results, err := api.LogMultipleTimes(plano)
	if err != nil {
		t.Fatalf("LogMultipleTimes devolveu erro: %v", err)
	}

	want := []struct {
		taskID  int
		date    string
		success bool
		entryID int
	}{
		{1, "2025-09-01", true, 100},
		{2, "2025-09-01", true, 200},
		{3, "2025-09-03", false, 0},
		{0, "2025-09-03", false, 0},
	}

	if len(results) != len(want) {
		t.Fatalf("devolveu %d resultados, esperava %d", len(results), len(want))
	}
	for i, w := range want {
		r := results[i]
		if r == nil {
			t.Fatalf("resultado %d nulo", i)
		}
		if r.TaskID != w.taskID || r.Date != w.date || r.Success != w.success || r.EntryID != w.entryID {
			t.Errorf("posição %d = %+v, esperava tarefa %d em %s (success=%v, entryId=%d)",
				i, *r, w.taskID, w.date, w.success, w.entryID)
		}
	}

	mu.Lock()
	defer mu.Unlock()
	if chamadasPorTarefa[3] != 1 {
		t.Errorf("POST da tarefa 3 enviado %d vezes; 500 em POST não pode ser repetido", chamadasPorTarefa[3])
	}
	if chamadasPorTarefa[0] != 0 {
		t.Error("tarefa com ID inválido não deveria chegar à API")
	}
}

func TestLancamentosInvalidamCacheDoDashboard(t *testing.T) {
	api, _ := newTestAPI(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			w.WriteHeader(http.StatusCreated)
			fmt.Fprint(w, `{"timelog":{"id":1}}`)
		case http.MethodPut:
			fmt.Fprint(w, `{"timelog":{"id":5,"taskId":77}}`)
		case http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		}
	})

	semear := func() {
		api.cache.Set("dashboard_stats_42", map[string]interface{}{"horasLogadas": 1.0}, time.Hour)
		api.cache.Set("recent_activities", []map[string]interface{}{}, time.Hour)
		api.cache.Set("projects", []Project{{ID: 1}}, time.Hour)
	}
	conferir := func(operacao string) {
		t.Helper()
		if _, ok := api.cache.Get("dashboard_stats_42"); ok {
			t.Errorf("%s: dashboard_stats continuou em cache", operacao)
		}
		if _, ok := api.cache.Get("recent_activities"); ok {
			t.Errorf("%s: recent_activities continuou em cache", operacao)
		}
		if _, ok := api.cache.Get("projects"); !ok {
			t.Errorf("%s: cache de projetos não depende de lançamentos e deveria ficar", operacao)
		}
	}

	entrada := TimeEntry{Minutes: 30, Date: "2025-09-01", Time: "09:00"}

	semear()
	if _, err := api.LogTime(10, entrada); err != nil {
		t.Fatalf("LogTime: %v", err)
	}
	conferir("LogTime")

	semear()
	if _, err := api.UpdateTimeEntry(5, entrada); err != nil {
		t.Fatalf("UpdateTimeEntry: %v", err)
	}
	conferir("UpdateTimeEntry")

	semear()
	if err := api.DeleteTimeEntry(5); err != nil {
		t.Fatalf("DeleteTimeEntry: %v", err)
	}
	conferir("DeleteTimeEntry")

	semear()
	if _, err := api.DeleteMultipleTimeEntries([]int{5, 6}); err != nil {
		t.Fatalf("DeleteMultipleTimeEntries: %v", err)
	}
	conferir("DeleteMultipleTimeEntries")

	semear()
	if _, err := api.LogMultipleTimes([]WorkDay{{Date: "2025-09-01", Entries: []EntryTask{{TaskID: 10, Entry: entrada}}}}); err != nil {
		t.Fatalf("LogMultipleTimes: %v", err)
	}
	conferir("LogMultipleTimes")
}

func TestUpdateTimeEntryDevolveIDDoLancamentoEmEntryID(t *testing.T) {
	var corpo string
	api, _ := newTestAPI(t, func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		corpo = string(b)
		if r.URL.Path != "/projects/api/v3/time/5.json" {
			t.Errorf("caminho = %q, esperava /projects/api/v3/time/5.json", r.URL.Path)
		}
		fmt.Fprint(w, `{"timelog":{"id":5,"taskId":77}}`)
	})

	result, err := api.UpdateTimeEntry(5, TimeEntry{Minutes: 30, Date: "2025-09-01", Time: "09:00"})
	if err != nil {
		t.Fatalf("UpdateTimeEntry: %v", err)
	}
	if result.EntryID != 5 {
		t.Errorf("EntryID = %d, esperava 5", result.EntryID)
	}
	if result.TaskID != 77 {
		t.Errorf("TaskID = %d, esperava 77 (a tarefa do lançamento, não o ID do lançamento)", result.TaskID)
	}
	if strings.Contains(corpo, "taskId") {
		t.Errorf("payload do PUT não deveria levar taskId: %s", corpo)
	}
}

func TestCacheRemoveEntradaExpiradaNaLeitura(t *testing.T) {
	c := NewCache()
	c.Set("velha", 1, -time.Second)
	c.Set("nova", 2, time.Hour)

	if _, ok := c.Get("velha"); ok {
		t.Fatal("entrada expirada não deveria ser devolvida")
	}

	c.mutex.RLock()
	_, aindaExiste := c.data["velha"]
	c.mutex.RUnlock()
	if aindaExiste {
		t.Error("entrada expirada deveria ser removida do mapa na leitura")
	}
	if v, ok := c.Get("nova"); !ok || v != 2 {
		t.Errorf("entrada válida = %v, %v", v, ok)
	}
}

func TestCacheDeletePrefix(t *testing.T) {
	c := NewCache()
	c.Set("dashboard_stats_1", 1, time.Hour)
	c.Set("dashboard_stats_2", 2, time.Hour)
	c.Set("projects", 3, time.Hour)

	c.DeletePrefix("dashboard_stats_")

	if _, ok := c.Get("dashboard_stats_1"); ok {
		t.Error("dashboard_stats_1 deveria ter sido removida")
	}
	if _, ok := c.Get("dashboard_stats_2"); ok {
		t.Error("dashboard_stats_2 deveria ter sido removida")
	}
	if _, ok := c.Get("projects"); !ok {
		t.Error("projects não tem o prefixo e deveria ficar")
	}
}
