package audit

import (
	"reflect"
	"testing"

	"logTime-go/backend/api"
)

// Os lançamentos seguem o formato que GetTimeEntriesForPeriodV2 produz a
// partir da fixture real backend/api/testdata/time_v2.json (data AAAA-MM-DD,
// minutos totais, IDs de tarefa/projeto e nomes).
const userID = 1004

func entry(id int, date string, minutes int, taskID int, desc string) api.TimeEntryReport {
	return api.TimeEntryReport{
		ID:          id,
		ProjectID:   1007,
		ProjectName: "Nome ficticio 5",
		TaskID:      taskID,
		TaskName:    "Nome ficticio 26",
		UserID:      userID,
		Date:        date,
		Minutes:     minutes,
		Description: desc,
		IsBillable:  true,
		StartTime:   "09:00:00",
	}
}

// Outubro de 2026: dia 1 é quinta. Dias úteis da semana 1-2 e 5-9, mais o 12
// (feriado nacional) fora.
var workingOct = []string{"2026-10-01", "2026-10-02", "2026-10-05", "2026-10-06", "2026-10-07", "2026-10-08", "2026-10-09"}

var nonWorkingOct = []api.NonWorkingDay{
	{Date: "2026-10-03", Type: "weekend", Name: "Saturday"},
	{Date: "2026-10-04", Type: "weekend", Name: "Sunday"},
	{Date: "2026-10-12", Type: "holiday", Name: "Nossa Senhora Aparecida"},
	{Date: "2026-10-13", Type: api.NonWorkingDayBridge, Name: "Ponte"},
	{Date: "2026-10-14", Type: api.NonWorkingDayVacation, Name: "Férias"},
	{Date: "2026-10-15", Type: api.NonWorkingDayStateHoliday, Name: "Dia do Professor"},
	{Date: "2026-10-16", Type: api.NonWorkingDayMunicipal, Name: "Aniversário da cidade"},
}

func baseInput(entries ...api.TimeEntryReport) Input {
	return Input{
		Year:              2026,
		Month:             10,
		Today:             "2026-10-02",
		Entries:           entries,
		UserID:            userID,
		WorkingDays:       workingOct,
		NonWorkingDays:    nonWorkingOct,
		MinutesPerDay:     480,
		DailyLimitMinutes: DefaultDailyLimitMinutes,
	}
}

// fullDays completa a jornada de 1 e 2/10 com lançamentos válidos, para que
// cada caso só veja o problema que monta.
func fullDays() []api.TimeEntryReport {
	return []api.TimeEntryReport{
		entry(1, "2026-10-01", 480, 10, "Implementa tela de login"),
		entry(2, "2026-10-02", 480, 10, "Corrige relatório mensal"),
	}
}

func keysOf(issues []Issue, t IssueType) []string {
	var out []string
	for _, is := range issues {
		if is.Type == t {
			out = append(out, is.Key)
		}
	}
	return out
}

func TestRunMesSemProblemasFicaProntoParaEntregar(t *testing.T) {
	r := Run(baseInput(fullDays()...))
	if len(r.Issues) != 0 {
		t.Fatalf("esperava nenhum problema, veio %+v", r.Issues)
	}
	if !r.Summary.Ready || r.Summary.LoggedMinutes != 960 || r.Summary.ExpectedToDateMinutes != 960 ||
		r.Summary.ExpectedMinutes != 7*480 || r.Summary.WorkingDaysToDate != 2 || r.Summary.WorkingDays != 7 {
		t.Errorf("resumo = %+v", r.Summary)
	}
	if r.Issues == nil || r.Counts == nil {
		t.Error("listas vazias devem ser [] e não nil (viram null no JS)")
	}
}

