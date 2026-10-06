package backend

import (
	"fmt"
	"time"

	"logTime-go/backend/planning"
)

// Bindings da grade semanal (timesheet tarefa × dia).

// GetWeekGrid devolve a semana (segunda a domingo) que contém a data, com os
// lançamentos do usuário agregados por tarefa × dia e as tarefas salvas sem
// lançamento como linhas vazias.
func (a *App) GetWeekGrid(date string) (*planning.WeekGrid, error) {
	client, err := a.client()
	if err != nil {
		return nil, err
	}

	_, dias, err := planning.WeekDays(date)
	if err != nil {
		return nil, err
	}

	entries, err := client.GetTimeEntriesForPeriodV2(dias[0], dias[6], false)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler os lançamentos da semana: %v", err)
	}

	grid, err := planning.BuildWeekGrid(dias[0], entries, a.configManager.GetSavedTasks())
	if err != nil {
		return nil, err
	}
	return &grid, nil
}

// BuildCopyPreviousWeekPlan monta (sem enviar) o plano que replica na semana
// de `date` os lançamentos da semana anterior, pulando dias não úteis. O envio
// segue o caminho de sempre: CheckPlanConflicts + LogMultipleTimes.
func (a *App) BuildCopyPreviousWeekPlan(date string) (*planning.CopyPlan, error) {
	client, err := a.client()
	if err != nil {
		return nil, err
	}

	_, dias, err := planning.WeekDays(date)
	if err != nil {
		return nil, err
	}
	segunda, _ := time.ParseInLocation("2006-01-02", dias[0], time.Local)
	inicioAnterior := segunda.AddDate(0, 0, -7).Format("2006-01-02")
	fimAnterior := segunda.AddDate(0, 0, -1).Format("2006-01-02")

	anteriores, err := client.GetTimeEntriesForPeriodV2(inicioAnterior, fimAnterior, false)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler os lançamentos da semana anterior: %v", err)
	}

	isWorkDay := func(d string) bool {
		t, err := time.ParseInLocation("2006-01-02", d, time.Local)
		return err == nil && client.IsWorkDay(t)
	}

	plano, err := planning.BuildCopyPlan(anteriores, dias[0], isWorkDay)
	if err != nil {
		return nil, err
	}
	return &plano, nil
}
