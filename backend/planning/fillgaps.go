// Package planning reúne regras de montagem de planos de lançamento que não
// dependem da rede: recebem o que já foi lido do Teamwork e devolvem
// []api.WorkDay prontos para CheckPlanConflicts/LogMultipleTimes. Ficam aqui,
// puras, para que cada regra seja coberta por teste de tabela.
package planning

import (
	"fmt"
	"time"

	"logTime-go/backend/api"
)

const (
	// GranularidadePadrao é o menor déficit (em minutos) que vale um
	// lançamento. Abaixo disso o dia é considerado completo: lançar 5 minutos
	// para "fechar a conta" polui o Teamwork sem ganho real.
	GranularidadePadrao = 15

	// InicioPadrao é o horário usado quando o template não informa um.
	InicioPadrao = "09:00"

	layoutData = "2006-01-02"
)

// FillOptions controla o preenchimento de lacunas.
type FillOptions struct {
	// MinutosPorDia é a jornada diária a atingir.
	MinutosPorDia int
	// Granularidade é o menor déficit/entrada gerado; <= 0 usa o padrão.
	Granularidade int
	// Hoje ('YYYY-MM-DD') limita o plano: dias depois dele só entram com
	// IncluirFuturos. Vem de fora para que os testes não dependam do relógio.
	Hoje string
	// IncluirFuturos libera dias posteriores a Hoje ("mês inteiro").
	IncluirFuturos bool
}

// DaySummary resume um dia útil do período: quanto já existe, quanto falta
// para a jornada e quanto o plano vai de fato lançar (pode ser menos que o
// faltante se as entradas escolhidas não somarem o suficiente, ou zero em dia
// futuro/abaixo da granularidade).
type DaySummary struct {
	Date      string `json:"date"`
	Logged    int    `json:"logged"`
	Missing   int    `json:"missing"`
	ToLog     int    `json:"toLog"`
	Future    bool   `json:"future"`
	Skipped   bool   `json:"skipped"`
	SkipCause string `json:"skipCause,omitempty"`
}

// FillGaps monta um plano que completa só o que falta em cada dia útil.
//
// Regras:
//   - dias com lançado >= jornada, ou déficit < granularidade, ficam de fora;
//   - dias após Hoje só entram com IncluirFuturos;
//   - as entradas das tarefas são usadas na ordem recebida (tarefa a tarefa,
//     entrada a entrada), respeitando workingDays de cada tarefa, até cobrir o
//     déficit; a última é truncada para não ultrapassar a jornada. Uma sobra
//     menor que a granularidade não vira entrada. As entradas não são repetidas:
//     se não somarem o déficit, o dia fica parcialmente preenchido (o resumo
//     mostra a diferença entre Missing e ToLog);
//   - horários: a lista do Teamwork (v2) não informa o horário dos lançamentos
//     existentes, então o "fim do que já existe" é estimado como o horário da
//     primeira entrada do template (ou 09:00) somado aos minutos já lançados.
//     Cada entrada mantém o próprio horário se ele for posterior a esse cursor
//     (preserva intervalos como o almoço) e é empurrada para depois dele caso
//     contrário, de modo que as entradas do dia nunca se sobreponham.
func FillGaps(diasUteis []string, lancado map[string]int, tarefas []api.Task, opts FillOptions) ([]api.WorkDay, []DaySummary) {
	granularidade := opts.Granularidade
	if granularidade <= 0 {
		granularidade = GranularidadePadrao
	}

	plano := make([]api.WorkDay, 0)
	resumo := make([]DaySummary, 0, len(diasUteis))

	for _, dia := range diasUteis {
		data, err := time.ParseInLocation(layoutData, dia, time.Local)
		if err != nil {
			continue
		}

		s := DaySummary{Date: dia, Logged: lancado[dia]}
		s.Missing = max(opts.MinutosPorDia-s.Logged, 0)
		s.Future = opts.Hoje != "" && dia > opts.Hoje

		switch {
		case s.Missing == 0:
			s.Skipped, s.SkipCause = true, "completo"
		case s.Future && !opts.IncluirFuturos:
			s.Skipped, s.SkipCause = true, "futuro"
		case s.Missing < granularidade:
			s.Skipped, s.SkipCause = true, "abaixo da granularidade"
		}
		if s.Skipped {
			resumo = append(resumo, s)
			continue
		}

		entradas := fillDay(int(data.Weekday()), s.Logged, s.Missing, granularidade, tarefas)
		if len(entradas) == 0 {
			s.Skipped, s.SkipCause = true, "sem tarefa para o dia"
			resumo = append(resumo, s)
			continue
		}

		wd := api.WorkDay{Date: dia, Entries: entradas}
		for _, e := range entradas {
			wd.TotalMin += e.Entry.Minutes
		}
		s.ToLog = wd.TotalMin
		plano = append(plano, wd)
		resumo = append(resumo, s)
	}

	return plano, resumo
}

// fillDay escolhe as entradas de um dia até cobrir o déficit.
func fillDay(diaSemana, lancado, deficit, granularidade int, tarefas []api.Task) []api.EntryTask {
	candidatas := make([]api.EntryTask, 0)
	for _, tarefa := range tarefas {
		if tarefa.TaskID <= 0 || !worksOn(tarefa, diaSemana) {
			continue
		}
		for _, e := range tarefa.Entries {
			if e.Minutes > 0 {
				candidatas = append(candidatas, api.EntryTask{TaskID: tarefa.TaskID, Entry: e})
			}
		}
	}
	if len(candidatas) == 0 {
		return nil
	}

	inicio, ok := parseClock(candidatas[0].Entry.Time)
	if !ok {
		inicio, _ = parseClock(InicioPadrao)
	}
	cursor := inicio + lancado

	restante := deficit
	escolhidas := make([]api.EntryTask, 0, len(candidatas))
	for _, c := range candidatas {
		if restante < granularidade {
			break
		}
		minutos := min(c.Entry.Minutes, restante)

		horario := cursor
		if proprio, ok := parseClock(c.Entry.Time); ok && proprio > cursor {
			horario = proprio
		}

		entrada := c.Entry
		entrada.Minutes = minutos
		entrada.Time = formatClock(horario)
		entrada.Date = ""
		escolhidas = append(escolhidas, api.EntryTask{TaskID: c.TaskID, Entry: entrada})

		cursor = horario + minutos
		restante -= minutos
	}
	return escolhidas
}

// worksOn espelha a regra de api.taskWorksOn: sem workingDays, a tarefa vale
// para todos os dias úteis.
func worksOn(tarefa api.Task, diaSemana int) bool {
	if len(tarefa.WorkingDays) == 0 {
		return true
	}
	for _, d := range tarefa.WorkingDays {
		if d == diaSemana {
			return true
		}
	}
	return false
}

// parseClock lê "HH:MM" ou "HH:MM:SS" como minutos desde a meia-noite.
func parseClock(valor string) (int, bool) {
	for _, layout := range []string{"15:04:05", "15:04"} {
		if t, err := time.Parse(layout, valor); err == nil {
			return t.Hour()*60 + t.Minute(), true
		}
	}
	return 0, false
}

// formatClock devolve "HH:MM:SS", o formato que o TimeLog já envia. Passar de
// 23:59 é limitado a 23:59 para não gerar um horário inválido.
func formatClock(minutos int) string {
	minutos = min(max(minutos, 0), 23*60+59)
	return fmt.Sprintf("%02d:%02d:00", minutos/60, minutos%60)
}
