package reminders

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"logTime-go/backend/config"
	"logTime-go/backend/notify"
)

// fonteFalsa simula o Teamwork: minutos por dia e feriados.
type fonteFalsa struct {
	mu        sync.Mutex
	minutos   map[string]int
	feriados  map[string]bool
	err       error
	consultas int
}

func (f *fonteFalsa) DailyLoggedMinutes(start, end string) (map[string]int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.consultas++
	if f.err != nil {
		return nil, f.err
	}
	out := map[string]int{}
	for k, v := range f.minutos {
		if k >= start && k <= end {
			out[k] = v
		}
	}
	return out, nil
}

func (f *fonteFalsa) IsWorkDay(d time.Time) bool {
	if d.Weekday() == time.Saturday || d.Weekday() == time.Sunday {
		return false
	}
	return !f.feriados[d.Format("2006-01-02")]
}

type relogio struct{ agora time.Time }

func (r *relogio) Now() time.Time { return r.agora }

func data(s string) time.Time {
	t, err := time.ParseInLocation("2006-01-02 15:04", s, time.Local)
	if err != nil {
		panic(err)
	}
	return t
}

func novoAgendador(t *testing.T, fonte *fonteFalsa, agora string, ajuste func(*config.ReminderSettings)) (*Scheduler, *notify.Recorder, *relogio, string) {
	t.Helper()
	cfg := config.DefaultReminderSettings()
	if ajuste != nil {
		ajuste(&cfg)
	}
	rec := &notify.Recorder{}
	rel := &relogio{agora: data(agora)}
	path := filepath.Join(t.TempDir(), "reminders.json")
	s := New(Options{
		Settings:  func() config.ReminderSettings { return cfg },
		Source:    func() (Source, int, error) { return fonte, 480, nil },
		Sender:    rec,
		Now:       rel.Now,
		StatePath: path,
	})
	return s, rec, rel, path
}

func TestLembreteDiarioDisparaNoHorarioENaoRepete(t *testing.T) {
	fonte := &fonteFalsa{minutos: map[string]int{"2026-09-09": 330}}
	s, rec, rel, _ := novoAgendador(t, fonte, "2026-09-09 17:59", nil)

	if got := s.Check(); len(got) != 0 {
		t.Fatalf("antes do horário enviou %v", got)
	}
	if fonte.consultas != 0 {
		t.Errorf("consultou a API antes do horário")
	}

	rel.agora = data("2026-09-09 18:00")
	if got := s.Check(); len(got) != 1 || got[0] != CategoryDaily {
		t.Fatalf("no horário enviou %v, esperava o diário", got)
	}
	enviado := rec.Sent()[0]
	if enviado.Body != "Faltam 2h 30min para fechar o dia" {
		t.Errorf("corpo = %q", enviado.Body)
	}
	if enviado.Data["route"] != RouteTimeLog || enviado.Data["kind"] != Kind || enviado.Data["date"] != "2026-09-09" {
		t.Errorf("data = %v", enviado.Data)
	}

	rel.agora = data("2026-09-09 21:00")
	if got := s.Check(); len(got) != 0 {
		t.Errorf("repetiu no mesmo dia: %v", got)
	}
	if len(rec.Sent()) != 1 {
		t.Errorf("enviou %d notificações, esperava 1", len(rec.Sent()))
	}
}

func TestLembreteNaoRepeteAposReiniciar(t *testing.T) {
	fonte := &fonteFalsa{}
	s, rec, _, path := novoAgendador(t, fonte, "2026-09-09 18:30", nil)
	s.Check()
	if len(rec.Sent()) != 1 {
		t.Fatalf("esperava 1 envio, houve %d", len(rec.Sent()))
	}

	cfg := config.DefaultReminderSettings()
	rec2 := &notify.Recorder{}
	reiniciado := New(Options{
		Settings:  func() config.ReminderSettings { return cfg },
		Source:    func() (Source, int, error) { return fonte, 480, nil },
		Sender:    rec2,
		Now:       func() time.Time { return data("2026-09-09 19:00") },
		StatePath: path,
	})
	reiniciado.Check()
	if len(rec2.Sent()) != 0 {
		t.Errorf("repetiu depois de reiniciar: %v", rec2.Sent())
	}
}

