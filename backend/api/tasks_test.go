package api

import (
	"fmt"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
)

// Página v3 de tarefas: projectId não vem na tarefa, só pela lista.
const tarefasPagina1 = `{
	"tasks":[
		{"id":1,"name":"Revisar contrato","tasklistId":10},
		{"id":2,"name":"Deploy","tasklist":{"id":20,"type":"tasklists"}}
	],
	"included":{
		"tasklists":{"10":{"id":10,"name":"Jurídico","projectId":100},"20":{"id":20,"name":"Infra","projectId":200}},
		"projects":{"100":{"id":100,"name":"Projeto A"},"200":{"id":200,"name":"Projeto B"}}
	},
	"meta":{"page":{"hasMore":true}}
}`

const tarefasPagina2 = `{
	"tasks":[{"id":3,"name":"","tasklistId":30}],
	"included":{"tasklists":{"30":{"id":30,"name":"Lista sem projeto"}}},
	"meta":{"page":{"hasMore":false}}
}`

func TestGetTasksResolveIncludedPaginaEEvitaNMais1(t *testing.T) {
	var detalhes, paginas int32
	api, _ := newTestAPI(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/projects/api/v3/tasks.json":
			atomic.AddInt32(&paginas, 1)
			include := r.URL.Query().Get("include")
			if !strings.Contains(include, "projects") || !strings.Contains(include, "tasklists") {
				t.Errorf("include = %q, esperava projects e tasklists", include)
			}
			if r.URL.Query().Get("pageSize") != fmt.Sprint(listPageSize) {
				t.Errorf("pageSize = %q, esperava %d", r.URL.Query().Get("pageSize"), listPageSize)
			}
			if r.URL.Query().Get("page") == "1" {
				fmt.Fprint(w, tarefasPagina1)
			} else {
				fmt.Fprint(w, tarefasPagina2)
			}
		case strings.HasPrefix(r.URL.Path, "/projects/api/v3/tasks/"):
			atomic.AddInt32(&detalhes, 1)
			// Só a tarefa 3 (sem nome e sem projeto) deveria chegar aqui.
			if r.URL.Path != "/projects/api/v3/tasks/3.json" {
				t.Errorf("detalhe pedido para %q; só a tarefa incompleta deveria ser consultada", r.URL.Path)
			}
			fmt.Fprint(w, `{"task":{"id":3,"name":"Tarefa real","tasklistId":30},
				"included":{"tasklists":{"30":{"name":"Lista sem projeto","projectId":300}},
				"projects":{"300":{"name":"Projeto C"}}}}`)
		default:
			t.Errorf("caminho inesperado %q", r.URL.Path)
		}
	})

	tasks, err := api.GetTasks()
	if err != nil {
		t.Fatalf("GetTasks: %v", err)
	}
	if len(tasks) != 3 {
		t.Fatalf("devolveu %d tarefas, esperava 3 (duas páginas)", len(tasks))
	}

	want := []struct {
		content, project, tasklist string
		projectID                  int
	}{
		{"Revisar contrato", "Projeto A", "Jurídico", 100},
		{"Deploy", "Projeto B", "Infra", 200},
		{"Tarefa real", "Projeto C", "Lista sem projeto", 300},
	}
	for i, w := range want {
		got := tasks[i]
		if got.Content != w.content || got.ProjectName != w.project || got.TasklistName != w.tasklist || got.ProjectID != w.projectID {
			t.Errorf("tarefa %d = %+v, esperava %+v", i, got, w)
		}
	}

	if got := atomic.LoadInt32(&paginas); got != 2 {
		t.Errorf("listagem chamada %d vezes, esperava 2 páginas", got)
	}
	if got := atomic.LoadInt32(&detalhes); got != 1 {
		t.Errorf("detalhe consultado %d vezes, esperava 1 (só a tarefa incompleta)", got)
	}
}

