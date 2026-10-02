package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// taskIncludes pede, junto com as tarefas, as listas e os projetos a que elas
// pertencem. A v3 não traz projectId/nome do projeto na tarefa: o caminho é
// tarefa -> tasklistId -> included.tasklists[id].projectId ->
// included.projects[id].name. Sem isso toda tarefa parecia "incompleta" e
// GetTasks fazia uma requisição de detalhe por tarefa (N+1).
const taskIncludes = "projects,tasklists"

func (t *TeamworkAPI) GetTasks() ([]TeamworkTask, error) {
	cacheKey := fmt.Sprintf("tasks_user_%d", t.Config.UserID)
	if cached, found := getCached[[]TeamworkTask](t.cache, cacheKey); found {
		return cached, nil
	}

	if !t.IsConfigured() {
		return nil, fmt.Errorf("API não configurada")
	}

	path := fmt.Sprintf("/projects/api/v3/tasks.json?assignedTo=%d&filter=active&include=%s&includeTasklists=true&includeTaskAssignees=true&includeCompletionStatus=true&includeEstimatedTime=true&includeTaskTags=true",
		t.Config.UserID, taskIncludes)

	slog.Debug("Buscando tarefas do usuário", "usuario", t.Config.UserID)

	tasks, err := t.fetchTaskPages(t.buildURL(path), "tarefas")
	if err != nil {
		return nil, err
	}

	t.enrichTasksWithDetails(&tasks)

	t.cache.Set(cacheKey, tasks, 15*time.Minute)
	return tasks, nil
}

// fetchTaskPages percorre um endpoint v3 de tarefas e devolve as tarefas já
// resolvidas com os dados de `included`.
func (t *TeamworkAPI) fetchTaskPages(baseURL, what string) ([]TeamworkTask, error) {
	tasks := make([]TeamworkTask, 0)

	err := t.fetchPages(baseURL, listPageSize, maxListPages, what, func(body []byte) (pageInfo, error) {
		var page TasksResponse
		if err := json.Unmarshal(body, &page); err != nil {
			return pageInfo{}, err
		}
		tasks = append(tasks, page.resolvedTasks()...)
		return pageInfo{items: len(page.Tasks), hasMore: page.Meta.Page.HasMore}, nil
	})
	if err != nil {
		return nil, err
	}

	return tasks, nil
}

// resolvedTasks preenche lista, projeto e nome de cada tarefa a partir do
// bloco `included` da própria resposta.
func (r *TasksResponse) resolvedTasks() []TeamworkTask {
	tasks := make([]TeamworkTask, 0, len(r.Tasks))

	for _, wire := range r.Tasks {
		task := wire.TeamworkTask

		if task.TasklistID == 0 && wire.Tasklist != nil {
			task.TasklistID = wire.Tasklist.ID
		}
		if task.ProjectID == 0 && wire.Project != nil {
			task.ProjectID = wire.Project.ID
		}

		if task.TasklistID > 0 {
			if tasklist, ok := r.Included.TaskLists[strconv.Itoa(task.TasklistID)]; ok {
				task.TasklistName = tasklist.Name
				if task.ProjectID == 0 {
					task.ProjectID = tasklist.ProjectID
				}
			}
		}

		if task.ProjectID > 0 && task.ProjectName == "" {
			if project, ok := r.Included.Projects[strconv.Itoa(task.ProjectID)]; ok {
				task.ProjectName = project.Name
			}
		}

		task.Content = taskDisplayName(task)
		tasks = append(tasks, task)
	}

	return tasks
}

// taskDisplayName escolhe o nome exibido da tarefa: Content, depois Name.
// O nome da lista de tarefas NÃO serve de fallback — antes ele era usado
// quando Content vinha vazio (o normal na v3, que manda "name"), e todas as
// tarefas da mesma lista apareciam com o mesmo título.
func taskDisplayName(task TeamworkTask) string {
	if task.Content != "" && !isPlaceholderName(task.Content) {
		return task.Content
	}
	if task.Name != "" {
		return task.Name
	}
	if task.Content != "" {
		return task.Content
	}

	projectInfo := ""
	if task.ProjectName != "" {
		projectInfo = fmt.Sprintf(" (%s)", task.ProjectName)
	}
	return fmt.Sprintf("Tarefa #%d%s", task.ID, projectInfo)
}

func isPlaceholderName(name string) bool {
	return strings.HasPrefix(name, "Tarefa #")
}

