package api

// Tipos do dashboard e do calendário. Os nomes JSON são exatamente os das
// chaves dos mapas que os bindings expõem hoje, para que a troca de
// map[string]interface{} por estes tipos não mude nada para o frontend.
//
// Os métodos que devolvem mapas continuam existindo porque os bindings em
// backend/*.go declaram esse retorno; eles apenas convertem estes tipos.

// DashboardStats são os números dos cards do dashboard.
type DashboardStats struct {
	TarefasPendentes   int     `json:"tarefasPendentes"`
	Projetos           int     `json:"projetos"`
	HorasLogadas       float64 `json:"horasLogadas"`
	HorasLogadasChange int     `json:"horasLogadasChange"`
	DiasUteisMes       int     `json:"diasUteisMes"`
	DiasUteisRestantes int     `json:"diasUteisRestantes"`
	DiasUteisPassados  int     `json:"diasUteisPassados"`
}

func (s DashboardStats) toMap() map[string]interface{} {
	return map[string]interface{}{
		"tarefasPendentes":   s.TarefasPendentes,
		"projetos":           s.Projetos,
		"horasLogadas":       s.HorasLogadas,
		"horasLogadasChange": s.HorasLogadasChange,
		"diasUteisMes":       s.DiasUteisMes,
		"diasUteisRestantes": s.DiasUteisRestantes,
		"diasUteisPassados":  s.DiasUteisPassados,
	}
}

// RecentActivity é um lançamento recente exibido no dashboard.
type RecentActivity struct {
	ID          int    `json:"id"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Minutes     int    `json:"minutes"`
	Date        string `json:"date"`
	ProjectID   int    `json:"projectId"`
	ProjectName string `json:"projectName"`
	TaskID      int    `json:"taskId"`
	TaskName    string `json:"taskName"`
}

func (a RecentActivity) toMap() map[string]interface{} {
	return map[string]interface{}{
		"id":          a.ID,
		"type":        a.Type,
		"description": a.Description,
		"minutes":     a.Minutes,
		"date":        a.Date,
		"projectId":   a.ProjectID,
		"projectName": a.ProjectName,
		"taskId":      a.TaskID,
		"taskName":    a.TaskName,
	}
}

// UpcomingDeadline é uma tarefa com prazo próximo exibida no dashboard.
type UpcomingDeadline struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	DueDate     string `json:"dueDate"`
	Priority    string `json:"priority"`
	ProjectID   int    `json:"projectId"`
	ProjectName string `json:"projectName"`
}

func (d UpcomingDeadline) toMap() map[string]interface{} {
	return map[string]interface{}{
		"id":          d.ID,
		"name":        d.Name,
		"dueDate":     d.DueDate,
		"priority":    d.Priority,
		"projectId":   d.ProjectID,
		"projectName": d.ProjectName,
	}
}

// NonWorkingDay é um fim de semana ou feriado do calendário mensal.
// Description e IsOptional só fazem sentido para feriados.
type NonWorkingDay struct {
	Date        string `json:"date"`
	Type        string `json:"type"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	IsOptional  bool   `json:"isOptional,omitempty"`
}

const (
	nonWorkingDayWeekend = "weekend"
	nonWorkingDayHoliday = "holiday"
)

// toMap reproduz o formato antigo: fins de semana só com date/type/name e
// feriados sempre com description e isOptional, mesmo vazios.
func (d NonWorkingDay) toMap() map[string]interface{} {
	m := map[string]interface{}{
		"date": d.Date,
		"type": d.Type,
		"name": d.Name,
	}
	if d.Type == nonWorkingDayHoliday {
		m["description"] = d.Description
		m["isOptional"] = d.IsOptional
	}
	return m
}

func toMaps[T interface{ toMap() map[string]interface{} }](items []T) []map[string]interface{} {
	maps := make([]map[string]interface{}, 0, len(items))
	for _, item := range items {
		maps = append(maps, item.toMap())
	}
	return maps
}
