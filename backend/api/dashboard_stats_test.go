package api

import (
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"
	"time"
)

// As horas do dashboard precisam sair de time/total.json (somado pelo
// Teamwork), uma chamada por mês, sem listar time.json com fromDate/toDate.
func TestGetDashboardStatsUsaTotaisDoTeamwork(t *testing.T) {
	now := time.Now()
	seedHolidayCache(t, now.Year())
	inicioMes := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).Format("2006-01-02")

	var totais, listagens int32
	api, _ := newTestAPI(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/projects/api/v3/time/total.json":
			atomic.AddInt32(&totais, 1)
			if r.URL.Query().Get("userId") != "42" {
				t.Errorf("userId = %q, esperava 42", r.URL.Query().Get("userId"))
			}
			minutos := 600 // mês anterior: 10h
			if r.URL.Query().Get("startDate") == inicioMes {
				minutos = 900 // mês atual: 15h
			}
			fmt.Fprintf(w, `{"time-totals":{"minutes":%d}}`, minutos)
		case "/projects/api/v3/time.json":
			atomic.AddInt32(&listagens, 1)
			fmt.Fprint(w, `{"timeEntries":[]}`)
		case "/projects/api/v3/tasks.json":
			if r.URL.Query().Get("responsiblePartyIds") != "42" {
				t.Errorf("responsiblePartyIds = %q, esperava 42 (assignedTo é ignorado pela v3)", r.URL.Query().Get("responsiblePartyIds"))
			}
			fmt.Fprint(w, `{"tasks":[],"meta":{"page":{"count":7,"hasMore":true}}}`)
		case "/projects/api/v3/projects.json":
			fmt.Fprint(w, `{"projects":[],"meta":{"page":{"count":3,"hasMore":false}}}`)
		default:
			t.Errorf("caminho inesperado %q", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	})

	stats, err := api.GetDashboardStats()
	if err != nil {
		t.Fatalf("GetDashboardStats devolveu erro: %v", err)
	}

	if stats["horasLogadas"] != 15.0 {
		t.Errorf("horasLogadas = %v, esperava 15", stats["horasLogadas"])
	}
	if stats["horasLogadasChange"] != 50 {
		t.Errorf("horasLogadasChange = %v, esperava 50", stats["horasLogadasChange"])
	}
	if stats["tarefasPendentes"] != 7 {
		t.Errorf("tarefasPendentes = %v, esperava 7", stats["tarefasPendentes"])
	}
	if stats["projetos"] != 3 {
		t.Errorf("projetos = %v, esperava 3 (de meta.page.count)", stats["projetos"])
	}
	if got := atomic.LoadInt32(&totais); got != 2 {
		t.Errorf("time/total.json chamado %d vezes, esperava 2 (mês atual e anterior)", got)
	}
	if got := atomic.LoadInt32(&listagens); got != 0 {
		t.Errorf("time.json chamado %d vezes; as horas devem vir dos totais", got)
	}
}
