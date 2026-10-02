package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

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

	slog.Debug("Lançando tempo", "tarefa", taskID, "data", entry.Date, "hora", entry.Time,
		"minutos", entry.Minutes, "descricao", entry.Description)

	reqBody := TimelogRequest{
		Timelog: entry,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("erro ao converter para JSON: %v", err)
	}

	req, err := t.createRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	resp, body, err := t.doRequest(req)
	if err != nil {
		return nil, err
	}

	slog.Debug("Resposta do servidor ao lançamento", "status", resp.StatusCode, "corpo", sanitizeForLog(truncateForError(body, 300)))

	result := &TimeLogResult{
		TaskID: taskID,
		Date:   entry.Date,
	}

	if resp.StatusCode == 201 {
		t.invalidateTimeEntryCaches()
		result.Success = true
		result.Message = fmt.Sprintf("Entrada de tempo enviada com sucesso: %s %s",
			entry.Date, entry.Time)

		if entryID, found := extractTimelogID(body); found {
			result.EntryID = entryID
			result.Message += fmt.Sprintf(" (ID: %d)", entryID)
		} else {
			// Sem o ID a entrada existe no Teamwork mas não pode ser desfeita
			// pela ferramenta. Registrar o corpo ajuda a mapear o formato.
			slog.Warn("Lançamento criado sem ID reconhecível na resposta", "corpo", sanitizeForLog(truncateForError(body, 300)))
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

		return result, errors.New(result.Message)
	}
}

// extractTimelogID procura o ID da entrada criada na resposta do POST. O
// formato varia conforme a versão do endpoint, então tentamos as formas
// conhecidas em vez de assumir uma só. Sem esse ID o lançamento não pode ser
// desfeito pela ferramenta.
func extractTimelogID(body []byte) (int, bool) {
	var shapes struct {
		ID        int `json:"id"`
		TimelogID int `json:"timelogId"`
		Timelog   struct {
			ID int `json:"id"`
		} `json:"timelog"`
		TimeEntry struct {
			ID int `json:"id"`
		} `json:"timeEntry"`
		TimeLog struct {
			ID int `json:"id"`
		} `json:"timeLog"`
	}

	if err := json.Unmarshal(body, &shapes); err != nil {
		return 0, false
	}

	for _, candidate := range []int{
		shapes.ID,
		shapes.TimelogID,
		shapes.Timelog.ID,
		shapes.TimeEntry.ID,
		shapes.TimeLog.ID,
	} {
		if candidate > 0 {
			return candidate, true
		}
	}

	return 0, false
}

// extractTimelogTaskID lê a tarefa do lançamento na resposta do PUT, quando a
// API a informa. Devolve 0 se não houver.
func extractTimelogTaskID(body []byte) int {
	var shapes struct {
		Timelog struct {
			TaskID int `json:"taskId"`
		} `json:"timelog"`
		TaskID int `json:"taskId"`
	}
	if err := json.Unmarshal(body, &shapes); err != nil {
		return 0
	}
	if shapes.Timelog.TaskID > 0 {
		return shapes.Timelog.TaskID
	}
	return shapes.TaskID
}

