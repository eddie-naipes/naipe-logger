package backend

import "logTime-go/backend/api"

// Bindings de tarefas e projetos remotos do Teamwork.

func (a *App) GetTasks() ([]api.TeamworkTask, error) {
	client, err := a.client()
	if err != nil {
		return nil, err
	}
	return client.GetTasks()
}

func (a *App) GetProjects() ([]api.Project, error) {
	client, err := a.client()
	if err != nil {
		return nil, err
	}
	return client.GetProjects()
}

func (a *App) GetTasksByProject(projectID int) ([]api.TeamworkTask, error) {
	client, err := a.client()
	if err != nil {
		return nil, err
	}
	return client.GetTasksByProject(projectID)
}
