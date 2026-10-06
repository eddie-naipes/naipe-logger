package planning

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"logTime-go/backend/api"
)

// WeekCell é o cruzamento tarefa × dia da grade semanal. Entries carrega os
// lançamentos (com ID) para que a interface possa editar ou apagar um a um.
type WeekCell struct {
	Date    string                `json:"date"`
	Minutes int                   `json:"minutes"`
	Entries []api.TimeEntryReport `json:"entries"`
}

// WeekRow é uma linha da grade. Saved indica que a tarefa está entre as
// tarefas salvas (a descrição/billable padrão vêm de lá).
type WeekRow struct {
	TaskID      int        `json:"taskId"`
	TaskName    string     `json:"taskName"`
	ProjectName string     `json:"projectName"`
	Saved       bool       `json:"saved"`
	Cells       []WeekCell `json:"cells"`
	Total       int        `json:"total"`
}

// WeekGrid é a semana de segunda a domingo. Days, DayTotals e as Cells de cada
// linha têm sempre 7 posições alinhadas; esconder sábado/domingo é decisão da
// interface.
type WeekGrid struct {
	WeekStart string    `json:"weekStart"`
	Days      []string  `json:"days"`
	Rows      []WeekRow `json:"rows"`
	DayTotals []int     `json:"dayTotals"`
	Total     int       `json:"total"`
}

// WeekDays devolve a segunda-feira da semana que contém a data e os 7 dias
// (segunda a domingo).
func WeekDays(data string) (string, []string, error) {
	d, err := time.ParseInLocation(layoutData, data, time.Local)
	if err != nil {
		return "", nil, fmt.Errorf("data inválida: %v", err)
	}
	// Weekday: domingo = 0. Recuar até a segunda.
	recuo := (int(d.Weekday()) + 6) % 7
	segunda := d.AddDate(0, 0, -recuo)

	dias := make([]string, 7)
	for i := range dias {
		dias[i] = segunda.AddDate(0, 0, i).Format(layoutData)
	}
	return dias[0], dias, nil
}

// BuildWeekGrid agrega os lançamentos da semana por tarefa × dia. As linhas
// são as tarefas com lançamento (ordenadas por projeto e nome) seguidas das
// tarefas salvas que ainda não têm nada na semana, na ordem em que foram
// salvas — assim o usuário pode lançar nelas direto pela grade. Lançamentos
// fora da semana são ignorados.
func BuildWeekGrid(weekStart string, entries []api.TimeEntryReport, salvas []api.Task) (WeekGrid, error) {
	inicio, dias, err := WeekDays(weekStart)
	if err != nil {
		return WeekGrid{}, err
	}

	indiceDia := make(map[string]int, len(dias))
	for i, d := range dias {
		indiceDia[d] = i
	}

	salvaPorID := make(map[int]api.Task, len(salvas))
	for _, s := range salvas {
		salvaPorID[s.TaskID] = s
	}

	linhas := make(map[int]*WeekRow)
	novaLinha := func(taskID int) *WeekRow {
		cells := make([]WeekCell, 7)
		for i, d := range dias {
			cells[i] = WeekCell{Date: d, Entries: []api.TimeEntryReport{}}
		}
		_, salva := salvaPorID[taskID]
		r := &WeekRow{TaskID: taskID, Saved: salva, Cells: cells}
		linhas[taskID] = r
		return r
	}

	grid := WeekGrid{WeekStart: inicio, Days: dias, Rows: []WeekRow{}, DayTotals: make([]int, 7)}

	for _, e := range entries {
		i, ok := indiceDia[e.Date]
		if !ok || e.Minutes <= 0 {
			continue
		}
		r, ok := linhas[e.TaskID]
		if !ok {
			r = novaLinha(e.TaskID)
		}
		if r.TaskName == "" {
			r.TaskName = e.TaskName
		}
		if r.ProjectName == "" {
			r.ProjectName = e.ProjectName
		}
		r.Cells[i].Minutes += e.Minutes
		r.Cells[i].Entries = append(r.Cells[i].Entries, e)
		r.Total += e.Minutes
		grid.DayTotals[i] += e.Minutes
		grid.Total += e.Minutes
	}

	comLancamento := make([]WeekRow, 0, len(linhas))
	for _, r := range linhas {
		if r.TaskID == 0 && r.TaskName == "" {
			r.TaskName = "Sem tarefa"
		}
		if s, ok := salvaPorID[r.TaskID]; ok {
			if r.TaskName == "" {
				r.TaskName = s.TaskName
			}
			if r.ProjectName == "" {
				r.ProjectName = s.ProjectName
			}
		}
		comLancamento = append(comLancamento, *r)
	}
	sort.SliceStable(comLancamento, func(i, j int) bool {
		a, b := comLancamento[i], comLancamento[j]
		if pa, pb := strings.ToLower(a.ProjectName), strings.ToLower(b.ProjectName); pa != pb {
			return pa < pb
		}
		if na, nb := strings.ToLower(a.TaskName), strings.ToLower(b.TaskName); na != nb {
			return na < nb
		}
		return a.TaskID < b.TaskID
	})
	grid.Rows = append(grid.Rows, comLancamento...)

	for _, s := range salvas {
		if s.TaskID <= 0 {
			continue
		}
		if _, ok := linhas[s.TaskID]; ok {
			continue
		}
		r := novaLinha(s.TaskID)
		r.TaskName, r.ProjectName = s.TaskName, s.ProjectName
		grid.Rows = append(grid.Rows, *r)
	}

	// Cada célula com os lançamentos em ordem estável (horário, depois ID).
	for ri := range grid.Rows {
		for ci := range grid.Rows[ri].Cells {
			es := grid.Rows[ri].Cells[ci].Entries
			sort.SliceStable(es, func(i, j int) bool {
				if es[i].StartTime != es[j].StartTime {
					return es[i].StartTime < es[j].StartTime
				}
				return es[i].ID < es[j].ID
			})
		}
	}

	return grid, nil
}

