// Package audit faz o "fechamento do mês": examina os lançamentos do usuário
// num mês e aponta o que está errado antes da entrega (dias incompletos,
// descrições vazias ou genéricas, duplicatas, lançamentos em dia não útil,
// dias acima do limite e lançamentos sem tarefa).
//
// É lógica pura: quem busca lançamentos, dias úteis e configuração no
// Teamwork/disco é o binding (app_audit.go).
package audit

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"logTime-go/backend/api"
)

const dateLayout = "2006-01-02"

// Severity diferencia o que impede a entrega (erro) do que merece só uma
// conferida (aviso).
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
)

// IssueType identifica a verificação que gerou o problema.
type IssueType string

const (
	TypeIncompleteDay      IssueType = "incomplete_day"
	TypeMissingDescription IssueType = "missing_description"
	TypeGenericDescription IssueType = "generic_description"
	TypeDuplicate          IssueType = "duplicate"
	TypeNonWorkingDay      IssueType = "non_working_day"
	TypeOverDailyLimit     IssueType = "over_daily_limit"
	TypeNoTask             IssueType = "no_task"
)

// typeOrder é a ordem de exibição dos grupos (e das contagens).
var typeOrder = []IssueType{
	TypeIncompleteDay,
	TypeMissingDescription,
	TypeDuplicate,
	TypeGenericDescription,
	TypeNonWorkingDay,
	TypeOverDailyLimit,
	TypeNoTask,
}

// Action é a correção sugerida, que o frontend transforma num botão.
type Action string

const (
	// ActionCompletePeriod abre "Completar período" com o mês.
	ActionCompletePeriod Action = "complete_period"
	// ActionEdit abre a edição do lançamento.
	ActionEdit Action = "edit"
	// ActionDeleteDuplicates apaga as cópias excedentes (DeleteEntryIDs).
	ActionDeleteDuplicates Action = "delete_duplicates"
	// ActionEditOrDelete oferece editar (mudar a data) ou apagar.
	ActionEditOrDelete Action = "edit_or_delete"
	// ActionViewDay lista os lançamentos do dia para editar/apagar.
	ActionViewDay Action = "view_day"
)

// DefaultDailyLimitMinutes é o limite diário padrão (10h).
const DefaultDailyLimitMinutes = 10 * 60

// Issue é um problema encontrado.
type Issue struct {
	// Key identifica o problema de forma estável entre execuções; é o que
	// fica gravado quando o usuário o ignora.
	Key      string    `json:"key"`
	Type     IssueType `json:"type"`
	Severity Severity  `json:"severity"`
	// Date é o dia do problema (AAAA-MM-DD).
	Date     string `json:"date"`
	EntryIDs []int  `json:"entryIds"`
	// Entries traz os lançamentos envolvidos, prontos para a edição.
	Entries []api.TimeEntryReport `json:"entries"`
	Message string                `json:"message"`
	Action  Action                `json:"action"`
	// DeleteEntryIDs são as cópias a apagar numa duplicata (todas menos a
	// mais antiga, KeepEntryID).
	DeleteEntryIDs []int `json:"deleteEntryIds,omitempty"`
	KeepEntryID    int   `json:"keepEntryId,omitempty"`
	// Minutes é o total do dia (dia incompleto / acima do limite).
	Minutes int `json:"minutes,omitempty"`
	// MissingMinutes é quanto falta para a jornada (dia incompleto).
	MissingMinutes int `json:"missingMinutes,omitempty"`
	// Reason explica por que o dia não é útil (lançamento em dia não útil).
	Reason  string `json:"reason,omitempty"`
	Ignored bool   `json:"ignored"`
}

// TypeCount é a contagem de problemas (não ignorados) de um tipo.
type TypeCount struct {
	Type     IssueType `json:"type"`
	Severity Severity  `json:"severity"`
	Count    int       `json:"count"`
}

