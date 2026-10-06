package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Testes de contrato: o servidor de teste devolve respostas REAIS da API do
// Teamwork, capturadas e anonimizadas por tools/capturefixtures, e as funções
// do cliente precisam decodificá-las. Três bugs (total em meta.page.totalItems,
// filtro assignedTo ignorado e loggedtime.json com números) passaram porque os
// mocks imitavam um formato que a API não usa; mocks novos devem partir
// destas fixtures.

// fixtureRoutes associa cada endpoint consumido pelo cliente a uma fixture.
// A ordem importa: o primeiro padrão que casar vence.
var fixtureRoutes = []struct {
	pattern *regexp.Regexp
	query   string // parâmetro que precisa estar presente (opcional)
	fixture string
}{
	{regexp.MustCompile(`^/projects/api/v3/me\.json$`), "", "me"},
	{regexp.MustCompile(`^/projects/api/v3/tasks\.json$`), "responsiblePartyIds", "tasks_count"},
	{regexp.MustCompile(`^/projects/api/v3/tasks\.json$`), "", "tasks"},
	{regexp.MustCompile(`^/projects/api/v3/tasks/\d+\.json$`), "", "task_detail"},
	{regexp.MustCompile(`^/projects/api/v3/projects\.json$`), "includeProjectUserInfo", "projects"},
	{regexp.MustCompile(`^/projects/api/v3/projects\.json$`), "", "projects_count"},
	{regexp.MustCompile(`^/projects/api/v3/projects/\d+/tasks\.json$`), "", "project_tasks"},
	{regexp.MustCompile(`^/projects/api/v3/projects/\d+/tasklists\.json$`), "", "tasklists"},
	{regexp.MustCompile(`^/projects/api/v3/tasklists/\d+/tasks\.json$`), "", "tasklist_tasks"},
	{regexp.MustCompile(`^/tasks\.json$`), "", "tasks_v1"},
	{regexp.MustCompile(`^/projects/api/v3/time\.json$`), "", "time_v3"},
	{regexp.MustCompile(`^/projects/api/v3/time/total\.json$`), "", "time_total"},
	{regexp.MustCompile(`^/projects/api/v3/time/\d+\.json$`), "", "time_entry_detail"},
	{regexp.MustCompile(`^/people/\d+/loggedtime\.json$`), "", "loggedtime"},
	{regexp.MustCompile(`^/projects/api/v2/time\.json$`), "", "time_v2"},
}

func loadFixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name+".json"))
	if err != nil {
		t.Fatalf("fixture %s: %v", name, err)
	}
	return data
}

// newContractAPI sobe o servidor de fixtures. A página 1 devolve a resposta
// real (capturada com pageSize pequeno, então hasMore pode ser true); as
// seguintes devolvem uma página vazia, o que encerra a paginação.
func newContractAPI(t *testing.T) *TeamworkAPI {
	t.Helper()
	api, _ := newTestAPI(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("método inesperado %s %s", r.Method, r.URL.Path)
			http.Error(w, "somente GET", http.StatusMethodNotAllowed)
			return
		}
		if p := r.URL.Query().Get("page"); p != "" && p != "1" {
			fmt.Fprint(w, `{}`)
			return
		}
		for _, route := range fixtureRoutes {
			if !route.pattern.MatchString(r.URL.Path) {
				continue
			}
			if route.query != "" && !r.URL.Query().Has(route.query) {
				continue
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(loadFixture(t, route.fixture))
			return
		}
		t.Errorf("endpoint sem fixture: %s", r.URL.String())
		http.NotFound(w, r)
	})

	// O usuário configurado é o da fixture me.json, para que filtros como
	// "atribuídas a mim" casem com os IDs (já remapeados) das demais.
	api.Config.UserID = fixtureUserID(t)
	return api
}

func fixtureUserID(t *testing.T) int {
	t.Helper()
	var me struct {
		Person struct {
			ID int `json:"id"`
		} `json:"person"`
	}
	if err := json.Unmarshal(loadFixture(t, "me"), &me); err != nil || me.Person.ID == 0 {
		t.Fatalf("me.json sem person.id: %v", err)
	}
	return me.Person.ID
}

