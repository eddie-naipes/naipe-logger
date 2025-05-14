package api

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type TeamworkAPI struct {
	Config Config
}

func NewTeamworkAPI(config Config) *TeamworkAPI {
	if config.MinutosPorDia == 0 {
		config.MinutosPorDia = 8 * 60
	}

	return &TeamworkAPI{
		Config: config,
	}
}

func (t *TeamworkAPI) IsConfigured() bool {
	return t.Config.AuthToken != "" && t.Config.ApiHost != ""
}

func (t *TeamworkAPI) GetTasks() ([]TeamworkTask, error) {
	if !t.IsConfigured() {
		return nil, fmt.Errorf("API não configurada")
	}

	var url string
	if strings.HasPrefix(t.Config.ApiHost, "http://") || strings.HasPrefix(t.Config.ApiHost, "https://") {
		url = fmt.Sprintf("%s/projects/api/v3/tasks.json?assignedTo=%d&filter=active&includeTasklists=true&includeTaskAssignees=true&includeCompletionStatus=true&includeEstimatedTime=true&includeTaskTags=true",
			t.Config.ApiHost, t.Config.UserID)
	} else {
		url = fmt.Sprintf("https://%s/projects/api/v3/tasks.json?assignedTo=%d&filter=active&includeTasklists=true&includeTaskAssignees=true&includeCompletionStatus=true&includeEstimatedTime=true&includeTaskTags=true",
			t.Config.ApiHost, t.Config.UserID)
	}

	fmt.Printf("Fazendo requisição para URL: %s\n", url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar requisição: %v", err)
	}

	auth := base64.StdEncoding.EncodeToString([]byte(t.Config.AuthToken + ":X"))
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
		return nil, fmt.Errorf("erro ao obter tarefas: %d %s - %s",
			resp.StatusCode, resp.Status, string(body))
	}

	var response TasksResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, fmt.Errorf("erro ao decodificar resposta: %v", err)
	}

	enrichTasksWithIncludedData(&response)

	if len(response.Tasks) > 0 {
		t.enrichTasksWithDetails(&response.Tasks)
	}

	return response.Tasks, nil
}

func (t *TeamworkAPI) enrichTasksWithDetails(tasks *[]TeamworkTask) {
	for i, task := range *tasks {
		if task.Content == "" || task.ProjectID == 0 || task.ProjectName == "" {
			taskDetail, err := t.GetTaskDetails(task.ID)
			if err == nil {
				if task.Content == "" {
					if taskDetail.Name != "" {
						(*tasks)[i].Content = taskDetail.Name
					} else if taskDetail.Content != "" {
						(*tasks)[i].Content = taskDetail.Content
					}
				}
				if task.ProjectID == 0 && taskDetail.ProjectID != 0 {
					(*tasks)[i].ProjectID = taskDetail.ProjectID
				}
				if task.ProjectName == "" && taskDetail.ProjectName != "" {
					(*tasks)[i].ProjectName = taskDetail.ProjectName
				}
				if task.Description == "" && taskDetail.Description != "" {
					(*tasks)[i].Description = taskDetail.Description
				}
				if task.TasklistName == "" && taskDetail.TasklistName != "" {
					(*tasks)[i].TasklistName = taskDetail.TasklistName
				}
				if taskDetail.LoggedMinutes > 0 {
					(*tasks)[i].LoggedMinutes = taskDetail.LoggedMinutes
				}
			}

			if (*tasks)[i].Content == "" {
				projectInfo := ""
				if (*tasks)[i].ProjectName != "" {
					projectInfo = fmt.Sprintf(" (%s)", (*tasks)[i].ProjectName)
				}
				(*tasks)[i].Content = fmt.Sprintf("Tarefa #%d%s", task.ID, projectInfo)
			}
		}
	}
}

func enrichTasksWithIncludedData(response *TasksResponse) {
	if len(response.Included.TaskLists) > 0 {
		for i, task := range response.Tasks {
			if task.TasklistID > 0 {
				tasklistIDStr := strconv.Itoa(task.TasklistID)

				if tasklistInfo, exists := response.Included.TaskLists[tasklistIDStr]; exists {
					if task.Content == "" {
						response.Tasks[i].Content = fmt.Sprintf("Tarefa #%d - %s", task.ID, tasklistInfo.Name)
					}

					response.Tasks[i].TasklistName = tasklistInfo.Name
				}
			}

			if response.Tasks[i].Content == "" {
				response.Tasks[i].Content = fmt.Sprintf("Tarefa #%d", task.ID)
			}
		}
	} else {
		for i, task := range response.Tasks {
			if task.Content == "" {
				response.Tasks[i].Content = fmt.Sprintf("Tarefa #%d", task.ID)
			}
		}
	}
}