func (t *TeamworkAPI) GetTaskDetails(taskID int) (TeamworkTask, error) {
	if !t.IsConfigured() {
		return TeamworkTask{}, fmt.Errorf("API não configurada")
	}

	taskIDStr := strconv.Itoa(taskID)
	path := fmt.Sprintf("/projects/api/v3/tasks/%s.json?include=projects,tasklists,timeTotals,tags", taskIDStr)
	url := t.buildURL(path)

	slog.Debug("Buscando detalhes da tarefa", "tarefa", taskID)

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
				Name      string `json:"name"`
				ProjectID int    `json:"projectId"`
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

	tasklistIDStr := strconv.Itoa(taskResponseV3.Task.TasklistID)
	if tlist, ok := taskResponseV3.Included.Tasklists[tasklistIDStr]; ok {
		result.TasklistName = tlist.Name
		// A tarefa v3 não traz projectId; ele vem da lista.
		if result.ProjectID == 0 {
			result.ProjectID = tlist.ProjectID
		}
	}

	projectIDStr := strconv.Itoa(result.ProjectID)
	if proj, ok := taskResponseV3.Included.Projects[projectIDStr]; ok {
		result.ProjectName = proj.Name
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
	if cached, found := getCached[[]TeamworkTask](t.cache, cacheKey); found {
		return cached, nil
	}

	if !t.IsConfigured() {
		return nil, fmt.Errorf("API não configurada")
	}

	path := fmt.Sprintf("/projects/api/v3/projects/%d/tasks.json?include=%s,users,companies,teams,timeTotals,tags,completedBy&includeCustomFields=true&includeLoggedTime=true",
		projectID, taskIncludes)

	slog.Debug("Buscando tarefas do projeto", "projeto", projectID)

	tasks, err := t.fetchTaskPages(t.buildURL(path), "tarefas do projeto")
	if err != nil {
		// Caminho alternativo para contas em que o endpoint por projeto falha:
		// monta a lista a partir das listas de tarefas (e, por fim, da v1).
		slog.Debug("Erro ao obter tarefas do projeto pela API v3", "projeto", projectID, "err", err)
		tasks, err = t.getTasksByTasklists(projectID)
		if err != nil {
			return nil, err
		}
	}

	if len(tasks) > 0 {
		t.enrichTasksWithProjectContext(&tasks, projectID)
	}

	t.cache.Set(cacheKey, tasks, 5*time.Minute)
	return tasks, nil
}

func (t *TeamworkAPI) enrichTasksWithProjectContext(tasks *[]TeamworkTask, expectedProjectID int) {
	if tasks == nil || len(*tasks) == 0 {
		return
	}

	projectName := ""
	needsProjectName := false
	for _, task := range *tasks {
		if task.ProjectName == "" {
			needsProjectName = true
			break
		}
	}

	if expectedProjectID > 0 && needsProjectName {
		projects, err := t.GetProjects()
		if err == nil {
			for _, p := range projects {
				if p.ID == expectedProjectID {
					projectName = p.Name
					break
				}
			}
		}
	}

	for i := range *tasks {
		task := &(*tasks)[i]

		if task.ProjectID == 0 && expectedProjectID > 0 {
			task.ProjectID = expectedProjectID
		}

		if task.ProjectName == "" && projectName != "" {
			task.ProjectName = projectName
		}

		task.Content = taskDisplayName(*task)
	}

	t.enrichTasksWithDetails(tasks)
}

// taskNeedsDetail diz se a tarefa ainda está sem nome real ou sem projeto
// depois de aproveitar o `included`. Só essas justificam uma requisição de
// detalhe.
func taskNeedsDetail(task TeamworkTask) bool {
	return task.Content == "" ||
		isPlaceholderName(task.Content) ||
		task.ProjectID == 0 ||
		task.ProjectName == ""
}

// enrichTasksWithDetails busca o detalhe só das tarefas que realmente
// precisam, com no máximo 5 requisições simultâneas.
func (t *TeamworkAPI) enrichTasksWithDetails(tasks *[]TeamworkTask) {
	if tasks == nil || len(*tasks) == 0 {
		return
	}

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 5)

	for i := range *tasks {
		if !taskNeedsDetail((*tasks)[i]) {
			continue
		}

		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			t.enrichTaskDetail(&(*tasks)[idx])
		}(i)
	}

	wg.Wait()
}

