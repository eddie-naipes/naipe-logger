// Package agenda lê agendas no formato iCalendar (.ics) — link iCal privado
// do Google Agenda/Outlook ou arquivo local —, expande as recorrências de um
// período e monta, a partir de regras, um plano de lançamentos com as
// reuniões. Nada aqui fala com o Teamwork.
//
// O parse estrutural (linhas dobradas, parâmetros, escapes) fica com
// github.com/arran4/golang-ical e a expansão de RRULE com
// github.com/teambition/rrule-go; datas e fusos são interpretados aqui, para
// controlar o fuso local (injetável nos testes) e os TZID do Outlook.
package agenda

import (
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"
	// Embute a base de fusos: no Windows o Go não acha o zoneinfo fora do
	// GOROOT, e um TZID como America/Sao_Paulo precisa ser resolvido.
	_ "time/tzdata"

	ics "github.com/arran4/golang-ical"
)

// Attendee é um participante do evento.
type Attendee struct {
	Email string
	// PartStat é o PARTSTAT em maiúsculas (ACCEPTED, DECLINED, ...).
	PartStat string
}

// Event é um VEVENT lido do arquivo, ainda sem expandir a recorrência.
type Event struct {
	UID     string
	Summary string
	Start   time.Time
	End     time.Time
	// AllDay indica DTSTART só com data (evento de dia inteiro).
	AllDay bool
	// Status é o STATUS em maiúsculas (CONFIRMED, TENTATIVE, CANCELLED).
	Status string
	// Transparent indica TRANSP:TRANSPARENT (o evento não ocupa a agenda).
	Transparent bool
	Attendees   []Attendee
	// RRule é o valor cru da RRULE ("" quando o evento não se repete).
	RRule string
	// ExDates são as ocorrências excluídas; exDateDays guarda as EXDATE só
	// com data (AAAA-MM-DD), que excluem o dia inteiro.
	ExDates    []time.Time
	exDateDays []string
	// RecurrenceID marca uma ocorrência alterada de um evento recorrente: ela
	// substitui a ocorrência original que começaria nesse instante.
	RecurrenceID    time.Time
	HasRecurrenceID bool
}

// Duration devolve a duração do evento (nunca negativa).
func (e Event) Duration() time.Duration {
	if e.End.Before(e.Start) {
		return 0
	}
	return e.End.Sub(e.Start)
}

// ErrNotCalendar indica que o conteúdo não é um iCalendar.
var ErrNotCalendar = errors.New("o conteúdo não é uma agenda iCalendar (.ics)")

// Parse lê os VEVENT de um iCalendar. Datas flutuantes (sem fuso) e TZID
// desconhecidos são interpretados em local. Os horários mantêm o fuso de
// origem (a RRULE é expandida nele); Expand os converte para o fuso local.
// Eventos sem UID ou sem DTSTART são descartados.
func Parse(r io.Reader, local *time.Location) ([]Event, error) {
	if local == nil {
		local = time.Local
	}
	cal, err := ics.ParseCalendarWithOptions(r,
		ics.WithUnknownPropertyHandler(ics.AcceptUnknownPropertyHandler))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrNotCalendar, err)
	}

	events := make([]Event, 0)
	for _, ve := range cal.Events() {
		ev, ok := convertEvent(ve, local)
		if ok {
			events = append(events, ev)
		}
	}
	return events, nil
}

func convertEvent(ve *ics.VEvent, local *time.Location) (Event, bool) {
	uid := strings.TrimSpace(ve.Id())
	startProp := ve.GetProperty(ics.ComponentPropertyDtStart)
	if uid == "" || startProp == nil {
		return Event{}, false
	}

	start, allDay, err := parseICalTime(startProp.Value, startProp.ICalParameters, local)
	if err != nil {
		return Event{}, false
	}

	ev := Event{
		UID:     uid,
		Summary: strings.TrimSpace(textValue(ve, ics.ComponentPropertySummary)),
		Start:   start,
		AllDay:  allDay,
		Status:  strings.ToUpper(strings.TrimSpace(textValue(ve, ics.ComponentPropertyStatus))),
	}
	ev.Transparent = strings.EqualFold(strings.TrimSpace(textValue(ve, ics.ComponentPropertyTransp)), "TRANSPARENT")

	ev.End = eventEnd(ve, ev, local)

	for _, at := range ve.Attendees() {
		ev.Attendees = append(ev.Attendees, Attendee{
			Email:    normalizeEmail(at.Value),
			PartStat: strings.ToUpper(strings.TrimSpace(firstParam(at.ICalParameters, "PARTSTAT"))),
		})
	}

	if p := ve.GetProperty(ics.ComponentPropertyRrule); p != nil {
		ev.RRule = strings.TrimSpace(p.Value)
	}

	for _, p := range ve.GetProperties(ics.ComponentPropertyExdate) {
		for _, v := range strings.Split(p.Value, ",") {
			v = strings.TrimSpace(v)
			if v == "" {
				continue
			}
			t, dateOnly, err := parseICalTime(v, p.ICalParameters, local)
			if err != nil {
				continue
			}
			if dateOnly {
				ev.exDateDays = append(ev.exDateDays, t.Format(dateLayout))
			} else {
				ev.ExDates = append(ev.ExDates, t)
			}
		}
	}

	if p := ve.GetProperty(ics.ComponentPropertyRecurrenceId); p != nil {
		if t, _, err := parseICalTime(p.Value, p.ICalParameters, local); err == nil {
			ev.RecurrenceID = t
			ev.HasRecurrenceID = true
		}
	}

	return ev, true
}