func (t *TeamworkAPI) GetTaskDetails(taskID int) (TeamworkTask, error) {
	if !t.IsConfigured() {
		return TeamworkTask{}, fmt.Errorf("API não configurada")
	}

	taskIDStr := strconv.Itoa(taskID)

	var url string
	if strings.HasPrefix(t.Config.ApiHost, "http://") || strings.HasPrefix(t.Config.ApiHost, "https://") {
		url = fmt.Sprintf("%s/projects/api/v3/tasks/%s.json?include=tags,assignees,time,project,tasklist",
			t.Config.ApiHost, taskIDStr)
	} else {
		url = fmt.Sprintf("https://%s/projects/api/v3/tasks/%s.json?include=tags,assignees,time,project,tasklist",
			t.Config.ApiHost, taskIDStr)
	}

	fmt.Printf("Buscando detalhes da tarefa ID %d...\n", taskID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return TeamworkTask{}, fmt.Errorf("erro ao criar requisição: %v", err)
	}

	auth := base64.StdEncoding.EncodeToString([]byte(t.Config.AuthToken + ":X"))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return TeamworkTask{}, fmt.Errorf("erro na requisição: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return TeamworkTask{}, fmt.Errorf("erro ao ler resposta: %v", err)
	}

	if resp.StatusCode != 200 {
		return TeamworkTask{}, fmt.Errorf("erro ao obter detalhes da tarefa: %d %s - %s",
			resp.StatusCode, resp.Status, string(body))
	}

	var taskResponseV3 struct {
		Task struct {
			ID          int    `json:"id"`
			Name        string `json:"name"`
			Description string `json:"description"`
			Status      string `json:"status"`
			ProjectID   int    `json:"projectId"`
			TasklistID  int    `json:"tasklistId"`
			CreatedAt   string `json:"createdAt"`
		} `json:"task"`
		Included struct {
			Projects map[string]struct {
				Name string `json:"name"`
			} `json:"projects"`
			Tasklists map[string]struct {
				Name string `json:"name"`
			} `json:"tasklists"`
			TimeTotals map[string]struct {
				LoggedMinutes         int `json:"loggedMinutes"`
				BillableLoggedMinutes int `json:"billableLoggedMinutes"`
			} `json:"timeTotals"`
		} `json:"included"`
	}

	err = json.Unmarshal(body, &taskResponseV3)
	if err == nil {
		result := TeamworkTask{
			ID:          taskResponseV3.Task.ID,
			Content:     taskResponseV3.Task.Name,
			Name:        taskResponseV3.Task.Name,
			Description: taskResponseV3.Task.Description,
			Status:      taskResponseV3.Task.Status,
			ProjectID:   taskResponseV3.Task.ProjectID,
			TasklistID:  taskResponseV3.Task.TasklistID,
			CreatedAt:   taskResponseV3.Task.CreatedAt,
		}

		projectIDStr := strconv.Itoa(taskResponseV3.Task.ProjectID)
		if proj, ok := taskResponseV3.Included.Projects[projectIDStr]; ok {
			result.ProjectName = proj.Name
		}

		tasklistIDStr := strconv.Itoa(taskResponseV3.Task.TasklistID)
		if tlist, ok := taskResponseV3.Included.Tasklists[tasklistIDStr]; ok {
			result.TasklistName = tlist.Name
		}

		if time, ok := taskResponseV3.Included.TimeTotals[taskIDStr]; ok {
			result.LoggedMinutes = time.LoggedMinutes
		}

		return result, nil
	}

	var taskWrapper struct {
		Task TeamworkTask `json:"task"`
	}

	err = json.Unmarshal(body, &taskWrapper)
	if err != nil {
		return TeamworkTask{}, fmt.Errorf("erro ao decodificar resposta: %v", err)
	}

	if taskWrapper.Task.Content == "" && taskWrapper.Task.Name != "" {
		taskWrapper.Task.Content = taskWrapper.Task.Name
	}

	return taskWrapper.Task, nil
}

