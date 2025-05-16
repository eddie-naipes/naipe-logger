package api

import (
	"fmt"
	"sync"
	"time"
)

type TeamworkAPI struct {
	Config Config
	cache  *Cache
}

func NewTeamworkAPI(config Config) *TeamworkAPI {
	if config.MinutosPorDia == 0 {
		config.MinutosPorDia = 8 * 60
	}

	return &TeamworkAPI{
		Config: config,
		cache:  NewCache(),
	}
}

func m(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (t *TeamworkAPI) getProjectInfo(projectID int) []Project {
	projects, err := t.GetProjects()
	if err != nil {
		fmt.Printf("Erro ao obter informações do projeto: %v\n", err)
		return []Project{}
	}
	return projects
}

func (t *TeamworkAPI) GetDashboardStats() (map[string]interface{}, error) {
	cacheKey := fmt.Sprintf("dashboard_stats_%d", t.Config.UserID)
	if cachedData, found := t.cache.Get(cacheKey); found {
		return cachedData.(map[string]interface{}), nil
	}

	if !t.IsConfigured() {
		return nil, fmt.Errorf("API não configurada")
	}

	stats := make(map[string]interface{})

	now := time.Now()
	firstDay := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	lastDay := time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, now.Location())

	startDate := firstDay.Format("2006-01-02")
	endDate := lastDay.Format("2006-01-02")

	// Executar solicitações em paralelo
	var wg sync.WaitGroup
	var taskCountErr, projectCountErr, hoursLoggedErr, workDaysErr error
	var tarefasPendentes, projetosAtivos int
	var horasLogadas, horasLogadasAnterior float64
	var diasUteis []string

	wg.Add(4)

	go func() {
		defer wg.Done()
		tarefasPendentes, taskCountErr = t.GetTaskCount()
	}()

	go func() {
		defer wg.Done()
		projetosAtivos, projectCountErr = t.GetProjectCount()
	}()

	go func() {
		defer wg.Done()
		horasLogadas, hoursLoggedErr = t.GetHoursLoggedInPeriod(startDate, endDate)
		if hoursLoggedErr == nil {
			mesAnteriorPrimeiroDia := time.Date(firstDay.Year(), firstDay.Month()-1, 1, 0, 0, 0, 0, firstDay.Location())
			mesAnteriorUltimoDia := time.Date(firstDay.Year(), firstDay.Month(), 0, 0, 0, 0, 0, firstDay.Location())
			startDateAnterior := mesAnteriorPrimeiroDia.Format("2006-01-02")
			endDateAnterior := mesAnteriorUltimoDia.Format("2006-01-02")
			horasLogadasAnterior, _ = t.GetHoursLoggedInPeriod(startDateAnterior, endDateAnterior)
		}
	}()

	go func() {
		defer wg.Done()
		diasUteis, workDaysErr = t.GetWorkingDays(startDate, endDate)
	}()

	wg.Wait()

	// Processa os resultados
	if taskCountErr != nil {
		stats["tarefasPendentes"] = 0
	} else {
		stats["tarefasPendentes"] = tarefasPendentes
	}

	if projectCountErr != nil {
		stats["projetos"] = 0
	} else {
		stats["projetos"] = projetosAtivos
	}

	if hoursLoggedErr != nil {
		stats["horasLogadas"] = 0.0
		stats["horasLogadasChange"] = 0
	} else {
		stats["horasLogadas"] = horasLogadas
		if horasLogadasAnterior > 0 {
			horasChange := ((horasLogadas - horasLogadasAnterior) / horasLogadasAnterior) * 100
			stats["horasLogadasChange"] = int(horasChange)
		} else {
			stats["horasLogadasChange"] = 0
		}
	}

	if workDaysErr != nil {
		stats["diasUteisMes"] = 0
		stats["diasUteisRestantes"] = 0
		stats["diasUteisPassados"] = 0
	} else {
		diasUteisMes := len(diasUteis)
		stats["diasUteisMes"] = diasUteisMes

		hoje := time.Now().Format("2006-01-02")
		diasUteisRestantes := 0
		diasUteisPassados := 0

		for _, dia := range diasUteis {
			if dia >= hoje {
				diasUteisRestantes++
			} else {
				diasUteisPassados++
			}
		}

		stats["diasUteisRestantes"] = diasUteisRestantes
		stats["diasUteisPassados"] = diasUteisPassados
	}

	t.cache.Set(cacheKey, stats, 1*time.Hour)
	return stats, nil
}
