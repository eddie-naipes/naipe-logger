package api

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type TeamworkAPI struct {
	Config Config
	cache  *Cache

	// ctx é o contexto de vida da aplicação. Toda requisição o carrega, de modo
	// que fechar o aplicativo aborta as chamadas em voo e interrompe as esperas
	// de backoff em vez de deixá-las rodando até o fim.
	ctxMutex sync.RWMutex
	ctx      context.Context

	// baseURL é o host já normalizado no construtor, para não refazer o parse
	// a cada requisição. normalizedFrom guarda o valor bruto que o originou.
	baseURL        string
	normalizedFrom string
	hostNormalized bool

	// httpClient substitui o cliente compartilhado; usado pelos testes.
	httpClient *http.Client
}

func NewTeamworkAPI(config Config) *TeamworkAPI {
	if config.MinutosPorDia == 0 {
		config.MinutosPorDia = 8 * 60
	}

	return &TeamworkAPI{
		Config:         config,
		cache:          NewCache(),
		baseURL:        normalizeHostOrEmpty(config.ApiHost),
		normalizedFrom: config.ApiHost,
		hostNormalized: true,
	}
}

// SetContext associa o contexto da aplicação ao cliente. Chamado ao criar ou
// substituir o cliente; sem ele as requisições usam context.Background().
func (t *TeamworkAPI) SetContext(ctx context.Context) {
	t.ctxMutex.Lock()
	defer t.ctxMutex.Unlock()
	t.ctx = ctx
}

// requestContext devolve o contexto a usar numa requisição, sempre não-nulo.
func (t *TeamworkAPI) requestContext() context.Context {
	t.ctxMutex.RLock()
	defer t.ctxMutex.RUnlock()
	if t.ctx == nil {
		return context.Background()
	}
	return t.ctx
}

func (t *TeamworkAPI) GetDashboardStats() (map[string]interface{}, error) {
	cacheKey := fmt.Sprintf("%s%d", cacheKeyDashboardStatsPrefix, t.Config.UserID)
	if cached, found := getCached[map[string]interface{}](t.cache, cacheKey); found {
		// Cópia: o chamador ajusta "horasLogadas" no mapa devolvido, e mutar o
		// objeto em cache contaminaria as próximas leituras.
		return copyStats(cached), nil
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

	mesAnteriorInicio := firstDay.AddDate(0, -1, 0).Format("2006-01-02")
	mesAnteriorFim := firstDay.AddDate(0, 0, -1).Format("2006-01-02")

	var wg sync.WaitGroup
	var taskCountErr, projectCountErr, hoursLoggedErr, hoursPrevErr, workDaysErr error
	var tarefasPendentes, projetosAtivos int
	var horasLogadas, horasLogadasAnterior float64
	var diasUteis []string

	wg.Add(5)

	go func() {
		defer wg.Done()
		tarefasPendentes, taskCountErr = t.GetTaskCount()
	}()

	go func() {
		defer wg.Done()
		projetosAtivos, projectCountErr = t.GetProjectCount()
	}()

	// As horas vêm de time/total.json, que já soma no servidor. A versão
	// anterior listava time.json com fromDate/toDate — parâmetros que a v3
	// ignora — e sem paginar, então o número não batia com o Teamwork e o
	// binding precisava sobrescrevê-lo.
	go func() {
		defer wg.Done()
		horasLogadas, hoursLoggedErr = t.GetHoursLoggedInPeriod(startDate, endDate)
	}()

	go func() {
		defer wg.Done()
		horasLogadasAnterior, hoursPrevErr = t.GetHoursLoggedInPeriod(mesAnteriorInicio, mesAnteriorFim)
	}()

	go func() {
		defer wg.Done()
		diasUteis, workDaysErr = t.GetWorkingDays(startDate, endDate)
	}()

	wg.Wait()

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
		t.logWarn("Erro ao obter horas do mês: %v", hoursLoggedErr)
		stats["horasLogadas"] = 0.0
		stats["horasLogadasChange"] = 0
	} else {
		stats["horasLogadas"] = horasLogadas
		stats["horasLogadasChange"] = 0
		if hoursPrevErr == nil && horasLogadasAnterior > 0 {
			horasChange := ((horasLogadas - horasLogadasAnterior) / horasLogadasAnterior) * 100
			stats["horasLogadasChange"] = int(horasChange)
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

	t.cache.Set(cacheKey, copyStats(stats), 1*time.Hour)
	return stats, nil
}

func copyStats(stats map[string]interface{}) map[string]interface{} {
	copied := make(map[string]interface{}, len(stats))
	for k, v := range stats {
		copied[k] = v
	}
	return copied
}