// fixtureCount lê meta.page.count da fixture, o total que a v3 informa.
func fixtureCount(t *testing.T, name string) int {
	t.Helper()
	var r struct {
		Meta struct {
			Page struct {
				Count *int `json:"count"`
			} `json:"page"`
		} `json:"meta"`
	}
	if err := json.Unmarshal(loadFixture(t, name), &r); err != nil || r.Meta.Page.Count == nil {
		t.Fatalf("%s sem meta.page.count: %v", name, err)
	}
	return *r.Meta.Page.Count
}

var reDataISO = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

func TestContratoMe(t *testing.T) {
	api := newContractAPI(t)
	id, err := api.GetCurrentUserId()
	if err != nil {
		t.Fatalf("GetCurrentUserId: %v", err)
	}
	if id != fixtureUserID(t) {
		t.Errorf("GetCurrentUserId = %d, esperava %d", id, fixtureUserID(t))
	}
}

func TestContratoContagens(t *testing.T) {
	api := newContractAPI(t)

	tarefas, err := api.GetTaskCount()
	if err != nil {
		t.Fatalf("GetTaskCount: %v", err)
	}
	if want := fixtureCount(t, "tasks_count"); tarefas != want {
		t.Errorf("GetTaskCount = %d, esperava %d (meta.page.count)", tarefas, want)
	}

	projetos, err := api.GetProjectCount()
	if err != nil {
		t.Fatalf("GetProjectCount: %v", err)
	}
	want := fixtureCount(t, "projects_count")
	if projetos != want {
		t.Errorf("GetProjectCount = %d, esperava %d (meta.page.count)", projetos, want)
	}
	if want > 0 && projetos == 0 {
		t.Error("contagem de projetos zerada com fixture não-zero")
	}
}

func TestContratoProjetos(t *testing.T) {
	api := newContractAPI(t)
	projetos, err := api.GetProjects()
	if err != nil {
		t.Fatalf("GetProjects: %v", err)
	}
	if len(projetos) == 0 {
		t.Fatal("nenhum projeto decodificado")
	}
	for _, p := range projetos {
		if p.ID == 0 || p.Name == "" {
			t.Errorf("projeto incompleto: %+v", p)
		}
	}
}

func checarTarefas(t *testing.T, origem string, tarefas []TeamworkTask) {
	t.Helper()
	if len(tarefas) == 0 {
		t.Fatalf("%s: nenhuma tarefa decodificada", origem)
	}
	resolvidas := 0
	for _, task := range tarefas {
		if task.ID == 0 {
			t.Errorf("%s: tarefa sem ID: %+v", origem, task)
		}
		if task.Content == "" || isPlaceholderName(task.Content) {
			t.Errorf("%s: tarefa %d sem nome real: %q", origem, task.ID, task.Content)
		}
		if task.ProjectName != "" && task.TasklistName != "" {
			resolvidas++
		}
	}
	// included precisa resolver lista e projeto, senão a tela mostra a
	// tarefa sem contexto e o cliente cai em N+1 requisições de detalhe.
	if resolvidas == 0 {
		t.Errorf("%s: nenhuma tarefa com projeto e lista resolvidos via included", origem)
	}
}

func TestContratoTarefas(t *testing.T) {
	api := newContractAPI(t)

	tarefas, err := api.GetTasks()
	if err != nil {
		t.Fatalf("GetTasks: %v", err)
	}
	checarTarefas(t, "GetTasks", tarefas)

	porLista, err := api.GetTasksByTasklist(1)
	if err != nil {
		t.Fatalf("GetTasksByTasklist: %v", err)
	}
	checarTarefas(t, "GetTasksByTasklist", porLista)

	porProjeto, err := api.GetTasksByProject(1)
	if err != nil {
		t.Fatalf("GetTasksByProject: %v", err)
	}
	if len(porProjeto) == 0 {
		t.Error("GetTasksByProject: nenhuma tarefa decodificada")
	}

	listas, err := api.GetTasklistsByProject(1)
	if err != nil {
		t.Fatalf("GetTasklistsByProject: %v", err)
	}
	if len(listas) == 0 || listas[0].ID == 0 || listas[0].Name == "" {
		t.Errorf("GetTasklistsByProject = %+v", listas)
	}
}

func TestContratoDetalheDaTarefa(t *testing.T) {
	api := newContractAPI(t)
	task, err := api.GetTaskDetails(1)
	if err != nil {
		t.Fatalf("GetTaskDetails: %v", err)
	}
	if task.ID == 0 || task.Name == "" {
		t.Errorf("detalhe incompleto: %+v", task)
	}
	if task.TasklistID != 0 && task.TasklistName == "" {
		t.Errorf("lista %d não resolvida via included", task.TasklistID)
	}
	if task.ProjectID == 0 || task.ProjectName == "" {
		t.Errorf("projeto não resolvido: id=%d nome=%q", task.ProjectID, task.ProjectName)
	}
}

