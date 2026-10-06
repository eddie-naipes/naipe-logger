package agenda

import (
	"strings"
	"testing"
	"time"
)

// Agendas fictícias: nenhum dado real.

func mustLoc(t *testing.T, name string) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation(name)
	if err != nil {
		t.Fatal(err)
	}
	return loc
}

func cal(eventos ...string) string {
	return "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//Teste//PT\r\n" +
		strings.Join(eventos, "") + "END:VCALENDAR\r\n"
}

func vevent(linhas ...string) string {
	return "BEGIN:VEVENT\r\n" + strings.Join(linhas, "\r\n") + "\r\nEND:VEVENT\r\n"
}

func parseExpand(t *testing.T, ics string, loc *time.Location, de, ate string) []Occurrence {
	t.Helper()
	evs, err := Parse(strings.NewReader(ics), loc)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	from, _ := time.ParseInLocation(dateLayout, de, loc)
	to, _ := time.ParseInLocation(dateLayout, ate, loc)
	return Expand(evs, "Trabalho", from, to.AddDate(0, 0, 1), loc)
}

func horarios(occs []Occurrence) []string {
	out := make([]string, len(occs))
	for i, o := range occs {
		out[i] = o.Start.Format("2006-01-02 15:04") + "-" + o.End.Format("15:04")
	}
	return out
}

func iguais(t *testing.T, got, want []string) {
	t.Helper()
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("obtido %v\nesperado %v", got, want)
	}
}

func TestParseConverteUTCETZIDParaOFusoLocal(t *testing.T) {
	sp := mustLoc(t, "America/Sao_Paulo")
	ics := cal(
		vevent("UID:a@x", "SUMMARY:Daily", "DTSTART:20261005T120000Z", "DTEND:20261005T121500Z"),
		vevent("UID:b@x", "SUMMARY:Com Lisboa", "DTSTART;TZID=Europe/Lisbon:20261005T150000", "DTEND;TZID=Europe/Lisbon:20261005T160000"),
		vevent("UID:c@x", "SUMMARY:Outlook", "DTSTART;TZID=E. South America Standard Time:20261005T170000", "DURATION:PT1H30M"),
		vevent("UID:d@x", "SUMMARY:Flutuante", "DTSTART:20261005T190000", "DTEND:20261005T193000"),
		vevent("UID:e@x", "SUMMARY:Mozilla", "DTSTART;TZID=/mozilla.org/20050126_1/America/Sao_Paulo:20261005T200000", "DTEND;TZID=/mozilla.org/20050126_1/America/Sao_Paulo:20261005T201000"),
	)
	iguais(t, horarios(parseExpand(t, ics, sp, "2026-10-05", "2026-10-05")), []string{
		"2026-10-05 09:00-09:15",
		"2026-10-05 11:00-12:00",
		"2026-10-05 17:00-18:30",
		"2026-10-05 19:00-19:30",
		"2026-10-05 20:00-20:10",
	})
}

func TestParseTituloComEscapesEDiaInteiro(t *testing.T) {
	sp := mustLoc(t, "America/Sao_Paulo")
	ics := cal(
		vevent("UID:a@x", `SUMMARY:Reunião\, planejamento\; Q4`, "DTSTART;VALUE=DATE:20261005", "DTEND;VALUE=DATE:20261006"),
	)
	occs := parseExpand(t, ics, sp, "2026-10-05", "2026-10-05")
	if len(occs) != 1 || !occs[0].AllDay || occs[0].Summary != "Reunião, planejamento; Q4" {
		t.Fatalf("ocorrências = %+v", occs)
	}
}

func TestParseRecusaConteudoQueNaoEAgenda(t *testing.T) {
	if _, err := Parse(strings.NewReader("<html>login</html>"), time.UTC); err == nil {
		t.Error("HTML aceito como agenda")
	}
}

func TestExpandSemanalComByDayIntervalEUntil(t *testing.T) {
	sp := mustLoc(t, "America/Sao_Paulo")
	ics := cal(vevent("UID:r@x", "SUMMARY:Status",
		"DTSTART;TZID=America/Sao_Paulo:20261005T100000",
		"DTEND;TZID=America/Sao_Paulo:20261005T103000",
		"RRULE:FREQ=WEEKLY;INTERVAL=2;BYDAY=MO,WE;UNTIL=20261031T235959Z"))
	iguais(t, horarios(parseExpand(t, ics, sp, "2026-10-01", "2026-11-30")), []string{
		"2026-10-05 10:00-10:30",
		"2026-10-07 10:00-10:30",
		"2026-10-19 10:00-10:30",
		"2026-10-21 10:00-10:30",
	})
}

func TestExpandDiarioComCountEExdate(t *testing.T) {
	sp := mustLoc(t, "America/Sao_Paulo")
	ics := cal(vevent("UID:d@x", "SUMMARY:Daily",
		"DTSTART;TZID=America/Sao_Paulo:20261005T090000",
		"DURATION:PT15M",
		"RRULE:FREQ=DAILY;COUNT=4",
		"EXDATE;TZID=America/Sao_Paulo:20261006T090000"))
	iguais(t, horarios(parseExpand(t, ics, sp, "2026-10-01", "2026-10-31")), []string{
		"2026-10-05 09:00-09:15",
		"2026-10-07 09:00-09:15",
		"2026-10-08 09:00-09:15",
	})
}

