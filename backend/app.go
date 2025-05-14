// backend/app.go
package backend

import (
	"context"
	"fmt"
	"logTime-go/backend/api"
	"logTime-go/backend/config"
)

// App é a estrutura principal da aplicação
type App struct {
	ctx           context.Context
	configManager *config.Manager
	teamworkAPI   *api.TeamworkAPI
}

// NewApp cria uma nova instância da aplicação
func NewApp(ctx context.Context) (*App, error) {
	configManager, err := config.NewManager()
	if err != nil {
		return nil, fmt.Errorf("erro ao inicializar gerenciador de configurações: %v", err)
	}

	teamworkConfig := configManager.GetTeamworkConfig()
	teamworkAPI := api.NewTeamworkAPI(teamworkConfig)

	return &App{
		ctx:           ctx,
		configManager: configManager,
		teamworkAPI:   teamworkAPI,
	}, nil
}

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx

	a.teamworkAPI = api.NewTeamworkAPI(a.configManager.GetTeamworkConfig())

	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Erro crítico durante a inicialização: %v\n", r)
		}
	}()

	if err := a.configManager.MigrateToSecureStorage(); err != nil {
		fmt.Printf("Aviso: não foi possível migrar para armazenamento seguro: %v\n", err)
	}

	if err := config.CheckAndMoveConfigFromExecDir(); err != nil {
		fmt.Printf("Aviso: não foi possível verificar/mover configurações: %v\n", err)
	}
}

func (a *App) Shutdown(ctx context.Context) {
	_ = a.configManager.Save()
}

func (a *App) GetConfig() api.Config {
	return a.configManager.GetTeamworkConfig()
}

func (a *App) SaveConfig(config api.Config) error {
	a.teamworkAPI = api.NewTeamworkAPI(config)
	return a.configManager.SetTeamworkConfig(config)
}

func (a *App) TestConnection(config api.Config) ([]interface{}, error) {
	success, message := a.teamworkAPI.TestConnection(config)
	return []interface{}{success, message}, nil
}

func (a *App) GetAppSettings() config.AppSettings {
	return a.configManager.GetAppSettings()
}

func (a *App) SaveAppSettings(settings config.AppSettings) error {
	return a.configManager.SetAppSettings(settings)
}

func (a *App) GetTasks() ([]api.TeamworkTask, error) {
	return a.teamworkAPI.GetTasks()
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

func (a *App) GetTaskDetails(taskID int) (api.TeamworkTask, error) {
	return a.teamworkAPI.GetTaskDetails(taskID)
}

func (a *App) GetTemplates() map[string]api.Template {
	return a.configManager.GetTemplates()
}

// GetTemplate retorna um template específico
func (a *App) GetTemplate(name string) (api.Template, bool) {
	return a.configManager.GetTemplate(name)
}

// SaveTemplate salva um novo template ou atualiza um existente
func (a *App) SaveTemplate(template api.Template) error {
	return a.configManager.SaveTemplate(template)
}

// DeleteTemplate remove um template
func (a *App) DeleteTemplate(name string) error {
	return a.configManager.DeleteTemplate(name)
}

//
// API de gerenciamento de tempo
//

// CalcularTotalMinutos calcula o total de minutos de um conjunto de tarefas
func (a *App) CalcularTotalMinutos(tarefas []api.Task) int {
	return a.teamworkAPI.CalcularTotalMinutos(tarefas)
}

// ObterDiasUteis retorna os dias úteis entre duas datas
func (a *App) ObterDiasUteis(inicio, fim string) ([]string, error) {
	return a.teamworkAPI.ObterDiasUteis(inicio, fim)
}

// CriarPlanoDistribuicao cria um plano de distribuição para os dias úteis com base nas tarefas padrão
func (a *App) CriarPlanoDistribuicao(diasUteis []string, tarefas []api.Task) []api.WorkDay {
	return a.teamworkAPI.CriarPlanoDistribuicao(diasUteis, tarefas)
}

// LogMultipleTimes registra múltiplas entradas de tempo para diferentes tarefas e dias
func (a *App) LogMultipleTimes(workDays []api.WorkDay) ([]*api.TimeLogResult, error) {
	return a.teamworkAPI.LogMultipleTimes(workDays)
}

// LogTime registra uma entrada de tempo para uma tarefa
func (a *App) LogTime(taskID int, entry api.TimeEntry) (*api.TimeLogResult, error) {
	return a.teamworkAPI.LogTime(taskID, entry)
}

// Adicione esta função ao arquivo backend/app.go

// GetCurrentUserId obtém o ID do usuário atual autenticado
func (a *App) GetCurrentUserId() (int, error) {
	return a.teamworkAPI.GetCurrentUserId()
}

// Adicione estas funções ao arquivo backend/app.go

// GetProjects obtém a lista de projetos
func (a *App) GetProjects() ([]api.Project, error) {
	return a.teamworkAPI.GetProjects()
}

func (a *App) GetTasksByProject(projectID int) ([]api.TeamworkTask, error) {
	return a.teamworkAPI.GetTasksByProject(projectID)
}

func (a *App) GetCurrentUserIdWithConfig(config api.Config) (int, error) {
	if config.AuthToken == "" || config.ApiHost == "" {
		return 0, fmt.Errorf("configuração incompleta: token ou host da API ausentes")
	}

	tempAPI := api.NewTeamworkAPI(config)
	userId, err := tempAPI.GetCurrentUserId()

	if err != nil {
		fmt.Printf("Erro ao obter ID do usuário: %v\n", err)
		return 0, err
	}

	fmt.Printf("ID do usuário obtido com sucesso: %d\n", userId)
	return userId, nil
}
