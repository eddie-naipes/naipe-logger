package backend

import "logTime-go/backend/api"

// Bindings de planejamento e lançamento de horas.

// GetWorkingDays e CreateDistributionPlan não falam com o Teamwork (só com a
// jornada configurada e o calendário de feriados), por isso não exigem conexão.

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
	client, err := a.client()
	if err != nil {
		return nil, err
	}
	return client.CheckPlanConflicts(workDays)
}

func (a *App) LogMultipleTimes(workDays []api.WorkDay) ([]*api.TimeLogResult, error) {
	client, err := a.client()
	if err != nil {
		return nil, err
	}
	return client.LogMultipleTimes(workDays)
}

func (a *App) GetLoggedTimeFromCalendarAPI(month, year int) (*api.LoggedTimeResponse, error) {
	client, err := a.client()
	if err != nil {
		return nil, err
	}
	return client.GetLoggedTimeFromCalendarAPI(month, year)
}
