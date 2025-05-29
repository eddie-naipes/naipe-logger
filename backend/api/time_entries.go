package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

func (t *TeamworkAPI) GetEntriesFromLoggedTime(month, year int) ([]map[string]interface{}, error) {
	response, err := t.GetLoggedTimeFromCalendarAPI(month, year)
	if err != nil {
		return nil, fmt.Errorf("erro ao obter dados de tempo do calendário: %v", err)
	}

	if response.STATUS != "OK" || (len(response.User.Billable) == 0 && len(response.User.Nonbillable) == 0) {
		return nil, fmt.Errorf("nenhuma entrada de tempo válida encontrada")
	}

	entries := make([]map[string]interface{}, 0)

	// Processar entradas billable
	if response.User.Billable != nil {
		for _, entry := range response.User.Billable {
			if len(entry) < 3 {
				continue
			}

			timestamp, err := strconv.ParseInt(entry[0], 10, 64)
			if err != nil {
				continue
			}

			// Convertendo milissegundos para segundos
			date := time.Unix(timestamp/1000, 0)
			dateStr := date.Format("2006-01-02")

			hours, _ := strconv.ParseFloat(entry[1], 64)
			minutes, _ := strconv.ParseInt(entry[2], 10, 64)

			// Criar uma entrada para o dia
			entryData := map[string]interface{}{
				"date":        dateStr,
				"minutes":     minutes,
				"hours":       hours,
				"description": "Tempo registrado (cobrável)",
				"projectName": "Teamwork",
				"isBillable":  true,
				"timestamp":   timestamp,
				"type":        "billable",
			}

			entries = append(entries, entryData)
		}
	}

	// Processar entradas não billable
	if response.User.Nonbillable != nil {
		for _, entry := range response.User.Nonbillable {
			if len(entry) < 3 {
				continue
			}

			timestamp, err := strconv.ParseInt(entry[0], 10, 64)
			if err != nil {
				continue
			}

			// Convertendo milissegundos para segundos
			date := time.Unix(timestamp/1000, 0)
			dateStr := date.Format("2006-01-02")

			hours, _ := strconv.ParseFloat(entry[1], 64)
			minutes, _ := strconv.ParseInt(entry[2], 10, 64)

			// Só adiciona se tiver minutos
			if minutes > 0 {
				// Criar uma entrada para o dia
				entryData := map[string]interface{}{
					"date":        dateStr,
					"minutes":     minutes,
					"hours":       hours,
					"description": "Tempo registrado (não cobrável)",
					"projectName": "Teamwork",
					"isBillable":  false,
					"timestamp":   timestamp,
					"type":        "nonbillable",
				}

				entries = append(entries, entryData)
			}
		}
	}

	sort.Slice(entries, func(i, j int) bool {
		date1, _ := time.Parse("2006-01-02", entries[i]["date"].(string))
		date2, _ := time.Parse("2006-01-02", entries[j]["date"].(string))
		return date1.After(date2)
	})

	return entries, nil
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
	path := fmt.Sprintf("/projects/api/v3/tasks/%s/time.json", taskIDStr)
	url := t.buildURL(path)

	if t.Config.UserID <= 0 {
		return nil, fmt.Errorf("ID do usuário não configurado")
	}

	entry.UserID = t.Config.UserID

	t.logDebug("Lançando tempo para tarefa #%d: %s %s - %d minutos - %s",
		taskID, entry.Date, entry.Time, entry.Minutes, entry.Description)

	reqBody := TimelogRequest{
		Timelog: entry,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("erro ao converter para JSON: %v", err)
	}

	t.logDebug("JSON do lançamento: %s", string(jsonData))

	req, err := t.createRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	resp, body, err := t.doRequest(req)
	if err != nil {
		return nil, err
	}

	t.logDebug("Resposta do servidor (%d): %s", resp.StatusCode, string(body))

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

func (t *TeamworkAPI) CreateDistributionPlanFromLoggedTime(month, year int, tasks []Task) ([]WorkDay, error) {
	entries, err := t.GetEntriesFromLoggedTime(month, year)
	if err != nil {
		return nil, err
	}

	// Agrupar entradas por dia
	entriesByDay := make(map[string][]map[string]interface{})
	for _, entry := range entries {
		date := entry["date"].(string)
		if _, ok := entriesByDay[date]; !ok {
			entriesByDay[date] = make([]map[string]interface{}, 0)
		}
		entriesByDay[date] = append(entriesByDay[date], entry)
	}

	// Criar dias de trabalho
	workDays := make([]WorkDay, 0, len(entriesByDay))

	for date, dayEntries := range entriesByDay {
		workDay := WorkDay{
			Date:     date,
			Entries:  []EntryTask{},
			TotalMin: 0,
		}

		// Total de minutos para o dia
		totalMin := 0
		for _, entry := range dayEntries {
			mins := int(entry["minutes"].(int64))
			totalMin += mins
		}

		// Distribuir os minutos entre as tarefas
		if len(tasks) > 0 {
			// Se temos tarefas, distribuímos entre elas
			minsPerTask := totalMin / len(tasks)
			remainingMins := totalMin % len(tasks)

			for _, task := range tasks {
				taskMins := minsPerTask
				if remainingMins > 0 {
					taskMins++
					remainingMins--
				}

				if taskMins <= 0 {
					continue
				}

				// Criar entrada para a tarefa
				workDay.Entries = append(workDay.Entries, EntryTask{
					TaskID: task.TaskID,
					Entry: TimeEntry{
						Minutes:     taskMins,
						Description: task.TaskName,
						IsBillable:  true,
						Time:        "09:00",
						Date:        date,
					},
				})
			}
		} else {
			// Se não temos tarefas, criamos uma entrada genérica
			workDay.Entries = append(workDay.Entries, EntryTask{
				TaskID: 0, // TaskID 0 indica que precisamos selecionar uma tarefa
				Entry: TimeEntry{
					Minutes:     totalMin,
					Description: "Tempo importado do calendário",
					IsBillable:  true,
					Time:        "09:00",
					Date:        date,
				},
			})
		}

		workDay.TotalMin = totalMin
		workDays = append(workDays, workDay)
	}

	sort.Slice(workDays, func(i, j int) bool {
		return workDays[i].Date < workDays[j].Date
	})

	return workDays, nil
}

func (t *TeamworkAPI) LogMultipleTimes(workDays []WorkDay) ([]*TimeLogResult, error) {
	if len(workDays) == 0 {
		return nil, fmt.Errorf("nenhum dia de trabalho fornecido para lançamento")
	}

	t.logDebug("Iniciando lançamento de horas para %d dias", len(workDays))

	totalEntries := 0
	for _, day := range workDays {
		totalEntries += len(day.Entries)
	}

	results := make([]*TimeLogResult, 0, totalEntries)
	resultChan := make(chan *TimeLogResult, totalEntries)
	errorChan := make(chan error, totalEntries)

	var wg sync.WaitGroup
	// Limitar processamento paralelo a 3 para não sobrecarregar a API
	semaphore := make(chan struct{}, 3)

	for _, dia := range workDays {
		if len(dia.Entries) == 0 {
			continue
		}

		for _, alocacao := range dia.Entries {
			wg.Add(1)
			go func(d string, a EntryTask) {
				defer wg.Done()
				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				entrada := a.Entry
				entrada.Date = d

				if a.TaskID <= 0 {
					resultChan <- &TimeLogResult{
						Success: false,
						Message: fmt.Sprintf("ID de tarefa inválido: %d", a.TaskID),
						Date:    d,
						TaskID:  a.TaskID,
					}
					return
				}

				result, err := t.LogTime(a.TaskID, entrada)
				if err != nil {
					if result == nil {
						resultChan <- &TimeLogResult{
							Success: false,
							Message: err.Error(),
							Date:    d,
							TaskID:  a.TaskID,
						}
					} else {
						resultChan <- result
					}
					errorChan <- err
				} else {
					resultChan <- result
				}

				// Pequena pausa para não sobrecarregar a API
				time.Sleep(500 * time.Millisecond)
			}(dia.Date, alocacao)
		}
	}

	go func() {
		wg.Wait()
		close(resultChan)
		close(errorChan)
	}()

	for result := range resultChan {
		results = append(results, result)
	}

	if len(results) == 0 {
		if len(errorChan) > 0 {
			return nil, <-errorChan
		}
		return nil, fmt.Errorf("nenhum resultado de lançamento de horas")
	}

	return results, nil
}

func (t *TeamworkAPI) GetWorkingDays(inicio, fim string) ([]string, error) {
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
		if t.IsWorkDay(atual) {
			diasUteis = append(diasUteis, formatDate(atual))
		}
		atual = atual.AddDate(0, 0, 1)
	}

	if len(diasUteis) == 0 {
		return nil, fmt.Errorf("não foram encontrados dias úteis no período especificado")
	}

	return diasUteis, nil
}

