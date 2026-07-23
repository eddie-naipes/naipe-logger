package api

import (
	"fmt"
	"sort"
)

// CheckPlanConflicts consulta os lançamentos já existentes no período coberto
// pelo plano e devolve os dias que já possuem tempo registrado.
//
// Existe porque a ferramenta não tem rollback: uma vez enviado, um lançamento
// duplicado só sai manualmente. Avisar antes é mais barato que corrigir depois.
func (t *TeamworkAPI) CheckPlanConflicts(workDays []WorkDay) ([]DayConflict, error) {
	if !t.IsConfigured() {
		return nil, fmt.Errorf("API não configurada")
	}

	inicio, fim, ok := planDateRange(workDays)
	if !ok {
		return []DayConflict{}, nil
	}

	existentes, err := t.GetTimeEntriesForPeriodV2(inicio, fim, false)
	if err != nil {
		return nil, fmt.Errorf("erro ao verificar lançamentos existentes: %v", err)
	}

	return detectConflicts(workDays, existentes), nil
}

// planDateRange devolve a menor e a maior data do plano.
func planDateRange(workDays []WorkDay) (string, string, bool) {
	inicio, fim := "", ""

	for _, dia := range workDays {
		if dia.Date == "" || len(dia.Entries) == 0 {
			continue
		}
		if inicio == "" || dia.Date < inicio {
			inicio = dia.Date
		}
		if fim == "" || dia.Date > fim {
			fim = dia.Date
		}
	}

	if inicio == "" {
		return "", "", false
	}
	return inicio, fim, true
}

// detectConflicts cruza o plano com os lançamentos existentes. É puro de
// propósito: toda a regra de colisão fica testável sem tocar na rede.
func detectConflicts(workDays []WorkDay, existentes []TimeEntryReport) []DayConflict {
	// Índice dos lançamentos existentes por dia.
	type diaExistente struct {
		minutos      int
		entradas     int
		porTarefa    map[int]int
		nomeDaTarefa map[int]string
	}

	existentePorDia := make(map[string]*diaExistente)
	for _, entry := range existentes {
		if entry.Date == "" {
			continue
		}
		dia, ok := existentePorDia[entry.Date]
		if !ok {
			dia = &diaExistente{
				porTarefa:    make(map[int]int),
				nomeDaTarefa: make(map[int]string),
			}
			existentePorDia[entry.Date] = dia
		}
		dia.minutos += entry.Minutes
		dia.entradas++
		dia.porTarefa[entry.TaskID] += entry.Minutes
		if entry.TaskName != "" {
			dia.nomeDaTarefa[entry.TaskID] = entry.TaskName
		}
	}

	conflitos := make([]DayConflict, 0)

	for _, planejado := range workDays {
		if len(planejado.Entries) == 0 {
			continue
		}

		existente, temLancamento := existentePorDia[planejado.Date]
		if !temLancamento || existente.entradas == 0 {
			continue
		}

		// Minutos que o plano pretende lançar em cada tarefa neste dia.
		planejadoPorTarefa := make(map[int]int)
		totalPlanejado := 0
		for _, entrada := range planejado.Entries {
			planejadoPorTarefa[entrada.TaskID] += entrada.Entry.Minutes
			totalPlanejado += entrada.Entry.Minutes
		}

		conflito := DayConflict{
			Date:            planejado.Date,
			ExistingMinutes: existente.minutos,
			ExistingEntries: existente.entradas,
			PlannedMinutes:  totalPlanejado,
			SameTask:        make([]TaskConflict, 0),
		}

		for taskID, minutosPlanejados := range planejadoPorTarefa {
			minutosExistentes, colide := existente.porTarefa[taskID]
			if !colide {
				continue
			}
			conflito.SameTask = append(conflito.SameTask, TaskConflict{
				TaskID:          taskID,
				TaskName:        existente.nomeDaTarefa[taskID],
				ExistingMinutes: minutosExistentes,
				PlannedMinutes:  minutosPlanejados,
			})
		}

		sort.Slice(conflito.SameTask, func(i, j int) bool {
			return conflito.SameTask[i].TaskID < conflito.SameTask[j].TaskID
		})

		conflitos = append(conflitos, conflito)
	}

	sort.Slice(conflitos, func(i, j int) bool {
		return conflitos[i].Date < conflitos[j].Date
	})

	return conflitos
}
