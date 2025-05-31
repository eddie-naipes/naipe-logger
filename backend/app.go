package backend

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"logTime-go/backend/api"
	"logTime-go/backend/config"
	"net/http"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

type App struct {
	ctx           context.Context
	configManager *config.Manager
	teamworkAPI   *api.TeamworkAPI
}

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

func (a *App) GetTemplate(name string) (api.Template, bool) {
	return a.configManager.GetTemplate(name)
}

func (a *App) SaveTemplate(template api.Template) error {
	return a.configManager.SaveTemplate(template)
}

func (a *App) DeleteTemplate(name string) error {
	return a.configManager.DeleteTemplate(name)
}

func (a *App) CalculateTotalMinutes(tarefas []api.Task) int {
	return a.teamworkAPI.CalculateTotalMinutes(tarefas)
}

func (a *App) GetWorkingDays(inicio, fim string) ([]string, error) {
	return a.teamworkAPI.GetWorkingDays(inicio, fim)
}

func (a *App) CreateDistributionPlan(diasUteis []string, tarefas []api.Task) []api.WorkDay {
	return a.teamworkAPI.CreateDistributionPlan(diasUteis, tarefas)
}

func (a *App) LogMultipleTimes(workDays []api.WorkDay) ([]*api.TimeLogResult, error) {
	return a.teamworkAPI.LogMultipleTimes(workDays)
}

func (a *App) LogTime(taskID int, entry api.TimeEntry) (*api.TimeLogResult, error) {
	return a.teamworkAPI.LogTime(taskID, entry)
}

func (a *App) GetCurrentUserId() (int, error) {
	return a.teamworkAPI.GetCurrentUserId()
}

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

func (a *App) LoginWithCredentials(email, password, host string) (*api.LoginResponse, error) {
	if email == "" || password == "" || host == "" {
		return nil, fmt.Errorf("email, senha e host são obrigatórios")
	}

	loginResponse, err := api.GetTokenWithCredentials(email, password, host)
	if err != nil {
		return nil, fmt.Errorf("erro na autenticação: %v", err)
	}

	if loginResponse.Success && loginResponse.Token != "" {
		config := a.configManager.GetTeamworkConfig()
		config.AuthToken = loginResponse.Token
		config.ApiHost = host

		if loginResponse.UserID <= 0 {
			tempAPI := api.NewTeamworkAPI(config)
			userID, err := tempAPI.GetCurrentUserId()
			if err != nil {
				fmt.Printf("Erro ao obter ID do usuário após login: %v\n", err)
				config.UserID = loginResponse.UserID
			} else {
				config.UserID = userID
				loginResponse.UserID = userID
			}
		} else {
			config.UserID = loginResponse.UserID
		}

		if err := a.configManager.SetTeamworkConfig(config); err != nil {
			return nil, fmt.Errorf("erro ao salvar configuração: %v", err)
		}

		a.teamworkAPI = api.NewTeamworkAPI(config)
	}

	return loginResponse, nil
}

func (a *App) DownloadCurrentMonthReport() (string, error) {
	if !a.teamworkAPI.IsConfigured() {
		return "", fmt.Errorf("API não configurada. Configure sua conta antes de exportar relatórios")
	}

	filePath, err := a.teamworkAPI.DownloadCurrentMonthTimeReport()
	if err != nil {
		return "", fmt.Errorf("erro ao baixar relatório: %v", err)
	}

	return filePath, nil
}

func (a *App) DownloadTimeReport(startDate, endDate string) (string, error) {
	if !a.teamworkAPI.IsConfigured() {
		return "", fmt.Errorf("API não configurada. Configure sua conta antes de exportar relatórios")
	}

	filePath, err := a.teamworkAPI.GetDefaultReportPath()
	if err != nil {
		return "", fmt.Errorf("erro ao obter caminho padrão de relatório: %v", err)
	}

	filePath = strings.TrimSuffix(filePath, filepath.Ext(filePath)) + "_" + startDate + "_" + endDate + ".pdf"

	err = a.teamworkAPI.DownloadTimeReportPDF(startDate, endDate, filePath)
	if err != nil {
		return "", fmt.Errorf("erro ao baixar relatório: %v", err)
	}

	return filePath, nil
}