func (t *TeamworkAPI) LogMultipleTimes(workDays []WorkDay) ([]*TimeLogResult, error) {
	if len(workDays) == 0 {
		return nil, fmt.Errorf("nenhum dia de trabalho fornecido para lançamento")
	}

	type pendente struct {
		date  string
		entry EntryTask
	}

	fila := make([]pendente, 0)
	for _, dia := range workDays {
		for _, alocacao := range dia.Entries {
			fila = append(fila, pendente{date: dia.Date, entry: alocacao})
		}
	}

	if len(fila) == 0 {
		return nil, fmt.Errorf("nenhum resultado de lançamento de horas")
	}

	slog.Debug("Iniciando lançamento em lote", "entradas", len(fila), "dias", len(workDays))

	// Cada goroutine escreve na sua posição: o painel de resultados e o
	// "reenviar só as falhas" leem na ordem do plano, não na ordem em que as
	// requisições terminaram (mesmo critério do delete em lote).
	results := make([]*TimeLogResult, len(fila))
	ctx := t.requestContext()

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 3)

	for i, item := range fila {
		wg.Add(1)
		go func(pos int, d string, a EntryTask) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			if a.TaskID <= 0 {
				results[pos] = &TimeLogResult{
					Success: false,
					Message: fmt.Sprintf("ID de tarefa inválido: %d", a.TaskID),
					Date:    d,
					TaskID:  a.TaskID,
				}
				return
			}

			entrada := a.Entry
			entrada.Date = d

			result, err := t.LogTime(a.TaskID, entrada)
			if result == nil {
				msg := "falha desconhecida ao lançar"
				if err != nil {
					msg = err.Error()
				}
				result = &TimeLogResult{Success: false, Message: msg, Date: d, TaskID: a.TaskID}
			}
			results[pos] = result

			// Respiro entre chamadas para não esbarrar no rate limit. Sai antes
			// se o aplicativo estiver fechando.
			_ = sleepContext(ctx, 500*time.Millisecond)
		}(i, item.date, item.entry)
	}

	wg.Wait()

	// Invalida de novo ao fim: um dashboard carregado no meio do lote teria
	// recolocado no cache números já desatualizados.
	t.invalidateTimeEntryCaches()

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

	// Feriados estaduais/municipais, pontes e férias vêm da configuração
	// local; consultados antes do nacional por não dependerem da rede.
	if _, extra := t.extraNonWorkingDay(data); extra {
		return false
	}

	isHoliday, _, err := t.IsHoliday(data)
	if err != nil {
		slog.Warn("Erro ao verificar feriado", "data", data.Format("2006-01-02"), "err", err)
		return true
	}
	return !isHoliday
}

func formatDate(data time.Time) string {
	return data.Format("2006-01-02")
}

func (t *TeamworkAPI) CreateDistributionPlan(diasUteis []string, tarefas []Task) []WorkDay {
	planoDistribuicao := make([]WorkDay, 0, len(diasUteis))

	// Sem log por tarefa×dia: um mês com dez tarefas gerava centenas de
	// linhas a cada pré-visualização. Fica só o resumo no fim.
	for _, dia := range diasUteis {
		workDay := WorkDay{
			Date:     dia,
			Entries:  []EntryTask{},
			TotalMin: 0,
		}

		diaData, err := time.Parse("2006-01-02", dia)
		if err != nil {
			slog.Warn("Data inválida ignorada no plano de distribuição", "data", dia, "err", err)
			continue
		}
		diaSemana := int(diaData.Weekday())

		for _, tarefa := range tarefas {
			if !taskWorksOn(tarefa, diaSemana) {
				continue
			}

			for _, entrada := range tarefa.Entries {
				workDay.Entries = append(workDay.Entries, EntryTask{
					TaskID: tarefa.TaskID,
					Entry:  entrada,
				})
				workDay.TotalMin += entrada.Minutes
			}
		}

		if len(workDay.Entries) > 0 {
			planoDistribuicao = append(planoDistribuicao, workDay)
		}
	}

	slog.Debug("Plano de distribuição gerado", "diasComLancamentos", len(planoDistribuicao),
		"diasUteis", len(diasUteis), "tarefas", len(tarefas))
	return planoDistribuicao
}

// taskWorksOn diz se a tarefa entra no dia da semana informado. Sem
// workingDays definidos, a tarefa vale para todos os dias úteis.
func taskWorksOn(tarefa Task, diaSemana int) bool {
	if len(tarefa.WorkingDays) == 0 {
		return true
	}
	for _, workingDay := range tarefa.WorkingDays {
		if workingDay == diaSemana {
			return true
		}
	}
	return false
}