func (t *TeamworkAPI) enrichTaskDetail(taskPtr *TeamworkTask) {
	// O nome da própria tarefa tem prioridade e não custa requisição.
	taskPtr.Content = taskDisplayName(*taskPtr)

	if !taskNeedsDetail(*taskPtr) {
		return
	}

	taskDetail, err := t.GetTaskDetails(taskPtr.ID)
	if err != nil {
		slog.Debug("Erro ao buscar detalhes da tarefa", "tarefa", taskPtr.ID, "err", err)
		return
	}

	if taskPtr.Name == "" && taskDetail.Name != "" {
		taskPtr.Name = taskDetail.Name
	}
	if isPlaceholderName(taskPtr.Content) || taskPtr.Content == "" {
		if taskDetail.Name != "" {
			taskPtr.Content = taskDetail.Name
		} else if taskDetail.Content != "" {
			taskPtr.Content = taskDetail.Content
		}
	}

	if taskPtr.ProjectID == 0 && taskDetail.ProjectID != 0 {
		taskPtr.ProjectID = taskDetail.ProjectID
	}
	if taskPtr.ProjectName == "" && taskDetail.ProjectName != "" {
		taskPtr.ProjectName = taskDetail.ProjectName
	}
	if taskPtr.Description == "" && taskDetail.Description != "" {
		taskPtr.Description = taskDetail.Description
	}
	if taskPtr.TasklistName == "" && taskDetail.TasklistName != "" {
		taskPtr.TasklistName = taskDetail.TasklistName
	}
	if taskDetail.LoggedMinutes > 0 {
		taskPtr.LoggedMinutes = taskDetail.LoggedMinutes
	}

	taskPtr.Content = taskDisplayName(*taskPtr)
}

func (t *TeamworkAPI) getTasksByTasklists(projectID int) ([]TeamworkTask, error) {
	slog.Debug("Tentando método alternativo: obter tarefas através das listas de tarefas", "projeto", projectID)

	tasklists, err := t.GetTasklistsByProject(projectID)
	if err != nil {
		slog.Debug("Erro ao obter listas de tarefas; tentando fallback para API v2", "projeto", projectID, "err", err)
		return t.fallbackGetTasksByProject(projectID)
	}

	if len(tasklists) == 0 {
		slog.Debug("Nenhuma lista de tarefas encontrada; tentando fallback para API v2", "projeto", projectID)
		return t.fallbackGetTasksByProject(projectID)
	}

	// Uma requisição por lista, até 3 em paralelo. Sequencial, um projeto com
	// dezenas de listas levava dezenas de idas e voltas. Cada lista grava na
	// sua posição para o resultado sair na ordem das listas.
	porLista := make([][]TeamworkTask, len(tasklists))

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 3)

	for i, tasklist := range tasklists {
		wg.Add(1)
		go func(pos int, tl TaskListItem) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			tasks, err := t.GetTasksByTasklist(tl.ID)
			if err != nil {
				slog.Debug("Erro ao obter tarefas da lista", "lista", tl.ID, "err", err)
				return
			}

			for j := range tasks {
				if tasks[j].ProjectID == 0 {
					tasks[j].ProjectID = projectID
				}
				if tasks[j].TasklistName == "" {
					tasks[j].TasklistName = tl.Name
				}
			}
			porLista[pos] = tasks
		}(i, tasklist)
	}

	wg.Wait()

	var allTasks []TeamworkTask
	for _, tasks := range porLista {
		allTasks = append(allTasks, tasks...)
	}

	if len(allTasks) == 0 {
		slog.Debug("Nenhuma tarefa encontrada via listas; tentando fallback para API v2", "projeto", projectID)
		return t.fallbackGetTasksByProject(projectID)
	}

	return allTasks, nil
}

func (t *TeamworkAPI) GetTasklistsByProject(projectID int) ([]TaskListItem, error) {
	path := fmt.Sprintf("/projects/api/v3/projects/%d/tasklists.json", projectID)

	slog.Debug("Buscando listas de tarefas", "projeto", projectID)

	tasklists := make([]TaskListItem, 0)
	err := t.fetchPages(t.buildURL(path), listPageSize, maxListPages, "listas de tarefas",
		func(body []byte) (pageInfo, error) {
			var page struct {
				Tasklists []TaskListItem `json:"tasklists"`
				pageMeta
			}
			if err := json.Unmarshal(body, &page); err != nil {
				return pageInfo{}, err
			}
			tasklists = append(tasklists, page.Tasklists...)
			return pageInfo{items: len(page.Tasklists), hasMore: page.Meta.Page.HasMore}, nil
		})
	if err != nil {
		return nil, err
	}

	return tasklists, nil
}

