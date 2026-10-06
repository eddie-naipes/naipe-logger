// Comando capturefixtures grava respostas reais e ANONIMIZADAS da API do
// Teamwork em backend/api/testdata, para os testes de contrato.
//
//	go run ./tools/capturefixtures [-out backend/api/testdata]
//
// Usa a conexão configurada no app (config.json + token do cofre do sistema)
// e faz SOMENTE requisições GET, com páginas pequenas, aos endpoints que o
// cliente em backend/api consome. Nada é criado, alterado ou apagado na conta.
// O token e os cabeçalhos nunca são gravados: só o corpo, depois de passar
// pelo anonimizador (ver anonymize.go). Um arquivo em que ainda apareça nome,
// e-mail, empresa ou host do usuário é descartado.
//
// Revise os arquivos gerados antes de comitar.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"logTime-go/backend/api"
	"logTime-go/backend/config"
)

type fixture struct {
	name string
	path string
}

func main() {
	out := flag.String("out", filepath.Join("backend", "api", "testdata"), "diretório de saída")
	from := flag.String("from", "", "reanonimiza os .json deste diretório em vez de consultar a API")
	flag.Parse()

	var err error
	if *from != "" {
		err = reprocess(*from, *out)
	} else {
		err = run(*out)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "erro:", err)
		os.Exit(1)
	}
}

func run(outDir string) error {
	manager, err := config.NewManager()
	if err != nil {
		return fmt.Errorf("configuração: %v", err)
	}
	cfg := manager.GetTeamworkConfig()
	if cfg.AuthToken == "" || cfg.ApiHost == "" || cfg.UserID == 0 {
		return fmt.Errorf("app não está conectado ao Teamwork (configure-o primeiro)")
	}
	client := api.NewTeamworkAPI(cfg)

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}

	uid := cfg.UserID
	agora := time.Now()
	inicioMesAtual := time.Date(agora.Year(), agora.Month(), 1, 0, 0, 0, 0, time.Local)
	// O mês anterior está fechado e tem mais chance de ter lançamentos.
	inicio := inicioMesAtual.AddDate(0, -1, 0)
	fim := inicioMesAtual.AddDate(0, 0, -1)
	ini, fi := inicio.Format("2006-01-02"), fim.Format("2006-01-02")

	raw := map[string][]byte{}
	get := func(f fixture) {
		body, status, err := client.GetJSON(f.path)
		switch {
		case err != nil:
			slog.Warn("Falha ao capturar", "fixture", f.name, "err", err)
		case status != 200:
			slog.Warn("Status inesperado", "fixture", f.name, "status", status)
		default:
			raw[f.name] = body
		}
	}

	// Primeira leva: endpoints que não dependem de IDs de outras respostas.
	for _, f := range []fixture{
		{"me", "/projects/api/v3/me.json"},
		{"tasks", fmt.Sprintf("/projects/api/v3/tasks.json?assignedTo=%d&filter=active&include=projects,tasklists&includeTasklists=true&includeTaskAssignees=true&includeCompletionStatus=true&includeEstimatedTime=true&includeTaskTags=true&page=1&pageSize=3", uid)},
		{"tasks_count", fmt.Sprintf("/projects/api/v3/tasks.json?responsiblePartyIds=%d&page=1&pageSize=1", uid)},
		{"projects", "/projects/api/v3/projects.json?includeProjectUserInfo=true&include=tags,projectTaskStats,projectCategories,companies&projectStatuses=active&page=1&pageSize=3"},
		{"projects_count", "/projects/api/v3/projects.json?projectStatuses=active&page=1&pageSize=1"},
		{"time_v3", fmt.Sprintf("/projects/api/v3/time.json?startDate=%s&endDate=%s&userId=%d&assignedToUserIds=%d&page=1&pageSize=3", ini, fi, uid, uid)},
		{"time_total", fmt.Sprintf("/projects/api/v3/time/total.json?startDate=%s&endDate=%s&userId=%d", ini, fi, uid)},
		{"loggedtime", fmt.Sprintf("/people/%d/loggedtime.json?m=%d&y=%d&projectId=0&page=1&pageSize=100", uid, int(inicio.Month()), inicio.Year())},
		{"time_v2", fmt.Sprintf("/projects/api/v2/time.json?getTotals=true&skipCounts=false&projectId=&companyId=0&userId=%d&assignedTeamIds=&invoicedType=all&billableType=all&fromDate=%s&toDate=%s&sortBy=date&sortOrder=desc&onlyStarredProjects=false&includeArchivedProjects=true&matchAllTags=true&projectStatus=all&showDeleted=0&page=1&pageSize=3", uid, strings.ReplaceAll(ini, "-", ""), strings.ReplaceAll(fi, "-", ""))},
	} {
		get(f)
	}

	// Segunda leva: detalhes, a partir dos IDs da primeira.
	if id := firstID(raw["tasks"], "tasks"); id != 0 {
		get(fixture{"task_detail", fmt.Sprintf("/projects/api/v3/tasks/%d.json?include=projects,tasklists,timeTotals,tags", id)})
	}
	if id := firstID(raw["time_v3"], "timelogs"); id != 0 {
		get(fixture{"time_entry_detail", fmt.Sprintf("/projects/api/v3/time/%d.json", id)})
	}
	if pid := firstID(raw["projects"], "projects"); pid != 0 {
		get(fixture{"project_tasks", fmt.Sprintf("/projects/api/v3/projects/%d/tasks.json?include=projects,tasklists,users,companies,teams,timeTotals,tags,completedBy&includeCustomFields=true&includeLoggedTime=true&page=1&pageSize=3", pid)})
		get(fixture{"tasklists", fmt.Sprintf("/projects/api/v3/projects/%d/tasklists.json?page=1&pageSize=3", pid)})
		get(fixture{"tasks_v1", fmt.Sprintf("/tasks.json?project_id=%d&page=1&pageSize=3", pid)})
	}
	if tlid := firstID(raw["tasklists"], "tasklists"); tlid != 0 {
		get(fixture{"tasklist_tasks", fmt.Sprintf("/projects/api/v3/tasklists/%d/tasks.json?include=projects,tasklists&includeTaskDetails=true&page=1&pageSize=3", tlid)})
	}

	return write(raw, sensitiveTerms(raw["me"], raw["loggedtime"], cfg.ApiHost), outDir)
}

