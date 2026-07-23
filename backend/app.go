package backend

import (
	"context"
	"encoding/json"
	"fmt"
	"logTime-go/backend/api"
	"logTime-go/backend/config"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

type App struct {
	ctx           context.Context
	configManager *config.Manager

	// apiMutex protege teamworkAPI, que é substituído quando a conexão muda
	// enquanto outros bindings o leem em paralelo.
	apiMutex    sync.RWMutex
	teamworkAPI *api.TeamworkAPI
}

// api devolve o cliente atual sob lock de leitura.
func (a *App) api() *api.TeamworkAPI {
	a.apiMutex.RLock()
	defer a.apiMutex.RUnlock()
	return a.teamworkAPI
}

// setAPI substitui o cliente e lhe entrega o contexto da aplicação, para que as
// requisições do cliente novo também sejam abortadas no encerramento.
func (a *App) setAPI(client *api.TeamworkAPI) {
	a.apiMutex.Lock()
	defer a.apiMutex.Unlock()
	if a.ctx != nil {
		client.SetContext(a.ctx)
	}
	a.teamworkAPI = client
}

func NewApp(ctx context.Context) (*App, error) {
	configManager, err := config.NewManager()
	if err != nil {
		return nil, fmt.Errorf("erro ao inicializar gerenciador de configurações: %v", err)
	}

	app := &App{configManager: configManager}
	app.setContext(ctx)
	app.setAPI(api.NewTeamworkAPI(configManager.GetTeamworkConfig()))

	return app, nil
}

// setContext guarda o contexto sob o mesmo lock que protege o cliente, já que
// setAPI o lê para repassá-lo a cada cliente novo.
func (a *App) setContext(ctx context.Context) {
	a.apiMutex.Lock()
	defer a.apiMutex.Unlock()
	a.ctx = ctx
}

func (a *App) Startup(ctx context.Context) {
	a.setContext(ctx)
	a.setAPI(api.NewTeamworkAPI(a.configManager.GetTeamworkConfig()))

	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Erro crítico durante a inicialização: %v\n", r)
		}
	}()

	a.api().ClearExpiredHolidayCache()

	go func() {
		if err := a.api().PreloadUpcomingHolidays(); err != nil {
			fmt.Printf("Aviso: erro ao pré-carregar feriados: %v\n", err)
		}
	}()
}

func (a *App) Shutdown(ctx context.Context) {
	_ = a.configManager.Save()
}

// GetPublicConfig devolve ao frontend apenas o que ele precisa saber. O token
// de API nunca atravessa a fronteira Go -> JavaScript.
func (a *App) GetPublicConfig() api.PublicConfig {
	cfg := a.configManager.GetTeamworkConfig()
	return api.PublicConfig{
		Configured:    a.api().IsConfigured(),
		ApiHost:       cfg.ApiHost,
		UserID:        cfg.UserID,
		MinutosPorDia: cfg.MinutosPorDia,
	}
}

// IsConfigured informa se há uma conexão utilizável configurada.
func (a *App) IsConfigured() bool {
	return a.api().IsConfigured()
}

// LegacyCredentialPurged informa que uma credencial antiga (email:senha) foi
// encontrada e apagada, para que a UI oriente o usuário a trocar a senha.
func (a *App) LegacyCredentialPurged() bool {
	return a.configManager.LegacyCredentialPurged()
}

// SetMinutosPorDia ajusta a jornada diária usada nos cálculos.
func (a *App) SetMinutosPorDia(minutos int) error {
	if err := a.configManager.SetMinutosPorDia(minutos); err != nil {
		return err
	}
	a.setAPI(api.NewTeamworkAPI(a.configManager.GetTeamworkConfig()))
	return nil
}

func (a *App) TestConnection() ([]interface{}, error) {
	success, message := a.api().TestConnection()
	return []interface{}{success, message}, nil
}

func (a *App) GetAppSettings() config.AppSettings {
	return a.configManager.GetAppSettings()
}

func (a *App) SaveAppSettings(settings config.AppSettings) error {
	return a.configManager.SetAppSettings(settings)
}

func (a *App) GetTasks() ([]api.TeamworkTask, error) {
	return a.api().GetTasks()
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
	return a.api().GetTaskDetails(taskID)
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
	return a.api().CalculateTotalMinutes(tarefas)
}

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

func (a *App) GetProjects() ([]api.Project, error) {
	return a.api().GetProjects()
}

func (a *App) GetTasksByProject(projectID int) ([]api.TeamworkTask, error) {
	return a.api().GetTasksByProject(projectID)
}

// ConnectWithToken valida um token de API do Teamwork e, em caso de sucesso,
// grava-o no cofre de credenciais do sistema. O token nunca é devolvido ao
// frontend nem gravado em config.json.
func (a *App) ConnectWithToken(token, host string) (*api.LoginResponse, error) {
	loginResponse, err := api.ValidateToken(token, host)
	if err != nil {
		return nil, err
	}

	if !loginResponse.Success {
		return loginResponse, nil
	}

	if err := a.configManager.SetConnection(loginResponse.InstanceID, loginResponse.UserID, token); err != nil {
		return nil, fmt.Errorf("erro ao salvar configuração: %v", err)
	}

	a.setAPI(api.NewTeamworkAPI(a.configManager.GetTeamworkConfig()))

	return loginResponse, nil
}

// Logout remove o token do cofre do sistema e limpa a conexão.
func (a *App) Logout() error {
	if err := a.configManager.ClearConnection(); err != nil {
		return err
	}
	a.setAPI(api.NewTeamworkAPI(a.configManager.GetTeamworkConfig()))
	return nil
}

