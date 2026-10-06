package backend

import (
	"errors"
	"testing"
	"time"

	"logTime-go/backend/api"
	"logTime-go/backend/audit"
	"logTime-go/backend/config"
	"logTime-go/backend/notify"
	"logTime-go/backend/reminders"
)

// fonteAuditoria imita o cliente: lançamentos no formato de
// GetTimeEntriesForPeriodV2 (fixture real time_v2.json), dias úteis e não
// úteis do mês.
type fonteAuditoria struct {
	entries     []api.TimeEntryReport
	working     []string
	workingErr  error
	nonWorking  []api.NonWorkingDay
	periodo     [2]string
	pediuApagou bool
}

func (f *fonteAuditoria) GetTimeEntriesForPeriodV2(start, end string, includeDeleted bool) ([]api.TimeEntryReport, error) {
	f.periodo = [2]string{start, end}
	f.pediuApagou = includeDeleted
	return f.entries, nil
}

func (f *fonteAuditoria) GetWorkingDays(string, string) ([]string, error) {
	return f.working, f.workingErr
}

func (f *fonteAuditoria) ListNonWorkingDays(int, int) ([]api.NonWorkingDay, error) {
	return f.nonWorking, nil
}

func TestAuditoriaExigeConexaoEValidaMes(t *testing.T) {
	a := appSemConexao()
	if _, err := a.RunMonthAudit(2026, 10); !errors.Is(err, errAPINaoConfigurada) {
		t.Errorf("sem conexão = %v", err)
	}
	if _, err := a.RunMonthAudit(2026, 13); err == nil || errors.Is(err, errAPINaoConfigurada) {
		t.Errorf("mês inválido = %v", err)
	}
	if got := a.GetAuditSettings(); got.DailyLimitMinutes != config.DefaultAuditDailyLimitMinutes {
		t.Errorf("padrões = %+v", got)
	}
}

func TestRunMonthAuditBuscaOMesEAplicaConfiguracao(t *testing.T) {
	fonte := &fonteAuditoria{
		entries: []api.TimeEntryReport{
			{ID: 1, TaskID: 10, TaskName: "Tarefa", UserID: 7, Date: "2026-10-01", Minutes: 480, Description: "Reunião"},
			{ID: 2, TaskID: 10, TaskName: "Tarefa", UserID: 7, Date: "2026-10-03", Minutes: 60, Description: "Plantão"},
		},
		working:    []string{"2026-10-01", "2026-10-02"},
		nonWorking: []api.NonWorkingDay{{Date: "2026-10-03", Type: "weekend", Name: "Saturday"}},
	}
	settings := config.AuditSettings{
		DailyLimitMinutes:   600,
		GenericDescriptions: []string{"reunião"},
		IgnoredIssues:       []string{"incomplete_day:2026-10-02"},
	}
	agora := time.Date(2026, 10, 2, 15, 0, 0, 0, time.Local)

	r, err := runMonthAudit(fonte, 7, 480, settings, 2026, 10, agora)
	if err != nil {
		t.Fatal(err)
	}
	if fonte.periodo != [2]string{"2026-10-01", "2026-10-31"} || fonte.pediuApagou {
		t.Errorf("consulta = %v (excluídos: %v)", fonte.periodo, fonte.pediuApagou)
	}

	tipos := map[audit.IssueType]bool{}
	for _, is := range r.Issues {
		tipos[is.Type] = true
		if is.Type == audit.TypeIncompleteDay && !is.Ignored {
			t.Errorf("dia ignorado voltou: %+v", is)
		}
	}
	for _, want := range []audit.IssueType{audit.TypeGenericDescription, audit.TypeNonWorkingDay, audit.TypeIncompleteDay} {
		if !tipos[want] {
			t.Errorf("faltou %s em %+v", want, r.Issues)
		}
	}
	if r.Summary.IgnoredCount != 1 || r.Summary.ErrorCount != 0 || !r.Summary.Ready || r.PendingCount() != 2 {
		t.Errorf("resumo = %+v", r.Summary)
	}
}

// Mês inteiro de férias: GetWorkingDays falha, e tudo vira dia não útil.
func TestRunMonthAuditMesSemDiasUteis(t *testing.T) {
	fonte := &fonteAuditoria{
		entries:    []api.TimeEntryReport{{ID: 1, TaskID: 10, Date: "2026-10-05", Minutes: 60, Description: "Urgência"}},
		workingErr: errors.New("não foram encontrados dias úteis no período especificado"),
		nonWorking: []api.NonWorkingDay{{Date: "2026-10-05", Type: api.NonWorkingDayVacation, Name: "Férias"}},
	}
	r, err := runMonthAudit(fonte, 7, 0, config.DefaultAuditSettings(), 2026, 10, time.Date(2026, 10, 20, 0, 0, 0, 0, time.Local))
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Issues) != 1 || r.Issues[0].Type != audit.TypeNonWorkingDay || r.Issues[0].Reason != "férias: Férias" {
		t.Errorf("problemas = %+v", r.Issues)
	}
	if r.Summary.MinutesPerDay != 480 {
		t.Errorf("jornada padrão = %d", r.Summary.MinutesPerDay)
	}
}

func TestCliqueNoLembreteDeFechamentoAbreAPagina(t *testing.T) {
	eventos, _, _ := stubJanela(t)
	a, _ := appComRecursos(t, time.Now())
	a.handleNotificationResponse(notify.Response{
		ActionID: reminders.ActionReviewMonth,
		Data:     map[string]any{"kind": reminders.Kind, "route": reminders.RouteMonthClose},
	})
	if len(*eventos) != 1 {
		t.Fatalf("eventos = %v", *eventos)
	}
	if got := (*eventos)[0].dados.(ReminderOpen); got.Route != "/fechamento" {
		t.Errorf("rota = %q", got.Route)
	}
}
