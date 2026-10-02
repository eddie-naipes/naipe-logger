package api

import (
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
)

// Regressão: a paginação fazia Unmarshal sempre na mesma variável, e o
// resultado de duas páginas virava [p2, p2] em vez de [p1, p2].
func TestGetTimeEntriesForPeriodPaginaSemSobrescrever(t *testing.T) {
	var chamadas int32
	api, _ := newTestAPI(t, func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&chamadas, 1)
		if r.URL.Path != "/projects/api/v3/time.json" {
			t.Errorf("caminho inesperado %q", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("startDate") != "2025-09-01" || q.Get("endDate") != "2025-09-30" {
			t.Errorf("datas devem ir em startDate/endDate, recebido %v", q)
		}
		if q.Get("assignedToUserIds") != "42" {
			t.Errorf("assignedToUserIds = %q, esperava 42", q.Get("assignedToUserIds"))
		}

		switch q.Get("page") {
		case "1":
			fmt.Fprint(w, `{"timeEntries":[{"id":1,"date":"2025-09-01","minutes":60},{"id":2,"date":"2025-09-02","minutes":30}],
				"meta":{"page":{"hasMore":true}}}`)
		case "2":
			// Página 2 no formato atual da v3 ("timelogs", data com hora).
			fmt.Fprint(w, `{"timelogs":[{"id":3,"date":"2025-09-03T00:00:00Z","minutes":45,"taskId":9}],
				"meta":{"page":{"hasMore":false}}}`)
		default:
			t.Errorf("página inesperada %q", q.Get("page"))
		}
	})

	entries, err := api.GetTimeEntriesForPeriod("2025-09-01", "2025-09-30")
	if err != nil {
		t.Fatalf("GetTimeEntriesForPeriod devolveu erro: %v", err)
	}

	if len(entries) != 3 {
		t.Fatalf("devolveu %d entradas, esperava 3: %+v", len(entries), entries)
	}
	for i, wantID := range []int{1, 2, 3} {
		if entries[i].ID != wantID {
			t.Errorf("posição %d: id = %d, esperava %d", i, entries[i].ID, wantID)
		}
	}
	if entries[2].Date != "2025-09-03" || entries[2].Minutes != 45 || entries[2].TaskID != 9 {
		t.Errorf("timelog da v3 mal convertido: %+v", entries[2])
	}
	if got := atomic.LoadInt32(&chamadas); got != 2 {
		t.Errorf("servidor recebeu %d chamadas, esperava 2", got)
	}
}

func TestGetTimeEntriesForPeriodPropagaErroDaSegundaPagina(t *testing.T) {
	api, _ := newTestAPI(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("page") == "1" {
			fmt.Fprint(w, `{"timeEntries":[{"id":1,"date":"2025-09-01","minutes":60}],"meta":{"page":{"hasMore":true}}}`)
			return
		}
		// 404 não é repetido pelo doRequest: falha direto.
		w.WriteHeader(http.StatusNotFound)
	})

	entries, err := api.GetTimeEntriesForPeriod("2025-09-01", "2025-09-30")
	if err == nil {
		t.Fatalf("esperava erro na página 2, recebeu %d entradas sem erro", len(entries))
	}
	if !strings.Contains(err.Error(), "página 2") {
		t.Errorf("erro deveria indicar a página que falhou: %v", err)
	}
}

func TestGetTimeEntriesForPeriodRespeitaTetoDePaginas(t *testing.T) {
	var chamadas int32
	api, _ := newTestAPI(t, func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&chamadas, 1)
		// API defeituosa: hasMore=true para sempre.
		fmt.Fprintf(w, `{"timeEntries":[{"id":%d,"date":"2025-09-01","minutes":1}],"meta":{"page":{"hasMore":true}}}`, n)
	})

	entries, err := api.GetTimeEntriesForPeriod("2025-09-01", "2025-09-30")
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if got := atomic.LoadInt32(&chamadas); got != maxTimeEntryPages {
		t.Errorf("servidor recebeu %d chamadas, esperava o teto de %d", got, maxTimeEntryPages)
	}
	if len(entries) != maxTimeEntryPages {
		t.Errorf("devolveu %d entradas, esperava %d", len(entries), maxTimeEntryPages)
	}
}
