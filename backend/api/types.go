package api

type Config struct {
	AuthToken     string `json:"authToken"`
	UserID        int    `json:"userId"`
	ApiHost       string `json:"apiHost"`
	MinutosPorDia int    `json:"minutosPorDia"`
}

type TimeEntry struct {
	Minutes     int    `json:"minutes"`
	UserID      int    `json:"userId"`
	Time        string `json:"time"`
	Description string `json:"description"`
	IsBillable  bool   `json:"isBillable"`
	Date        string `json:"date,omitempty"`
}

type Task struct {
	TaskID      int         `json:"taskId"`
	TaskName    string      `json:"taskName"`
	ProjectID   int         `json:"projectId"`
	ProjectName string      `json:"projectName"`
	Entries     []TimeEntry `json:"entries"`
}

type TimelogRequest struct {
	Timelog TimeEntry `json:"timelog"`
}

type WorkDay struct {
	Date     string      `json:"date"`
	Entries  []EntryTask `json:"entries"`
	TotalMin int         `json:"totalMin"`
}

type EntryTask struct {
	TaskID int       `json:"taskId"`
	Entry  TimeEntry `json:"entry"`
}

// Estruturas de resposta da API Teamwork
// Modifique essa estrutura no arquivo types.go

type TeamworkTask struct {
	ID           int    `json:"id"`
	Content      string `json:"content"`
	Name         string `json:"name,omitempty"` // Adicionando campo Name para compatibilidade com a API v3
	Description  string `json:"description,omitempty"`
	ProjectID    int    `json:"projectId"`
	ProjectName  string `json:"projectName"`
	Status       string `json:"status,omitempty"`
	Priority     string `json:"priority,omitempty"`
	CreatedAt    string `json:"createdAt,omitempty"`
	StartDate    string `json:"startDate,omitempty"`
	DueDate      string `json:"dueDate,omitempty"`
	TasklistID   int    `json:"tasklistId,omitempty"`
	TasklistName string `json:"tasklistName,omitempty"`
	Tags         []struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"tags,omitempty"`
	Assignees []struct {
		ID   int    `json:"id"`
		Type string `json:"type"`
	} `json:"assignees,omitempty"`
	LoggedMinutes int `json:"loggedMinutes,omitempty"` // Adicionando campo para tempo registrado
}

type TasksResponse struct {
	Tasks    []TeamworkTask `json:"tasks"`
	Included struct {
		TaskLists map[string]struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		} `json:"tasklists"`
	} `json:"included,omitempty"`
	Meta struct {
		Page struct {
			Count       int  `json:"count"`
			HasMore     bool `json:"hasMore"`
			ItemsOnPage int  `json:"itemsOnPage"`
			Page        int  `json:"page"`
			PageOffset  int  `json:"pageOffset"`
			PageSize    int  `json:"pageSize"`
			TotalItems  int  `json:"totalItems"`
			TotalPages  int  `json:"totalPages"`
		} `json:"page"`
	} `json:"meta"`
}

type Template struct {
	Name     string `json:"name"`
	Tasks    []Task `json:"tasks"`
	TotalMin int    `json:"totalMin"`
}

type TimeLogResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Date    string `json:"date"`
	TaskID  int    `json:"taskId"`
}

type Project struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Status      string `json:"status,omitempty"`
	Company     struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	} `json:"company,omitempty"`
}

type ProjectsResponse struct {
	Projects    []Project `json:"projects"`
	Page        int       `json:"page"`
	TotalPages  int       `json:"totalPages"`
	TotalItems  int       `json:"totalItems"`
	ItemsOnPage int       `json:"itemsOnPage"`
}
