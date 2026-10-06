package security

import (
	"errors"
	"testing"

	"github.com/zalando/go-keyring"
)

func TestAgendaURLIdaEVoltaSemMisturarComOToken(t *testing.T) {
	keyring.MockInit()

	if err := StoreToken("tkn"); err != nil {
		t.Fatal(err)
	}
	if err := StoreAgendaURL("a1", "https://agenda.exemplo.com/privado.ics"); err != nil {
		t.Fatal(err)
	}
	got, err := LoadAgendaURL("a1")
	if err != nil || got != "https://agenda.exemplo.com/privado.ics" {
		t.Fatalf("LoadAgendaURL = %q, %v", got, err)
	}
	if tok, _ := LoadToken(); tok != "tkn" {
		t.Errorf("token alterado: %q", tok)
	}

	if err := DeleteAgendaURL("a1"); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadAgendaURL("a1"); !errors.Is(err, ErrNoAgendaURL) {
		t.Errorf("depois de remover: %v", err)
	}
	if err := DeleteAgendaURL("a1"); err != nil {
		t.Errorf("remover de novo deveria ser no-op: %v", err)
	}
}

func TestAgendaURLRecusaIDInvalidoELinkVazio(t *testing.T) {
	keyring.MockInit()
	if err := StoreAgendaURL("", "https://x/a.ics"); err == nil {
		t.Error("id vazio aceito")
	}
	if err := StoreAgendaURL("a/b", "https://x/a.ics"); err == nil {
		t.Error("id com barra aceito")
	}
	if err := StoreAgendaURL("a", "  "); err == nil {
		t.Error("link vazio aceito")
	}
}