func TestLembretePulaFimDeSemanaEFeriado(t *testing.T) {
	fonte := &fonteFalsa{feriados: map[string]bool{"2026-09-07": true}}
	s, rec, rel, _ := novoAgendador(t, fonte, "2026-09-06 18:00", nil) // domingo
	s.Check()
	rel.agora = data("2026-09-07 18:00") // feriado
	s.Check()
	if len(rec.Sent()) != 0 {
		t.Errorf("notificou em dia não útil: %v", rec.Sent())
	}
	if fonte.consultas != 0 {
		t.Errorf("consultou a API em dia não útil")
	}

	// Com "só dias úteis" desligado, avisa também no domingo.
	s2, rec2, _, _ := novoAgendador(t, fonte, "2026-09-06 18:00", func(c *config.ReminderSettings) { c.WorkDaysOnly = false })
	s2.Check()
	if len(rec2.Sent()) != 1 {
		t.Errorf("com WorkDaysOnly=false esperava 1 envio, houve %d", len(rec2.Sent()))
	}
}

func TestLembreteNaoNotificaComJornadaCompleta(t *testing.T) {
	fonte := &fonteFalsa{minutos: map[string]int{"2026-09-09": 480}}
	s, rec, rel, _ := novoAgendador(t, fonte, "2026-09-09 18:05", nil)
	s.Check()
	if len(rec.Sent()) != 0 {
		t.Errorf("notificou com a jornada completa: %v", rec.Sent())
	}
	rel.agora = data("2026-09-09 18:06")
	s.Check()
	if fonte.consultas != 1 {
		t.Errorf("consultou %d vezes, esperava 1 (dia já avaliado)", fonte.consultas)
	}
}

func TestLembreteDesligadoNaoFazNada(t *testing.T) {
	fonte := &fonteFalsa{}
	s, rec, _, _ := novoAgendador(t, fonte, "2026-09-09 20:00", func(c *config.ReminderSettings) { c.Enabled = false })
	s.Check()
	if len(rec.Sent()) != 0 || fonte.consultas != 0 {
		t.Errorf("lembrete desligado agiu: %v / %d consultas", rec.Sent(), fonte.consultas)
	}
}

func TestLembreteErroDaAPIEsperaAntesDeTentarDeNovo(t *testing.T) {
	fonte := &fonteFalsa{err: errors.New("fora do ar")}
	s, rec, rel, _ := novoAgendador(t, fonte, "2026-09-09 18:00", nil)
	s.Check()
	rel.agora = data("2026-09-09 18:01")
	s.Check()
	if fonte.consultas != 1 {
		t.Errorf("consultou %d vezes, esperava esperar %v", fonte.consultas, retryDelay)
	}

	fonte.err = nil
	rel.agora = data("2026-09-09 18:16")
	s.Check()
	if len(rec.Sent()) != 1 {
		t.Errorf("depois da espera esperava 1 envio, houve %d", len(rec.Sent()))
	}
}

func TestLembreteSemConexaoNaoMarcaODia(t *testing.T) {
	cfg := config.DefaultReminderSettings()
	s := New(Options{
		Settings: func() config.ReminderSettings { return cfg },
		Source:   func() (Source, int, error) { return nil, 0, errors.New("sem conexão") },
		Sender:   &notify.Recorder{},
		Now:      func() time.Time { return data("2026-09-09 18:00") },
	})
	s.Check()
	if s.State().LastDaily != "" {
		t.Errorf("marcou o dia sem conseguir avaliar: %+v", s.State())
	}
}