// Summary resume o mês: lançado × esperado.
type Summary struct {
	Year  int `json:"year"`
	Month int `json:"month"`
	// LoggedMinutes soma os lançamentos do usuário no mês.
	LoggedMinutes int `json:"loggedMinutes"`
	// ExpectedMinutes = dias úteis do mês × jornada.
	ExpectedMinutes int `json:"expectedMinutes"`
	// ExpectedToDateMinutes considera só os dias úteis até hoje.
	ExpectedToDateMinutes int `json:"expectedToDateMinutes"`
	// LoggedToDateMinutes soma os lançamentos até hoje (inclusive).
	LoggedToDateMinutes int `json:"loggedToDateMinutes"`
	WorkingDays         int `json:"workingDays"`
	WorkingDaysToDate   int `json:"workingDaysToDate"`
	MinutesPerDay       int `json:"minutesPerDay"`
	EntryCount          int `json:"entryCount"`
	ErrorCount          int `json:"errorCount"`
	WarningCount        int `json:"warningCount"`
	IgnoredCount        int `json:"ignoredCount"`
	// Ready = nenhum erro pendente (avisos não impedem a entrega).
	Ready bool `json:"ready"`
}

// Result é a auditoria de um mês.
type Result struct {
	// Issues traz todos os problemas, inclusive os ignorados (Ignored=true),
	// ordenados por tipo, dia e chave.
	Issues  []Issue     `json:"issues"`
	Counts  []TypeCount `json:"counts"`
	Summary Summary     `json:"summary"`
}

// Input reúne o que a auditoria precisa. Nada aqui toca a rede.
type Input struct {
	Year  int
	Month int
	// Today (AAAA-MM-DD) limita a verificação de dias incompletos.
	Today string
	// Entries são os lançamentos do período; Run descarta os excluídos, os
	// de outros usuários e os fora do mês.
	Entries []api.TimeEntryReport
	UserID  int
	// WorkingDays são os dias úteis do mês (GetWorkingDays).
	WorkingDays []string
	// NonWorkingDays explica os dias não úteis (ListNonWorkingDays).
	NonWorkingDays []api.NonWorkingDay
	MinutesPerDay  int
	// DailyLimitMinutes > 0 liga o aviso de dia acima do limite.
	DailyLimitMinutes int
	// GenericDescriptions são descrições consideradas vagas demais.
	GenericDescriptions []string
	// Ignored são as chaves de problemas que o usuário mandou ignorar.
	Ignored []string
}

// MonthRange devolve o primeiro e o último dia do mês (AAAA-MM-DD).
func MonthRange(year, month int) (string, string, error) {
	if year < 2000 || year > 2100 || month < 1 || month > 12 {
		return "", "", fmt.Errorf("mês inválido: %02d/%d", month, year)
	}
	first := time.Date(year, time.Month(month), 1, 12, 0, 0, 0, time.UTC)
	last := first.AddDate(0, 1, -1)
	return first.Format(dateLayout), last.Format(dateLayout), nil
}

// Run executa todas as verificações.
func Run(in Input) Result {
	start, end, err := MonthRange(in.Year, in.Month)
	if err != nil {
		return Result{Issues: []Issue{}, Counts: []TypeCount{}, Summary: Summary{Year: in.Year, Month: in.Month}}
	}

	entries := filterEntries(in.Entries, start, end, in.UserID)

	working := make(map[string]bool, len(in.WorkingDays))
	for _, d := range in.WorkingDays {
		if d >= start && d <= end {
			working[d] = true
		}
	}
	nonWorking := make(map[string]api.NonWorkingDay, len(in.NonWorkingDays))
	for _, d := range in.NonWorkingDays {
		// Um dia pode aparecer mais de uma vez; vale o primeiro (fim de
		// semana e feriado nacional vêm antes dos extras).
		if _, ok := nonWorking[d.Date]; !ok {
			nonWorking[d.Date] = d
		}
	}

	perDay := make(map[string][]api.TimeEntryReport)
	for _, e := range entries {
		perDay[e.Date] = append(perDay[e.Date], e)
	}

	var issues []Issue
	issues = append(issues, incompleteDays(in, working, perDay)...)
	issues = append(issues, descriptionIssues(entries, in.GenericDescriptions)...)
	issues = append(issues, duplicates(entries)...)
	issues = append(issues, nonWorkingDayEntries(entries, working, nonWorking)...)
	issues = append(issues, overDailyLimit(perDay, in.DailyLimitMinutes)...)
	issues = append(issues, noTask(entries)...)

	ignored := make(map[string]bool, len(in.Ignored))
	for _, k := range in.Ignored {
		ignored[k] = true
	}
	for i := range issues {
		issues[i].Ignored = ignored[issues[i].Key]
	}
	sortIssues(issues)
	if issues == nil {
		issues = []Issue{}
	}

	return Result{
		Issues:  issues,
		Counts:  countByType(issues),
		Summary: summarize(in, entries, working, issues),
	}
}