func (t *TeamworkAPI) LogTime(taskID int, entry TimeEntry) (*TimeLogResult, error) {
	if !t.IsConfigured() {
		return nil, fmt.Errorf("API não configurada")
	}

	if taskID <= 0 {
		return nil, fmt.Errorf("ID de tarefa inválido: %d", taskID)
	}

	if entry.Date == "" {
		return nil, fmt.Errorf("data não especificada para o lançamento")
	}

	if entry.Minutes <= 0 {
		return nil, fmt.Errorf("minutos devem ser maiores que zero: %d", entry.Minutes)
	}

	taskIDStr := strconv.Itoa(taskID)

	var url string
	if strings.HasPrefix(t.Config.ApiHost, "http://") || strings.HasPrefix(t.Config.ApiHost, "https://") {
		url = fmt.Sprintf("%s/projects/api/v3/tasks/%s/time.json",
			t.Config.ApiHost, taskIDStr)
	} else {
		url = fmt.Sprintf("https://%s/projects/api/v3/tasks/%s/time.json",
			t.Config.ApiHost, taskIDStr)
	}

	if t.Config.UserID <= 0 {
		return nil, fmt.Errorf("ID do usuário não configurado")
	}

	entry.UserID = t.Config.UserID

	fmt.Printf("Lançando tempo para tarefa #%d: %s %s - %d minutos - %s\n",
		taskID, entry.Date, entry.Time, entry.Minutes, entry.Description)

	reqBody := TimelogRequest{
		Timelog: entry,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("erro ao converter para JSON: %v", err)
	}

	fmt.Printf("JSON do lançamento: %s\n", string(jsonData))

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("erro ao criar requisição: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	auth := base64.StdEncoding.EncodeToString([]byte(t.Config.AuthToken + ":X"))
	req.Header.Set("Authorization", "Basic "+auth)

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

	fmt.Printf("Resposta do servidor (%d): %s\n", resp.StatusCode, string(body))

	result := &TimeLogResult{
		TaskID: taskID,
		Date:   entry.Date,
	}

	if resp.StatusCode == 201 {
		result.Success = true
		result.Message = fmt.Sprintf("Entrada de tempo enviada com sucesso: %s %s",
			entry.Date, entry.Time)

		var successResponse struct {
			ID     int    `json:"id"`
			Status string `json:"status"`
		}

		if err := json.Unmarshal(body, &successResponse); err == nil && successResponse.ID > 0 {
			result.Message += fmt.Sprintf(" (ID: %d)", successResponse.ID)
		}

		return result, nil
	} else {
		result.Success = false

		var errorResponse struct {
			Errors []string `json:"errors"`
		}

		if err := json.Unmarshal(body, &errorResponse); err == nil && len(errorResponse.Errors) > 0 {
			result.Message = fmt.Sprintf("Erro ao enviar entrada: %s", strings.Join(errorResponse.Errors, ", "))
		} else {
			result.Message = fmt.Sprintf("Erro ao enviar entrada: %d %s - %s",
				resp.StatusCode, resp.Status, string(body))
		}

		return result, fmt.Errorf(result.Message)
	}
}

func (t *TeamworkAPI) LogMultipleTimes(workDays []WorkDay) ([]*TimeLogResult, error) {
	results := make([]*TimeLogResult, 0)

	if len(workDays) == 0 {
		return nil, fmt.Errorf("nenhum dia de trabalho fornecido para lançamento")
	}

	fmt.Printf("Iniciando lançamento de horas para %d dias\n", len(workDays))

	for _, dia := range workDays {
		fmt.Printf("Processando dia %s com %d entradas\n", dia.Date, len(dia.Entries))

		if len(dia.Entries) == 0 {
			fmt.Printf("Nenhuma entrada para o dia %s, pulando\n", dia.Date)
			continue
		}

		for _, alocacao := range dia.Entries {
			entrada := alocacao.Entry
			entrada.Date = dia.Date

			if alocacao.TaskID <= 0 {
				result := &TimeLogResult{
					Success: false,
					Message: fmt.Sprintf("ID de tarefa inválido: %d", alocacao.TaskID),
					Date:    dia.Date,
					TaskID:  alocacao.TaskID,
				}
				results = append(results, result)
				fmt.Printf("ID de tarefa inválido: %d para o dia %s\n", alocacao.TaskID, dia.Date)
				continue
			}

			fmt.Printf("Enviando lançamento: Tarefa #%d, Data: %s, Minutos: %d, Descrição: %s\n",
				alocacao.TaskID, entrada.Date, entrada.Minutes, entrada.Description)

			result, err := t.LogTime(alocacao.TaskID, entrada)

			if err != nil {
				fmt.Printf("Erro ao lançar horas: %v\n", err)
				if result == nil {
					result = &TimeLogResult{
						Success: false,
						Message: err.Error(),
						Date:    dia.Date,
						TaskID:  alocacao.TaskID,
					}
				}
			} else {
				fmt.Printf("Lançamento bem-sucedido para Tarefa #%d no dia %s\n", alocacao.TaskID, dia.Date)
			}

			results = append(results, result)

			time.Sleep(1 * time.Second)
		}
	}

	fmt.Printf("Lançamento de horas concluído. Total de resultados: %d\n", len(results))

	sucessos := 0
	for _, result := range results {
		if result.Success {
			sucessos++
		}
	}

	fmt.Printf("Lançamentos bem-sucedidos: %d, Falhas: %d\n", sucessos, len(results)-sucessos)

	if len(results) == 0 {
		return nil, fmt.Errorf("nenhum resultado de lançamento de horas")
	}

	return results, nil
}

func (t *TeamworkAPI) ObterDiasUteis(inicio, fim string) ([]string, error) {
	inicioDate, err := time.Parse("2006-01-02", inicio)
	if err != nil {
		return nil, fmt.Errorf("data inicial inválida: %v", err)
	}

	fimDate, err := time.Parse("2006-01-02", fim)
	if err != nil {
		return nil, fmt.Errorf("data final inválida: %v", err)
	}

	if fimDate.Before(inicioDate) {
		return nil, fmt.Errorf("a data final deve ser igual ou posterior à data inicial")
	}

	diasUteis := make([]string, 0)

	atual := inicioDate
	for !atual.After(fimDate) {
		if isDiaUtil(atual) {
			diasUteis = append(diasUteis, formatarData(atual))
		}
		atual = atual.AddDate(0, 0, 1)
	}

	if len(diasUteis) == 0 {
		return nil, fmt.Errorf("não foram encontrados dias úteis no período especificado")
	}

	return diasUteis, nil
}

func isDiaUtil(data time.Time) bool {
	diaSemana := data.Weekday()
	return diaSemana != time.Saturday && diaSemana != time.Sunday
}

func formatarData(data time.Time) string {
	return data.Format("2006-01-02")
}

func (t *TeamworkAPI) CriarPlanoDistribuicao(diasUteis []string, tarefas []Task) []WorkDay {
	planoDistribuicao := make([]WorkDay, 0, len(diasUteis))

	for _, dia := range diasUteis {
		workDay := WorkDay{
			Date:     dia,
			Entries:  []EntryTask{},
			TotalMin: 0,
		}

		for _, tarefa := range tarefas {
			for _, entrada := range tarefa.Entries {
				workDay.Entries = append(workDay.Entries, EntryTask{
					TaskID: tarefa.TaskID,
					Entry:  entrada,
				})
				workDay.TotalMin += entrada.Minutes
			}
		}

		planoDistribuicao = append(planoDistribuicao, workDay)
	}

	return planoDistribuicao
}

func (t *TeamworkAPI) CalcularTotalMinutos(tarefas []Task) int {
	total := 0
	for _, tarefa := range tarefas {
		for _, entrada := range tarefa.Entries {
			total += entrada.Minutes
		}
	}
	return total
}

func (t *TeamworkAPI) TestConnection(config Config) (bool, string) {
	tempAPI := &TeamworkAPI{
		Config: config,
	}

	if !tempAPI.IsConfigured() {
		return false, "API não configurada"
	}

	var url string
	if strings.HasPrefix(tempAPI.Config.ApiHost, "http://") || strings.HasPrefix(tempAPI.Config.ApiHost, "https://") {
		url = fmt.Sprintf("%s/projects/api/v3/me.json", tempAPI.Config.ApiHost)
	} else {
		url = fmt.Sprintf("https://%s/projects/api/v3/me.json", tempAPI.Config.ApiHost)
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false, fmt.Sprintf("Erro ao criar requisição: %v", err)
	}

	auth := base64.StdEncoding.EncodeToString([]byte(tempAPI.Config.AuthToken + ":X"))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Sprintf("Erro na requisição: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return false, fmt.Sprintf("Autenticação falhou: %d %s", resp.StatusCode, resp.Status)
	}

	return true, "Conexão estabelecida com sucesso!"
}

func (t *TeamworkAPI) GetCurrentUserId() (int, error) {
	if t.Config.AuthToken == "" || t.Config.ApiHost == "" {
		return 0, fmt.Errorf("API não configurada (falta token ou host)")
	}

	var url string
	if strings.HasPrefix(t.Config.ApiHost, "http://") || strings.HasPrefix(t.Config.ApiHost, "https://") {
		url = fmt.Sprintf("%s/projects/api/v3/me.json", t.Config.ApiHost)
	} else {
		url = fmt.Sprintf("https://%s/projects/api/v3/me.json", t.Config.ApiHost)
	}

	fmt.Printf("Consultando API do Teamwork em: %s\n", url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, fmt.Errorf("erro ao criar requisição: %v", err)
	}

	auth := base64.StdEncoding.EncodeToString([]byte(t.Config.AuthToken + ":X"))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("erro na requisição: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("erro ao ler resposta: %v", err)
	}

	fmt.Printf("Resposta da API (primeiros 500 caracteres): %s\n", string(body[:m(len(body), 500)]))

	if resp.StatusCode != 200 {
		return 0, fmt.Errorf("erro ao obter informações do usuário: %d %s - %s",
			resp.StatusCode, resp.Status, string(body))
	}

	var response struct {
		Person struct {
			ID int `json:"id"`
		} `json:"person"`
	}

	err = json.Unmarshal(body, &response)
	if err != nil || response.Person.ID == 0 {
		var responseV3 struct {
			User struct {
				ID int `json:"id"`
			} `json:"user"`
		}

		err = json.Unmarshal(body, &responseV3)
		if err != nil || responseV3.User.ID == 0 {
			var responseAlt map[string]interface{}
			err = json.Unmarshal(body, &responseAlt)
			if err != nil {
				return 0, fmt.Errorf("erro ao decodificar resposta: %v", err)
			}

			fmt.Printf("Estrutura da resposta: %+v\n", responseAlt)

			if userId, found := findUserIdInMap(responseAlt); found {
				return userId, nil
			}

			return 0, fmt.Errorf("não foi possível encontrar o ID do usuário na resposta")
		}

		return responseV3.User.ID, nil
	}

	return response.Person.ID, nil
}

func findUserIdInMap(data map[string]interface{}) (int, bool) {
	if id, ok := data["id"]; ok {
		if idInt, ok := id.(float64); ok {
			return int(idInt), true
		}
	}

	if person, ok := data["person"].(map[string]interface{}); ok {
		if id, ok := person["id"]; ok {
			if idInt, ok := id.(float64); ok {
				return int(idInt), true
			}
		}
	}

	if user, ok := data["user"].(map[string]interface{}); ok {
		if id, ok := user["id"]; ok {
			if idInt, ok := id.(float64); ok {
				return int(idInt), true
			}
		}
	}

	if me, ok := data["me"].(map[string]interface{}); ok {
		if id, ok := me["id"]; ok {
			if idInt, ok := id.(float64); ok {
				return int(idInt), true
			}
		}
	}

	return 0, false
}

func m(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (t *TeamworkAPI) GetProjects() ([]Project, error) {
	if !t.IsConfigured() {
		return nil, fmt.Errorf("API não configurada")
	}

	var url string
	if strings.HasPrefix(t.Config.ApiHost, "http://") || strings.HasPrefix(t.Config.ApiHost, "https://") {
		url = fmt.Sprintf("%s/projects/api/v3/projects.json?includeProjectUserInfo=true&include=tags,projectTaskStats,projectCategories,companies&projectStatuses=active",
			t.Config.ApiHost)
	} else {
		url = fmt.Sprintf("https://%s/projects/api/v3/projects.json?includeProjectUserInfo=true&include=tags,projectTaskStats,projectCategories,companies&projectStatuses=active",
			t.Config.ApiHost)
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar requisição: %v", err)
	}

	auth := base64.StdEncoding.EncodeToString([]byte(t.Config.AuthToken + ":X"))
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
		return nil, fmt.Errorf("erro ao obter projetos: %d %s - %s",
			resp.StatusCode, resp.Status, string(body))
	}

	var response ProjectsResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, fmt.Errorf("erro ao decodificar resposta: %v\nBody: %s", err, string(body))
	}

	return response.Projects, nil
}

