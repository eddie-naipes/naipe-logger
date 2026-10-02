package backend

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"logTime-go/backend/notify"
	"logTime-go/backend/reminders"
	"logTime-go/backend/timer"
)

type eventoEmitido struct {
	nome  string
	dados interface{}
}

// stubJanela troca emitEvent e bringWindowToFront por gravadores.
func stubJanela(t *testing.T) (*[]eventoEmitido, *int, *sync.Mutex) {
	t.Helper()
	var mu sync.Mutex
	var eventos []eventoEmitido
	frente := 0
	origEmit, origFront := emitEvent, bringWindowToFront
	emitEvent = func(_ context.Context, name string, data ...interface{}) {
		mu.Lock()
		defer mu.Unlock()
		var d interface{}
		if len(data) > 0 {
			d = data[0]
		}
		eventos = append(eventos, eventoEmitido{name, d})
	}
	bringWindowToFront = func(context.Context) {
		mu.Lock()
		defer mu.Unlock()
		frente++
	}
	t.Cleanup(func() { emitEvent, bringWindowToFront = origEmit, origFront })
	return &eventos, &frente, &mu
}

// appComRecursos monta um App sem tocar o HOME: cronômetro e lembretes em
// diretório temporário e notificações num Recorder.
func appComRecursos(t *testing.T, agora time.Time) (*App, *notify.Recorder) {
	t.Helper()
	a := appSemConexao()
	a.setContext(context.Background())
	rec := &notify.Recorder{}
	dir := t.TempDir()
	a.features.notifier = rec
	a.features.timer = timer.New(timer.Options{
		Path:     filepath.Join(dir, "timer.json"),
		Now:      func() time.Time { return agora },
		OnChange: a.emitTimerChanged,
	})
	a.features.reminders = reminders.New(reminders.Options{
		Settings: a.reminderSettings,
		Source:   a.reminderSource,
		Sender:   senderFunc(a.send),
		Now:      func() time.Time { return agora },
	})
	return a, rec
}

func TestCliqueNoLembreteEmiteEventoETrazJanela(t *testing.T) {
	eventos, frente, _ := stubJanela(t)
	a, _ := appComRecursos(t, time.Now())

	a.handleNotificationResponse(notify.Response{
		ActionID: reminders.ActionCompleteMonth,
		Data:     map[string]any{"kind": reminders.Kind, "route": reminders.RouteComplete, "date": "2026-09-30"},
	})
	if *frente != 1 || len(*eventos) != 1 {
		t.Fatalf("frente=%d eventos=%v", *frente, *eventos)
	}
	ev := (*eventos)[0]
	if ev.nome != EventReminderOpen {
		t.Errorf("evento = %s", ev.nome)
	}
	if got := ev.dados.(ReminderOpen); got.Route != "/completar" || got.Date != "2026-09-30" {
		t.Errorf("payload = %+v", got)
	}

	// Rota desconhecida cai no lançamento de horas.
	a.handleNotificationResponse(notify.Response{ActionID: notify.DefaultAction, Data: map[string]any{"kind": reminders.Kind, "route": "javascript:alert(1)"}})
	if got := (*eventos)[1].dados.(ReminderOpen); got.Route != "/timelog" {
		t.Errorf("rota não permitida virou %q", got.Route)
	}
}

func TestSendTestReminderUsaONotificador(t *testing.T) {
	a, rec := appComRecursos(t, time.Now())
	if err := a.SendTestReminder(); err != nil {
		t.Fatal(err)
	}
	if len(rec.Sent()) != 1 {
		t.Fatalf("enviados = %d", len(rec.Sent()))
	}

	a.features.notifier = unavailableNotifier{}
	if err := a.SendTestReminder(); !errors.Is(err, errNotificacoesIndisponiveis) {
		t.Errorf("sem notificações = %v", err)
	}
}

func TestCronometroEmiteTimerChanged(t *testing.T) {
	eventos, _, mu := stubJanela(t)
	a, _ := appComRecursos(t, time.Now())

	if _, err := a.StartTimer(timer.TaskRef{TaskID: 5, TaskName: "T"}, "", false); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(*eventos) != 1 || (*eventos)[0].nome != EventTimerChanged {
		t.Fatalf("eventos = %v", *eventos)
	}
	if st := (*eventos)[0].dados.(timer.State); !st.Running || st.TaskID != 5 {
		t.Errorf("estado emitido = %+v", st)
	}
}

func TestPararELancarPelaNotificacaoSemConexaoAvisaErro(t *testing.T) {
	eventos, frente, _ := stubJanela(t)
	a, _ := appComRecursos(t, time.Now())
	if _, err := a.StartTimer(timer.TaskRef{TaskID: 5}, "", false); err != nil {
		t.Fatal(err)
	}

	a.handleNotificationResponse(notify.Response{ActionID: timer.ActionStopLog, Data: map[string]any{"kind": timer.Kind}})

	ultimo := (*eventos)[len(*eventos)-1]
	payload, ok := ultimo.dados.(TimerLogged)
	if ultimo.nome != EventTimerLogged || !ok || payload.Logged || payload.Error == "" {
		t.Errorf("evento = %+v", ultimo)
	}
	if *frente != 1 {
		t.Errorf("a janela deveria vir para frente no erro")
	}
	if !a.GetTimerState().Running {
		t.Error("sem conexão o cronômetro não pode ser descartado")
	}
}

func TestStopTimerExigeConexaoParaLancar(t *testing.T) {
	stubJanela(t)
	a, _ := appComRecursos(t, time.Now())
	if _, err := a.StartTimer(timer.TaskRef{TaskID: 5}, "", false); err != nil {
		t.Fatal(err)
	}
	if _, err := a.StopTimer(true, "x", nil); !errors.Is(err, errAPINaoConfigurada) {
		t.Errorf("StopTimer sem conexão = %v", err)
	}
	if _, err := a.StopTimer(false, "", nil); err != nil {
		t.Errorf("descartar não exige conexão: %v", err)
	}
}

func TestLembretesSemConexaoNaoEnviam(t *testing.T) {
	a, rec := appComRecursos(t, time.Date(2026, 9, 9, 19, 0, 0, 0, time.Local))
	a.reminderScheduler().Check()
	if len(rec.Sent()) != 0 {
		t.Errorf("enviou sem conexão: %v", rec.Sent())
	}
}
