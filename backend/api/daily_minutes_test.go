package api

import (
	"fmt"
	"net/http"
	"testing"
)

func TestDailyLoggedMinutesSomaPorDia(t *testing.T) {
	api, _ := newTestAPI(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("showDeleted"); got != "0" {
			t.Errorf("showDeleted = %q, esperava 0 (sem entradas apagadas)", got)
		}
		fmt.Fprint(w, `{"timeEntries":[
			{"id":1,"userId":42,"date":"2025-09-01T00:00:00Z","hoursDecimal":2},
			{"id":2,"userId":42,"date":"2025-09-01","hours":0,"minutes":30},
			{"id":3,"userId":42,"date":"2025-09-02","hoursDecimal":8},
			{"id":4,"userId":7,"date":"2025-09-02","hoursDecimal":3},
			{"id":5,"date":"2025-09-03","hoursDecimal":1}
		],"meta":{"page":{"hasMore":false}}}`)
	})

	got, err := api.DailyLoggedMinutes("2025-09-01", "2025-09-30")
	if err != nil {
		t.Fatalf("erro: %v", err)
	}

	esperado := map[string]int{"2025-09-01": 150, "2025-09-02": 480, "2025-09-03": 60}
	if len(got) != len(esperado) {
		t.Fatalf("got = %v, esperava %v", got, esperado)
	}
	for dia, minutos := range esperado {
		if got[dia] != minutos {
			t.Errorf("%s = %d min, esperava %d (lançamentos de outro usuário não contam)", dia, got[dia], minutos)
		}
	}
}

func TestDailyLoggedMinutesPeriodoVazio(t *testing.T) {
	api, _ := newTestAPI(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"timeEntries":[],"meta":{"page":{"hasMore":false}}}`)
	})

	got, err := api.DailyLoggedMinutes("2025-09-01", "2025-09-30")
	if err != nil {
		t.Fatalf("erro: %v", err)
	}
	if got == nil || len(got) != 0 {
		t.Errorf("got = %v, esperava mapa vazio (não nil)", got)
	}
}

func TestDailyLoggedMinutesPropagaErro(t *testing.T) {
	api, _ := newTestAPI(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("não deveria chamar a API com data inválida")
	})

	if _, err := api.DailyLoggedMinutes("ontem", "2025-09-30"); err == nil {
		t.Error("esperava erro para data inválida")
	}
}
