// Package reports agrega os lançamentos de um período (totais, por projeto,
// por tarefa, por dia e por semana, comparação com a jornada) e os exporta em
// CSV. É lógica pura: quem busca os dados no Teamwork é o binding.
package reports

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"logTime-go/backend/api"
)

const dateLayout = "2006-01-02"

// MaxPeriodDays limita o período de um relatório (um ano e um dia, para
// cobrir um ano bissexto inteiro). Cada dia vira uma barra no gráfico e uma
// consulta de dia útil.
const MaxPeriodDays = 366

// Period é um intervalo de datas já validado (inclusivo nas duas pontas).
type Period struct {
	Start time.Time
	End   time.Time
}

// ParsePeriod valida datas AAAA-MM-DD, fim >= início e o tamanho máximo. As
// datas entram em nomes de arquivo, então nada além desse formato é aceito.
func ParsePeriod(startDate, endDate string) (Period, error) {
	start, err := time.Parse(dateLayout, startDate)
	if err != nil {
		return Period{}, fmt.Errorf("data inicial inválida (use AAAA-MM-DD): %q", startDate)
	}
	end, err := time.Parse(dateLayout, endDate)
	if err != nil {
		return Period{}, fmt.Errorf("data final inválida (use AAAA-MM-DD): %q", endDate)
	}
	if end.Before(start) {
		return Period{}, fmt.Errorf("a data final (%s) é anterior à inicial (%s)", endDate, startDate)
	}
	if days := int(end.Sub(start).Hours()/24) + 1; days > MaxPeriodDays {
		return Period{}, fmt.Errorf("período longo demais: %d dias (máximo %d)", days, MaxPeriodDays)
	}
	return Period{Start: start, End: end}, nil
}

// StartDate e EndDate devolvem as pontas no formato AAAA-MM-DD.
func (p Period) StartDate() string { return p.Start.Format(dateLayout) }
func (p Period) EndDate() string   { return p.End.Format(dateLayout) }

// Report é o resumo de um período.
type Report struct {
	StartDate          string `json:"startDate"`
	EndDate            string `json:"endDate"`
	TotalMinutes       int    `json:"totalMinutes"`
	BillableMinutes    int    `json:"billableMinutes"`
	NonBillableMinutes int    `json:"nonBillableMinutes"`
	EntryCount         int    `json:"entryCount"`
	// WorkingDays é o número de dias úteis do período (fins de semana,
	// feriados e férias fora).
	WorkingDays int `json:"workingDays"`
	// MinutesPerDay é a jornada diária configurada.
	MinutesPerDay int `json:"minutesPerDay"`
	// ExpectedMinutes = WorkingDays × MinutesPerDay.
	ExpectedMinutes int `json:"expectedMinutes"`
	// BalanceMinutes = TotalMinutes − ExpectedMinutes (negativo = faltando).
	BalanceMinutes int `json:"balanceMinutes"`
	// DaysWithEntries conta os dias (úteis ou não) com algum lançamento.
	DaysWithEntries int `json:"daysWithEntries"`
	// WorkingDaysWithoutEntries conta os dias úteis sem nenhum lançamento.
	WorkingDaysWithoutEntries int `json:"workingDaysWithoutEntries"`

	ByProject []ProjectTotal `json:"byProject"`
	ByTask    []TaskTotal    `json:"byTask"`
	ByDay     []DayTotal     `json:"byDay"`
	ByWeek    []WeekTotal    `json:"byWeek"`
}

// ProjectTotal soma os lançamentos de um projeto.
type ProjectTotal struct {
	ProjectID       int    `json:"projectId"`
	ProjectName     string `json:"projectName"`
	Minutes         int    `json:"minutes"`
	BillableMinutes int    `json:"billableMinutes"`
	EntryCount      int    `json:"entryCount"`
}

// TaskTotal soma os lançamentos de uma tarefa.
type TaskTotal struct {
	TaskID          int    `json:"taskId"`
	TaskName        string `json:"taskName"`
	ProjectID       int    `json:"projectId"`
	ProjectName     string `json:"projectName"`
	Minutes         int    `json:"minutes"`
	BillableMinutes int    `json:"billableMinutes"`
	EntryCount      int    `json:"entryCount"`
}

// DayTotal é um dia do período, com ou sem lançamento.
type DayTotal struct {
	Date            string `json:"date"`
	Minutes         int    `json:"minutes"`
	BillableMinutes int    `json:"billableMinutes"`
	IsWorkingDay    bool   `json:"isWorkingDay"`
	// ExpectedMinutes é a jornada nos dias úteis e 0 nos demais.
	ExpectedMinutes int `json:"expectedMinutes"`
}

// WeekTotal é uma semana (segunda a domingo) recortada pelo período.
type WeekTotal struct {
	WeekStart       string `json:"weekStart"`
	WeekEnd         string `json:"weekEnd"`
	Minutes         int    `json:"minutes"`
	BillableMinutes int    `json:"billableMinutes"`
	WorkingDays     int    `json:"workingDays"`
	ExpectedMinutes int    `json:"expectedMinutes"`
}

// Sem nome o lançamento ainda aparece, agrupado sob um rótulo explícito.
const (
	noProjectName = "(sem projeto)"
	noTaskName    = "(sem tarefa)"
)

// Include informa se um lançamento entra no relatório: só os do usuário
// atual (quando o ID é conhecido), não excluídos e dentro do período.
func Include(entry api.TimeEntryReport, period Period, userID int) bool {
	if userID != 0 && entry.UserID != 0 && entry.UserID != userID {
		return false
	}
	if entry.DeletedAt != "" || strings.EqualFold(entry.Status, "deleted") {
		return false
	}
	if entry.Minutes <= 0 {
		return false
	}
	return entry.Date >= period.StartDate() && entry.Date <= period.EndDate()
}