// fixtureNames é a ordem fixa de processamento: o remapeamento de IDs é
// compartilhado entre os arquivos, então a ordem define os IDs fictícios.
var fixtureNames = []string{"me", "tasks", "tasks_count", "task_detail", "projects", "projects_count",
	"project_tasks", "tasklists", "tasklist_tasks", "tasks_v1", "time_v3", "time_entry_detail",
	"time_total", "loggedtime", "time_v2"}

// reprocess passa de novo pelo anonimizador fixtures já gravadas, sem acessar
// a API — útil depois de endurecer as regras de anonimização.
func reprocess(fromDir, outDir string) error {
	raw := map[string][]byte{}
	for _, name := range fixtureNames {
		if data, err := os.ReadFile(filepath.Join(fromDir, name+".json")); err == nil {
			raw[name] = data
		}
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	return write(raw, nil, outDir)
}

func write(raw map[string][]byte, needles []string, outDir string) error {
	anon := newAnonymizer()
	for _, name := range fixtureNames {
		body, ok := raw[name]
		if !ok {
			fmt.Printf("%-18s não capturado\n", name)
			continue
		}
		data, err := anon.Anonymize(body)
		if err != nil {
			slog.Warn("Falha ao anonimizar", "fixture", name, "err", err)
			continue
		}
		if leaks := findLeaks(data, needles); len(leaks) > 0 {
			// Não mostra os termos: eles são justamente os dados pessoais.
			fmt.Printf("%-18s DESCARTADO: %d termo(s) sensível(is) restante(s)\n", name, len(leaks))
			continue
		}
		path := filepath.Join(outDir, name+".json")
		if err := os.WriteFile(path, data, 0o644); err != nil {
			return err
		}
		fmt.Printf("%-18s gravado em %s (%d bytes)\n", name, path, len(data))
	}

	fmt.Println("\nTextos mantidos sem anonimizar (revise):")
	for _, line := range anon.KeptSummary() {
		fmt.Println("  " + line)
	}
	return nil
}

// firstID devolve o id do primeiro item do array `key` da resposta.
func firstID(body []byte, key string) int {
	if body == nil {
		return 0
	}
	var r map[string][]struct {
		ID int `json:"id"`
	}
	// Outras chaves do objeto podem não ser arrays; decodifica só a desejada.
	var generic map[string]json.RawMessage
	if json.Unmarshal(body, &generic) != nil {
		return 0
	}
	if json.Unmarshal([]byte(`{"x":`+string(generic[key])+`}`), &r) != nil {
		return 0
	}
	if items := r["x"]; len(items) > 0 {
		return items[0].ID
	}
	return 0
}

// sensitiveTerms reúne nome, e-mail e empresa do usuário (de me.json e
// loggedtime.json) e o host da conta, para conferir que não sobraram.
func sensitiveTerms(me, loggedtime []byte, host string) []string {
	var terms []string
	var collect func(v any, key string)
	collect = func(v any, key string) {
		switch val := v.(type) {
		case map[string]any:
			for k, item := range val {
				collect(item, k)
			}
		case []any:
			for _, item := range val {
				collect(item, key)
			}
		case string:
			k := strings.ToLower(key)
			if strings.Contains(k, "name") || strings.Contains(k, "email") ||
				strings.Contains(k, "company") || strings.Contains(k, "title") {
				terms = append(terms, val)
				terms = append(terms, strings.Fields(val)...)
			}
		}
	}
	for _, body := range [][]byte{me, loggedtime} {
		var v any
		if json.Unmarshal(body, &v) == nil {
			collect(v, "")
		}
	}

	if u, err := url.Parse(host); err == nil && u.Hostname() != "" {
		terms = append(terms, u.Hostname(), strings.Split(u.Hostname(), ".")[0])
	} else if host != "" {
		terms = append(terms, host)
	}
	return terms
}