func (a *App) OpenDirectoryPath(filePath string) error {
	dirPath := filepath.Dir(filePath)

	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", dirPath)
	case "darwin":
		cmd = exec.Command("open", dirPath)
	case "linux":
		cmd = exec.Command("xdg-open", dirPath)
	default:
		return fmt.Errorf("sistema operacional não suportado: %s", runtime.GOOS)
	}

	return cmd.Start()
}

func (a *App) GetDashboardStats() (map[string]interface{}, error) {
	if !a.teamworkAPI.IsConfigured() {
		return nil, fmt.Errorf("API não configurada")
	}

	stats, err := a.teamworkAPI.GetDashboardStats()
	if err != nil {
		return nil, fmt.Errorf("erro ao obter estatísticas do dashboard: %v", err)
	}

	now := time.Now()
	startDate := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).Format("2006-01-02")
	endDate := time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, now.Location()).Format("2006-01-02")

	timeTotal, err := a.teamworkAPI.GetTimeTotalsForPeriod(startDate, endDate)
	if err == nil && timeTotal != nil && timeTotal.TimeTotals.Minutes > 0 {
		stats["horasLogadas"] = float64(timeTotal.TimeTotals.Minutes) / 60.0
	}

	return stats, nil
}

func (a *App) GetRecentActivities() ([]map[string]interface{}, error) {
	if !a.teamworkAPI.IsConfigured() {
		return nil, fmt.Errorf("API não configurada")
	}
	return a.teamworkAPI.GetRecentActivities()
}

func (a *App) GetTasksWithUpcomingDeadlines() ([]map[string]interface{}, error) {
	if !a.teamworkAPI.IsConfigured() {
		return nil, fmt.Errorf("API não configurada")
	}
	return a.teamworkAPI.GetTasksWithUpcomingDeadlines()
}

func (a *App) GetTimeTotalsForPeriod(startDate, endDate string) (*api.TimeTotal, error) {
	if !a.teamworkAPI.IsConfigured() {
		return nil, fmt.Errorf("API não configurada")
	}

	timeTotal, err := a.teamworkAPI.GetTimeTotalsForPeriod(startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("erro ao obter totais de tempo: %v", err)
	}

	return timeTotal, nil
}

func (a *App) GetTimeEntriesForPeriod(startDate, endDate string) ([]api.TimeEntryReport, error) {
	if !a.teamworkAPI.IsConfigured() {
		return nil, fmt.Errorf("API não configurada")
	}

	return a.teamworkAPI.GetTimeEntriesForPeriod(startDate, endDate)
}

func (a *App) GetLoggedTimeFromCalendarAPI(month, year int) (*api.LoggedTimeResponse, error) {
	if !a.teamworkAPI.IsConfigured() {
		return nil, fmt.Errorf("API não configurada")
	}

	return a.teamworkAPI.GetLoggedTimeFromCalendarAPI(month, year)
}

func (a *App) CreateDistributionPlanFromLoggedTime(month, year int, tasks []api.Task) ([]api.WorkDay, error) {
	if !a.teamworkAPI.IsConfigured() {
		return nil, fmt.Errorf("API não configurada")
	}

	return a.teamworkAPI.CreateDistributionPlanFromLoggedTime(month, year, tasks)
}

func (a *App) GetEntriesFromLoggedTime(month, year int) ([]map[string]interface{}, error) {
	if !a.teamworkAPI.IsConfigured() {
		return nil, fmt.Errorf("API não configurada")
	}

	return a.teamworkAPI.GetEntriesFromLoggedTime(month, year)
}

func (a *App) GetBrazilianHolidays(year int) (map[string]api.Holiday, error) {
	if !a.teamworkAPI.IsConfigured() {
		return nil, fmt.Errorf("API não configurada")
	}

	return a.teamworkAPI.GetBrazilianHolidays(year)
}

func (a *App) GetHolidaysForMonth(year, month int) ([]api.Holiday, error) {
	if !a.teamworkAPI.IsConfigured() {
		return nil, fmt.Errorf("API não configurada")
	}

	return a.teamworkAPI.GetHolidaysForMonth(year, month)
}