func TestRunVerificacoes(t *testing.T) {
	casos := []struct {
		nome     string
		extra    []api.TimeEntryReport
		mutate   func(*Input)
		tipo     IssueType
		chaves   []string
		severity Severity
		action   Action
	}{
		{
			nome:     "dia útil até hoje abaixo da jornada",
			mutate:   func(in *Input) { in.Today = "2026-10-05" },
			tipo:     TypeIncompleteDay,
			chaves:   []string{"incomplete_day:2026-10-05"},
			severity: SeverityError,
			action:   ActionCompletePeriod,
		},
		{
			nome:     "sem descrição (só espaços)",
			extra:    []api.TimeEntryReport{entry(30, "2026-10-01", 30, 10, "   ")},
			tipo:     TypeMissingDescription,
			chaves:   []string{"missing_description:30"},
			severity: SeverityError,
			action:   ActionEdit,
		},
		{
			nome:     "descrição igual ao nome da tarefa",
			extra:    []api.TimeEntryReport{entry(31, "2026-10-01", 30, 10, "nome FICTICIO 26.")},
			tipo:     TypeGenericDescription,
			chaves:   []string{"generic_description:31"},
			severity: SeverityWarning,
			action:   ActionEdit,
		},
		{
			nome:     "descrição na lista de genéricas",
			extra:    []api.TimeEntryReport{entry(32, "2026-10-02", 30, 10, "Reunião")},
			mutate:   func(in *Input) { in.GenericDescriptions = []string{"  reunião ", "ajustes"} },
			tipo:     TypeGenericDescription,
			chaves:   []string{"generic_description:32"},
			severity: SeverityWarning,
			action:   ActionEdit,
		},
		{
			nome: "duplicatas mesma tarefa, dia, minutos e descrição",
			extra: []api.TimeEntryReport{
				entry(41, "2026-10-01", 30, 10, "Daily"),
				entry(40, "2026-10-01", 30, 10, "daily "),
				entry(42, "2026-10-01", 30, 10, "Daily"),
			},
			tipo:     TypeDuplicate,
			chaves:   []string{"duplicate:40,41,42"},
			severity: SeverityError,
			action:   ActionDeleteDuplicates,
		},
		{
			nome:     "fim de semana",
			extra:    []api.TimeEntryReport{entry(50, "2026-10-03", 60, 10, "Plantão")},
			tipo:     TypeNonWorkingDay,
			chaves:   []string{"non_working_day:50"},
			severity: SeverityWarning,
			action:   ActionEditOrDelete,
		},
		{
			nome:     "acima do limite diário",
			extra:    []api.TimeEntryReport{entry(60, "2026-10-02", 180, 11, "Deploy noturno")},
			tipo:     TypeOverDailyLimit,
			chaves:   []string{"over_daily_limit:2026-10-02"},
			severity: SeverityWarning,
			action:   ActionViewDay,
		},
		{
			nome:     "sem tarefa",
			extra:    []api.TimeEntryReport{entry(70, "2026-10-01", 30, 0, "Atendimento")},
			tipo:     TypeNoTask,
			chaves:   []string{"no_task:70"},
			severity: SeverityWarning,
			action:   ActionEdit,
		},
	}

	for _, c := range casos {
		t.Run(c.nome, func(t *testing.T) {
			in := baseInput(append(fullDays(), c.extra...)...)
			if c.mutate != nil {
				c.mutate(&in)
			}
			r := Run(in)
			if got := keysOf(r.Issues, c.tipo); !reflect.DeepEqual(got, c.chaves) {
				t.Fatalf("chaves %s = %v, esperava %v (todos: %+v)", c.tipo, got, c.chaves, r.Issues)
			}
			for _, is := range r.Issues {
				if is.Type != c.tipo {
					t.Errorf("problema inesperado: %+v", is)
					continue
				}
				if is.Severity != c.severity || is.Action != c.action || is.Message == "" {
					t.Errorf("problema = %+v", is)
				}
			}
		})
	}
}