// CopyPlan é o resultado de "copiar semana anterior".
type CopyPlan struct {
	Plan []api.WorkDay `json:"plan"`
	// SkippedDays são os dias da semana atual que receberiam cópia mas não
	// são úteis (fim de semana/feriado).
	SkippedDays []string `json:"skippedDays"`
	// SkippedEntries conta os lançamentos da semana anterior que ficaram de
	// fora (dia não útil, sem tarefa ou sem minutos).
	SkippedEntries int `json:"skippedEntries"`
}

// BuildCopyPlan transforma os lançamentos da semana anterior num plano para a
// semana que começa em weekStart, cada um no mesmo dia da semana (+7 dias),
// com a mesma tarefa, minutos, descrição e billable.
//
// Horário: usa o StartTime do lançamento quando a API o informa; a listagem v2
// usada hoje não traz esse dado, então os lançamentos do dia são enfileirados
// a partir das 09:00 na ordem original (data, horário, ID), sem sobreposição.
//
// Lançamentos sem tarefa não são copiados: o envio em lote só lança em
// tarefas. isWorkDay vem de fora (feriados dependem do calendário remoto).
func BuildCopyPlan(anteriores []api.TimeEntryReport, weekStart string, isWorkDay func(string) bool) (CopyPlan, error) {
	inicio, dias, err := WeekDays(weekStart)
	if err != nil {
		return CopyPlan{}, err
	}
	fimSemana := dias[6]

	ordenados := append([]api.TimeEntryReport(nil), anteriores...)
	sort.SliceStable(ordenados, func(i, j int) bool {
		a, b := ordenados[i], ordenados[j]
		if a.Date != b.Date {
			return a.Date < b.Date
		}
		if a.StartTime != b.StartTime {
			return a.StartTime < b.StartTime
		}
		return a.ID < b.ID
	})

	resultado := CopyPlan{Plan: []api.WorkDay{}, SkippedDays: []string{}}
	porDia := make(map[string]*api.WorkDay)
	cursor := make(map[string]int)
	ordem := make([]string, 0)
	puladoVisto := make(map[string]bool)
	inicioPadrao, _ := parseClock(InicioPadrao)

	for _, e := range ordenados {
		if e.TaskID <= 0 || e.Minutes <= 0 {
			resultado.SkippedEntries++
			continue
		}
		origem, err := time.ParseInLocation(layoutData, e.Date, time.Local)
		if err != nil {
			resultado.SkippedEntries++
			continue
		}
		destino := origem.AddDate(0, 0, 7).Format(layoutData)
		if destino < inicio || destino > fimSemana {
			resultado.SkippedEntries++
			continue
		}
		if !isWorkDay(destino) {
			resultado.SkippedEntries++
			if !puladoVisto[destino] {
				puladoVisto[destino] = true
				resultado.SkippedDays = append(resultado.SkippedDays, destino)
			}
			continue
		}

		wd, ok := porDia[destino]
		if !ok {
			wd = &api.WorkDay{Date: destino, Entries: []api.EntryTask{}}
			porDia[destino] = wd
			cursor[destino] = inicioPadrao
			ordem = append(ordem, destino)
		}

		horario := cursor[destino]
		if proprio, ok := parseClock(e.StartTime); ok {
			horario = proprio
		}
		cursor[destino] = max(cursor[destino], horario+e.Minutes)

		descricao := e.Description
		if strings.TrimSpace(descricao) == "" {
			descricao = e.TaskName
		}

		wd.Entries = append(wd.Entries, api.EntryTask{
			TaskID: e.TaskID,
			Entry: api.TimeEntry{
				Minutes:     e.Minutes,
				Time:        formatClock(horario),
				Description: descricao,
				IsBillable:  e.IsBillable,
			},
		})
		wd.TotalMin += e.Minutes
	}

	sort.Strings(ordem)
	for _, d := range ordem {
		resultado.Plan = append(resultado.Plan, *porDia[d])
	}
	sort.Strings(resultado.SkippedDays)
	return resultado, nil
}
