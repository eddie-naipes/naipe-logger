package api

import (
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"
)

func (t *TeamworkAPI) GetTasks() ([]TeamworkTask, error) {
	cacheKey := fmt.Sprintf("tasks_user_%d", t.Config.UserID)
	if cachedData, found := t.cache.Get(cacheKey); found {
		return cachedData.([]TeamworkTask), nil
	}

	if !t.IsConfigured() {
		return nil, fmt.Errorf("API não configurada")
	}

	path := fmt.Sprintf("/projects/api/v3/tasks.json?assignedTo=%d&filter=active&includeTasklists=true&includeTaskAssignees=true&includeCompletionStatus=true&includeEstimatedTime=true&includeTaskTags=true",
		t.Config.UserID)

	url := t.buildURL(path)
	t.logDebug("Fazendo requisição para URL: %s", url)

	req, err := t.createRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, body, err := t.doRequest(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("erro ao obter tarefas: %d %s - %s",
			resp.StatusCode, resp.Status, string(body))
	}

	var response TasksResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("erro ao decodificar resposta: %v", err)
	}

	enrichTasksWithIncludedData(&response)

	if len(response.Tasks) > 0 {
		t.enrichTasksWithDetails(&response.Tasks)
	}

	t.cache.Set(cacheKey, response.Tasks, 15*time.Minute)
	return response.Tasks, nil
}

func (t *TeamworkAPI) GetTaskDetails(taskID int) (TeamworkTask, error) {
	if !t.IsConfigured() {
		return TeamworkTask{}, fmt.Errorf("API não configurada")
	}

	taskIDStr := strconv.Itoa(taskID)
	path := fmt.Sprintf("/projects/api/v3/tasks/%s.json?include=tags,assignees,time,project,tasklist", taskIDStr)
	url := t.buildURL(path)

	t.logDebug("Buscando detalhes da tarefa ID %d...", taskID)

	req, err := t.createRequest("GET", url, nil)
	if err != nil {
		return TeamworkTask{}, err
	}

	resp, body, err := t.doRequest(req)
	if err != nil {
		return TeamworkTask{}, err
	}

	if resp.StatusCode != 200 {
		return TeamworkTask{}, fmt.Errorf("erro ao obter detalhes da tarefa: %d %s - %s",
			resp.StatusCode, resp.Status, string(body))
	}

	result, err := parseTaskResponseV3(body, taskIDStr)
	if err == nil {
		return result, nil
	}

	return parseTaskResponseLegacy(body)
}

func parseTaskResponseV3(body []byte, taskIDStr string) (TeamworkTask, error) {
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

	err := json.Unmarshal(body, &taskResponseV3)
	if err != nil {
		return TeamworkTask{}, err
	}

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

	if timeLog, ok := taskResponseV3.Included.TimeTotals[taskIDStr]; ok {
		result.LoggedMinutes = timeLog.LoggedMinutes
	}

	return result, nil
}

func parseTaskResponseLegacy(body []byte) (TeamworkTask, error) {
	var taskWrapper struct {
		Task TeamworkTask `json:"task"`
	}

	err := json.Unmarshal(body, &taskWrapper)
	if err != nil {
		return TeamworkTask{}, fmt.Errorf("erro ao decodificar resposta: %v", err)
	}

	if taskWrapper.Task.Content == "" && taskWrapper.Task.Name != "" {
		taskWrapper.Task.Content = taskWrapper.Task.Name
	}

	return taskWrapper.Task, nil
}

func (t *TeamworkAPI) GetTaskCount() (int, error) {
	if !t.IsConfigured() {
		return 0, fmt.Errorf("API não configurada")
	}

	path := fmt.Sprintf("/projects/api/v3/tasks.json?assignedTo=%d&filter=active&page=1&pageSize=1",
		t.Config.UserID)
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
		return 0, fmt.Errorf("erro ao obter tarefas: %d %s", resp.StatusCode, resp.Status)
	}

	var response struct {
		Meta struct {
			Page struct {
				TotalItems int `json:"totalItems"`
			} `json:"page"`
		} `json:"meta"`
	}

	err = json.Unmarshal(body, &response)
	if err != nil {
		return 0, fmt.Errorf("erro ao decodificar resposta: %v", err)
	}

	return response.Meta.Page.TotalItems, nil
}