// filterEntries mantém só os lançamentos do usuário, não excluídos, com tempo
// e dentro do mês, em ordem de data, início e ID.
func filterEntries(all []api.TimeEntryReport, start, end string, userID int) []api.TimeEntryReport {
	out := make([]api.TimeEntryReport, 0, len(all))
	for _, e := range all {
		if e.DeletedAt != "" || strings.EqualFold(e.Status, "deleted") {
			continue
		}
		// UserID 0 = a resposta não informou o dono; o filtro da URL já
		// restringiu ao usuário atual.
		if userID != 0 && e.UserID != 0 && e.UserID != userID {
			continue
		}
		if e.Minutes <= 0 || e.Date < start || e.Date > end {
			continue
		}
		out = append(out, e)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Date != out[j].Date {
			return out[i].Date < out[j].Date
		}
		if out[i].StartTime != out[j].StartTime {
			return out[i].StartTime < out[j].StartTime
		}
		return out[i].ID < out[j].ID
	})
	return out
}

// 1. Dias úteis até hoje abaixo da jornada.
func incompleteDays(in Input, working map[string]bool, perDay map[string][]api.TimeEntryReport) []Issue {
	if in.MinutesPerDay <= 0 {
		return nil
	}
	days := make([]string, 0, len(working))
	for d := range working {
		if in.Today == "" || d <= in.Today {
			days = append(days, d)
		}
	}
	sort.Strings(days)

	var out []Issue
	for _, d := range days {
		logged := sumMinutes(perDay[d])
		if logged >= in.MinutesPerDay {
			continue
		}
		missing := in.MinutesPerDay - logged
		msg := fmt.Sprintf("Faltam %s (lançado %s de %s)",
			FormatMinutes(missing), FormatMinutes(logged), FormatMinutes(in.MinutesPerDay))
		if logged == 0 {
			msg = fmt.Sprintf("Nenhuma hora lançada (jornada de %s)", FormatMinutes(in.MinutesPerDay))
		}
		out = append(out, Issue{
			Key:            dayKey(TypeIncompleteDay, d),
			Type:           TypeIncompleteDay,
			Severity:       SeverityError,
			Date:           d,
			EntryIDs:       ids(perDay[d]),
			Entries:        nonNil(perDay[d]),
			Message:        msg,
			Action:         ActionCompletePeriod,
			Minutes:        logged,
			MissingMinutes: missing,
		})
	}
	return out
}

// 2. Lançamentos sem descrição ou com descrição genérica.
func descriptionIssues(entries []api.TimeEntryReport, generic []string) []Issue {
	genericSet := make(map[string]bool, len(generic))
	for _, g := range generic {
		if n := NormalizeDescription(g); n != "" {
			genericSet[n] = true
		}
	}

	var out []Issue
	for _, e := range entries {
		desc := NormalizeDescription(e.Description)
		switch {
		case desc == "":
			out = append(out, entryIssue(TypeMissingDescription, SeverityError, e,
				"Lançamento sem descrição", ActionEdit))
		case genericSet[desc]:
			out = append(out, entryIssue(TypeGenericDescription, SeverityWarning, e,
				fmt.Sprintf("Descrição genérica: %q", strings.TrimSpace(e.Description)), ActionEdit))
		case e.TaskName != "" && desc == NormalizeDescription(e.TaskName):
			out = append(out, entryIssue(TypeGenericDescription, SeverityWarning, e,
				"A descrição é só o nome da tarefa", ActionEdit))
		}
	}
	return out
}