func TestDuplicataMantemOMaisAntigoEApagaOsDemais(t *testing.T) {
	in := baseInput(append(fullDays(),
		entry(41, "2026-10-01", 60, 10, "Daily"),
		entry(40, "2026-10-01", 60, 10, "Daily"),
		// Mesmo texto mas outros minutos, outra tarefa ou outro dia: não é
		// duplicata.
		entry(43, "2026-10-01", 45, 10, "Daily"),
		entry(44, "2026-10-01", 60, 11, "Daily"),
		entry(45, "2026-10-02", 60, 10, "Daily"),
	)...)
	in.DailyLimitMinutes = 0
	r := Run(in)
	dups := keysOf(r.Issues, TypeDuplicate)
	if len(dups) != 1 {
		t.Fatalf("duplicatas = %v", dups)
	}
	for _, is := range r.Issues {
		if is.Type != TypeDuplicate {
			continue
		}
		if is.KeepEntryID != 40 || !reflect.DeepEqual(is.DeleteEntryIDs, []int{41}) || !reflect.DeepEqual(is.EntryIDs, []int{40, 41}) {
			t.Errorf("duplicata = %+v", is)
		}
	}
}

func TestDiaIncompletoIgnoraFuturoEDiasNaoUteis(t *testing.T) {
	in := baseInput()
	in.Today = "2026-10-06"
	r := Run(in)
	want := []string{
		"incomplete_day:2026-10-01", "incomplete_day:2026-10-02",
		"incomplete_day:2026-10-05", "incomplete_day:2026-10-06",
	}
	if got := keysOf(r.Issues, TypeIncompleteDay); !reflect.DeepEqual(got, want) {
		t.Errorf("dias incompletos = %v", got)
	}
	if r.Issues[0].MissingMinutes != 480 || r.Summary.Ready {
		t.Errorf("primeiro = %+v, resumo = %+v", r.Issues[0], r.Summary)
	}
}

func TestMotivoDoDiaNaoUtil(t *testing.T) {
	casos := map[string]string{
		"2026-10-03": "sábado",
		"2026-10-04": "domingo",
		"2026-10-12": "feriado nacional: Nossa Senhora Aparecida",
		"2026-10-13": "ponte: Ponte",
		"2026-10-14": "férias: Férias",
		"2026-10-15": "feriado estadual: Dia do Professor",
		"2026-10-16": "feriado municipal: Aniversário da cidade",
	}
	var entries []api.TimeEntryReport
	id := 100
	for d := range casos {
		entries = append(entries, entry(id, d, 60, 10, "Trabalho em "+d))
		id++
	}
	in := baseInput(append(fullDays(), entries...)...)
	r := Run(in)
	vistos := 0
	for _, is := range r.Issues {
		if is.Type != TypeNonWorkingDay {
			continue
		}
		vistos++
		if want := casos[is.Date]; is.Reason != want {
			t.Errorf("%s: motivo = %q, esperava %q", is.Date, is.Reason, want)
		}
	}
	if vistos != len(casos) {
		t.Errorf("esperava %d lançamentos em dia não útil, veio %d", len(casos), vistos)
	}
}

func TestFiltraExcluidosOutrosUsuariosEForaDoMes(t *testing.T) {
	apagado := entry(80, "2026-10-01", 30, 0, "")
	apagado.DeletedAt = "2026-10-02T10:00:00Z"
	outro := entry(81, "2026-10-01", 30, 0, "")
	outro.UserID = 999
	foraDoMes := entry(82, "2026-09-30", 30, 0, "")
	r := Run(baseInput(append(fullDays(), apagado, outro, foraDoMes)...))
	if len(r.Issues) != 0 || r.Summary.EntryCount != 2 {
		t.Errorf("problemas = %+v, resumo = %+v", r.Issues, r.Summary)
	}
}

