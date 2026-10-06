package security

import (
	"errors"
	"fmt"
	"strings"

	"github.com/zalando/go-keyring"
)

// Os links iCal privados das agendas também são segredos (dão leitura da
// agenda inteira): ficam no cofre do sistema, um por agenda, nunca em
// config.json nem no frontend.

// ErrNoAgendaURL indica que o cofre não tem o link da agenda.
var ErrNoAgendaURL = errors.New("link da agenda não encontrado no cofre de credenciais")

func agendaAccount(id string) string { return "agenda-url-" + id }

func validAgendaID(id string) error {
	if id == "" || strings.ContainsAny(id, " /\\") {
		return fmt.Errorf("identificador de agenda inválido: %q", id)
	}
	return nil
}

// StoreAgendaURL grava o link iCal da agenda id no cofre do sistema.
func StoreAgendaURL(id, link string) error {
	if err := validAgendaID(id); err != nil {
		return err
	}
	if strings.TrimSpace(link) == "" {
		return errors.New("link da agenda vazio")
	}
	if err := keyring.Set(serviceName, agendaAccount(id), link); err != nil {
		return fmt.Errorf("não foi possível gravar o link da agenda no cofre de credenciais do sistema: %w", err)
	}
	return nil
}

// LoadAgendaURL lê o link iCal da agenda id.
func LoadAgendaURL(id string) (string, error) {
	if err := validAgendaID(id); err != nil {
		return "", err
	}
	link, err := keyring.Get(serviceName, agendaAccount(id))
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return "", ErrNoAgendaURL
		}
		return "", fmt.Errorf("não foi possível ler o link da agenda do cofre de credenciais do sistema: %w", err)
	}
	return link, nil
}

// DeleteAgendaURL remove o link da agenda id. Remover um link inexistente não é erro.
func DeleteAgendaURL(id string) error {
	if err := validAgendaID(id); err != nil {
		return err
	}
	err := keyring.Delete(serviceName, agendaAccount(id))
	if err != nil && !errors.Is(err, keyring.ErrNotFound) {
		return fmt.Errorf("não foi possível remover o link da agenda do cofre de credenciais do sistema: %w", err)
	}
	return nil
}