func (t *TeamworkAPI) GetTasklistsByProject(projectID int) ([]TaskListItem, error) {
	var url string
	projectIDStr := strconv.Itoa(projectID)

	if strings.HasPrefix(t.Config.ApiHost, "http://") || strings.HasPrefix(t.Config.ApiHost, "https://") {
		url = fmt.Sprintf("%s/projects/api/v3/projects/%s/tasklists.json",
			t.Config.ApiHost, projectIDStr)
	} else {
		url = fmt.Sprintf("https://%s/projects/api/v3/projects/%s/tasklists.json",
			t.Config.ApiHost, projectIDStr)
	}

	fmt.Printf("Buscando listas de tarefas para o projeto %s: %s\n", projectIDStr, url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar requisição: %v", err)
	}

	auth := base64.StdEncoding.EncodeToString([]byte(t.Config.AuthToken + ":X"))
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
		return nil, fmt.Errorf("erro ao obter listas de tarefas: %d %s - %s",
			resp.StatusCode, resp.Status, string(body))
	}

	var response struct {
		Tasklists []TaskListItem `json:"tasklists"`
	}

	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, fmt.Errorf("erro ao decodificar resposta: %v", err)
	}

	return response.Tasklists, nil
}

func (t *TeamworkAPI) GetTasksByProject(projectID int) ([]TeamworkTask, error) {
	if !t.IsConfigured() {
		return nil, fmt.Errorf("API não configurada")
	}

	projectIDStr := strconv.Itoa(projectID)

	var url string
	if strings.HasPrefix(t.Config.ApiHost, "http://") || strings.HasPrefix(t.Config.ApiHost, "https://") {
		url = fmt.Sprintf("%s/projects/api/v3/projects/%s/tasks.json?include=projects,taskLists,users,companies,teams,timeTotals,tags,completedBy&includeCustomFields=true&includeLoggedTime=true",
			t.Config.ApiHost, projectIDStr)
	} else {
		url = fmt.Sprintf("https://%s/projects/api/v3/projects/%s/tasks.json?include=projects,taskLists,users,companies,teams,timeTotals,tags,completedBy&includeCustomFields=true&includeLoggedTime=true",
			t.Config.ApiHost, projectIDStr)
	}

	fmt.Printf("Buscando tarefas para o projeto %s: %s\n", projectIDStr, url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar requisição: %v", err)
	}

	auth := base64.StdEncoding.EncodeToString([]byte(t.Config.AuthToken + ":X"))
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
		fmt.Printf("Erro ao obter tarefas do projeto (API v3): %d %s - %s\n",
			resp.StatusCode, resp.Status, string(body))

		return t.getTasksByTasklists(projectID)
	}

	var responseV3 struct {
		Tasks []struct {
			ID          int    `json:"id"`
			Name        string `json:"name"`
			Description string `json:"description"`
			Status      string `json:"status"`
			ProjectID   int    `json:"projectId"`
			TasklistID  int    `json:"tasklistId"`
			CreatedAt   string `json:"createdAt"`
		} `json:"tasks"`
		Included struct {
			Projects map[string]struct {
				Name string `json:"name"`
			} `json:"projects"`
			Tasklists map[string]struct {
				Name string `json:"name"`
			} `json:"tasklists"`
		} `json:"included"`
	}

	err = json.Unmarshal(body, &responseV3)
	if err != nil {
		var response TasksResponse
		err = json.Unmarshal(body, &response)
		if err != nil {
			fmt.Printf("Erro ao decodificar resposta: %v\nTentando método alternativo...\n", err)
			return t.getTasksByTasklists(projectID)
		}

		enrichTasksWithIncludedData(&response)

		if len(response.Tasks) > 0 {
			t.enrichTasksWithDetails(&response.Tasks)
		}

		return response.Tasks, nil
	}

	var tasks []TeamworkTask
	projectName := ""

	projectIDKey := projectIDStr
	if proj, ok := responseV3.Included.Projects[projectIDKey]; ok {
		projectName = proj.Name
	} else {
		projects, _ := t.GetProjects()
		for _, p := range projects {
			if p.ID == projectID {
				projectName = p.Name
				break
			}
		}
	}

	for _, task := range responseV3.Tasks {
		tasklistName := ""
		tasklistIDStr := strconv.Itoa(task.TasklistID)

		if tlist, ok := responseV3.Included.Tasklists[tasklistIDStr]; ok {
			tasklistName = tlist.Name
		}

		tasks = append(tasks, TeamworkTask{
			ID:           task.ID,
			Content:      task.Name,
			Description:  task.Description,
			ProjectID:    projectID,
			ProjectName:  projectName,
			Status:       task.Status,
			CreatedAt:    task.CreatedAt,
			TasklistID:   task.TasklistID,
			TasklistName: tasklistName,
		})
	}

	return tasks, nil
}

