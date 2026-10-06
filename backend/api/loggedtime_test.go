package api

import (
	"encoding/json"
	"testing"
)

// Formato real do loggedtime.json: timestamp como texto, horas e minutos como
// números. Antes do FlexString, qualquer mês com horas lançadas falhava aqui.
func TestLoggedTimeResponseAceitaNumerosNosDias(t *testing.T) {
	body := `{"STATUS":"OK","user":{"billable":[["1790812800000",0.25,15],["1790899200000","1.5","90"]],"nonbillable":[],"firstname":"A","lastname":"B","id":"244599","endepoch":"1793404800000","startepoch":"1790812800000"}}`

	var r LoggedTimeResponse
	if err := json.Unmarshal([]byte(body), &r); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if len(r.User.Billable) != 2 {
		t.Fatalf("billable tem %d dias, esperava 2", len(r.User.Billable))
	}
	if got := r.User.Billable[0]; got != [3]FlexString{"1790812800000", "0.25", "15"} {
		t.Errorf("primeiro dia = %v", got)
	}
	if got := r.User.Billable[1]; got != [3]FlexString{"1790899200000", "1.5", "90"} {
		t.Errorf("segundo dia = %v", got)
	}

	// O frontend continua recebendo texto, como antes.
	out, err := json.Marshal(r.User.Billable[0])
	if err != nil || string(out) != `["1790812800000","0.25","15"]` {
		t.Errorf("Marshal = %s, %v", out, err)
	}
}

func TestFlexStringRecusaObjeto(t *testing.T) {
	var f FlexString
	if err := json.Unmarshal([]byte(`{"a":1}`), &f); err == nil {
		t.Error("objeto deveria ser recusado")
	}
}