// GetHoursLoggedInPeriod devolve as horas lançadas pelo usuário no período,
// somadas pelo próprio Teamwork em time/total.json. Listar time.json e somar
// aqui exigiria paginar tudo — e a versão anterior ainda usava fromDate/toDate,
// que a v3 ignora (o correto é startDate/endDate).
func (t *TeamworkAPI) GetHoursLoggedInPeriod(startDate, endDate string) (float64, error) {
	total, err := t.GetTimeTotalsForPeriod(startDate, endDate)
	if err != nil {
		return 0, err
	}
	return float64(total.TimeTotals.Minutes) / 60.0, nil
}

const (
	// Janela e limite usados no card de atividades recentes do dashboard.
	recentActivitiesDays  = 30
	recentActivitiesLimit = 5
)

// GetRecentActivities devolve os últimos lançamentos de tempo do usuário.
//
// A versão anterior listava tarefas do primeiro projeto e preenchia data e
// duração com valores gerados (`now.AddDate(0,0,-i)` e 60 minutos fixos), que o
// dashboard exibia como se fossem reais. Agora todos os campos vêm da API; se
// não houver lançamentos, o card fica vazio em vez de mostrar dado inventado.
func (t *TeamworkAPI) GetRecentActivities() ([]map[string]interface{}, error) {
	atividades, err := t.ListRecentActivities()
	if err != nil {
		return nil, err
	}
	return toMaps(atividades), nil
}

// ListRecentActivities é a versão tipada de GetRecentActivities.
func (t *TeamworkAPI) ListRecentActivities() ([]RecentActivity, error) {
	cacheKey := cacheKeyRecentActivities
	if cached, found := getCached[[]RecentActivity](t.cache, cacheKey); found {
		return cached, nil
	}

	if !t.IsConfigured() {
		return nil, fmt.Errorf("API não configurada")
	}

	now := time.Now()
	entries, err := t.GetTimeEntriesForPeriodV2(
		now.AddDate(0, 0, -recentActivitiesDays).Format("2006-01-02"),
		now.Format("2006-01-02"),
		false,
	)
	if err != nil {
		return nil, fmt.Errorf("erro ao obter lançamentos recentes: %v", err)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Date > entries[j].Date
	})

	if len(entries) > recentActivitiesLimit {
		entries = entries[:recentActivitiesLimit]
	}

	atividades := make([]RecentActivity, 0, len(entries))
	for _, entry := range entries {
		descricao := entry.Description
		if descricao == "" {
			descricao = entry.TaskName
		}

		atividades = append(atividades, RecentActivity{
			ID:          entry.ID,
			Type:        "timelog",
			Description: descricao,
			Minutes:     entry.Minutes,
			Date:        entry.Date,
			ProjectID:   entry.ProjectID,
			ProjectName: entry.ProjectName,
			TaskID:      entry.TaskID,
			TaskName:    entry.TaskName,
		})
	}

	t.cache.Set(cacheKey, atividades, 30*time.Minute)
	return atividades, nil
}

// GetAllNonWorkingDays devolve fins de semana e feriados do mês no formato de
// mapa que o binding atual expõe; ListNonWorkingDays é a versão tipada.
func (t *TeamworkAPI) GetAllNonWorkingDays(year, month int) ([]map[string]interface{}, error) {
	days, err := t.ListNonWorkingDays(year, month)
	if err != nil {
		return nil, err
	}
	return toMaps(days), nil
}

