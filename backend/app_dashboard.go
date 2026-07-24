package backend

import (
	"encoding/json"
	"fmt"
	"time"
)

// Bindings do dashboard e do perfil do usuário.

func (a *App) GetDashboardStats() (map[string]interface{}, error) {
	if !a.api().IsConfigured() {
		return nil, fmt.Errorf("API não configurada")
	}

	stats, err := a.api().GetDashboardStats()
	if err != nil {
		return nil, fmt.Errorf("erro ao obter estatísticas do dashboard: %v", err)
	}

	now := time.Now()
	startDate := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).Format("2006-01-02")
	endDate := time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, now.Location()).Format("2006-01-02")

	timeTotal, err := a.api().GetTimeTotalsForPeriod(startDate, endDate)
	if err == nil && timeTotal != nil && timeTotal.TimeTotals.Minutes > 0 {
		stats["horasLogadas"] = float64(timeTotal.TimeTotals.Minutes) / 60.0
	}

	return stats, nil
}

func (a *App) GetRecentActivities() ([]map[string]interface{}, error) {
	if !a.api().IsConfigured() {
		return nil, fmt.Errorf("API não configurada")
	}
	return a.api().GetRecentActivities()
}

func (a *App) GetTasksWithUpcomingDeadlines() ([]map[string]interface{}, error) {
	if !a.api().IsConfigured() {
		return nil, fmt.Errorf("API não configurada")
	}
	return a.api().GetTasksWithUpcomingDeadlines()
}

func (a *App) GetUserProfile() (map[string]interface{}, error) {
	if !a.api().IsConfigured() {
		return nil, fmt.Errorf("API não configurada")
	}

	userID, err := a.api().GetCurrentUserId()
	if err != nil {
		return nil, fmt.Errorf("erro ao obter ID do usuário: %v", err)
	}

	body, status, err := a.api().GetJSON(fmt.Sprintf("/projects/api/v3/people/%d.json", userID))
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

	profile := map[string]interface{}{
		"id":        response.Person.ID,
		"firstName": response.Person.FirstName,
		"lastName":  response.Person.LastName,
		"email":     response.Person.Email,
		"avatarURL": response.Person.AvatarURL,
		"fullName":  fmt.Sprintf("%s %s", response.Person.FirstName, response.Person.LastName),
	}

	return profile, nil
}