func (a *App) DownloadCurrentMonthReport() (string, error) {
	if !a.api().IsConfigured() {
		return "", fmt.Errorf("API não configurada. Configure sua conta antes de exportar relatórios")
	}

	filePath, err := a.api().DownloadCurrentMonthTimeReport()
	if err != nil {
		return "", fmt.Errorf("erro ao baixar relatório: %v", err)
	}

	return filePath, nil
}

func (a *App) DownloadTimeReport(startDate, endDate string) (string, error) {
	if !a.api().IsConfigured() {
		return "", fmt.Errorf("API não configurada. Configure sua conta antes de exportar relatórios")
	}

	filePath, err := a.api().GetDefaultReportPath()
	if err != nil {
		return "", fmt.Errorf("erro ao obter caminho padrão de relatório: %v", err)
	}

	filePath = strings.TrimSuffix(filePath, filepath.Ext(filePath)) + "_" + startDate + "_" + endDate + ".pdf"

	err = a.api().DownloadTimeReportPDF(startDate, endDate, filePath)
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

func (a *App) GetTimeTotalsForPeriod(startDate, endDate string) (*api.TimeTotal, error) {
	if !a.api().IsConfigured() {
		return nil, fmt.Errorf("API não configurada")
	}

	timeTotal, err := a.api().GetTimeTotalsForPeriod(startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("erro ao obter totais de tempo: %v", err)
	}

	return timeTotal, nil
}

func (a *App) GetTimeEntriesForPeriod(startDate, endDate string) ([]api.TimeEntryReport, error) {
	if !a.api().IsConfigured() {
		return nil, fmt.Errorf("API não configurada")
	}

	return a.api().GetTimeEntriesForPeriod(startDate, endDate)
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

func (a *App) GetBrazilianHolidays(year int) (map[string]api.Holiday, error) {
	if !a.api().IsConfigured() {
		return nil, fmt.Errorf("API não configurada")
	}

	return a.api().GetBrazilianHolidays(year)
}

func (a *App) GetHolidaysForMonth(year, month int) ([]api.Holiday, error) {
	if !a.api().IsConfigured() {
		return nil, fmt.Errorf("API não configurada")
	}

	return a.api().GetHolidaysForMonth(year, month)
}

func (a *App) GetAllNonWorkingDays(year, month int) ([]map[string]interface{}, error) {
	if !a.api().IsConfigured() {
		return nil, fmt.Errorf("API não configurada")
	}

	return a.api().GetAllNonWorkingDays(year, month)
}

func (a *App) IsWorkDay(date string) (bool, error) {
	if !a.api().IsConfigured() {
		return false, fmt.Errorf("API não configurada")
	}

	dateObj, err := time.Parse("2006-01-02", date)
	if err != nil {
		return false, fmt.Errorf("formato de data inválido: %v", err)
	}

	return a.api().IsWorkDay(dateObj), nil
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
	if !a.api().IsConfigured() {
		return nil, fmt.Errorf("API não configurada")
	}

	return a.api().GetTimeEntriesWithDetails(startDate, endDate)
}

func (a *App) DeleteTimeEntry(entryID int) error {
	if !a.api().IsConfigured() {
		return fmt.Errorf("API não configurada")
	}

	return a.api().DeleteTimeEntry(entryID)
}

func (a *App) DeleteMultipleTimeEntries(entryIDs []int) ([]api.DeleteTimeEntryResult, error) {
	if !a.api().IsConfigured() {
		return nil, fmt.Errorf("API não configurada")
	}

	return a.api().DeleteMultipleTimeEntries(entryIDs)
}

func (a *App) GetTimeEntriesForPeriodV2(startDate, endDate string, includeDeleted bool) ([]api.TimeEntryReport, error) {
	if !a.api().IsConfigured() {
		return nil, fmt.Errorf("API não configurada")
	}

	return a.api().GetTimeEntriesForPeriodV2(startDate, endDate, includeDeleted)
}

func (a *App) GetAllTimeEntriesForDay(date string) ([]api.TimeEntryReport, error) {
	if !a.api().IsConfigured() {
		return nil, fmt.Errorf("API não configurada")
	}

	return a.api().GetAllTimeEntriesForDay(date)
}

func (a *App) GetDeletedTimeEntries(startDate, endDate string) ([]api.TimeEntryReport, error) {
	if !a.api().IsConfigured() {
		return nil, fmt.Errorf("API não configurada")
	}

	return a.api().GetDeletedTimeEntries(startDate, endDate)
}

func (a *App) UpdateTimeEntry(entryID int, entry api.TimeEntry) (*api.TimeLogResult, error) {
	if !a.api().IsConfigured() {
		return nil, fmt.Errorf("API não configurada")
	}

	return a.api().UpdateTimeEntry(entryID, entry)
}

func (a *App) GetHolidayCacheStats() (map[string]interface{}, error) {
	if !a.api().IsConfigured() {
		return nil, fmt.Errorf("API não configurada")
	}

	return a.api().GetHolidayCacheStats(), nil
}

func (a *App) ClearHolidayCache() error {
	if !a.api().IsConfigured() {
		return fmt.Errorf("API não configurada")
	}

	a.api().ClearExpiredHolidayCache()
	return nil
}

func (a *App) PreloadHolidays() error {
	if !a.api().IsConfigured() {
		return fmt.Errorf("API não configurada")
	}

	return a.api().PreloadUpcomingHolidays()
}

func (a *App) RefreshHolidaysForYear(year int) (map[string]api.Holiday, error) {
	if !a.api().IsConfigured() {
		return nil, fmt.Errorf("API não configurada")
	}

	a.api().ClearHolidaysCacheForYear(year)

	return a.api().GetBrazilianHolidays(year)
}
