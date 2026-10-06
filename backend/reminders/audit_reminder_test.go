package reminders

import (
	"errors"
	"testing"
	"time"

	"logTime-go/backend/config"
	"logTime-go/backend/notify"
)

func agendadorComAuditoria(t *testing.T, fonte *fonteFalsa, agora string, auditoria func(int, int) (int, error)) (*Scheduler, *notify.Recorder) {
	t.Helper()
	cfg := config.DefaultReminderSettings()
	rec := &notify.Recorder{}
	s := New(Options{
		Settings:   func() config.ReminderSettings { return cfg },
		Source:     func() (Source, int, error) { return fonte, 480, nil },
		Sender:     rec,
		Now:        func() time.Time { return data(agora) },
		AuditCount: auditoria,
	})
	return s, rec
}

func mesCompleto(ano, mes, ultimo int) *fonteFalsa {
	fonte := &fonteFalsa{minutos: map[string]int{}}
	for d := 1; d <= ultimo; d++ {
		fonte.minutos[time.Date(ano, time.Month(mes), d, 0, 0, 0, 0, time.Local).Format("2006-01-02")] = 480
	}
	return fonte
}

func TestLembreteFimDeMesComProblemasDeAuditoriaLevaAoFechamento(t *testing.T) {
	fonte := mesCompleto(2026, 9, 30)
	fonte.minutos["2026-09-15"] = 240
	var pedido [2]int
	s, rec := agendadorComAuditoria(t, fonte, "2026-09-29 18:00", func(ano, mes int) (int, error) {
		pedido = [2]int{ano, mes}
		return 4, nil
	})
	got := s.Check()
	if len(got) != 1 || got[0] != CategoryMonthClose {
		t.Fatalf("enviou %v", got)
	}
	n := rec.Sent()[0]
	if n.Body != "1 dia pendente: 15 · 4 problemas no fechamento do mês" {
		t.Errorf("corpo = %q", n.Body)
	}
	if n.Data["route"] != RouteMonthClose || n.CategoryID != CategoryMonthClose {
		t.Errorf("notificação = %+v", n)
	}
	if pedido != [2]int{2026, 9} {
		t.Errorf("auditou %v", pedido)
	}
}

// Mês com todas as horas, mas com problemas (ex.: duplicata): o lembrete sai
// mesmo sem dias pendentes.
func TestLembreteFimDeMesSoComProblemasDeAuditoria(t *testing.T) {
	s, rec := agendadorComAuditoria(t, mesCompleto(2026, 9, 30), "2026-09-30 18:00", func(int, int) (int, error) { return 1, nil })
	if got := s.Check(); len(got) != 1 || got[0] != CategoryMonthClose {
		t.Fatalf("enviou %v", got)
	}
	if body := rec.Sent()[0].Body; body != "1 problema no fechamento do mês" {
		t.Errorf("corpo = %q", body)
	}
}

func TestLembreteFimDeMesSemProblemasOuComErroNaAuditoriaMantemOComportamento(t *testing.T) {
	for nome, auditoria := range map[string]func(int, int) (int, error){
		"zero problemas": func(int, int) (int, error) { return 0, nil },
		"erro":           func(int, int) (int, error) { return 0, errors.New("fora do ar") },
	} {
		t.Run(nome, func(t *testing.T) {
			fonte := mesCompleto(2026, 9, 30)
			fonte.minutos["2026-09-02"] = 0
			s, rec := agendadorComAuditoria(t, fonte, "2026-09-29 18:00", auditoria)
			if got := s.Check(); len(got) != 1 || got[0] != CategoryMonthEnd {
				t.Fatalf("enviou %v", got)
			}
			n := rec.Sent()[0]
			if n.Body != "1 dia pendente: 02" || n.Data["route"] != RouteComplete {
				t.Errorf("notificação = %+v", n)
			}
		})
	}

	s, rec := agendadorComAuditoria(t, mesCompleto(2026, 9, 30), "2026-09-30 18:00", func(int, int) (int, error) { return 0, nil })
	if got := s.Check(); len(got) != 0 || len(rec.Sent()) != 0 {
		t.Errorf("mês fechado sem problemas não deveria lembrar: %v", got)
	}
}
