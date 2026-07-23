// Package security guarda o segredo de API no cofre de credenciais do sistema
// operacional (Credential Manager no Windows, Keychain no macOS, Secret Service
// no Linux). O segredo nunca é gravado em disco pela aplicação.
package security

import (
	"errors"
	"fmt"
	"strings"

	"github.com/zalando/go-keyring"
)

const (
	// serviceName identifica a aplicação dentro do cofre do sistema.
	serviceName = "com.teamwork-logger"
	// accountName identifica qual segredo desta aplicação estamos guardando.
	accountName = "api-token"
)

// ErrNoToken indica que nenhum token foi gravado ainda (ou que ele foi removido).
var ErrNoToken = errors.New("nenhum token de API armazenado")

// StoreToken grava o token de API no cofre do sistema, substituindo o anterior.
func StoreToken(token string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return errors.New("token de API vazio")
	}

	if err := keyring.Set(serviceName, accountName, token); err != nil {
		return fmt.Errorf("não foi possível gravar o token no cofre de credenciais do sistema: %w", err)
	}
	return nil
}

// LoadToken lê o token de API do cofre do sistema. Retorna ErrNoToken quando
// ainda não existe um token gravado.
func LoadToken() (string, error) {
	token, err := keyring.Get(serviceName, accountName)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return "", ErrNoToken
		}
		return "", fmt.Errorf("não foi possível ler o token do cofre de credenciais do sistema: %w", err)
	}
	return token, nil
}

// DeleteToken remove o token do cofre. Remover um token inexistente não é erro.
func DeleteToken() error {
	err := keyring.Delete(serviceName, accountName)
	if err != nil && !errors.Is(err, keyring.ErrNotFound) {
		return fmt.Errorf("não foi possível remover o token do cofre de credenciais do sistema: %w", err)
	}
	return nil
}
