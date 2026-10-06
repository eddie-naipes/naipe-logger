package backend

import (
	"fmt"
	"time"

	"logTime-go/backend/api"
	"logTime-go/backend/planning"
)

// Bindings de "Completar período": gera um plano que preenche só o que falta
// em cada dia útil para atingir a jornada configurada.

// FillGapsRequest descreve o período e a origem das entradas: um template
// (TemplateName) ou tarefas salvas (TaskIDs). Com template, TaskIDs é ignorado.
type FillGapsRequest struct {
	Start        string `json:"start"`
	End          string `json:"end"`
	TemplateName string `json:"templateName"`
	TaskIDs      []int  `json:"taskIds"`
	// IncludeFuture libera dias depois de hoje ("mês inteiro").
	IncludeFuture bool `json:"includeFuture"`
	// Granularity é o menor déficit lançado, em minutos (<= 0 usa 15).
	Granularity int `json:"granularity"`
}

// FillGapsResult traz o plano e o resumo de todos os dias úteis do período,
// inclusive os que ficaram de fora (completos, futuros...).
type FillGapsResult struct {
	Plan          []api.WorkDay         `json:"plan"`
	Days          []planning.DaySummary `json:"days"`
	MinutesPerDay int                   `json:"minutesPerDay"`
}

func (a *App) PlanFillGaps(req FillGapsRequest) (*FillGapsResult, error) {
	client, err := a.client()
	if err != nil {
		return nil, err
	}

	tarefas, err := tasksForFill(req, a.configManager.GetTemplate, a.configManager.GetSavedTasks)
	if err != nil {
		return nil, err
	}

	diasUteis, err := client.GetWorkingDays(req.Start, req.End)
	if err != nil {
		return nil, err
	}

	lancado, err := client.DailyLoggedMinutes(req.Start, req.End)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler as horas já lançadas: %v", err)
	}

	jornada := client.Config.MinutosPorDia
	plano, dias := planning.FillGaps(diasUteis, lancado, tarefas, planning.FillOptions{
		MinutosPorDia:  jornada,
		Granularidade:  req.Granularity,
		Hoje:           time.Now().Format("2006-01-02"),
		IncluirFuturos: req.IncludeFuture,
	})

	return &FillGapsResult{Plan: plano, Days: dias, MinutesPerDay: jornada}, nil
}

// tasksForFill resolve as tarefas de origem do pedido. As funções de leitura
// da configuração vêm por parâmetro para o teste não depender do HOME.
func tasksForFill(
	req FillGapsRequest,
	getTemplate func(string) (api.Template, bool),
	getSaved func() []api.Task,
) ([]api.Task, error) {
	if req.TemplateName != "" {
		tpl, ok := getTemplate(req.TemplateName)
		if !ok {
			return nil, fmt.Errorf("template '%s' não encontrado", req.TemplateName)
		}
		if len(tpl.Tasks) == 0 {
			return nil, fmt.Errorf("o template '%s' não tem tarefas", req.TemplateName)
		}
		return tpl.Tasks, nil
	}

	if len(req.TaskIDs) == 0 {
		return nil, fmt.Errorf("selecione um template ou ao menos uma tarefa salva")
	}

	// Mantém a ordem das tarefas salvas: é ela que decide quem é preenchido
	// primeiro (e quem é truncado) em cada dia.
	escolhidas := make(map[int]bool, len(req.TaskIDs))
	for _, id := range req.TaskIDs {
		escolhidas[id] = true
	}
	tarefas := make([]api.Task, 0, len(req.TaskIDs))
	for _, t := range getSaved() {
		if escolhidas[t.TaskID] {
			tarefas = append(tarefas, t)
		}
	}
	if len(tarefas) == 0 {
		return nil, fmt.Errorf("nenhuma das tarefas selecionadas está salva")
	}
	return tarefas, nil
}
