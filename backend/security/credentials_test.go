package security

import (
	"errors"
	"testing"

	"github.com/zalando/go-keyring"
)

// Todos os testes usam o provedor em memória do go-keyring (MockInit), de modo
// que nunca tocam no cofre real do sistema — necessário para rodar em CI
// headless, onde não há Keychain/Secret Service.

func TestStoreELoadTokenIdaEVolta(t *testing.T) {
	keyring.MockInit()

	const token = "tkn-abc123"
	if err := StoreToken(token); err != nil {
		t.Fatalf("StoreToken devolveu erro: %v", err)
	}

	got, err := LoadToken()
	if err != nil {
		t.Fatalf("LoadToken devolveu erro: %v", err)
	}
	if got != token {
		t.Errorf("LoadToken = %q, esperava %q", got, token)
	}
}

func TestStoreTokenRemoveEspacos(t *testing.T) {
	keyring.MockInit()

	if err := StoreToken("  tkn-com-espacos  "); err != nil {
		t.Fatalf("StoreToken devolveu erro: %v", err)
	}

	got, err := LoadToken()
	if err != nil {
		t.Fatalf("LoadToken devolveu erro: %v", err)
	}
	if got != "tkn-com-espacos" {
		t.Errorf("LoadToken = %q, esperava o token sem espaços nas pontas", got)
	}
}

func TestStoreTokenRecusaVazio(t *testing.T) {
	keyring.MockInit()

	for _, entrada := range []string{"", "   ", "\t\n"} {
		if err := StoreToken(entrada); err == nil {
			t.Errorf("StoreToken(%q) = nil, esperava erro para token vazio", entrada)
		}
	}

	// Um token vazio recusado não pode deixar resíduo no cofre.
	if _, err := LoadToken(); !errors.Is(err, ErrNoToken) {
		t.Errorf("LoadToken após recusa = %v, esperava ErrNoToken", err)
	}
}

func TestLoadTokenSemNadaGravado(t *testing.T) {
	keyring.MockInit()

	got, err := LoadToken()
	if !errors.Is(err, ErrNoToken) {
		t.Errorf("LoadToken sem token = %v, esperava ErrNoToken", err)
	}
	if got != "" {
		t.Errorf("LoadToken sem token devolveu %q, esperava string vazia", got)
	}
}

func TestDeleteTokenRemove(t *testing.T) {
	keyring.MockInit()

	if err := StoreToken("tkn-a-remover"); err != nil {
		t.Fatalf("StoreToken devolveu erro: %v", err)
	}
	if err := DeleteToken(); err != nil {
		t.Fatalf("DeleteToken devolveu erro: %v", err)
	}

	if _, err := LoadToken(); !errors.Is(err, ErrNoToken) {
		t.Errorf("LoadToken após DeleteToken = %v, esperava ErrNoToken", err)
	}
}

func TestDeleteTokenInexistenteNaoEErro(t *testing.T) {
	keyring.MockInit()

	// Remover quando não há nada gravado é operação idempotente, não erro:
	// o Logout pode ser acionado sem uma conexão ativa.
	if err := DeleteToken(); err != nil {
		t.Errorf("DeleteToken sem token = %v, esperava nil", err)
	}
}

func TestStoreTokenSobrescreveAnterior(t *testing.T) {
	keyring.MockInit()

	if err := StoreToken("token-antigo"); err != nil {
		t.Fatalf("StoreToken (antigo) devolveu erro: %v", err)
	}
	if err := StoreToken("token-novo"); err != nil {
		t.Fatalf("StoreToken (novo) devolveu erro: %v", err)
	}

	got, err := LoadToken()
	if err != nil {
		t.Fatalf("LoadToken devolveu erro: %v", err)
	}
	if got != "token-novo" {
		t.Errorf("LoadToken = %q, esperava o token mais recente", got)
	}
}