// ListNonWorkingDays lista fins de semana e feriados (em dia útil) do mês,
// mais os dias extras da configuração (ver appendExtraNonWorkingDays).
func (t *TeamworkAPI) ListNonWorkingDays(year, month int) ([]NonWorkingDay, error) {
	startDate := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)

	var endDate time.Time
	if month == 12 {
		endDate = time.Date(year+1, 1, 0, 0, 0, 0, 0, time.Local)
	} else {
		endDate = time.Date(year, time.Month(month+1), 0, 0, 0, 0, 0, time.Local)
	}

	holidays, err := t.GetHolidaysForMonth(year, month)
	if err != nil {
		slog.Warn("Erro ao obter feriados do mês", "mes", month, "ano", year, "err", err)
		holidays = []Holiday{}
	}

	nonWorkingDays := make([]NonWorkingDay, 0)

	current := startDate
	for !current.After(endDate) {
		if current.Weekday() == time.Saturday || current.Weekday() == time.Sunday {
			nonWorkingDays = append(nonWorkingDays, NonWorkingDay{
				Date: formatDate(current),
				Type: nonWorkingDayWeekend,
				Name: current.Weekday().String(),
			})
		}
		current = current.AddDate(0, 0, 1)
	}

	for _, holiday := range holidays {
		holidayDate, err := time.Parse("2006-01-02", holiday.Date)
		if err != nil {
			slog.Debug("Data de feriado inválida", "data", holiday.Date, "err", err)
			continue
		}

		if holidayDate.Weekday() != time.Saturday && holidayDate.Weekday() != time.Sunday {
			nonWorkingDays = append(nonWorkingDays, NonWorkingDay{
				Date:        holiday.Date,
				Type:        nonWorkingDayHoliday,
				Name:        holiday.Name,
				Description: holiday.Description,
				IsOptional:  holiday.IsOptional,
			})
		}
	}

	return t.appendExtraNonWorkingDays(nonWorkingDays, startDate, endDate), nil
}

