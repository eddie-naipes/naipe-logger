package backend

import (
	"fmt"

	"logTime-go/backend/api"
	"logTime-go/backend/config"
)

// Bindings de configuração local: jornada, preferências, tarefas salvas e
// templates. Nada aqui toca a API remota, exceto a recriação do cliente quando a
// jornada muda.

// SetMinutosPorDia ajusta a jornada diária usada nos cálculos.
func (a *App) SetMinutosPorDia(minutos int) error {
	if err := a.configManager.SetMinutosPorDia(minutos); err != nil {
		return err
	}
	a.setAPI(api.NewTeamworkAPI(a.configManager.GetTeamworkConfig()))
	return nil
}

func (a *App) GetAppSettings() config.AppSettings {
	return a.configManager.GetAppSettings()
}

func (a *App) SaveAppSettings(settings config.AppSettings) error {
	return a.configManager.SetAppSettings(settings)
}

func (a *App) GetSavedTasks() []api.Task {
	return a.configManager.GetSavedTasks()
}

func (a *App) SaveTask(task api.Task) error {
	return a.configManager.AddSavedTask(task)
}

func (a *App) RemoveTask(taskID int) error {
	return a.configManager.RemoveSavedTask(taskID)
}

func (a *App) ClearSavedTasks() error {
	return a.configManager.SetSavedTasks([]api.Task{})
}

func (a *App) CalculateTotalMinutes(tarefas []api.Task) int {
	return a.api().CalculateTotalMinutes(tarefas)
}

func (a *App) GetTemplates() map[string]api.Template {
	return a.configManager.GetTemplates()
}

func (a *App) GetTemplate(name string) (api.Template, bool) {
	return a.configManager.GetTemplate(name)
}

func (a *App) SaveTemplate(template api.Template) error {
	return a.configManager.SaveTemplate(template)
}

func (a *App) DeleteTemplate(name string) error {
	return a.configManager.DeleteTemplate(name)
}

func (a *App) ApplyTemplate(templateName string) error {
	template, exists := a.configManager.GetTemplate(templateName)
	if !exists {
		return fmt.Errorf("template '%s' não encontrado", templateName)
	}

	for _, task := range template.Tasks {
		err := a.configManager.AddSavedTask(task)
		if err != nil {
			return fmt.Errorf("erro ao aplicar tarefa do template: %v", err)
		}
	}

	return nil
}
