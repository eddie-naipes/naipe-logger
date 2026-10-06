package agenda

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Situação de cada evento no plano.
const (
	StatusMapped   = "mapped"   // tem tarefa (regra ou tarefa padrão)
	StatusUnmapped = "unmapped" // nenhuma regra casou e não há tarefa padrão
	StatusIgnored  = "ignored"  // fora do plano; Reason explica
	StatusImported = "imported" // já lançado numa importação anterior
)

// TaskRef identifica a tarefa do Teamwork que recebe o lançamento.
type TaskRef struct {
	TaskID      int    `json:"taskId"`
	TaskName    string `json:"taskName"`
	ProjectID   int    `json:"projectId"`
	ProjectName string `json:"projectName"`
}

// Rule associa eventos a uma tarefa pelo título.
type Rule struct {
	// Match é uma palavra-chave (procurada no título, sem diferenciar
	// maiúsculas) ou, com Regex, uma expressão regular.
	Match string
	Regex bool
	Task  TaskRef
	// Description substitui o título do evento na descrição do lançamento.
	Description string
}

// PlanOptions são as regras de mapeamento já resolvidas.
type PlanOptions struct {
	Rules []Rule
	// DefaultTask recebe eventos sem regra; TaskID 0 os deixa "sem regra".
	DefaultTask TaskRef
	// IgnoreWords descartam eventos cujo título contém alguma delas.
	IgnoreWords []string
	// MinMinutes descarta eventos mais curtos (depois do corte de sobreposição).
	MinMinutes int
	// RoundTo arredonda a duração para o múltiplo mais próximo (0 = exato).
	RoundTo int
	// UserEmail identifica o usuário entre os participantes, para ignorar
	// convites que ele recusou.
	UserEmail string
	// IncludeTransparent mantém eventos marcados como "livre".
	IncludeTransparent bool
	Billable           bool
	// Imported informa as chaves já lançadas antes.
	Imported func(key string) bool
}

// PlanItem é um evento do período e o que será feito com ele.
type PlanItem struct {
	Key    string `json:"key"`
	Source string `json:"source"`
	Title  string `json:"title"`
	Date   string `json:"date"`
	// StartTime/EndTime (HH:MM) já consideram o corte de sobreposição.
	StartTime string `json:"startTime"`
	EndTime   string `json:"endTime"`
	Minutes   int    `json:"minutes"`
	Status    string `json:"status"`
	// Reason explica o status (motivo de ter sido ignorado, regra usada...).
	Reason string `json:"reason"`
	// RuleIndex é a regra que casou (-1 nenhuma).
	RuleIndex   int     `json:"ruleIndex"`
	Task        TaskRef `json:"task"`
	Description string  `json:"description"`
	Billable    bool    `json:"billable"`
}

// CompileRules valida as regras: uma expressão regular inválida é erro.
func CompileRules(rules []Rule) ([]*regexp.Regexp, error) {
	out := make([]*regexp.Regexp, len(rules))
	for i, r := range rules {
		pattern := strings.TrimSpace(r.Match)
		if pattern == "" {
			return nil, fmt.Errorf("regra %d: informe a palavra-chave ou a expressão", i+1)
		}
		if !r.Regex {
			pattern = regexp.QuoteMeta(pattern)
		}
		re, err := regexp.Compile("(?i)" + pattern)
		if err != nil {
			return nil, fmt.Errorf("regra %d: expressão regular inválida: %v", i+1, err)
		}
		out[i] = re
	}
	return out, nil
}