// 3. Possíveis duplicatas: mesma tarefa, dia, minutos e descrição.
func duplicates(entries []api.TimeEntryReport) []Issue {
	type groupKey struct {
		taskID  int
		date    string
		minutes int
		desc    string
	}
	groups := make(map[groupKey][]api.TimeEntryReport)
	var order []groupKey
	for _, e := range entries {
		k := groupKey{e.TaskID, e.Date, e.Minutes, NormalizeDescription(e.Description)}
		if _, ok := groups[k]; !ok {
			order = append(order, k)
		}
		groups[k] = append(groups[k], e)
	}

	var out []Issue
	for _, k := range order {
		group := groups[k]
		if len(group) < 2 {
			continue
		}
		sorted := append([]api.TimeEntryReport(nil), group...)
		sort.Slice(sorted, func(i, j int) bool { return sorted[i].ID < sorted[j].ID })
		keep := sorted[0].ID
		toDelete := ids(sorted[1:])
		out = append(out, Issue{
			Key:      idsKey(TypeDuplicate, ids(sorted)),
			Type:     TypeDuplicate,
			Severity: SeverityError,
			Date:     k.date,
			EntryIDs: ids(sorted),
			Entries:  sorted,
			Message: fmt.Sprintf("%d lançamentos iguais de %s em %q",
				len(sorted), FormatMinutes(k.minutes), taskLabel(sorted[0])),
			Action:         ActionDeleteDuplicates,
			DeleteEntryIDs: toDelete,
			KeepEntryID:    keep,
			Minutes:        k.minutes,
		})
	}
	return out
}

// 4. Lançamentos em dia não útil.
func nonWorkingDayEntries(entries []api.TimeEntryReport, working map[string]bool, nonWorking map[string]api.NonWorkingDay) []Issue {
	var out []Issue
	for _, e := range entries {
		if working[e.Date] {
			continue
		}
		reason := NonWorkingReason(e.Date, nonWorking[e.Date])
		issue := entryIssue(TypeNonWorkingDay, SeverityWarning, e,
			fmt.Sprintf("Lançamento em dia não útil (%s)", reason), ActionEditOrDelete)
		issue.Reason = reason
		out = append(out, issue)
	}
	return out
}

// 5. Dias acima do limite diário.
func overDailyLimit(perDay map[string][]api.TimeEntryReport, limit int) []Issue {
	if limit <= 0 {
		return nil
	}
	days := make([]string, 0, len(perDay))
	for d := range perDay {
		days = append(days, d)
	}
	sort.Strings(days)

	var out []Issue
	for _, d := range days {
		total := sumMinutes(perDay[d])
		if total <= limit {
			continue
		}
		out = append(out, Issue{
			Key:      dayKey(TypeOverDailyLimit, d),
			Type:     TypeOverDailyLimit,
			Severity: SeverityWarning,
			Date:     d,
			EntryIDs: ids(perDay[d]),
			Entries:  perDay[d],
			Message: fmt.Sprintf("%s lançadas no dia (limite de %s)",
				FormatMinutes(total), FormatMinutes(limit)),
			Action:  ActionViewDay,
			Minutes: total,
		})
	}
	return out
}

// 6. Lançamentos sem tarefa.
func noTask(entries []api.TimeEntryReport) []Issue {
	var out []Issue
	for _, e := range entries {
		if e.TaskID != 0 {
			continue
		}
		out = append(out, entryIssue(TypeNoTask, SeverityWarning, e,
			"Lançamento sem tarefa (só no projeto)", ActionEdit))
	}
	return out
}

func entryIssue(t IssueType, sev Severity, e api.TimeEntryReport, msg string, action Action) Issue {
	return Issue{
		Key:      idsKey(t, []int{e.ID}),
		Type:     t,
		Severity: sev,
		Date:     e.Date,
		EntryIDs: []int{e.ID},
		Entries:  []api.TimeEntryReport{e},
		Message:  msg,
		Action:   action,
		Minutes:  e.Minutes,
	}
}

// dayKey e idsKey montam as chaves estáveis dos problemas: "tipo:dia" para os
// problemas do dia e "tipo:id1,id2" (IDs em ordem crescente) para os de
// lançamentos.
func dayKey(t IssueType, date string) string { return string(t) + ":" + date }

func idsKey(t IssueType, entryIDs []int) string {
	sorted := append([]int(nil), entryIDs...)
	sort.Ints(sorted)
	parts := make([]string, len(sorted))
	for i, id := range sorted {
		parts[i] = strconv.Itoa(id)
	}
	return string(t) + ":" + strings.Join(parts, ",")
}

// NormalizeDescription compara descrições sem diferenciar maiúsculas,
// espaços repetidos e pontuação final.
func NormalizeDescription(s string) string {
	s = strings.ToLower(strings.Join(strings.Fields(s), " "))
	return strings.TrimRight(s, ".;:!,-… ")
}