func (t *TeamworkAPI) GetTasksByTasklist(tasklistID int) ([]TeamworkTask, error) {
	path := fmt.Sprintf("/projects/api/v3/tasklists/%d/tasks.json?include=%s&includeTaskDetails=true",
		tasklistID, taskIncludes)

	slog.Debug("Buscando tarefas da lista", "lista", tasklistID)

	return t.fetchTaskPages(t.buildURL(path), "tarefas da lista")
}

func (t *TeamworkAPI) fallbackGetTasksByProject(projectID int) ([]TeamworkTask, error) {
	slog.Debug("Tentando método alternativo (API v2) para obter tarefas", "projeto", projectID)

	projectIDStr := strconv.Itoa(projectID)
	path := fmt.Sprintf("/tasks.json?project_id=%s", projectIDStr)
	url := t.buildURL(path)

	slog.Debug("Fazendo requisição alternativa", "url", url)

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

// upcomingDeadlinesLimit é quantas tarefas o card do dashboard exibe.
const upcomingDeadlinesLimit = 5

// GetTasksWithUpcomingDeadlines devolve as próximas tarefas do usuário com data
// de vencimento definida, ordenadas da mais próxima para a mais distante.
//
// A versão anterior gerava a data com `now.AddDate(0, 0, i+1)` — ou seja, a
// primeira tarefa "vencia" amanhã, a segunda depois de amanhã, e assim por
// diante, independentemente do prazo real — e fixava a prioridade em "Normal".
// Ambos os campos já vinham da API e simplesmente não eram usados. Tarefas sem
// prazo definido são omitidas em vez de receberem um prazo inventado.
func (t *TeamworkAPI) GetTasksWithUpcomingDeadlines() ([]map[string]interface{}, error) {
	tarefas, err := t.ListUpcomingDeadlines()
	if err != nil {
		return nil, err
	}
	return toMaps(tarefas), nil
}

// ListUpcomingDeadlines é a versão tipada de GetTasksWithUpcomingDeadlines.
func (t *TeamworkAPI) ListUpcomingDeadlines() ([]UpcomingDeadline, error) {
	cacheKey := "upcoming_tasks"
	if cached, found := getCached[[]UpcomingDeadline](t.cache, cacheKey); found {
		return cached, nil
	}

	tasks, err := t.GetTasks()
	if err != nil {
		return nil, fmt.Errorf("erro ao obter tarefas: %v", err)
	}

	tarefas := filtrarEOrdenarPrazos(tasks, startOfToday(), upcomingDeadlinesLimit)

	t.cache.Set(cacheKey, tarefas, 30*time.Minute)
	return tarefas, nil
}

// filtrarEOrdenarPrazos mantém apenas as tarefas com prazo real a partir de
// hoje, ordena da mais próxima para a mais distante e corta no limite.
// Tarefas sem prazo, ou com prazo irreconhecível, são descartadas.
func filtrarEOrdenarPrazos(tasks []TeamworkTask, hoje time.Time, limite int) []UpcomingDeadline {
	type tarefaComPrazo struct {
		task TeamworkTask
		due  time.Time
	}

	comPrazo := make([]tarefaComPrazo, 0, len(tasks))
	for _, task := range tasks {
		due, ok := parseTeamworkDate(task.DueDate)
		if !ok || due.Before(hoje) {
			continue
		}
		comPrazo = append(comPrazo, tarefaComPrazo{task: task, due: due})
	}

	sort.Slice(comPrazo, func(i, j int) bool {
		if comPrazo[i].due.Equal(comPrazo[j].due) {
			return comPrazo[i].task.ID < comPrazo[j].task.ID
		}
		return comPrazo[i].due.Before(comPrazo[j].due)
	})

	if len(comPrazo) > limite {
		comPrazo = comPrazo[:limite]
	}

	tarefas := make([]UpcomingDeadline, 0, len(comPrazo))
	for _, item := range comPrazo {
		nome := item.task.Content
		if nome == "" {
			nome = item.task.Name
		}

		tarefas = append(tarefas, UpcomingDeadline{
			ID:          item.task.ID,
			Name:        nome,
			DueDate:     item.due.Format("2006-01-02"),
			Priority:    item.task.Priority,
			ProjectID:   item.task.ProjectID,
			ProjectName: item.task.ProjectName,
		})
	}

	return tarefas
}
