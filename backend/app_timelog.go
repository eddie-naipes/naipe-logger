package backend

import (
	"fmt"

	"logTime-go/backend/api"
)

// Bindings de planejamento e lançamento de horas.

func (a *App) GetWorkingDays(inicio, fim string) ([]string, error) {
	return a.api().GetWorkingDays(inicio, fim)
}

func (a *App) CreateDistributionPlan(diasUteis []string, tarefas []api.Task) []api.WorkDay {
	return a.api().CreateDistributionPlan(diasUteis, tarefas)
}

// CheckPlanConflicts avisa quais dias do plano já possuem tempo lançado, para
// que o usuário confirme antes de enviar. A ferramenta não tem rollback, então
// um lote duplicado só se desfaz apagando entrada por entrada.
func (a *App) CheckPlanConflicts(workDays []api.WorkDay) ([]api.DayConflict, error) {
	return a.api().CheckPlanConflicts(workDays)
}

func (a *App) LogMultipleTimes(workDays []api.WorkDay) ([]*api.TimeLogResult, error) {
	return a.api().LogMultipleTimes(workDays)
}

func (a *App) LogTime(taskID int, entry api.TimeEntry) (*api.TimeLogResult, error) {
	return a.api().LogTime(taskID, entry)
}

func (a *App) GetCurrentUserId() (int, error) {
	return a.api().GetCurrentUserId()
}

func (a *App) GetLoggedTimeFromCalendarAPI(month, year int) (*api.LoggedTimeResponse, error) {
	if !a.api().IsConfigured() {
		return nil, fmt.Errorf("API não configurada")
	}

	return a.api().GetLoggedTimeFromCalendarAPI(month, year)
}

func (a *App) CreateDistributionPlanFromLoggedTime(month, year int, tasks []api.Task) ([]api.WorkDay, error) {
	if !a.api().IsConfigured() {
		return nil, fmt.Errorf("API não configurada")
	}

	return a.api().CreateDistributionPlanFromLoggedTime(month, year, tasks)
}

func (a *App) GetEntriesFromLoggedTime(month, year int) ([]map[string]interface{}, error) {
	if !a.api().IsConfigured() {
		return nil, fmt.Errorf("API não configurada")
	}

	return a.api().GetEntriesFromLoggedTime(month, year)
}