func TestExpandMensalComExdateSoComData(t *testing.T) {
	sp := mustLoc(t, "America/Sao_Paulo")
	ics := cal(vevent("UID:m@x", "SUMMARY:Retro",
		"DTSTART:20260105T170000Z", "DTEND:20260105T180000Z",
		"RRULE:FREQ=MONTHLY;BYMONTHDAY=5",
		"EXDATE;VALUE=DATE:20260305"))
	iguais(t, horarios(parseExpand(t, ics, sp, "2026-02-01", "2026-04-30")), []string{
		"2026-02-05 14:00-15:00",
		"2026-04-05 14:00-15:00",
	})
}

func TestExpandRecurrenceIDSubstituiAOcorrenciaOriginal(t *testing.T) {
	sp := mustLoc(t, "America/Sao_Paulo")
	ics := cal(
		vevent("UID:w@x", "SUMMARY:Semanal",
			"DTSTART;TZID=America/Sao_Paulo:20261005T140000",
			"DTEND;TZID=America/Sao_Paulo:20261005T150000",
			"RRULE:FREQ=WEEKLY;COUNT=3"),
		// A de 12/10 foi para as 16h; a de 19/10 foi movida para fora do período.
		vevent("UID:w@x", "SUMMARY:Semanal (remarcada)",
			"RECURRENCE-ID;TZID=America/Sao_Paulo:20261012T140000",
			"DTSTART;TZID=America/Sao_Paulo:20261012T160000",
			"DTEND;TZID=America/Sao_Paulo:20261012T163000"),
		vevent("UID:w@x", "SUMMARY:Semanal",
			"RECURRENCE-ID;TZID=America/Sao_Paulo:20261019T140000",
			"DTSTART;TZID=America/Sao_Paulo:20261102T140000",
			"DTEND;TZID=America/Sao_Paulo:20261102T150000"),
	)
	occs := parseExpand(t, ics, sp, "2026-10-01", "2026-10-31")
	iguais(t, horarios(occs), []string{
		"2026-10-05 14:00-15:00",
		"2026-10-12 16:00-16:30",
	})
	if occs[1].Summary != "Semanal (remarcada)" {
		t.Errorf("título da ocorrência alterada = %q", occs[1].Summary)
	}
	if occs[1].Key != "w@x|20261012T170000Z" {
		t.Errorf("a ocorrência alterada deve manter a chave da original, veio %q", occs[1].Key)
	}
	if occs[0].Key != "w@x|20261005T170000Z" {
		t.Errorf("chave = %q", occs[0].Key)
	}
}

func TestExpandOcorrenciaCanceladaPorRecurrenceID(t *testing.T) {
	sp := mustLoc(t, "America/Sao_Paulo")
	ics := cal(
		vevent("UID:c@x", "SUMMARY:1:1", "DTSTART;TZID=America/Sao_Paulo:20261005T110000",
			"DTEND;TZID=America/Sao_Paulo:20261005T113000", "RRULE:FREQ=DAILY;COUNT=2"),
		vevent("UID:c@x", "SUMMARY:1:1", "STATUS:CANCELLED",
			"RECURRENCE-ID;TZID=America/Sao_Paulo:20261006T110000",
			"DTSTART;TZID=America/Sao_Paulo:20261006T110000",
			"DTEND;TZID=America/Sao_Paulo:20261006T113000"),
	)
	occs := parseExpand(t, ics, sp, "2026-10-05", "2026-10-06")
	if len(occs) != 2 || occs[0].Cancelled || !occs[1].Cancelled {
		t.Fatalf("ocorrências = %+v", occs)
	}
}

// Uma reunião semanal às 10h de Nova York continua às 10h de lá depois do fim
// do horário de verão americano (1º/11/2026); vista de São Paulo (sem horário
// de verão), passa das 11h para as 12h.
func TestExpandRespeitaHorarioDeVeraoDoFusoDoEvento(t *testing.T) {
	sp := mustLoc(t, "America/Sao_Paulo")
	ics := cal(vevent("UID:ny@x", "SUMMARY:Sync EUA",
		"DTSTART;TZID=America/New_York:20261026T100000",
		"DTEND;TZID=America/New_York:20261026T103000",
		"RRULE:FREQ=WEEKLY;BYDAY=MO;COUNT=2"))
	iguais(t, horarios(parseExpand(t, ics, sp, "2026-10-01", "2026-11-30")), []string{
		"2026-10-26 11:00-11:30",
		"2026-11-02 12:00-12:30",
	})
}

func TestExpandRRuleInvalidaFicaSoComAPrimeira(t *testing.T) {
	ics := cal(vevent("UID:x@x", "SUMMARY:X", "DTSTART:20261005T120000Z", "DTEND:20261005T130000Z", "RRULE:FREQ=QUANDOQUISER"))
	if n := len(parseExpand(t, ics, time.UTC, "2026-10-01", "2026-10-31")); n != 1 {
		t.Errorf("ocorrências = %d", n)
	}
}

func TestDedupeRemoveMesmaReuniaoEmDuasAgendas(t *testing.T) {
	occs := []Occurrence{{Key: "a"}, {Key: "b"}, {Key: "a"}}
	if n := len(Dedupe(occs)); n != 2 {
		t.Errorf("Dedupe = %d", n)
	}
}

func TestParseICalDuration(t *testing.T) {
	casos := map[string]time.Duration{
		"PT1H30M": 90 * time.Minute,
		"P1D":     24 * time.Hour,
		"P1W":     7 * 24 * time.Hour,
		"PT45S":   45 * time.Second,
		"-PT15M":  -15 * time.Minute,
	}
	for in, want := range casos {
		got, err := parseICalDuration(in)
		if err != nil || got != want {
			t.Errorf("%s = %v, %v; esperava %v", in, got, err, want)
		}
	}
	for _, in := range []string{"", "P", "PT", "1H"} {
		if _, err := parseICalDuration(in); err == nil {
			t.Errorf("%q aceito", in)
		}
	}
}