func (t *TeamworkAPI) getTasksByTasklists(projectID int) ([]TeamworkTask, error) {
	fmt.Println("Tentando método alternativo: obter tarefas através das listas de tarefas")

	tasklists, err := t.GetTasklistsByProject(projectID)
	if err != nil {
		fmt.Printf("Erro ao obter listas de tarefas: %v\nTentando fallback para API v2...\n", err)
		return t.fallbackGetTasksByProject(projectID)
	}

	if len(tasklists) == 0 {
		fmt.Println("Nenhuma lista de tarefas encontrada. Tentando fallback para API v2...")
		return t.fallbackGetTasksByProject(projectID)
	}

	projectName := ""
	projects, _ := t.GetProjects()
	for _, p := range projects {
		if p.ID == projectID {
			projectName = p.Name
			break
		}
	}

	var allTasks []TeamworkTask

	for _, tasklist := range tasklists {
		fmt.Printf("Buscando tarefas da lista %d: %s\n", tasklist.ID, tasklist.Name)

		tasks, err := t.GetTasksByTasklist(tasklist.ID)
		if err != nil {
			fmt.Printf("Erro ao obter tarefas da lista %d: %v\n", tasklist.ID, err)
			continue
		}

		for i := range tasks {
			if tasks[i].ProjectID == 0 {
				tasks[i].ProjectID = projectID
			}
			if tasks[i].ProjectName == "" {
				tasks[i].ProjectName = projectName
			}
			if tasks[i].TasklistName == "" {
				tasks[i].TasklistName = tasklist.Name
			}
		}

		allTasks = append(allTasks, tasks...)
	}

	if len(allTasks) == 0 {
		fmt.Println("Nenhuma tarefa encontrada via listas. Tentando fallback para API v2...")
		return t.fallbackGetTasksByProject(projectID)
	}

	return allTasks, nil
}