func (a *App) GetAllNonWorkingDays(year, month int) ([]map[string]interface{}, error) {
	if !a.teamworkAPI.IsConfigured() {
		return nil, fmt.Errorf("API não configurada")
	}

	return a.teamworkAPI.GetAllNonWorkingDays(year, month)
}

func (a *App) IsWorkDay(date string) (bool, error) {
	if !a.teamworkAPI.IsConfigured() {
		return false, fmt.Errorf("API não configurada")
	}

	dateObj, err := time.Parse("2006-01-02", date)
	if err != nil {
		return false, fmt.Errorf("formato de data inválido: %v", err)
	}

	return a.teamworkAPI.IsWorkDay(dateObj), nil
}

func (a *App) GetUserProfile() (map[string]interface{}, error) {
	if !a.teamworkAPI.IsConfigured() {
		return nil, fmt.Errorf("API não configurada")
	}

	userID, err := a.teamworkAPI.GetCurrentUserId()
	if err != nil {
		return nil, fmt.Errorf("erro ao obter ID do usuário: %v", err)
	}

	config := a.configManager.GetTeamworkConfig()
	baseURL := config.ApiHost
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		baseURL = "https://" + baseURL
	}

	url := fmt.Sprintf("%s/projects/api/v3/people/%d.json", baseURL, userID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar requisição: %v", err)
	}

	auth := base64.StdEncoding.EncodeToString([]byte(config.AuthToken + ":X"))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("erro na requisição: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler resposta: %v", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("erro ao obter perfil do usuário: %d %s", resp.StatusCode, resp.Status)
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

func (a *App) ClearSavedTasks() error {
	return a.configManager.SetSavedTasks([]api.Task{})
}

func (a *App) GetTimeEntriesWithDetails(startDate, endDate string) ([]api.TimeEntryReport, error) {
	if !a.teamworkAPI.IsConfigured() {
		return nil, fmt.Errorf("API não configurada")
	}

	return a.teamworkAPI.GetTimeEntriesWithDetails(startDate, endDate)
}

func (a *App) DeleteTimeEntry(entryID int) error {
	if !a.teamworkAPI.IsConfigured() {
		return fmt.Errorf("API não configurada")
	}

	return a.teamworkAPI.DeleteTimeEntry(entryID)
}

func (a *App) DeleteMultipleTimeEntries(entryIDs []int) ([]api.DeleteTimeEntryResult, error) {
	if !a.teamworkAPI.IsConfigured() {
		return nil, fmt.Errorf("API não configurada")
	}

	return a.teamworkAPI.DeleteMultipleTimeEntries(entryIDs)
}

func (a *App) GetTimeEntriesForPeriodV2(startDate, endDate string, includeDeleted bool) ([]api.TimeEntryReport, error) {
	if !a.teamworkAPI.IsConfigured() {
		return nil, fmt.Errorf("API não configurada")
	}

	return a.teamworkAPI.GetTimeEntriesForPeriodV2(startDate, endDate, includeDeleted)
}

func (a *App) GetAllTimeEntriesForDay(date string) ([]api.TimeEntryReport, error) {
	if !a.teamworkAPI.IsConfigured() {
		return nil, fmt.Errorf("API não configurada")
	}

	return a.teamworkAPI.GetAllTimeEntriesForDay(date)
}

func (a *App) GetDeletedTimeEntries(startDate, endDate string) ([]api.TimeEntryReport, error) {
	if !a.teamworkAPI.IsConfigured() {
		return nil, fmt.Errorf("API não configurada")
	}

	return a.teamworkAPI.GetDeletedTimeEntries(startDate, endDate)
}

func (a *App) DeleteTimeEntryV2(entryID int) error {
	if !a.teamworkAPI.IsConfigured() {
		return fmt.Errorf("API não configurada")
	}

	return a.teamworkAPI.DeleteTimeEntryV2(entryID)
}

func (a *App) UpdateTimeEntry(entryID int, entry api.TimeEntry) (*api.TimeLogResult, error) {
	if !a.teamworkAPI.IsConfigured() {
		return nil, fmt.Errorf("API não configurada")
	}

	return a.teamworkAPI.UpdateTimeEntry(entryID, entry)
}