func TestIgnoradosNaoContamMasContinuamNaLista(t *testing.T) {
	in := baseInput(append(fullDays(), entry(30, "2026-10-01", 30, 10, ""))...)
	in.Ignored = []string{"missing_description:30"}
	r := Run(in)
	if len(r.Issues) != 1 || !r.Issues[0].Ignored {
		t.Fatalf("problemas = %+v", r.Issues)
	}
	if r.Summary.ErrorCount != 0 || r.Summary.IgnoredCount != 1 || !r.Summary.Ready || len(r.Counts) != 0 || r.PendingCount() != 0 {
		t.Errorf("resumo = %+v, contagens = %+v", r.Summary, r.Counts)
	}
}

// As chaves não podem depender da ordem em que o Teamwork devolve os
// lançamentos, senão um problema ignorado voltaria a aparecer.
func TestChavesEstaveis(t *testing.T) {
	a := []api.TimeEntryReport{
		entry(42, "2026-10-01", 60, 10, "Daily"),
		entry(40, "2026-10-01", 60, 10, "Daily"),
	}
	b := []api.TimeEntryReport{a[1], a[0]}
	ra := Run(baseInput(append(fullDays(), a...)...))
	rb := Run(baseInput(append(fullDays(), b...)...))
	if !reflect.DeepEqual(keysOf(ra.Issues, TypeDuplicate), keysOf(rb.Issues, TypeDuplicate)) {
		t.Errorf("chaves diferentes: %v x %v", keysOf(ra.Issues, TypeDuplicate), keysOf(rb.Issues, TypeDuplicate))
	}
	if got := idsKey(TypeDuplicate, []int{3, 1, 2}); got != "duplicate:1,2,3" {
		t.Errorf("idsKey = %q", got)
	}
	if got := dayKey(TypeOverDailyLimit, "2026-10-02"); got != "over_daily_limit:2026-10-02" {
		t.Errorf("dayKey = %q", got)
	}
}

func TestContagensPorTipoNaOrdemDeExibicao(t *testing.T) {
	in := baseInput(append(fullDays(),
		entry(70, "2026-10-01", 30, 0, "Atendimento"),
		entry(30, "2026-10-01", 30, 10, ""),
		entry(31, "2026-10-02", 30, 10, ""),
	)...)
	in.DailyLimitMinutes = 0
	r := Run(in)
	want := []TypeCount{
		{Type: TypeMissingDescription, Severity: SeverityError, Count: 2},
		{Type: TypeNoTask, Severity: SeverityWarning, Count: 1},
	}
	if !reflect.DeepEqual(r.Counts, want) {
		t.Errorf("contagens = %+v", r.Counts)
	}
	if r.Summary.ErrorCount != 2 || r.Summary.WarningCount != 1 || r.Summary.Ready {
		t.Errorf("resumo = %+v", r.Summary)
	}
}

func TestLimiteDiarioZeroDesliga(t *testing.T) {
	in := baseInput(entry(1, "2026-10-01", 900, 10, "Maratona"), entry(2, "2026-10-02", 480, 10, "Normal"))
	in.DailyLimitMinutes = 0
	if keys := keysOf(Run(in).Issues, TypeOverDailyLimit); len(keys) != 0 {
		t.Errorf("limite desligado ainda gerou %v", keys)
	}
}

func TestMonthRange(t *testing.T) {
	ini, fim, err := MonthRange(2028, 2)
	if err != nil || ini != "2028-02-01" || fim != "2028-02-29" {
		t.Errorf("MonthRange(2028, 2) = %s, %s, %v", ini, fim, err)
	}
	for _, c := range [][2]int{{2026, 0}, {2026, 13}, {1, 1}} {
		if _, _, err := MonthRange(c[0], c[1]); err == nil {
			t.Errorf("MonthRange(%d, %d) deveria falhar", c[0], c[1])
		}
	}
}

func TestFormatMinutes(t *testing.T) {
	for in, want := range map[int]string{0: "0min", 45: "45min", 60: "1h", 150: "2h 30min"} {
		if got := FormatMinutes(in); got != want {
			t.Errorf("FormatMinutes(%d) = %q, esperava %q", in, got, want)
		}
	}
}