func (t *TeamworkAPI) getProjectInfo(projectID int) []Project {
	projects, err := t.GetProjects()
	if err != nil {
		fmt.Printf("Erro ao obter informações do projeto: %v\n", err)
		return []Project{}
	}
	return projects
}

func (t *TeamworkAPI) GetTasksByTasklist(tasklistID int) ([]TeamworkTask, error) {
	var url string
	tasklistIDStr := strconv.Itoa(tasklistID)

	if strings.HasPrefix(t.Config.ApiHost, "http://") || strings.HasPrefix(t.Config.ApiHost, "https://") {
		url = fmt.Sprintf("%s/projects/api/v3/tasklists/%s/tasks.json?includeTaskDetails=true",
			t.Config.ApiHost, tasklistIDStr)
	} else {
		url = fmt.Sprintf("https://%s/projects/api/v3/tasklists/%s/tasks.json?includeTaskDetails=true",
			t.Config.ApiHost, tasklistIDStr)
	}

	fmt.Printf("Buscando tarefas da lista %s: %s\n", tasklistIDStr, url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar requisição: %v", err)
	}

	auth := base64.StdEncoding.EncodeToString([]byte(t.Config.AuthToken + ":X"))
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
		return nil, fmt.Errorf("erro ao obter tarefas da lista: %d %s - %s",
			resp.StatusCode, resp.Status, string(body))
	}

	var response TasksResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, fmt.Errorf("erro ao decodificar resposta: %v", err)
	}

	enrichTasksWithIncludedData(&response)

	return response.Tasks, nil
}

