package agenda

import (
	"sort"
	"strings"
	"time"

	"github.com/teambition/rrule-go"
)

// Occurrence é uma ocorrência concreta de um evento dentro do período.
type Occurrence struct {
	// Key identifica a ocorrência entre importações: o UID, mais o início
	// original (UTC) quando o evento é recorrente. Uma ocorrência alterada
	// (RECURRENCE-ID) mantém a chave da original.
	Key         string
	UID         string
	Summary     string
	Start       time.Time
	End         time.Time
	AllDay      bool
	Cancelled   bool
	Transparent bool
	Attendees   []Attendee
	// Source é o nome da agenda de onde veio.
	Source string
}

const keyTimeLayout = "20060102T150405Z"

func occurrenceKey(uid string, original time.Time, recurring bool) string {
	if !recurring {
		return uid
	}
	return uid + "|" + original.UTC().Format(keyTimeLayout)
}

// Expand devolve as ocorrências que começam em [from, to), em loc, ordenadas
// pelo início. Trata RRULE (com INTERVAL, BYDAY, UNTIL, COUNT...), EXDATE e
// RECURRENCE-ID (a ocorrência alterada substitui a original, mesmo quando foi
// movida para fora do período). Uma RRULE inválida faz o evento valer só
// pela primeira ocorrência.
func Expand(events []Event, source string, from, to time.Time, loc *time.Location) []Occurrence {
	if loc == nil {
		loc = time.Local
	}

	masters := map[string]Event{}
	overrides := map[string][]Event{}
	order := []string{}
	for _, ev := range events {
		if _, seen := masters[ev.UID]; !seen && len(overrides[ev.UID]) == 0 {
			order = append(order, ev.UID)
		}
		if ev.HasRecurrenceID {
			overrides[ev.UID] = append(overrides[ev.UID], ev)
			continue
		}
		masters[ev.UID] = ev
	}

	inWindow := func(t time.Time) bool { return !t.Before(from) && t.Before(to) }
	out := make([]Occurrence, 0)

	for _, uid := range order {
		master, hasMaster := masters[uid]
		alterados := map[int64]bool{}
		for _, ov := range overrides[uid] {
			alterados[ov.RecurrenceID.Unix()] = true
		}
		recurring := len(overrides[uid]) > 0 || (hasMaster && master.RRule != "")

		if hasMaster {
			for _, start := range masterStarts(master, from, to) {
				if alterados[start.Unix()] || master.excluded(start) || !inWindow(start) {
					continue
				}
				out = append(out, toOccurrence(master, source, start, start.Add(master.Duration()),
					occurrenceKey(uid, start, recurring), loc))
			}
		}

		for _, ov := range overrides[uid] {
			if !inWindow(ov.Start) {
				continue
			}
			out = append(out, toOccurrence(ov, source, ov.Start, ov.End,
				occurrenceKey(uid, ov.RecurrenceID, true), loc))
		}
	}

	sort.SliceStable(out, func(i, j int) bool {
		if !out[i].Start.Equal(out[j].Start) {
			return out[i].Start.Before(out[j].Start)
		}
		return out[i].Key < out[j].Key
	})
	return out
}

// masterStarts devolve os inícios do evento-mestre que podem cair no período.
func masterStarts(ev Event, from, to time.Time) []time.Time {
	if ev.RRule == "" {
		return []time.Time{ev.Start}
	}
	// A RRULE é expandida no fuso do DTSTART, para que "toda segunda às 10h"
	// continue às 10h depois de uma mudança de horário de verão.
	opt, err := rrule.StrToROptionInLocation(ev.RRule, ev.Start.Location())
	if err != nil {
		return []time.Time{ev.Start}
	}
	opt.Dtstart = ev.Start
	rule, err := rrule.NewRRule(*opt)
	if err != nil {
		return []time.Time{ev.Start}
	}
	return rule.Between(from, to, true)
}

// excluded informa se a ocorrência que começa em start foi removida por EXDATE.
func (e Event) excluded(start time.Time) bool {
	for _, ex := range e.ExDates {
		if ex.Equal(start) {
			return true
		}
	}
	if len(e.exDateDays) > 0 {
		dia := start.Format(dateLayout)
		for _, d := range e.exDateDays {
			if d == dia {
				return true
			}
		}
	}
	return false
}

func toOccurrence(ev Event, source string, start, end time.Time, key string, loc *time.Location) Occurrence {
	return Occurrence{
		Key:         key,
		UID:         ev.UID,
		Summary:     ev.Summary,
		Start:       start.In(loc),
		End:         end.In(loc),
		AllDay:      ev.AllDay,
		Cancelled:   strings.EqualFold(ev.Status, "CANCELLED"),
		Transparent: ev.Transparent,
		Attendees:   ev.Attendees,
		Source:      source,
	}
}

// Dedupe remove ocorrências repetidas (mesma chave), mantendo a primeira — a
// mesma reunião pode estar em duas agendas configuradas.
func Dedupe(occs []Occurrence) []Occurrence {
	seen := map[string]bool{}
	out := make([]Occurrence, 0, len(occs))
	for _, o := range occs {
		if seen[o.Key] {
			continue
		}
		seen[o.Key] = true
		out = append(out, o)
	}
	return out
}
