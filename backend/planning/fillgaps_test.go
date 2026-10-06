package planning

import (
	"reflect"
	"testing"

	"logTime-go/backend/api"
)

// 2025-09-15 é segunda; 2025-09-16 terça; 2025-09-17 quarta.

func tarefa(id int, workingDays []int, entradas ...api.TimeEntry) api.Task {
	return api.Task{TaskID: id, TaskName: "T", Entries: entradas, WorkingDays: workingDays}
}

func ent(minutos int, horario, descricao string) api.TimeEntry {
	return api.TimeEntry{Minutes: minutos, Time: horario, Description: descricao, IsBillable: true}
}

// lancamentoPlano é a forma compacta de comparar uma entrada do plano.
type lancamentoPlano struct {
	TaskID  int
	Minutos int
	Horario string
}

func achatar(dia api.WorkDay) []lancamentoPlano {
	out := make([]lancamentoPlano, 0, len(dia.Entries))
	for _, e := range dia.Entries {
		out = append(out, lancamentoPlano{e.TaskID, e.Entry.Minutes, e.Entry.Time})
	}
	return out
}

func TestFillGaps(t *testing.T) {
	// Template típico: 4h de manhã, 4h à tarde com intervalo de almoço.
	template := []api.Task{
		tarefa(1, nil, ent(240, "09:00:00", "Manhã")),
		tarefa(2, nil, ent(240, "14:00:00", "Tarde")),
	}

	casos := []struct {
		nome       string
		dias       []string
		lancado    map[string]int
		tarefas    []api.Task
		opts       FillOptions
		esperado   map[string][]lancamentoPlano
		resumoPula map[string]string
	}{
		{
			nome:    "dia vazio recebe o template inteiro com os horários dele",
			dias:    []string{"2025-09-15"},
			tarefas: template,
			opts:    FillOptions{MinutosPorDia: 480, Hoje: "2025-09-30"},
			esperado: map[string][]lancamentoPlano{
				"2025-09-15": {{1, 240, "09:00:00"}, {2, 240, "14:00:00"}},
			},
		},
		{
			nome:    "dia parcial preenche só o que falta, continuando após o que existe",
			dias:    []string{"2025-09-15"},
			lancado: map[string]int{"2025-09-15": 300},
			tarefas: template,
			opts:    FillOptions{MinutosPorDia: 480, Hoje: "2025-09-30"},
			esperado: map[string][]lancamentoPlano{
				// 09:00 + 300 min = 14:00; a primeira entrada é truncada em 180.
				"2025-09-15": {{1, 180, "14:00:00"}},
			},
		},
		{
			nome:       "dia completo fica de fora",
			dias:       []string{"2025-09-15"},
			lancado:    map[string]int{"2025-09-15": 480},
			tarefas:    template,
			opts:       FillOptions{MinutosPorDia: 480, Hoje: "2025-09-30"},
			esperado:   map[string][]lancamentoPlano{},
			resumoPula: map[string]string{"2025-09-15": "completo"},
		},
		{
			nome:       "dia acima da jornada fica de fora",
			dias:       []string{"2025-09-15"},
			lancado:    map[string]int{"2025-09-15": 600},
			tarefas:    template,
			opts:       FillOptions{MinutosPorDia: 480, Hoje: "2025-09-30"},
			esperado:   map[string][]lancamentoPlano{},
			resumoPula: map[string]string{"2025-09-15": "completo"},
		},
		{
			nome:    "feriado não está nos dias úteis e não recebe nada",
			dias:    []string{"2025-09-15", "2025-09-17"}, // 16 seria o feriado
			tarefas: template[:1],
			opts:    FillOptions{MinutosPorDia: 240, Hoje: "2025-09-30"},
			esperado: map[string][]lancamentoPlano{
				"2025-09-15": {{1, 240, "09:00:00"}},
				"2025-09-17": {{1, 240, "09:00:00"}},
			},
		},
		{
			nome:    "trunca a última entrada para não passar da jornada",
			dias:    []string{"2025-09-15"},
			lancado: map[string]int{"2025-09-15": 60},
			tarefas: template,
			opts:    FillOptions{MinutosPorDia: 480, Hoje: "2025-09-30"},
			esperado: map[string][]lancamentoPlano{
				// Cursor 10:00; a tarde mantém 14:00 por ser posterior.
				"2025-09-15": {{1, 240, "10:00:00"}, {2, 180, "14:00:00"}},
			},
		},
		{
			nome:       "déficit menor que a granularidade não gera entrada",
			dias:       []string{"2025-09-15"},
			lancado:    map[string]int{"2025-09-15": 470},
			tarefas:    template,
			opts:       FillOptions{MinutosPorDia: 480, Hoje: "2025-09-30"},
			esperado:   map[string][]lancamentoPlano{},
			resumoPula: map[string]string{"2025-09-15": "abaixo da granularidade"},
		},
		{
			nome:    "granularidade configurável e sobra pequena não vira entrada",
			dias:    []string{"2025-09-15"},
			lancado: map[string]int{"2025-09-15": 0},
			tarefas: []api.Task{
				tarefa(1, nil, ent(445, "09:00:00", "A")),
				tarefa(2, nil, ent(60, "", "B")),
			},
			opts: FillOptions{MinutosPorDia: 480, Granularidade: 60, Hoje: "2025-09-30"},
			esperado: map[string][]lancamentoPlano{
				// Sobram 35 min < 60: a tarefa 2 não entra.
				"2025-09-15": {{1, 445, "09:00:00"}},
			},
		},
		{
			nome: "workingDays: tarefa só de terça não entra na segunda",
			dias: []string{"2025-09-15", "2025-09-16"},
			tarefas: []api.Task{
				tarefa(9, []int{2}, ent(120, "08:00:00", "Daily de terça")),
				tarefa(1, nil, ent(480, "", "Dev")),
			},
			opts: FillOptions{MinutosPorDia: 480, Hoje: "2025-09-30"},
			esperado: map[string][]lancamentoPlano{
				"2025-09-15": {{1, 480, "09:00:00"}},
				"2025-09-16": {{9, 120, "08:00:00"}, {1, 360, "10:00:00"}},
			},
		},
		{
			nome:       "dia futuro fica de fora sem 'mês inteiro'",
			dias:       []string{"2025-09-15", "2025-09-16"},
			tarefas:    template,
			opts:       FillOptions{MinutosPorDia: 480, Hoje: "2025-09-15"},
			esperado:   map[string][]lancamentoPlano{"2025-09-15": {{1, 240, "09:00:00"}, {2, 240, "14:00:00"}}},
			resumoPula: map[string]string{"2025-09-16": "futuro"},
		},
		{
			nome:    "dia futuro entra com 'mês inteiro'",
			dias:    []string{"2025-09-16"},
			tarefas: template[:1],
			opts:    FillOptions{MinutosPorDia: 240, Hoje: "2025-09-15", IncluirFuturos: true},
			esperado: map[string][]lancamentoPlano{
				"2025-09-16": {{1, 240, "09:00:00"}},
			},
		},
		{
			nome:       "sem tarefa para o dia da semana",
			dias:       []string{"2025-09-15"},
			tarefas:    []api.Task{tarefa(9, []int{2}, ent(60, "", "Só terça"))},
			opts:       FillOptions{MinutosPorDia: 480, Hoje: "2025-09-30"},
			esperado:   map[string][]lancamentoPlano{},
			resumoPula: map[string]string{"2025-09-15": "sem tarefa para o dia"},
		},
	}

	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			plano, resumo := FillGaps(c.dias, c.lancado, c.tarefas, c.opts)

			got := make(map[string][]lancamentoPlano)
			for _, dia := range plano {
				got[dia.Date] = achatar(dia)
				soma := 0
				for _, e := range dia.Entries {
					soma += e.Entry.Minutes
				}
				if soma != dia.TotalMin {
					t.Errorf("%s: TotalMin = %d, soma das entradas = %d", dia.Date, dia.TotalMin, soma)
				}
			}
			if !reflect.DeepEqual(got, c.esperado) {
				t.Errorf("plano = %v\nesperava %v", got, c.esperado)
			}

			if len(resumo) != len(c.dias) {
				t.Fatalf("resumo tem %d dias, esperava %d", len(resumo), len(c.dias))
			}
			for _, s := range resumo {
				causa, deveriaPular := c.resumoPula[s.Date]
				if s.Skipped != deveriaPular || s.SkipCause != causa {
					t.Errorf("%s: skipped=%v causa=%q, esperava skipped=%v causa=%q",
						s.Date, s.Skipped, s.SkipCause, deveriaPular, causa)
				}
				if s.Logged+s.ToLog > c.opts.MinutosPorDia && s.ToLog > 0 {
					t.Errorf("%s: lançado %d + a lançar %d ultrapassa a jornada", s.Date, s.Logged, s.ToLog)
				}
			}
		})
	}
}