// Setembro/2026: dias úteis 29 e 30 são os dois últimos.
func TestLembreteFimDeMesListaDiasPendentes(t *testing.T) {
	fonte := &fonteFalsa{minutos: map[string]int{}}
	for d := 1; d <= 30; d++ {
		fonte.minutos[time.Date(2026, 9, d, 0, 0, 0, 0, time.Local).Format("2006-01-02")] = 480
	}
	fonte.minutos["2026-09-02"] = 0
	fonte.minutos["2026-09-15"] = 240
	delete(fonte.minutos, "2026-09-25")

	// 28/09 ainda não está nos 2 últimos dias úteis.
	s, rec, rel, _ := novoAgendador(t, fonte, "2026-09-28 18:00", nil)
	s.Check()
	if len(rec.Sent()) != 0 {
		t.Fatalf("antes da janela enviou %v", rec.Sent())
	}

	rel.agora = data("2026-09-29 18:00")
	got := s.Check()
	if len(got) != 1 || got[0] != CategoryMonthEnd {
		t.Fatalf("enviou %v, esperava só o de fim de mês", got)
	}
	n := rec.Sent()[0]
	if n.Body != "3 dias pendentes: 02, 15, 25" {
		t.Errorf("corpo = %q", n.Body)
	}
	if n.Data["route"] != RouteComplete {
		t.Errorf("rota = %v", n.Data["route"])
	}
}

func TestLembreteFimDeMesIncluiHojeEDispensaODiario(t *testing.T) {
	fonte := &fonteFalsa{minutos: map[string]int{"2026-09-30": 60}}
	for d := 1; d < 30; d++ {
		fonte.minutos[time.Date(2026, 9, d, 0, 0, 0, 0, time.Local).Format("2006-01-02")] = 480
	}
	s, rec, _, _ := novoAgendador(t, fonte, "2026-09-30 18:00", nil)
	got := s.Check()
	if len(got) != 1 || got[0] != CategoryMonthEnd {
		t.Fatalf("enviou %v, esperava só o de fim de mês", got)
	}
	if !strings.HasPrefix(rec.Sent()[0].Body, "1 dia pendente: 30") {
		t.Errorf("corpo = %q", rec.Sent()[0].Body)
	}
}

func TestLembreteFimDeMesDesligadoUsaSoODiario(t *testing.T) {
	fonte := &fonteFalsa{}
	s, _, _, _ := novoAgendador(t, fonte, "2026-09-30 18:00", func(c *config.ReminderSettings) { c.MonthEndEnabled = false })
	if got := s.Check(); len(got) != 1 || got[0] != CategoryDaily {
		t.Errorf("enviou %v, esperava só o diário", got)
	}
}

func TestIsInLastWorkDaysConsideraFeriado(t *testing.T) {
	fonte := &fonteFalsa{feriados: map[string]bool{"2026-09-30": true}}
	// Com 30/09 feriado, os 2 últimos dias úteis são 28 e 29.
	if !isInLastWorkDays(data("2026-09-28 10:00"), 2, fonte.IsWorkDay) {
		t.Error("28/09 deveria estar na janela")
	}
	if isInLastWorkDays(data("2026-09-25 10:00"), 2, fonte.IsWorkDay) {
		t.Error("25/09 não deveria estar na janela")
	}
}

func TestFormatMinutes(t *testing.T) {
	casos := map[int]string{45: "45min", 60: "1h", 150: "2h 30min", 0: "0min"}
	for in, want := range casos {
		if got := FormatMinutes(in); got != want {
			t.Errorf("FormatMinutes(%d) = %q, esperava %q", in, got, want)
		}
	}
}

func TestSendTestUsaOSender(t *testing.T) {
	s, rec, _, _ := novoAgendador(t, &fonteFalsa{}, "2026-09-09 10:00", nil)
	if err := s.SendTest(); err != nil {
		t.Fatal(err)
	}
	if len(rec.Sent()) != 1 || rec.Sent()[0].CategoryID != CategoryDaily {
		t.Errorf("enviado = %v", rec.Sent())
	}
}

func TestRunEncerraComOContexto(t *testing.T) {
	fonte := &fonteFalsa{}
	s, rec, _, _ := novoAgendador(t, fonte, "2026-09-09 18:00", nil)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		s.Run(ctx, time.Millisecond)
		close(done)
	}()

	prazo := time.After(2 * time.Second)
	for len(rec.Sent()) == 0 {
		select {
		case <-prazo:
			t.Fatal("Run não verificou os lembretes")
		default:
			time.Sleep(time.Millisecond)
		}
	}
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Run não encerrou com o contexto")
	}
}
