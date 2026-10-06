package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAgendaPadraoParaConfigAntigo(t *testing.T) {
	fakeKeyring(t)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(legacyConfigJSON), 0600); err != nil {
		t.Fatal(err)
	}
	m, err := newManagerAt(dir)
	if err != nil {
		t.Fatalf("newManagerAt: %v", err)
	}
	s := m.GetAgendaSettings()
	if s.Rounding != AgendaRoundingExact || s.MinMinutes != 5 || !s.Billable || s.Rules == nil || s.Calendars == nil || s.IgnoreWords == nil {
		t.Errorf("padrão = %+v", s)
	}
}

func TestSetAgendaSettingsValidaEPreservaAgendas(t *testing.T) {
	fakeKeyring(t)
	dir := t.TempDir()
	m, err := newManagerAt(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.AddAgendaCalendar(AgendaCalendar{ID: "a1", Name: "Trabalho", MaskedURL: "agenda.exemplo.com…ic.ics"}); err != nil {
		t.Fatal(err)
	}
	if err := m.AddAgendaCalendar(AgendaCalendar{ID: "a1"}); err == nil {
		t.Error("id repetido aceito")
	}

	invalidas := []AgendaSettings{
		{Rules: []AgendaRule{{Match: "(", IsRegex: true, Task: AgendaTask{TaskID: 1}}}},
		{Rules: []AgendaRule{{Match: "  ", Task: AgendaTask{TaskID: 1}}}},
		{Rules: []AgendaRule{{Match: "daily"}}},
		{MinMinutes: -1},
		{UserEmail: "sem-arroba"},
	}
	for i, s := range invalidas {
		if err := m.SetAgendaSettings(s); err == nil {
			t.Errorf("caso %d aceito", i)
		}
	}

	err = m.SetAgendaSettings(AgendaSettings{
		Calendars:   []AgendaCalendar{{ID: "intruso"}},
		Rules:       []AgendaRule{{Match: " Daily ", Task: AgendaTask{TaskID: 7, TaskName: "Cerimônias"}}},
		IgnoreWords: []string{" almoço ", ""},
		Rounding:    "qualquer",
		UserEmail:   " eu@exemplo.com ",
	})
	if err != nil {
		t.Fatal(err)
	}

	m2, err := newManagerAt(dir)
	if err != nil {
		t.Fatal(err)
	}
	s := m2.GetAgendaSettings()
	if len(s.Calendars) != 1 || s.Calendars[0].ID != "a1" {
		t.Errorf("agendas = %+v", s.Calendars)
	}
	if s.Rules[0].Match != "Daily" || len(s.IgnoreWords) != 1 || s.Rounding != AgendaRoundingExact || s.UserEmail != "eu@exemplo.com" {
		t.Errorf("normalização = %+v", s)
	}

	if err := m2.RemoveAgendaCalendar("a1"); err != nil {
		t.Fatal(err)
	}
	if n := len(m2.GetAgendaSettings().Calendars); n != 0 {
		t.Errorf("agendas depois de remover = %d", n)
	}
}

func TestConfigJSONNaoGuardaLinkDaAgenda(t *testing.T) {
	fakeKeyring(t)
	dir := t.TempDir()
	m, err := newManagerAt(dir)
	if err != nil {
		t.Fatal(err)
	}
	_ = m.AddAgendaCalendar(AgendaCalendar{ID: "a1", MaskedURL: "agenda.exemplo.com…ic.ics"})
	data, err := os.ReadFile(filepath.Join(dir, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "https://") {
		t.Errorf("config.json contém um link: %s", data)
	}
}
