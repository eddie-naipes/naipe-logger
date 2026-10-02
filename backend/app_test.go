package backend

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"logTime-go/backend/api"
)

// appSemConexao monta um App com um cliente sem host nem token, sem passar por
// config.NewManager (que tocaria o HOME e o cofre reais).
func appSemConexao() *App {
	a := &App{}
	a.setAPI(api.NewTeamworkAPI(api.Config{}))
	return a
}

func TestClientSemConexaoDevolveErroPadronizado(t *testing.T) {
	a := appSemConexao()

	if _, err := a.client(); !errors.Is(err, errAPINaoConfigurada) {
		t.Fatalf("client() = %v, esperava errAPINaoConfigurada", err)
	}

	vazio := &App{}
	if _, err := vazio.client(); !errors.Is(err, errAPINaoConfigurada) {
		t.Fatalf("client() sem cliente algum = %v, esperava errAPINaoConfigurada", err)
	}
}

// Os bindings de tarefas e lançamento não checavam a conexão e iam até a
// camada HTTP com host vazio; agora devolvem o erro padronizado de imediato.
func TestBindingsRemotosExigemConexao(t *testing.T) {
	a := appSemConexao()

	chamadas := map[string]func() error{
		"GetTasks":                     func() error { _, err := a.GetTasks(); return err },
		"GetProjects":                  func() error { _, err := a.GetProjects(); return err },
		"GetTasksByProject":            func() error { _, err := a.GetTasksByProject(1); return err },
		"CheckPlanConflicts":           func() error { _, err := a.CheckPlanConflicts(nil); return err },
		"LogMultipleTimes":             func() error { _, err := a.LogMultipleTimes(nil); return err },
		"GetLoggedTimeFromCalendarAPI": func() error { _, err := a.GetLoggedTimeFromCalendarAPI(1, 2026); return err },
		"GetDashboardStats":            func() error { _, err := a.GetDashboardStats(); return err },
		"GetUserProfile":               func() error { _, err := a.GetUserProfile(); return err },
		"GetTimeTotalsForPeriod":       func() error { _, err := a.GetTimeTotalsForPeriod("2026-01-01", "2026-01-31"); return err },
		"DeleteMultipleTimeEntries":    func() error { _, err := a.DeleteMultipleTimeEntries([]int{1}); return err },
		"DownloadTimeReport":           func() error { _, err := a.DownloadTimeReport("2026-01-01", "2026-01-31"); return err },
		"RefreshHolidaysForYear":       func() error { _, err := a.RefreshHolidaysForYear(2026); return err },
	}

	for nome, chamar := range chamadas {
		if err := chamar(); !errors.Is(err, errAPINaoConfigurada) {
			t.Errorf("%s sem conexão = %v, esperava errAPINaoConfigurada", nome, err)
		}
	}
}

// Startup não recria o cliente de NewApp (que perderia o cache de feriados):
// apenas lhe entrega o contexto.
func TestSetContextNaoRecriaCliente(t *testing.T) {
	a := appSemConexao()
	cliente := a.api()

	a.setContext(context.Background())

	if a.api() != cliente {
		t.Fatal("setContext substituiu o cliente")
	}
}

func TestReportFilePathMontaNomeComAsDatas(t *testing.T) {
	padrao := filepath.Join("home", "TeamworkReports", "TeamworkReport_2026-10.pdf")

	got, err := reportFilePath(padrao, "2026-09-01", "2026-09-30")
	if err != nil {
		t.Fatalf("reportFilePath falhou: %v", err)
	}

	want := filepath.Join("home", "TeamworkReports", "TeamworkReport_2026-10_2026-09-01_2026-09-30.pdf")
	if got != want {
		t.Errorf("reportFilePath = %q, esperava %q", got, want)
	}
}

func TestReportFilePathRecusaDatasInvalidas(t *testing.T) {
	padrao := filepath.Join("home", "TeamworkReports", "TeamworkReport_2026-10.pdf")

	casos := []struct{ inicio, fim string }{
		{"../../etc/passwd", "2026-09-30"},
		{"2026-09-01", `..\..\Windows\x`},
		{"2026-09-01/x", "2026-09-30"},
		{"01/09/2026", "30/09/2026"},
		{"", "2026-09-30"},
		{"2026-09-30", "2026-09-01"}, // fim antes do início
	}

	for _, c := range casos {
		if got, err := reportFilePath(padrao, c.inicio, c.fim); err == nil {
			t.Errorf("reportFilePath(%q, %q) = %q, esperava erro", c.inicio, c.fim, got)
		}
	}
}