func checarLancamentos(t *testing.T, origem string, entries []TimeEntryReport) {
	t.Helper()
	if len(entries) == 0 {
		t.Fatalf("%s: nenhum lançamento decodificado", origem)
	}
	for _, e := range entries {
		if e.ID == 0 || e.TaskID == 0 || e.ProjectID == 0 {
			t.Errorf("%s: lançamento sem IDs: %+v", origem, e)
		}
		if e.Minutes <= 0 {
			t.Errorf("%s: lançamento %d com %d minutos", origem, e.ID, e.Minutes)
		}
		if !reDataISO.MatchString(e.Date) {
			t.Errorf("%s: lançamento %d com data %q fora de YYYY-MM-DD", origem, e.ID, e.Date)
		}
	}
}

func TestContratoLancamentosV3(t *testing.T) {
	api := newContractAPI(t)
	entries, err := api.GetTimeEntriesForPeriod("2026-09-01", "2026-09-30")
	if err != nil {
		t.Fatalf("GetTimeEntriesForPeriod: %v", err)
	}
	checarLancamentos(t, "GetTimeEntriesForPeriod", entries)
}

func TestContratoLancamentosV2(t *testing.T) {
	api := newContractAPI(t)
	entries, err := api.GetTimeEntriesForPeriodV2("2026-09-01", "2026-09-30", false)
	if err != nil {
		t.Fatalf("GetTimeEntriesForPeriodV2: %v", err)
	}
	checarLancamentos(t, "GetTimeEntriesForPeriodV2", entries)
	for _, e := range entries {
		if e.ProjectName == "" || e.TaskName == "" {
			t.Errorf("lançamento %d sem nomes de projeto/tarefa", e.ID)
		}
	}
}

func TestContratoDetalheDoLancamento(t *testing.T) {
	api := newContractAPI(t)
	entry, err := api.GetTimeEntryDetails(1)
	if err != nil {
		t.Fatalf("GetTimeEntryDetails: %v", err)
	}
	if entry.ID == 0 || entry.Minutes <= 0 || entry.TaskID == 0 {
		t.Errorf("detalhe do lançamento incompleto: %+v", entry)
	}
	if !reDataISO.MatchString(entry.Date) {
		t.Errorf("data %q fora de YYYY-MM-DD", entry.Date)
	}
}

func TestContratoTotais(t *testing.T) {
	api := newContractAPI(t)
	total, err := api.GetTimeTotalsForPeriod("2026-09-01", "2026-09-30")
	if err != nil {
		t.Fatalf("GetTimeTotalsForPeriod: %v", err)
	}
	if total.TimeTotals.Minutes <= 0 {
		t.Errorf("time-totals.minutes = %d, a fixture tem horas no período", total.TimeTotals.Minutes)
	}

	horas, err := api.GetHoursLoggedInPeriod("2026-09-01", "2026-09-30")
	if err != nil {
		t.Fatalf("GetHoursLoggedInPeriod: %v", err)
	}
	if horas != float64(total.TimeTotals.Minutes)/60 {
		t.Errorf("GetHoursLoggedInPeriod = %v, esperava %v", horas, float64(total.TimeTotals.Minutes)/60)
	}
}

func TestContratoCalendario(t *testing.T) {
	api := newContractAPI(t)
	resp, err := api.GetLoggedTimeFromCalendarAPI(9, 2026)
	if err != nil {
		t.Fatalf("GetLoggedTimeFromCalendarAPI: %v", err)
	}
	if len(resp.User.Billable) == 0 {
		t.Fatal("nenhum dia faturável decodificado")
	}
	minutos := 0
	for _, dia := range resp.User.Billable {
		if !regexp.MustCompile(`^\d{13}$`).MatchString(string(dia[0])) {
			t.Errorf("timestamp do dia = %q", dia[0])
		}
		var m int
		if _, err := fmt.Sscan(strings.TrimSpace(string(dia[2])), &m); err != nil {
			t.Errorf("minutos do dia %q não numéricos: %v", dia[2], err)
		}
		minutos += m
	}
	if minutos == 0 {
		t.Error("calendário sem minutos, mas a fixture tem horas lançadas")
	}
}
