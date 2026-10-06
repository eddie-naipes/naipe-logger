package backend

import (
	"errors"
	"testing"

	"logTime-go/backend/api"
)

func TestBindingsDePlanejamentoExigemConexao(t *testing.T) {
	a := appSemConexao()

	chamadas := map[string]func() error{
		"PlanFillGaps":              func() error { _, err := a.PlanFillGaps(FillGapsRequest{}); return err },
		"GetWeekGrid":               func() error { _, err := a.GetWeekGrid("2026-01-05"); return err },
		"BuildCopyPreviousWeekPlan": func() error { _, err := a.BuildCopyPreviousWeekPlan("2026-01-05"); return err },
	}
	for nome, chamar := range chamadas {
		if err := chamar(); !errors.Is(err, errAPINaoConfigurada) {
			t.Errorf("%s sem conexão = %v, esperava errAPINaoConfigurada", nome, err)
		}
	}
}

func TestTasksForFill(t *testing.T) {
	salvas := []api.Task{{TaskID: 1, TaskName: "A"}, {TaskID: 2, TaskName: "B"}, {TaskID: 3, TaskName: "C"}}
	templates := map[string]api.Template{
		"Padrão": {Name: "Padrão", Tasks: []api.Task{{TaskID: 9}}},
		"Vazio":  {Name: "Vazio"},
	}
	getTemplate := func(n string) (api.Template, bool) { tpl, ok := templates[n]; return tpl, ok }
	getSaved := func() []api.Task { return salvas }

	got, err := tasksForFill(FillGapsRequest{TemplateName: "Padrão", TaskIDs: []int{1}}, getTemplate, getSaved)
	if err != nil || len(got) != 1 || got[0].TaskID != 9 {
		t.Errorf("com template = %v, %v; esperava as tarefas do template", got, err)
	}

	// A ordem é a das tarefas salvas, não a da seleção.
	got, err = tasksForFill(FillGapsRequest{TaskIDs: []int{3, 1}}, getTemplate, getSaved)
	if err != nil || len(got) != 2 || got[0].TaskID != 1 || got[1].TaskID != 3 {
		t.Errorf("com tarefas = %v, %v; esperava [1 3]", got, err)
	}

	for nome, req := range map[string]FillGapsRequest{
		"template inexistente": {TemplateName: "Nenhum"},
		"template vazio":       {TemplateName: "Vazio"},
		"sem seleção":          {},
		"tarefa não salva":     {TaskIDs: []int{42}},
	} {
		if _, err := tasksForFill(req, getTemplate, getSaved); err == nil {
			t.Errorf("%s: esperava erro", nome)
		}
	}
}
