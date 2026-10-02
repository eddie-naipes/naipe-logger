package backend

import (
	"encoding/json"
	"fmt"
)

// Bindings do dashboard e do perfil do usuário.

// UserProfile é o perfil exibido no cabeçalho. Os nomes JSON são os que o
// frontend já consome.
type UserProfile struct {
	ID        int    `json:"id"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatarURL"`
	FullName  string `json:"fullName"`
}

// GetDashboardStats devolve as estatísticas do mês. As horas logadas já vêm
// calculadas por api.GetDashboardStats.
func (a *App) GetDashboardStats() (map[string]interface{}, error) {
	client, err := a.client()
	if err != nil {
		return nil, err
	}

	stats, err := client.GetDashboardStats()
	if err != nil {
		return nil, fmt.Errorf("erro ao obter estatísticas do dashboard: %v", err)
	}

	return stats, nil
}

func (a *App) GetRecentActivities() ([]map[string]interface{}, error) {
	client, err := a.client()
	if err != nil {
		return nil, err
	}
	return client.GetRecentActivities()
}

func (a *App) GetTasksWithUpcomingDeadlines() ([]map[string]interface{}, error) {
	client, err := a.client()
	if err != nil {
		return nil, err
	}
	return client.GetTasksWithUpcomingDeadlines()
}

func (a *App) GetUserProfile() (*UserProfile, error) {
	client, err := a.client()
	if err != nil {
		return nil, err
	}

	userID, err := client.GetCurrentUserId()
	if err != nil {
		return nil, fmt.Errorf("erro ao obter ID do usuário: %v", err)
	}

	body, status, err := client.GetJSON(fmt.Sprintf("/projects/api/v3/people/%d.json", userID))
	if err != nil {
		return nil, err
	}
	if status != 200 {
		return nil, fmt.Errorf("erro ao obter perfil do usuário: %d", status)
	}

	var response struct {
		Person struct {
			ID        int    `json:"id"`
			FirstName string `json:"firstName"`
			LastName  string `json:"lastName"`
			Email     string `json:"email"`
			AvatarURL string `json:"avatar-url"`
		} `json:"person"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("erro ao decodificar resposta: %v", err)
	}

	return &UserProfile{
		ID:        response.Person.ID,
		FirstName: response.Person.FirstName,
		LastName:  response.Person.LastName,
		Email:     response.Person.Email,
		AvatarURL: response.Person.AvatarURL,
		FullName:  fmt.Sprintf("%s %s", response.Person.FirstName, response.Person.LastName),
	}, nil
}
