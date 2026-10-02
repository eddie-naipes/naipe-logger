package api

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

// 2025-09-01T00:00:00Z em milissegundos: o formato do loggedtime.json.
const primeiroDeSetembroUTC int64 = 1756684800000

func TestLoggedTimeDateIgnoraFusoLocal(t *testing.T) {
	saoPaulo := time.FixedZone("America/Sao_Paulo", -3*60*60)

	// Documenta o bug: no fuso de São Paulo a meia-noite UTC cai na véspera.
	if got := time.UnixMilli(primeiroDeSetembroUTC).In(saoPaulo).Format("2006-01-02"); got != "2025-08-31" {
		t.Fatalf("premissa do teste falhou: %s", got)
	}

	// Simula a máquina do usuário em São Paulo sem depender do TZ real.
	original := time.Local
	time.Local = saoPaulo
	t.Cleanup(func() { time.Local = original })

	if got := loggedTimeDate(primeiroDeSetembroUTC); got != "2025-09-01" {
		t.Errorf("loggedTimeDate em São Paulo = %s, esperava 2025-09-01", got)
	}

	// O resultado também não pode mudar a leste de Greenwich.
	time.Local = time.FixedZone("Asia/Tokyo", 9*60*60)
	if got := loggedTimeDate(primeiroDeSetembroUTC); got != "2025-09-01" {
		t.Errorf("loggedTimeDate em Tóquio = %s, esperava 2025-09-01", got)
	}
}

func TestGetEntriesFromLoggedTimeUsaDataUTCEOrdenaDecrescente(t *testing.T) {
	segundoDeSetembro := primeiroDeSetembroUTC + 24*60*60*1000

	api, _ := newTestAPI(t, func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"STATUS":"OK","user":{
			"billable":[["%d","1.0","60"],["%d","0.5","30"]],
			"nonbillable":[["%d","0.0","0"]]}}`,
			primeiroDeSetembroUTC, segundoDeSetembro, primeiroDeSetembroUTC)
	})

	entries, err := api.GetEntriesFromLoggedTime(9, 2025)
	if err != nil {
		t.Fatalf("GetEntriesFromLoggedTime: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("devolveu %d entradas, esperava 2 (não cobrável com 0 min é omitida)", len(entries))
	}
	if entries[0]["date"] != "2025-09-02" || entries[1]["date"] != "2025-09-01" {
		t.Errorf("datas = %v, %v; esperava 2025-09-02 e 2025-09-01", entries[0]["date"], entries[1]["date"])
	}
}
