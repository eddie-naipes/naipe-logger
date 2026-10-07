package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestAuditoriaConfigAntigoRecebePadroes(t *testing.T) {
	fakeKeyring(t)
	dir := t.TempDir()
	antigo := `{"appSettings":{"darkMode":true},"reminders":{"dailyTime":"17:30"}}`
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(antigo), 0600); err != nil {
		t.Fatal(err)
	}
	m, err := newManagerAt(dir)
	if err != nil {
		t.Fatal(err)
	}
	got := m.GetAuditSettings()
	if !reflect.DeepEqual(got, DefaultAuditSettings()) {
		t.Errorf("auditoria = %+v, esperava os padrões", got)
	}
	if got.GenericDescriptions == nil || got.IgnoredIssues == nil {
		t.Error("listas devem ser [] e não nil")
	}
}

// Um config com só parte da seção mantém o padrão do resto; valores
// absurdos editados à mão voltam ao padrão.
func TestAuditoriaConfigParcialEInvalido(t *testing.T) {
	fakeKeyring(t)
	dir := t.TempDir()
	parcial := `{"audit":{"genericDescriptions":[" reunião ","Reunião",""],"ignoredIssues":["duplicate:1,2","../x","duplicate:1,2"]}}`
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(parcial), 0600); err != nil {
		t.Fatal(err)
	}
	m, err := newManagerAt(dir)
	if err != nil {
		t.Fatal(err)
	}
	got := m.GetAuditSettings()
	want := AuditSettings{
		DailyLimitMinutes:   DefaultAuditDailyLimitMinutes,
		GenericDescriptions: []string{"reunião"},
		IgnoredIssues:       []string{"duplicate:1,2"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("auditoria = %+v, esperava %+v", got, want)
	}

	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(`{"audit":{"dailyLimitMinutes":-5}}`), 0600); err != nil {
		t.Fatal(err)
	}
	m2, err := newManagerAt(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := m2.GetAuditSettings().DailyLimitMinutes; got != DefaultAuditDailyLimitMinutes {
		t.Errorf("limite = %d", got)
	}
}

func TestSetAuditSettingsValidaPersisteEPreservaIgnorados(t *testing.T) {
	fakeKeyring(t)
	dir := t.TempDir()
	m, err := newManagerAt(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.IgnoreAuditIssue("missing_description:10"); err != nil {
		t.Fatal(err)
	}

	if err := m.SetAuditSettings(AuditSettings{DailyLimitMinutes: 25 * 60}); err == nil {
		t.Error("esperava erro para limite acima de 24h")
	}
	if err := m.SetAuditSettings(AuditSettings{DailyLimitMinutes: -1}); err == nil {
		t.Error("esperava erro para limite negativo")
	}

	novo := AuditSettings{DailyLimitMinutes: 0, GenericDescriptions: []string{"Ajustes", "ajustes "}, IgnoredIssues: []string{"no_task:99"}}
	if err := m.SetAuditSettings(novo); err != nil {
		t.Fatal(err)
	}

	recarregado, err := newManagerAt(dir)
	if err != nil {
		t.Fatal(err)
	}
	want := AuditSettings{DailyLimitMinutes: 0, GenericDescriptions: []string{"Ajustes"}, IgnoredIssues: []string{"missing_description:10"}}
	if got := recarregado.GetAuditSettings(); !reflect.DeepEqual(got, want) {
		t.Errorf("auditoria = %+v, esperava %+v", got, want)
	}
}

func TestIgnorarEReexibirProblema(t *testing.T) {
	fakeKeyring(t)
	dir := t.TempDir()
	m, err := newManagerAt(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, invalida := range []string{"", "sem-dois-pontos", "tipo:abc", "TIPO:1", "x:1;rm"} {
		if err := m.IgnoreAuditIssue(invalida); err == nil {
			t.Errorf("chave %q deveria ser recusada", invalida)
		}
	}

	for _, k := range []string{"duplicate:1,2", "over_daily_limit:2026-10-02", "duplicate:1,2"} {
		if err := m.IgnoreAuditIssue(k); err != nil {
			t.Fatal(err)
		}
	}
	if got := m.GetAuditSettings().IgnoredIssues; !reflect.DeepEqual(got, []string{"duplicate:1,2", "over_daily_limit:2026-10-02"}) {
		t.Errorf("ignorados = %v", got)
	}

	if err := m.UnignoreAuditIssue("duplicate:1,2"); err != nil {
		t.Fatal(err)
	}
	if err := m.UnignoreAuditIssue("inexistente:1"); err != nil {
		t.Fatal(err)
	}
	recarregado, err := newManagerAt(dir)
	if err != nil {
		t.Fatal(err)
	}
	if got := recarregado.GetAuditSettings().IgnoredIssues; !reflect.DeepEqual(got, []string{"over_daily_limit:2026-10-02"}) {
		t.Errorf("ignorados após recarregar = %v", got)
	}
}
