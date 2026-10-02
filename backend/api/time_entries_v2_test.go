package api

import (
	"fmt"
	"net/http"
	"sync/atomic"
	"testing"
)

// Antes só page=1&pageSize=500 era lido; o restante do período sumia.
func TestGetTimeEntriesForPeriodV2Pagina(t *testing.T) {
	var chamadas int32
	api, _ := newTestAPI(t, func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&chamadas, 1)
		q := r.URL.Query()
		if q.Get("pageSize") != "500" {
			t.Errorf("pageSize = %q, esperava 500", q.Get("pageSize"))
		}
		if len(q["page"]) != 1 {
			t.Errorf("parâmetro page repetido na URL: %v", q["page"])
		}

		switch q.Get("page") {
		case "1":
			// Sem meta: a decisão vem do cabeçalho X-Pages.
			w.Header().Set("X-Pages", "2")
			fmt.Fprint(w, `{"timeEntries":[{"id":10,"date":"2025-09-01T00:00:00Z","hoursDecimal":1.75}]}`)
		case "2":
			w.Header().Set("X-Pages", "2")
			fmt.Fprint(w, `{"timeEntries":[{"id":11,"date":"2025-09-02","hours":0,"minutes":30}]}`)
		default:
			t.Errorf("página inesperada %q", q.Get("page"))
		}
	})

	entries, err := api.GetTimeEntriesForPeriodV2("2025-09-01", "2025-09-30", false)
	if err != nil {
		t.Fatalf("erro: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("devolveu %d entradas, esperava 2", len(entries))
	}
	if entries[0].ID != 10 || entries[0].Date != "2025-09-01" || entries[0].Minutes != 105 {
		t.Errorf("primeira entrada = %+v", entries[0])
	}
	if entries[1].ID != 11 || entries[1].Date != "2025-09-02" || entries[1].Minutes != 30 {
		t.Errorf("segunda entrada = %+v", entries[1])
	}
	if got := atomic.LoadInt32(&chamadas); got != 2 {
		t.Errorf("servidor recebeu %d chamadas, esperava 2", got)
	}
}

func TestGetTimeEntriesForPeriodV2NaoInventaData(t *testing.T) {
	api, _ := newTestAPI(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"timeEntries":[{"id":1,"date":"data estranha","minutes":10}],"meta":{"page":{"hasMore":false}}}`)
	})

	entries, err := api.GetTimeEntriesForPeriodV2("2025-09-01", "2025-09-30", false)
	if err != nil {
		t.Fatalf("erro: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("devolveu %d entradas, esperava 1", len(entries))
	}
	if entries[0].Date == "0001-01-01" {
		t.Error("data irreconhecível virou 0001-01-01")
	}
}