// NonWorkingReason descreve por que o dia não é útil.
func NonWorkingReason(date string, d api.NonWorkingDay) string {
	label := func(prefix string) string {
		if d.Name != "" {
			return prefix + ": " + d.Name
		}
		return prefix
	}
	switch d.Type {
	case "holiday":
		return label("feriado nacional")
	case api.NonWorkingDayStateHoliday:
		return label("feriado estadual")
	case api.NonWorkingDayMunicipal:
		return label("feriado municipal")
	case api.NonWorkingDayBridge:
		return label("ponte")
	case api.NonWorkingDayVacation:
		return label("férias")
	case api.NonWorkingDayCustom:
		return label("dia sem expediente")
	}
	if t, err := time.Parse(dateLayout, date); err == nil {
		switch t.Weekday() {
		case time.Saturday:
			return "sábado"
		case time.Sunday:
			return "domingo"
		}
	}
	return "dia sem expediente"
}

func taskLabel(e api.TimeEntryReport) string {
	if e.TaskName != "" {
		return e.TaskName
	}
	if e.ProjectName != "" {
		return e.ProjectName
	}
	return "sem tarefa"
}

func sumMinutes(entries []api.TimeEntryReport) int {
	total := 0
	for _, e := range entries {
		total += e.Minutes
	}
	return total
}

func ids(entries []api.TimeEntryReport) []int {
	out := make([]int, len(entries))
	for i, e := range entries {
		out[i] = e.ID
	}
	return out
}

func nonNil(entries []api.TimeEntryReport) []api.TimeEntryReport {
	if entries == nil {
		return []api.TimeEntryReport{}
	}
	return entries
}

func sortIssues(issues []Issue) {
	rank := make(map[IssueType]int, len(typeOrder))
	for i, t := range typeOrder {
		rank[t] = i
	}
	sort.SliceStable(issues, func(i, j int) bool {
		a, b := issues[i], issues[j]
		if rank[a.Type] != rank[b.Type] {
			return rank[a.Type] < rank[b.Type]
		}
		if a.Date != b.Date {
			return a.Date < b.Date
		}
		return a.Key < b.Key
	})
}

// countByType conta os problemas não ignorados, na ordem de exibição; tipos
// sem problema ficam de fora.
func countByType(issues []Issue) []TypeCount {
	counts := make(map[IssueType]*TypeCount)
	for _, is := range issues {
		if is.Ignored {
			continue
		}
		c, ok := counts[is.Type]
		if !ok {
			c = &TypeCount{Type: is.Type, Severity: is.Severity}
			counts[is.Type] = c
		}
		c.Count++
		// Um tipo com algum erro conta como erro.
		if is.Severity == SeverityError {
			c.Severity = SeverityError
		}
	}
	out := []TypeCount{}
	for _, t := range typeOrder {
		if c, ok := counts[t]; ok {
			out = append(out, *c)
		}
	}
	return out
}

func summarize(in Input, entries []api.TimeEntryReport, working map[string]bool, issues []Issue) Summary {
	s := Summary{
		Year:          in.Year,
		Month:         in.Month,
		MinutesPerDay: in.MinutesPerDay,
		EntryCount:    len(entries),
		WorkingDays:   len(working),
	}
	for d := range working {
		if in.Today == "" || d <= in.Today {
			s.WorkingDaysToDate++
		}
	}
	for _, e := range entries {
		s.LoggedMinutes += e.Minutes
		if in.Today == "" || e.Date <= in.Today {
			s.LoggedToDateMinutes += e.Minutes
		}
	}
	s.ExpectedMinutes = s.WorkingDays * in.MinutesPerDay
	s.ExpectedToDateMinutes = s.WorkingDaysToDate * in.MinutesPerDay

	for _, is := range issues {
		switch {
		case is.Ignored:
			s.IgnoredCount++
		case is.Severity == SeverityError:
			s.ErrorCount++
		default:
			s.WarningCount++
		}
	}
	s.Ready = s.ErrorCount == 0
	return s
}

// PendingCount é o número de problemas não ignorados (erros + avisos).
func (r Result) PendingCount() int {
	return r.Summary.ErrorCount + r.Summary.WarningCount
}

// FormatMinutes escreve uma duração como "2h 30min", "45min" ou "3h".
func FormatMinutes(total int) string {
	if total < 0 {
		total = -total
	}
	h, m := total/60, total%60
	switch {
	case h == 0:
		return fmt.Sprintf("%dmin", m)
	case m == 0:
		return fmt.Sprintf("%dh", h)
	default:
		return fmt.Sprintf("%dh %dmin", h, m)
	}
}