// Filter devolve os lançamentos que entram no relatório (ver Include),
// ordenados por data, hora de início e ID.
func Filter(entries []api.TimeEntryReport, period Period, userID int) []api.TimeEntryReport {
	out := make([]api.TimeEntryReport, 0, len(entries))
	for _, e := range entries {
		if Include(e, period, userID) {
			out = append(out, e)
		}
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

// Build agrega os lançamentos (já filtrados ou não: Build filtra de novo).
// workingDays são as datas AAAA-MM-DD úteis do período (GetWorkingDays).
func Build(entries []api.TimeEntryReport, period Period, workingDays []string, minutesPerDay, userID int) Report {
	entries = Filter(entries, period, userID)

	working := make(map[string]bool, len(workingDays))
	for _, d := range workingDays {
		if d >= period.StartDate() && d <= period.EndDate() {
			working[d] = true
		}
	}

	r := Report{
		StartDate:     period.StartDate(),
		EndDate:       period.EndDate(),
		EntryCount:    len(entries),
		WorkingDays:   len(working),
		MinutesPerDay: minutesPerDay,
		ByProject:     []ProjectTotal{},
		ByTask:        []TaskTotal{},
		ByDay:         []DayTotal{},
		ByWeek:        []WeekTotal{},
	}
	r.ExpectedMinutes = r.WorkingDays * minutesPerDay

	projects := map[string]*ProjectTotal{}
	tasks := map[string]*TaskTotal{}
	days := map[string]*DayTotal{}

	for _, e := range entries {
		billable := 0
		if e.IsBillable {
			billable = e.Minutes
		}
		r.TotalMinutes += e.Minutes
		r.BillableMinutes += billable

		projectName := nameOr(e.ProjectName, noProjectName)
		pk := groupKey(e.ProjectID, projectName)
		p, ok := projects[pk]
		if !ok {
			p = &ProjectTotal{ProjectID: e.ProjectID, ProjectName: projectName}
			projects[pk] = p
		}
		p.Minutes += e.Minutes
		p.BillableMinutes += billable
		p.EntryCount++

		taskName := nameOr(e.TaskName, noTaskName)
		tk := pk + "|" + groupKey(e.TaskID, taskName)
		t, ok := tasks[tk]
		if !ok {
			t = &TaskTotal{TaskID: e.TaskID, TaskName: taskName, ProjectID: e.ProjectID, ProjectName: projectName}
			tasks[tk] = t
		}
		t.Minutes += e.Minutes
		t.BillableMinutes += billable
		t.EntryCount++

		d, ok := days[e.Date]
		if !ok {
			d = &DayTotal{Date: e.Date}
			days[e.Date] = d
		}
		d.Minutes += e.Minutes
		d.BillableMinutes += billable
	}
	r.NonBillableMinutes = r.TotalMinutes - r.BillableMinutes
	r.BalanceMinutes = r.TotalMinutes - r.ExpectedMinutes
	r.DaysWithEntries = len(days)

	for _, p := range projects {
		r.ByProject = append(r.ByProject, *p)
	}
	sort.Slice(r.ByProject, func(i, j int) bool {
		a, b := r.ByProject[i], r.ByProject[j]
		if a.Minutes != b.Minutes {
			return a.Minutes > b.Minutes
		}
		return a.ProjectName < b.ProjectName
	})

	for _, t := range tasks {
		r.ByTask = append(r.ByTask, *t)
	}
	sort.Slice(r.ByTask, func(i, j int) bool {
		a, b := r.ByTask[i], r.ByTask[j]
		if a.Minutes != b.Minutes {
			return a.Minutes > b.Minutes
		}
		if a.ProjectName != b.ProjectName {
			return a.ProjectName < b.ProjectName
		}
		return a.TaskName < b.TaskName
	})

	// Todo dia do período entra na série, para o gráfico mostrar os buracos.
	var week *WeekTotal
	for day := period.Start; !day.After(period.End); day = day.AddDate(0, 0, 1) {
		date := day.Format(dateLayout)
		dt := DayTotal{Date: date, IsWorkingDay: working[date]}
		if d, ok := days[date]; ok {
			dt.Minutes, dt.BillableMinutes = d.Minutes, d.BillableMinutes
		}
		if dt.IsWorkingDay {
			dt.ExpectedMinutes = minutesPerDay
			if dt.Minutes == 0 {
				r.WorkingDaysWithoutEntries++
			}
		}
		r.ByDay = append(r.ByDay, dt)

		if week == nil || day.Weekday() == time.Monday {
			r.ByWeek = append(r.ByWeek, WeekTotal{WeekStart: date})
			week = &r.ByWeek[len(r.ByWeek)-1]
		}
		week.WeekEnd = date
		week.Minutes += dt.Minutes
		week.BillableMinutes += dt.BillableMinutes
		week.ExpectedMinutes += dt.ExpectedMinutes
		if dt.IsWorkingDay {
			week.WorkingDays++
		}
	}

	return r
}

func nameOr(name, fallback string) string {
	if strings.TrimSpace(name) == "" {
		return fallback
	}
	return strings.TrimSpace(name)
}

// groupKey agrupa pelo ID quando há um; sem ID, pelo nome.
func groupKey(id int, name string) string {
	if id != 0 {
		return fmt.Sprintf("#%d", id)
	}
	return "n:" + name
}