func (t *TeamworkAPI) fallbackGetTasksByProject(projectID int) ([]TeamworkTask, error) {
	fmt.Println("Tentando método alternativo (API v2) para obter tarefas...")

	var url string
	projectIDStr := strconv.Itoa(projectID)

	if strings.HasPrefix(t.Config.ApiHost, "http://") || strings.HasPrefix(t.Config.ApiHost, "https://") {
		url = fmt.Sprintf("%s/tasks.json?project_id=%s",
			t.Config.ApiHost, projectIDStr)
	} else {
		url = fmt.Sprintf("https://%s/tasks.json?project_id=%s",
			t.Config.ApiHost, projectIDStr)
	}

	fmt.Printf("Fazendo requisição alternativa para URL: %s\n", url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar requisição: %v", err)
	}

	auth := base64.StdEncoding.EncodeToString([]byte(t.Config.AuthToken + ":X"))
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
		return nil, fmt.Errorf("erro ao obter tarefas do projeto (modo alternativo): %d %s - %s",
			resp.StatusCode, resp.Status, string(body))
	}

	var responseV2 struct {
		TodoItems []struct {
			ID         int    `json:"id"`
			Content    string `json:"content"`
			ProjectID  int    `json:"project-id"`
			TodoListID int    `json:"todo-list-id"`
			Status     string `json:"status"`
		} `json:"todo-items"`
	}

	err = json.Unmarshal(body, &responseV2)
	if err != nil {
		return nil, fmt.Errorf("erro ao decodificar resposta v2: %v", err)
	}

	var tasks []TeamworkTask
	projectName := ""

	projects, _ := t.GetProjects()
	for _, proj := range projects {
		if proj.ID == projectID {
			projectName = proj.Name
			break
		}
	}

	for _, item := range responseV2.TodoItems {
		task := TeamworkTask{
			ID:          item.ID,
			Content:     item.Content,
			ProjectID:   item.ProjectID,
			ProjectName: projectName,
			Status:      item.Status,
			TasklistID:  item.TodoListID,
		}
		tasks = append(tasks, task)
	}

	return tasks, nil
}