func (t *TeamworkAPI) GetTimeEntryDetails(entryID int) (*TimeEntryReport, error) {
	if !t.IsConfigured() {
		return nil, fmt.Errorf("API não configurada")
	}

	entryIDStr := strconv.Itoa(entryID)
	path := fmt.Sprintf("/projects/api/v3/time/%s.json", entryIDStr)
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
		return nil, fmt.Errorf("erro ao obter detalhes da entrada de tempo: %d %s", resp.StatusCode, resp.Status)
	}

	var response struct {
		TimeEntry TimeEntryReport `json:"timeEntry"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("erro ao decodificar resposta: %v", err)
	}

	return &response.TimeEntry, nil
}

func (t *TeamworkAPI) DeleteTimeEntry(entryID int) error {
	if !t.IsConfigured() {
		return fmt.Errorf("API não configurada")
	}

	entryIDStr := strconv.Itoa(entryID)
	path := fmt.Sprintf("/projects/api/v3/time/%s.json", entryIDStr)
	url := t.buildURL(path)

	slog.Debug("Apagando entrada de tempo", "entrada", entryID, "url", url)

	req, err := t.createRequest("DELETE", url, nil)
	if err != nil {
		return err
	}

	resp, body, err := t.doRequest(req)
	if err != nil {
		return err
	}

	slog.Debug("Resposta da exclusão", "status", resp.StatusCode, "corpo", sanitizeForLog(truncateForError(body, 300)))

	if resp.StatusCode != 200 && resp.StatusCode != 204 {
		return fmt.Errorf("erro ao deletar entrada de tempo: %d %s - %s",
			resp.StatusCode, resp.Status, string(body))
	}

	t.invalidateTimeEntryCaches()
	return nil
}

func (t *TeamworkAPI) DeleteMultipleTimeEntries(entryIDs []int) ([]DeleteTimeEntryResult, error) {
	if !t.IsConfigured() {
		return nil, fmt.Errorf("API não configurada")
	}

	if len(entryIDs) == 0 {
		return []DeleteTimeEntryResult{}, nil
	}

	// Cada goroutine escreve na sua própria posição: o painel de resultados da
	// interface é lido de cima para baixo, então a ordem precisa ser a mesma em
	// que as entradas foram pedidas — e não a ordem em que terminaram.
	results := make([]DeleteTimeEntryResult, len(entryIDs))

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 3)

	for i, entryID := range entryIDs {
		wg.Add(1)
		go func(pos, id int) {
			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			result := DeleteTimeEntryResult{
				EntryID: id,
				Success: false,
			}

			err := t.DeleteTimeEntry(id)
			if err != nil {
				result.Message = err.Error()
			} else {
				result.Success = true
				result.Message = "Entrada deletada com sucesso"
			}

			results[pos] = result

			// Respiro entre chamadas para não esbarrar no rate limit. Sai antes
			// se o aplicativo estiver fechando.
			_ = sleepContext(t.requestContext(), 200*time.Millisecond)
		}(i, entryID)
	}

	wg.Wait()

	t.invalidateTimeEntryCaches()

	return results, nil
}

func (t *TeamworkAPI) GetTimeEntriesForPeriodV2(startDate, endDate string, includeDeleted bool) ([]TimeEntryReport, error) {
	if !t.IsConfigured() {
		return nil, fmt.Errorf("API não configurada")
	}

	_, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return nil, fmt.Errorf("data inicial inválida: %v", err)
	}

	_, err = time.Parse("2006-01-02", endDate)
	if err != nil {
		return nil, fmt.Errorf("data final inválida: %v", err)
	}

	startDateFormatted := strings.ReplaceAll(startDate, "-", "")
	endDateFormatted := strings.ReplaceAll(endDate, "-", "")

	showDeleted := "0"
	if includeDeleted {
		showDeleted = "1"
	}

	path := fmt.Sprintf("/projects/api/v2/time.json?getTotals=true&skipCounts=false&projectId=&companyId=0&userId=%d&assignedTeamIds=&invoicedType=all&billableType=all&fromDate=%s&toDate=%s&sortBy=date&sortOrder=desc&onlyStarredProjects=false&includeArchivedProjects=true&matchAllTags=true&projectStatus=all&showDeleted=%s",
		t.Config.UserID, startDateFormatted, endDateFormatted, showDeleted)

	slog.Debug("Obtendo entradas de tempo (API v2)", "inicio", startDate, "fim", endDate)

	// Antes só a primeira página (500 itens) era lida: um mês cheio de quem
	// lança por tarefa passava disso e a detecção de conflitos ficava cega
	// para o resto.
	entries := make([]TimeEntryReport, 0)
	err = t.fetchPages(t.buildURL(path), timeEntryPageSize, maxTimeEntryPages, "entradas de tempo",
		func(body []byte) (pageInfo, error) {
			var page struct {
				TimeEntries []v2TimeEntry `json:"timeEntries"`
				pageMeta
			}
			if err := json.Unmarshal(body, &page); err != nil {
				return pageInfo{}, err
			}

			for _, entry := range page.TimeEntries {
				entries = append(entries, t.v2EntryToReport(entry))
			}

			return pageInfo{items: len(page.TimeEntries), hasMore: page.Meta.Page.HasMore}, nil
		})
	if err != nil {
		return nil, err
	}

	return entries, nil
}

// v2TimeEntry é o formato de um lançamento em /projects/api/v2/time.json.
type v2TimeEntry struct {
	ID                int     `json:"id"`
	ProjectID         int     `json:"projectId"`
	ProjectName       string  `json:"projectName"`
	TaskID            int     `json:"taskId"`
	TaskName          string  `json:"taskName"`
	TasklistID        int     `json:"tasklistId"`
	TasklistName      string  `json:"tasklistName"`
	UserID            int     `json:"userId"`
	UserFirstName     string  `json:"userFirstName"`
	UserLastName      string  `json:"userLastName"`
	Date              string  `json:"date"`
	Hours             float64 `json:"hours"`
	HoursDecimal      float64 `json:"hoursDecimal"`
	Minutes           int     `json:"minutes"`
	Description       string  `json:"description"`
	IsBillable        bool    `json:"isBillable"`
	IsBilled          bool    `json:"isBilled"`
	HasStartTime      bool    `json:"hasStartTime"`
	Status            string  `json:"status"`
	CreatedAt         string  `json:"createdAt"`
	UpdatedDate       string  `json:"updatedDate"`
	DateDeleted       string  `json:"dateDeleted,omitempty"`
	DeletedByUserId   int     `json:"deletedByUserId,omitempty"`
	DeletedByUserName string  `json:"deletedByUserName,omitempty"`
}

func (t *TeamworkAPI) v2EntryToReport(entry v2TimeEntry) TimeEntryReport {
	// Data irreconhecível é mantida como veio: antes o erro de parse era
	// ignorado e o lançamento aparecia em "0001-01-01".
	formattedDate := entry.Date
	if parsed, ok := parseTeamworkDate(entry.Date); ok {
		formattedDate = parsed.Format("2006-01-02")
	} else {
		slog.Warn("Lançamento com data irreconhecível", "entrada", entry.ID, "data", entry.Date)
	}

	// hoursDecimal é a duração total em horas decimais; hours/minutes são as
	// partes inteira e fracionária da MESMA duração. Somar hours*60+minutes
	// só é válido como fallback — usar o total evita truncar 1.75h em 1h.
	totalMinutes := int(math.Round(entry.HoursDecimal * 60))
	if totalMinutes == 0 {
		totalMinutes = int(entry.Hours)*60 + entry.Minutes
	}

	return TimeEntryReport{
		ID:            entry.ID,
		ProjectID:     entry.ProjectID,
		ProjectName:   entry.ProjectName,
		TaskID:        entry.TaskID,
		TaskName:      entry.TaskName,
		TasklistID:    entry.TasklistID,
		TasklistName:  entry.TasklistName,
		UserID:        entry.UserID,
		UserFirstName: entry.UserFirstName,
		UserLastName:  entry.UserLastName,
		Date:          formattedDate,
		Hours:         entry.HoursDecimal,
		Minutes:       totalMinutes,
		Description:   entry.Description,
		IsBillable:    entry.IsBillable,
		IsBilled:      entry.IsBilled,
		StartTime:     "",
		EndTime:       "",
	}
}

func (t *TeamworkAPI) UpdateTimeEntry(entryID int, entry TimeEntry) (*TimeLogResult, error) {
	if !t.IsConfigured() {
		return nil, fmt.Errorf("API não configurada")
	}

	if entryID <= 0 {
		return nil, fmt.Errorf("ID de entrada inválido: %d", entryID)
	}

	if entry.Minutes <= 0 {
		return nil, fmt.Errorf("minutos devem ser maiores que zero: %d", entry.Minutes)
	}

	entryIDStr := strconv.Itoa(entryID)
	path := fmt.Sprintf("/projects/api/v3/time/%s.json", entryIDStr)
	url := t.buildURL(path)

	if t.Config.UserID <= 0 {
		return nil, fmt.Errorf("ID do usuário não configurado")
	}

	entry.UserID = t.Config.UserID

	slog.Debug("Atualizando entrada de tempo", "entrada", entryID, "data", entry.Date,
		"hora", entry.Time, "minutos", entry.Minutes, "descricao", entry.Description)

	reqBody := TimelogRequest{
		Timelog: entry,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("erro ao converter para JSON: %v", err)
	}

	req, err := t.createRequest("PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	resp, body, err := t.doRequest(req)
	if err != nil {
		return nil, err
	}

	// O payload (TimelogRequest) não leva tarefa: o PUT altera só o lançamento
	// identificado na URL. O ID editado vai em EntryID — antes ia em TaskID,
	// e quem lesse o resultado via um "ID de tarefa" que era de lançamento.
	result := &TimeLogResult{
		EntryID: entryID,
		TaskID:  extractTimelogTaskID(body),
		Date:    entry.Date,
	}

	if resp.StatusCode == 200 || resp.StatusCode == 201 {
		t.invalidateTimeEntryCaches()
		result.Success = true
		result.Message = fmt.Sprintf("Entrada de tempo atualizada com sucesso")
		return result, nil
	} else {
		result.Success = false
		var errorResponse struct {
			Errors []string `json:"errors"`
		}

		if err := json.Unmarshal(body, &errorResponse); err == nil && len(errorResponse.Errors) > 0 {
			result.Message = fmt.Sprintf("Erro ao atualizar entrada: %s", strings.Join(errorResponse.Errors, ", "))
		} else {
			result.Message = fmt.Sprintf("Erro ao atualizar entrada: %d %s - %s",
				resp.StatusCode, resp.Status, string(body))
		}

		return result, errors.New(result.Message)
	}
}
