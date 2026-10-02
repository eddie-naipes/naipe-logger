package api

import (
	"context"
	"fmt"
	"log/slog"
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
	registerSecretForLogs(config.AuthToken)

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

// GetDashboardStats devolve os números do dashboard no formato de mapa que o
// binding atual expõe. Mantido por compatibilidade; GetDashboardSummary é a
// versão tipada. Cada chamada monta um mapa novo, então o chamador pode
// alterá-lo sem contaminar o cache.
func (t *TeamworkAPI) GetDashboardStats() (map[string]interface{}, error) {
	stats, err := t.GetDashboardSummary()
	if err != nil {
		return nil, err
	}
	return stats.toMap(), nil
}

// GetDashboardSummary calcula os números do dashboard do mês atual.
func (t *TeamworkAPI) GetDashboardSummary() (DashboardStats, error) {
	cacheKey := fmt.Sprintf("%s%d", cacheKeyDashboardStatsPrefix, t.Config.UserID)
	if cached, found := getCached[DashboardStats](t.cache, cacheKey); found {
		return cached, nil
	}

	if !t.IsConfigured() {
		return DashboardStats{}, fmt.Errorf("API não configurada")
	}

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

	// Falhas parciais viram zero no card correspondente em vez de derrubar o
	// dashboard inteiro.
	var stats DashboardStats

	if taskCountErr != nil {
		slog.Warn("Erro ao contar tarefas pendentes", "err", taskCountErr)
	} else {
		stats.TarefasPendentes = tarefasPendentes
	}
	if projectCountErr != nil {
		slog.Warn("Erro ao contar projetos ativos", "err", projectCountErr)
	} else {
		stats.Projetos = projetosAtivos
	}

	if hoursLoggedErr != nil {
		slog.Warn("Erro ao obter horas do mês", "err", hoursLoggedErr)
	} else {
		stats.HorasLogadas = horasLogadas
		if hoursPrevErr == nil && horasLogadasAnterior > 0 {
			stats.HorasLogadasChange = int(((horasLogadas - horasLogadasAnterior) / horasLogadasAnterior) * 100)
		}
	}

	if workDaysErr == nil {
		stats.DiasUteisMes = len(diasUteis)

		hoje := time.Now().Format("2006-01-02")
		for _, dia := range diasUteis {
			if dia >= hoje {
				stats.DiasUteisRestantes++
			} else {
				stats.DiasUteisPassados++
			}
		}
	}

	t.cache.Set(cacheKey, stats, 1*time.Hour)
	return stats, nil
}