func (t *TeamworkAPI) IsWorkDay(data time.Time) bool {
	diaSemana := data.Weekday()

	if diaSemana == time.Saturday || diaSemana == time.Sunday {
		return false
	}

	isHoliday, _, _ := t.IsHoliday(data)
	return !isHoliday
}

func formatDate(data time.Time) string {
	return data.Format("2006-01-02")
}

func (t *TeamworkAPI) CreateDistributionPlan(diasUteis []string, tarefas []Task) []WorkDay {
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

func (t *TeamworkAPI) CalculateTotalMinutes(tarefas []Task) int {
	total := 0
	for _, tarefa := range tarefas {
		for _, entrada := range tarefa.Entries {
			total += entrada.Minutes
		}
	}
	return total
}

func (t *TeamworkAPI) GetHoursLoggedInPeriod(startDate, endDate string) (float64, error) {
	userID := strconv.Itoa(t.Config.UserID)
	path := fmt.Sprintf("/projects/api/v3/time.json?userId=%s&fromDate=%s&toDate=%s",
		userID, startDate, endDate)
	url := t.buildURL(path)

	req, err := t.createRequest("GET", url, nil)
	if err != nil {
		return 0, err
	}

	resp, body, err := t.doRequest(req)
	if err != nil {
		return 0, err
	}

	if resp.StatusCode != 200 {
		return 0, fmt.Errorf("erro ao obter registros de tempo: %d %s", resp.StatusCode, resp.Status)
	}

	var response struct {
		TimeEntries []struct {
			Minutes float64 `json:"minutes"`
		} `json:"timeEntries"`
	}

	err = json.Unmarshal(body, &response)
	if err != nil {
		return t.GetHoursLoggedInPeriodLegacy(startDate, endDate)
	}

	totalMinutos := 0.0
	for _, entry := range response.TimeEntries {
		totalMinutos += entry.Minutes
	}

	return totalMinutos / 60.0, nil
}

func (t *TeamworkAPI) GetHoursLoggedInPeriodLegacy(startDate, endDate string) (float64, error) {
	userID := strconv.Itoa(t.Config.UserID)
	path := fmt.Sprintf("/time/total.json?userId=%s&fromDate=%s&toDate=%s",
		userID, startDate, endDate)
	url := t.buildURL(path)

	req, err := t.createRequest("GET", url, nil)
	if err != nil {
		return 0, err
	}

	resp, body, err := t.doRequest(req)
	if err != nil {
		return 0, err
	}

	if resp.StatusCode != 200 {
		return 0, fmt.Errorf("erro ao obter registros de tempo legado: %d %s", resp.StatusCode, resp.Status)
	}

	var rawResponse map[string]interface{}
	if err := json.Unmarshal(body, &rawResponse); err != nil {
		return 0, err
	}

	totalMinutos := 0.0

	if timeEntriesRaw, ok := rawResponse["time-entries"]; ok {
		if timeEntriesArr, ok := timeEntriesRaw.([]interface{}); ok {
			for _, entryRaw := range timeEntriesArr {
				if entry, ok := entryRaw.(map[string]interface{}); ok {
					if mins, ok := entry["minutes"].(float64); ok {
						totalMinutos += mins
					}
				}
			}
		}
	}

	return totalMinutos / 60.0, nil
}

func (t *TeamworkAPI) GetTimeLogsForPeriod(startDate, endDate string) ([]map[string]interface{}, float64, map[string]interface{}, error) {
	userID := strconv.Itoa(t.Config.UserID)
	path := fmt.Sprintf("/time/total.json?userId=%s&fromDate=%s&toDate=%s&includeTaskInfo=true",
		userID, startDate, endDate)
	url := t.buildURL(path)

	t.logDebug("Obtendo registros de tempo: %s", url)

	req, err := t.createRequest("GET", url, nil)
	if err != nil {
		return nil, 0, nil, err
	}

	resp, body, err := t.doRequest(req)
	if err != nil {
		return nil, 0, nil, err
	}

	if resp.StatusCode != 200 {
		return nil, 0, nil, fmt.Errorf("erro ao obter registros de tempo: %d %s - %s",
			resp.StatusCode, resp.Status, string(body))
	}

	var rawResponse map[string]interface{}
	if err := json.Unmarshal(body, &rawResponse); err != nil {
		return nil, 0, nil, err
	}

	var entries []map[string]interface{}
	totalMinutos := 0.0

	if timeEntriesRaw, ok := rawResponse["time-entries"]; ok {
		if timeEntriesArr, ok := timeEntriesRaw.([]interface{}); ok {
			for _, entryRaw := range timeEntriesArr {
				if entry, ok := entryRaw.(map[string]interface{}); ok {
					entries = append(entries, entry)

					if mins, ok := entry["minutes"].(float64); ok {
						totalMinutos += mins
					}
				}
			}
		}
	}

	totalHoras := totalMinutos / 60.0

	var ultimoLancamento map[string]interface{}
	if len(entries) > 0 {
		ultimoLancamento = entries[0]

		for _, entry := range entries {
			if dataAtual, ok := entry["date"].(string); ok {
				if dataUltimo, ok := ultimoLancamento["date"].(string); ok {
					if dataAtual > dataUltimo {
						ultimoLancamento = entry
					}
				}
			}
		}
	}

	return entries, totalHoras, ultimoLancamento, nil
}

func (t *TeamworkAPI) GetRecentActivities() ([]map[string]interface{}, error) {

	cacheKey := "recent_activities"
	if cachedData, found := t.cache.Get(cacheKey); found {
		return cachedData.([]map[string]interface{}), nil
	}

	projects, err := t.GetProjects()
	if err != nil || len(projects) == 0 {
		return []map[string]interface{}{}, nil
	}

	projectID := projects[0].ID

	tasks, err := t.GetTasksByProject(projectID)
	if err != nil || len(tasks) == 0 {
		return []map[string]interface{}{}, nil
	}

	atividades := make([]map[string]interface{}, 0)

	sort.Slice(tasks, func(i, j int) bool {
		time1, _ := time.Parse(time.RFC3339, tasks[i].CreatedAt)
		time2, _ := time.Parse(time.RFC3339, tasks[j].CreatedAt)
		return time1.After(time2)
	})

	limit := 5
	if len(tasks) < limit {
		limit = len(tasks)
	}

	now := time.Now()

	for i := 0; i < limit; i++ {
		task := tasks[i]

		atividadeInfo := map[string]interface{}{
			"id":          task.ID,
			"type":        "task",
			"description": "Tarefa atualizada: " + task.Content,
			"minutes":     60.0,
			"date":        now.AddDate(0, 0, -i).Format("2006-01-02"),
			"projectId":   task.ProjectID,
			"projectName": task.ProjectName,
			"taskId":      task.ID,
			"taskName":    task.Content,
		}

		atividades = append(atividades, atividadeInfo)
	}

	t.cache.Set(cacheKey, atividades, 30*time.Minute)
	return atividades, nil
}

func (t *TeamworkAPI) GetAllNonWorkingDays(year, month int) ([]map[string]interface{}, error) {
	startDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)

	var endDate time.Time
	if month == 12 {
		endDate = time.Date(year+1, 1, 0, 0, 0, 0, 0, time.Local)
	} else {
		endDate = time.Date(year, time.Month(month+1), 0, 0, 0, 0, 0, time.Local)
	}

	holidays, err := t.GetHolidaysForMonth(year, month)
	if err != nil {
		return nil, err
	}

	nonWorkingDays := make([]map[string]interface{}, 0)

	current := startDate
	for !current.After(endDate) {
		if current.Weekday() == time.Saturday || current.Weekday() == time.Sunday {
			nonWorkingDays = append(nonWorkingDays, map[string]interface{}{
				"date": formatDate(current),
				"type": "weekend",
				"name": current.Weekday().String(),
			})
		}
		current = current.AddDate(0, 0, 1)
	}

	for _, holiday := range holidays {
		date, _ := time.Parse("2006-01-02", holiday.Date)
		if date.Weekday() != time.Saturday && date.Weekday() != time.Sunday {
			nonWorkingDays = append(nonWorkingDays, map[string]interface{}{
				"date":        holiday.Date,
				"type":        "holiday",
				"name":        holiday.Name,
				"description": holiday.Description,
				"isOptional":  holiday.IsOptional,
			})
		}
	}

	return nonWorkingDays, nil
}