func TestFillGapsResumoInformaFaltanteMesmoSemEntradasSuficientes(t *testing.T) {
	plano, resumo := FillGaps(
		[]string{"2025-09-15"},
		map[string]int{"2025-09-15": 120},
		[]api.Task{tarefa(1, nil, ent(60, "09:00", "Curta"))},
		FillOptions{MinutosPorDia: 480, Hoje: "2025-09-30"},
	)

	if len(plano) != 1 || len(resumo) != 1 {
		t.Fatalf("plano=%v resumo=%v", plano, resumo)
	}
	s := resumo[0]
	if s.Logged != 120 || s.Missing != 360 || s.ToLog != 60 {
		t.Errorf("resumo = %+v, esperava lançado 120, faltante 360, a lançar 60", s)
	}
	if got := plano[0].Entries[0].Entry.Time; got != "11:00:00" {
		t.Errorf("horário = %q, esperava 11:00:00 (09:00 + 120 min já lançados)", got)
	}
}

func TestFillGapsNaoAlteraAsTarefasRecebidas(t *testing.T) {
	tarefas := []api.Task{tarefa(1, nil, ent(480, "09:00:00", "Dev"))}
	FillGaps([]string{"2025-09-15"}, map[string]int{"2025-09-15": 400}, tarefas, FillOptions{MinutosPorDia: 480})

	if tarefas[0].Entries[0].Minutes != 480 {
		t.Error("FillGaps truncou a entrada original do template")
	}
}