// eventEnd resolve o fim por DTEND, DURATION ou, na falta dos dois, pela
// regra da RFC 5545: dia inteiro dura um dia; com horário, dura zero.
func eventEnd(ve *ics.VEvent, ev Event, local *time.Location) time.Time {
	if p := ve.GetProperty(ics.ComponentPropertyDtEnd); p != nil {
		if end, _, err := parseICalTime(p.Value, p.ICalParameters, local); err == nil && !end.Before(ev.Start) {
			return end
		}
	}
	if p := ve.GetProperty(ics.ComponentPropertyDuration); p != nil {
		if d, err := parseICalDuration(p.Value); err == nil && d >= 0 {
			return ev.Start.Add(d)
		}
	}
	if ev.AllDay {
		return ev.Start.AddDate(0, 0, 1)
	}
	return ev.Start
}

func textValue(ve *ics.VEvent, prop ics.ComponentProperty) string {
	p := ve.GetProperty(prop)
	if p == nil {
		return ""
	}
	return ics.FromText(p.Value)
}

func firstParam(params map[string][]string, name string) string {
	for k, v := range params {
		if strings.EqualFold(k, name) && len(v) > 0 {
			return v[0]
		}
	}
	return ""
}

// normalizeEmail tira o "mailto:" (em qualquer caixa) e deixa em minúsculas.
func normalizeEmail(v string) string {
	v = strings.TrimSpace(v)
	if len(v) >= 7 && strings.EqualFold(v[:7], "mailto:") {
		v = v[7:]
	}
	return strings.ToLower(strings.TrimSpace(v))
}

const (
	dateLayout      = "2006-01-02"
	icsDate         = "20060102"
	icsDateTime     = "20060102T150405"
	icsDateTimeUTC  = "20060102T150405Z"
	icsDateTimeNoSS = "20060102T1504"
)

// parseICalTime interpreta DATE, DATE-TIME UTC (Z), com TZID ou flutuante.
// dateOnly informa que o valor era só uma data (meia-noite em local).
func parseICalTime(value string, params map[string][]string, local *time.Location) (time.Time, bool, error) {
	value = strings.TrimSpace(value)
	if strings.EqualFold(firstParam(params, "VALUE"), "DATE") || len(value) == len(icsDate) {
		t, err := time.ParseInLocation(icsDate, value, local)
		return t, true, err
	}
	if strings.HasSuffix(value, "Z") || strings.HasSuffix(value, "z") {
		t, err := time.ParseInLocation(icsDateTimeUTC, strings.ToUpper(value), time.UTC)
		if err != nil {
			return time.Time{}, false, err
		}
		return t, false, nil
	}

	loc := local
	if tzid := strings.Trim(firstParam(params, "TZID"), `"`); tzid != "" {
		loc = resolveTZID(tzid, local)
	}
	t, err := time.ParseInLocation(icsDateTime, value, loc)
	if err != nil {
		// Alguns geradores omitem os segundos.
		t, err = time.ParseInLocation(icsDateTimeNoSS, value, loc)
		if err != nil {
			return time.Time{}, false, err
		}
	}
	return t, false, nil
}

// resolveTZID aceita nomes IANA, nomes de fuso do Windows (Outlook) e prefixos
// como "/mozilla.org/20050126_1/America/Sao_Paulo". Um TZID desconhecido (ex.:
// "Customized Time Zone") cai no fuso local, como um horário flutuante.
func resolveTZID(tzid string, local *time.Location) *time.Location {
	if loc, err := time.LoadLocation(tzid); err == nil {
		return loc
	}
	if loc := ics.WindowsTimezoneToIANA(tzid); loc != nil {
		return loc
	}
	parts := strings.Split(strings.Trim(tzid, "/"), "/")
	for i := 1; i < len(parts); i++ {
		if loc, err := time.LoadLocation(strings.Join(parts[i:], "/")); err == nil {
			return loc
		}
	}
	return local
}

var durationRe = regexp.MustCompile(`^([+-])?P(?:(\d+)W)?(?:(\d+)D)?(?:T(?:(\d+)H)?(?:(\d+)M)?(?:(\d+)S)?)?$`)

// parseICalDuration interpreta DURATION da RFC 5545 (ex.: PT1H30M, P1D, P2W).
func parseICalDuration(v string) (time.Duration, error) {
	m := durationRe.FindStringSubmatch(strings.ToUpper(strings.TrimSpace(v)))
	if m == nil || v == "P" || strings.HasSuffix(strings.ToUpper(v), "T") {
		return 0, fmt.Errorf("duração inválida: %q", v)
	}
	num := func(s string) time.Duration {
		if s == "" {
			return 0
		}
		n, _ := strconv.Atoi(s)
		return time.Duration(n)
	}
	d := num(m[2])*7*24*time.Hour + num(m[3])*24*time.Hour +
		num(m[4])*time.Hour + num(m[5])*time.Minute + num(m[6])*time.Second
	if m[1] == "-" {
		d = -d
	}
	return d, nil
}