func (t *TeamworkAPI) GetTasksByProject(projectID int) ([]TeamworkTask, error) {
	cacheKey := fmt.Sprintf("tasks_project_%d", projectID)
	if cachedData, found := t.cache.Get(cacheKey); found {
		return cachedData.([]TeamworkTask), nil
	}

	if !t.IsConfigured() {
		return nil, fmt.Errorf("API não configurada")
	}

	projectIDStr := strconv.Itoa(projectID)
	path := fmt.Sprintf("/projects/api/v3/projects/%s/tasks.json?include=projects,taskLists,users,companies,teams,timeTotals,tags,completedBy&includeCustomFields=true&includeLoggedTime=true",
		projectIDStr)
	url := t.buildURL(path)

	t.logDebug("Buscando tarefas para o projeto %s: %s", projectIDStr, url)

	req, err := t.createRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, body, err := t.doRequest(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		t.logDebug("Erro ao obter tarefas do projeto (API v3): %d %s - %s",
			resp.StatusCode, resp.Status, string(body))
		tasks, err := t.getTasksByTasklists(projectID)
		if err == nil && len(tasks) > 0 {
			t.cache.Set(cacheKey, tasks, 15*time.Minute)
		}
		return tasks, err
	}

	tasks, err := parseProjectTasksV3(body, projectID, t)
	if err == nil {
		t.cache.Set(cacheKey, tasks, 15*time.Minute)
		return tasks, nil
	}

	var response TasksResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		t.logDebug("Erro ao decodificar resposta: %v\nTentando método alternativo...", err)
		tasks, err := t.getTasksByTasklists(projectID)
		if err == nil && len(tasks) > 0 {
			t.cache.Set(cacheKey, tasks, 15*time.Minute)
		}
		return tasks, err
	}

	enrichTasksWithIncludedData(&response)
	if len(response.Tasks) > 0 {
		t.enrichTasksWithDetails(&response.Tasks)
	}

	t.cache.Set(cacheKey, response.Tasks, 15*time.Minute)
	return response.Tasks, nil
}

