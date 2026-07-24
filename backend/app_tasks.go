package backend

import "logTime-go/backend/api"

// Bindings de tarefas e projetos remotos do Teamwork.

func (a *App) GetTasks() ([]api.TeamworkTask, error) {
	return a.api().GetTasks()
}

func (a *App) GetTaskDetails(taskID int) (api.TeamworkTask, error) {
	return a.api().GetTaskDetails(taskID)
}

func (a *App) GetProjects() ([]api.Project, error) {
	return a.api().GetProjects()
}

func (a *App) GetTasksByProject(projectID int) ([]api.TeamworkTask, error) {
	return a.api().GetTasksByProject(projectID)
}