func TestNomeDaTarefaTemPrioridadeSobreNomeDaLista(t *testing.T) {
	resp := TasksResponse{}
	resp.Tasks = []taskWire{{TeamworkTask: TeamworkTask{ID: 1, Name: "Nome real", TasklistID: 10}}}
	resp.Included.TaskLists = map[string]struct {
		ID        int    `json:"id"`
		Name      string `json:"name"`
		ProjectID int    `json:"projectId"`
	}{"10": {ID: 10, Name: "Nome da lista"}}

	tasks := resp.resolvedTasks()
	if tasks[0].Content != "Nome real" {
		t.Errorf("Content = %q, esperava o nome da tarefa e não o da lista", tasks[0].Content)
	}
	if tasks[0].TasklistName != "Nome da lista" {
		t.Errorf("TasklistName = %q", tasks[0].TasklistName)
	}

	// Sem nome nenhum, vira um marcador identificável, nunca o nome da lista.
	if got := taskDisplayName(TeamworkTask{ID: 7, TasklistName: "Lista"}); got != "Tarefa #7" {
		t.Errorf("taskDisplayName sem nome = %q, esperava \"Tarefa #7\"", got)
	}
	// Um marcador antigo cede lugar ao nome real.
	if got := taskDisplayName(TeamworkTask{ID: 7, Content: "Tarefa #7", Name: "Real"}); got != "Real" {
		t.Errorf("taskDisplayName com marcador = %q, esperava \"Real\"", got)
	}
}

func TestEnrichTaskDetailUsaNameSemRequisicao(t *testing.T) {
	var chamadas int32
	api, _ := newTestAPI(t, func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&chamadas, 1)
		w.WriteHeader(http.StatusNotFound)
	})

	task := TeamworkTask{ID: 1, Name: "Já tenho nome", ProjectID: 5, ProjectName: "P"}
	api.enrichTaskDetail(&task)

	if task.Content != "Já tenho nome" {
		t.Errorf("Content = %q, esperava o Name da tarefa", task.Content)
	}
	if got := atomic.LoadInt32(&chamadas); got != 0 {
		t.Errorf("fez %d requisições; com nome e projeto não há o que buscar", got)
	}
}

func TestGetProjectsPagina(t *testing.T) {
	api, _ := newTestAPI(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("page") == "1" {
			fmt.Fprint(w, `{"projects":[{"id":1,"name":"A"}],"meta":{"page":{"hasMore":true}}}`)
			return
		}
		fmt.Fprint(w, `{"projects":[{"id":2,"name":"B"}],"meta":{"page":{"hasMore":false}}}`)
	})

	projects, err := api.GetProjects()
	if err != nil {
		t.Fatalf("GetProjects: %v", err)
	}
	if len(projects) != 2 || projects[0].ID != 1 || projects[1].ID != 2 {
		t.Errorf("projetos = %+v, esperava [1 2]", projects)
	}
}

func TestGetTasksByProjectCaiParaListasEmOrdem(t *testing.T) {
	var porLista int32
	api, _ := newTestAPI(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/projects/api/v3/projects/100/tasks.json":
			w.WriteHeader(http.StatusForbidden)
		case "/projects/api/v3/projects/100/tasklists.json":
			fmt.Fprint(w, `{"tasklists":[{"id":1,"name":"L1"},{"id":2,"name":"L2"},{"id":3,"name":"L3"}],"meta":{"page":{"hasMore":false}}}`)
		case "/projects/api/v3/projects.json":
			fmt.Fprint(w, `{"projects":[{"id":100,"name":"Projeto A"}],"meta":{"page":{"hasMore":false}}}`)
		default:
			if !strings.HasPrefix(r.URL.Path, "/projects/api/v3/tasklists/") {
				t.Errorf("caminho inesperado %q", r.URL.Path)
				return
			}
			atomic.AddInt32(&porLista, 1)
			var id int
			fmt.Sscanf(strings.TrimPrefix(r.URL.Path, "/projects/api/v3/tasklists/"), "%d", &id)
			fmt.Fprintf(w, `{"tasks":[{"id":%d,"name":"T%d","tasklistId":%d}],"meta":{"page":{"hasMore":false}}}`, id*10, id, id)
		}
	})

	tasks, err := api.GetTasksByProject(100)
	if err != nil {
		t.Fatalf("GetTasksByProject: %v", err)
	}
	if len(tasks) != 3 {
		t.Fatalf("devolveu %d tarefas, esperava 3", len(tasks))
	}
	for i, wantID := range []int{10, 20, 30} {
		if tasks[i].ID != wantID {
			t.Errorf("posição %d: id = %d, esperava %d (ordem das listas)", i, tasks[i].ID, wantID)
		}
		if tasks[i].ProjectID != 100 || tasks[i].ProjectName != "Projeto A" {
			t.Errorf("tarefa %d sem contexto do projeto: %+v", i, tasks[i])
		}
	}
	if got := atomic.LoadInt32(&porLista); got != 3 {
		t.Errorf("listas consultadas %d vezes, esperava 3", got)
	}
}