func (t *TeamworkAPI) GetTokenWithCredentials(email, password, host string) (*LoginResponse, error) {
	if email == "" || password == "" || host == "" {
		return nil, fmt.Errorf("email, senha e host são obrigatórios")
	}

	baseURL := host
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		baseURL = "https://" + baseURL
	}

	// Tentamos usar /projects/api/v3/me.json para verificar a autenticação
	// e obter detalhes do usuário diretamente
	url := fmt.Sprintf("%s/projects/api/v3/me.json", baseURL)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar requisição: %v", err)
	}

	// Usar as credenciais fornecidas para autenticação básica
	auth := base64.StdEncoding.EncodeToString([]byte(email + ":" + password))
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

	// Verificar se a autenticação foi bem-sucedida
	if resp.StatusCode != 200 {
		return &LoginResponse{
			Success: false,
			Message: fmt.Sprintf("Erro na autenticação: %d %s - %s",
				resp.StatusCode, resp.Status, string(body)),
		}, nil
	}

	// Tentar extrair o ID do usuário da resposta
	var userID int
	var userInfo map[string]interface{}

	if err := json.Unmarshal(body, &userInfo); err != nil {
		return nil, fmt.Errorf("erro ao decodificar resposta: %v", err)
	}

	// Imprimir resposta para diagnóstico
	fmt.Printf("Resposta da API ME: %s\n", string(body))

	// Tentar encontrar o ID do usuário de diferentes maneiras
	if user, ok := userInfo["user"].(map[string]interface{}); ok {
		if id, ok := user["id"].(float64); ok {
			userID = int(id)
		}
	} else if me, ok := userInfo["me"].(map[string]interface{}); ok {
		if id, ok := me["id"].(float64); ok {
			userID = int(id)
		}
	} else if id, ok := userInfo["id"].(float64); ok {
		userID = int(id)
	}

	// Na API v3, podemos não receber um token explícito
	// Em vez disso, continuamos usando as credenciais para autenticação básica
	result := &LoginResponse{
		Success:    true,
		Token:      email + ":" + password, // Armazenamos a combinação de credenciais
		UserID:     userID,
		InstanceID: host,
		Message:    "Autenticação bem-sucedida",
	}

	return result, nil
}

func GetTokenWithCredentials(email, password, host string) (*LoginResponse, error) {
	tempAPI := &TeamworkAPI{}
	return tempAPI.GetTokenWithCredentials(email, password, host)
}