// BuildPlan classifica as ocorrências e calcula horário e duração de cada uma.
// Num mesmo dia, um evento que começa antes de o anterior terminar é cortado
// (só o trecho livre é lançado); cada evento fica limitado ao próprio dia.
func BuildPlan(occs []Occurrence, opts PlanOptions) ([]PlanItem, error) {
	regras, err := CompileRules(opts.Rules)
	if err != nil {
		return nil, err
	}

	type candidato struct {
		occ   Occurrence
		start time.Time
		end   time.Time
	}

	items := make([]PlanItem, 0, len(occs))
	porDia := map[string][]candidato{}
	dias := []string{}

	for _, o := range occs {
		base := PlanItem{
			Key:       o.Key,
			Source:    o.Source,
			Title:     o.Summary,
			Date:      o.Start.Format(dateLayout),
			StartTime: o.Start.Format("15:04"),
			EndTime:   o.End.Format("15:04"),
			RuleIndex: -1,
			Billable:  opts.Billable,
		}
		if motivo := ignoreReason(o, opts); motivo != "" {
			base.Status = StatusIgnored
			base.Reason = motivo
			base.Minutes = int(o.End.Sub(o.Start).Minutes())
			items = append(items, base)
			continue
		}
		fimDoDia := startOfDay(o.Start).AddDate(0, 0, 1)
		end := o.End
		if end.After(fimDoDia) {
			end = fimDoDia
		}
		if _, ok := porDia[base.Date]; !ok {
			dias = append(dias, base.Date)
		}
		porDia[base.Date] = append(porDia[base.Date], candidato{occ: o, start: o.Start, end: end})
	}

	for _, dia := range dias {
		lista := porDia[dia]
		sort.SliceStable(lista, func(i, j int) bool {
			if !lista[i].start.Equal(lista[j].start) {
				return lista[i].start.Before(lista[j].start)
			}
			if !lista[i].end.Equal(lista[j].end) {
				return lista[i].end.After(lista[j].end)
			}
			return lista[i].occ.Key < lista[j].occ.Key
		})

		var cursor time.Time
		for _, c := range lista {
			item := PlanItem{
				Key:       c.occ.Key,
				Source:    c.occ.Source,
				Title:     c.occ.Summary,
				Date:      dia,
				RuleIndex: -1,
				Billable:  opts.Billable,
			}
			start := c.start
			if !cursor.IsZero() && cursor.After(start) {
				start = cursor
			}
			item.StartTime = start.Format("15:04")
			item.EndTime = c.end.Format("15:04")
			if c.end.Equal(startOfDay(c.start).AddDate(0, 0, 1)) {
				item.EndTime = "24:00"
			}

			if !c.end.After(start) {
				item.Status = StatusIgnored
				item.Reason = "Sobreposto a outro evento"
				items = append(items, item)
				continue
			}
			if c.end.After(cursor) {
				cursor = c.end
			}

			brutos := int(math.Round(c.end.Sub(start).Minutes()))
			item.Minutes = roundMinutes(brutos, opts.RoundTo)
			if start.After(c.start) {
				item.Reason = "Início ajustado por sobreposição"
			}
			if brutos < opts.MinMinutes || item.Minutes <= 0 {
				item.Status = StatusIgnored
				item.Reason = fmt.Sprintf("Duração menor que %d min", max(opts.MinMinutes, 1))
				items = append(items, item)
				continue
			}

			applyMapping(&item, c.occ.Summary, opts, regras)
			if opts.Imported != nil && opts.Imported(item.Key) {
				item.Status = StatusImported
				item.Reason = "Já lançado numa importação anterior"
			}
			items = append(items, item)
		}
	}

	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Date != items[j].Date {
			return items[i].Date < items[j].Date
		}
		return items[i].StartTime < items[j].StartTime
	})
	return items, nil
}

func applyMapping(item *PlanItem, titulo string, opts PlanOptions, regras []*regexp.Regexp) {
	for i, re := range regras {
		if !re.MatchString(titulo) {
			continue
		}
		r := opts.Rules[i]
		item.Status = StatusMapped
		item.RuleIndex = i
		item.Task = r.Task
		item.Description = strings.TrimSpace(r.Description)
		if item.Reason == "" {
			item.Reason = fmt.Sprintf("Regra %d: %s", i+1, r.Match)
		}
		break
	}
	if item.Status == "" {
		if opts.DefaultTask.TaskID > 0 {
			item.Status = StatusMapped
			item.Task = opts.DefaultTask
			if item.Reason == "" {
				item.Reason = "Tarefa padrão"
			}
		} else {
			item.Status = StatusUnmapped
			if item.Reason == "" {
				item.Reason = "Nenhuma regra corresponde ao título"
			}
		}
	}
	if item.Description == "" {
		item.Description = titulo
	}
}

// ignoreReason devolve por que a ocorrência fica fora do plano ("" se entra).
func ignoreReason(o Occurrence, opts PlanOptions) string {
	switch {
	case o.AllDay:
		return "Evento de dia inteiro"
	case o.Cancelled:
		return "Evento cancelado"
	case o.Transparent && !opts.IncludeTransparent:
		return "Marcado como livre (não ocupa a agenda)"
	case declined(o, opts.UserEmail):
		return "Convite recusado"
	case !o.End.After(o.Start):
		return "Evento sem duração"
	}
	titulo := strings.ToLower(o.Summary)
	for _, w := range opts.IgnoreWords {
		w = strings.TrimSpace(w)
		if w != "" && strings.Contains(titulo, strings.ToLower(w)) {
			return fmt.Sprintf("Contém a palavra ignorada %q", w)
		}
	}
	return ""
}

func declined(o Occurrence, email string) bool {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return false
	}
	for _, a := range o.Attendees {
		if a.Email == email && a.PartStat == "DECLINED" {
			return true
		}
	}
	return false
}

// roundMinutes arredonda para o múltiplo de step mais próximo, sem zerar um
// evento que tinha duração (vira um step).
func roundMinutes(min, step int) int {
	if step <= 1 || min <= 0 {
		return min
	}
	r := int(math.Round(float64(min)/float64(step))) * step
	if r == 0 {
		r = step
	}
	return r
}

func startOfDay(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}