func parseProjectTasksV3(body []byte, projectID int, t *TeamworkAPI) ([]TeamworkTask, error) {
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

	err := json.Unmarshal(body, &responseV3)
	if err != nil {
		return nil, err
	}

	var tasks []TeamworkTask
	projectName := ""
	projectIDStr := strconv.Itoa(projectID)

	if proj, ok := responseV3.Included.Projects[projectIDStr]; ok {
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

func (t *TeamworkAPI) enrichTasksWithDetails(tasks *[]TeamworkTask) {
	if len(*tasks) <= 1 {
		for i, task := range *tasks {
			t.enrichTaskDetail(&(*tasks)[i], task)
		}
		return
	}

	var wg sync.WaitGroup
	tasksCopy := *tasks

	semaphore := make(chan struct{}, 5)

	for i, task := range tasksCopy {
		if task.Content == "" || task.ProjectID == 0 || task.ProjectName == "" {
			wg.Add(1)
			go func(idx int, tsk TeamworkTask) {
				defer wg.Done()
				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				t.enrichTaskDetail(&(*tasks)[idx], tsk)
			}(i, task)
		}
	}

	wg.Wait()
}

func (t *TeamworkAPI) enrichTaskDetail(taskPtr *TeamworkTask, task TeamworkTask) {
	if task.Content == "" || task.ProjectID == 0 || task.ProjectName == "" {
		taskDetail, err := t.GetTaskDetails(task.ID)
		if err == nil {
			if task.Content == "" {
				if taskDetail.Name != "" {
					taskPtr.Content = taskDetail.Name
				} else if taskDetail.Content != "" {
					taskPtr.Content = taskDetail.Content
				}
			}
			if task.ProjectID == 0 && taskDetail.ProjectID != 0 {
				taskPtr.ProjectID = taskDetail.ProjectID
			}
			if task.ProjectName == "" && taskDetail.ProjectName != "" {
				taskPtr.ProjectName = taskDetail.ProjectName
			}
			if task.Description == "" && taskDetail.Description != "" {
				taskPtr.Description = taskDetail.Description
			}
			if task.TasklistName == "" && taskDetail.TasklistName != "" {
				taskPtr.TasklistName = taskDetail.TasklistName
			}
			if taskDetail.LoggedMinutes > 0 {
				taskPtr.LoggedMinutes = taskDetail.LoggedMinutes
			}
		}

		if taskPtr.Content == "" {
			projectInfo := ""
			if taskPtr.ProjectName != "" {
				projectInfo = fmt.Sprintf(" (%s)", taskPtr.ProjectName)
			}
			taskPtr.Content = fmt.Sprintf("Tarefa #%d%s", task.ID, projectInfo)
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

func (t *TeamworkAPI) getTasksByTasklists(projectID int) ([]TeamworkTask, error) {
	t.logDebug("Tentando método alternativo: obter tarefas através das listas de tarefas")

	tasklists, err := t.GetTasklistsByProject(projectID)
	if err != nil {
		t.logDebug("Erro ao obter listas de tarefas: %v\nTentando fallback para API v2...", err)
		return t.fallbackGetTasksByProject(projectID)
	}

	if len(tasklists) == 0 {
		t.logDebug("Nenhuma lista de tarefas encontrada. Tentando fallback para API v2...")
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
		t.logDebug("Buscando tarefas da lista %d: %s", tasklist.ID, tasklist.Name)

		tasks, err := t.GetTasksByTasklist(tasklist.ID)
		if err != nil {
			t.logDebug("Erro ao obter tarefas da lista %d: %v", tasklist.ID, err)
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
		t.logDebug("Nenhuma tarefa encontrada via listas. Tentando fallback para API v2...")
		return t.fallbackGetTasksByProject(projectID)
	}

	return allTasks, nil
}

func (t *TeamworkAPI) GetTasklistsByProject(projectID int) ([]TaskListItem, error) {
	projectIDStr := strconv.Itoa(projectID)
	path := fmt.Sprintf("/projects/api/v3/projects/%s/tasklists.json", projectIDStr)
	url := t.buildURL(path)

	t.logDebug("Buscando listas de tarefas para o projeto %s: %s", projectIDStr, url)

	req, err := t.createRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, body, err := t.doRequest(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("erro ao obter listas de tarefas: %d %s - %s",
			resp.StatusCode, resp.Status, string(body))
	}

	var response struct {
		Tasklists []TaskListItem `json:"tasklists"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("erro ao decodificar resposta: %v", err)
	}

	return response.Tasklists, nil
}

func (t *TeamworkAPI) GetTasksByTasklist(tasklistID int) ([]TeamworkTask, error) {
	tasklistIDStr := strconv.Itoa(tasklistID)
	path := fmt.Sprintf("/projects/api/v3/tasklists/%s/tasks.json?includeTaskDetails=true", tasklistIDStr)
	url := t.buildURL(path)

	t.logDebug("Buscando tarefas da lista %s: %s", tasklistIDStr, url)

	req, err := t.createRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, body, err := t.doRequest(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("erro ao obter tarefas da lista: %d %s - %s",
			resp.StatusCode, resp.Status, string(body))
	}

	var response TasksResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("erro ao decodificar resposta: %v", err)
	}

	enrichTasksWithIncludedData(&response)
	return response.Tasks, nil
}

func (t *TeamworkAPI) fallbackGetTasksByProject(projectID int) ([]TeamworkTask, error) {
	t.logDebug("Tentando método alternativo (API v2) para obter tarefas...")

	projectIDStr := strconv.Itoa(projectID)
	path := fmt.Sprintf("/tasks.json?project_id=%s", projectIDStr)
	url := t.buildURL(path)

	t.logDebug("Fazendo requisição alternativa para URL: %s", url)

	req, err := t.createRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, body, err := t.doRequest(req)
	if err != nil {
		return nil, err
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

	if err := json.Unmarshal(body, &responseV2); err != nil {
		return nil, fmt.Errorf("erro ao decodificar resposta v2: %v", err)
	}

	projectName := ""
	projects, _ := t.GetProjects()
	for _, proj := range projects {
		if proj.ID == projectID {
			projectName = proj.Name
			break
		}
	}

	var tasks []TeamworkTask
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

func (t *TeamworkAPI) GetTasksWithUpcomingDeadlines() ([]map[string]interface{}, error) {
	now := time.Now()
	proximos7Dias := now.AddDate(0, 0, 7).Format("2006-01-02")

	path := fmt.Sprintf("/projects/api/v3/tasks.json?assignedToUserIds=%d&filter=active&dueBefore=%s&include=project",
		t.Config.UserID, proximos7Dias)
	url := t.buildURL(path)

	req, err := t.createRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, body, err := t.doRequest(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("erro ao obter tarefas com vencimento próximo: %d", resp.StatusCode)
	}

	var responseData struct {
		Tasks []struct {
			ID        int    `json:"id"`
			Content   string `json:"content"`
			Name      string `json:"name"`
			DueDate   string `json:"dueDate"`
			Priority  string `json:"priority"`
			ProjectID int    `json:"projectId"`
		} `json:"tasks"`
		Included struct {
			Projects map[string]struct {
				Name string `json:"name"`
			} `json:"projects"`
		} `json:"included"`
	}

	if err := json.Unmarshal(body, &responseData); err != nil {
		return nil, err
	}

	tarefas := make([]map[string]interface{}, 0)

	for _, task := range responseData.Tasks {
		nome := task.Name
		if nome == "" {
			nome = task.Content
		}

		projectName := ""
		projectIDStr := strconv.Itoa(task.ProjectID)
		if project, ok := responseData.Included.Projects[projectIDStr]; ok {
			projectName = project.Name
		}

		priority := "Normal"
		if task.Priority != "" {
			priority = task.Priority
		}

		tarefaInfo := map[string]interface{}{
			"id":          task.ID,
			"name":        nome,
			"dueDate":     task.DueDate,
			"priority":    priority,
			"projectId":   task.ProjectID,
			"projectName": projectName,
		}

		tarefas = append(tarefas, tarefaInfo)
	}

	return tarefas, nil
}

func (t *TeamworkAPI) GetCompletedTasksByProject(projectID int) (int, error) {
	projectIDStr := strconv.Itoa(projectID)
	path := fmt.Sprintf("/projects/api/v3/tasks.json?projectIds=%s&completedStatus=completed", projectIDStr)
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
		return 0, fmt.Errorf("erro ao obter tarefas concluídas: %d", resp.StatusCode)
	}

	var responseData struct {
		Tasks []struct {
			ID int `json:"id"`
		} `json:"tasks"`
	}

	if err := json.Unmarshal(body, &responseData); err != nil {
		return 0, err
	}

	return len(responseData.Tasks), nil
}

func (t *TeamworkAPI) GetCompletedTasks(startDate, endDate string) (int, error) {
	path := fmt.Sprintf("/projects/api/v3/tasks.json?completedStatus=completed&updatedAfterDate=%s&updatedBeforeDate=%s",
		startDate, endDate)
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
		return 0, fmt.Errorf("erro ao obter tarefas concluídas: %d", resp.StatusCode)
	}

	var responseData struct {
		Tasks []struct {
			ID int `json:"id"`
		} `json:"tasks"`
	}

	if err := json.Unmarshal(body, &responseData); err != nil {
		return 0, err
	}

	return len(responseData.Tasks), nil
}
