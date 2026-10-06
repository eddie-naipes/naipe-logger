package backend

import (
	"context"
	"errors"
	"log/slog"
	"path/filepath"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"logTime-go/backend/api"
	"logTime-go/backend/config"
	"logTime-go/backend/internal/fsutil"
	"logTime-go/backend/notify"
	"logTime-go/backend/reminders"
	"logTime-go/backend/timer"
)

// Infraestrutura comum aos lembretes e ao cronômetro: notificações nativas do
// sistema (Wails 2.16), o roteamento dos cliques nelas e as goroutines de
// verificação, que vivem enquanto o app estiver aberto.
//
// Eventos emitidos para o frontend:
//   - "reminder:open" (ReminderOpen {route, date}): o usuário clicou num
//     lembrete; o frontend navega para a rota;
//   - "timer:changed" (timer.State): o cronômetro mudou, inclusive por ação
//     de notificação;
//   - "timer:logged" (TimerLogged): o cronômetro foi parado e lançado (ou
//     falhou) por ação de notificação.

const (
	EventReminderOpen = "reminder:open"
	EventTimerChanged = "timer:changed"
	EventTimerLogged  = "timer:logged"
)

// checkInterval é a frequência das verificações de lembrete e de cronômetro
// esquecido.
const checkInterval = time.Minute

var errNotificacoesIndisponiveis = errors.New("as notificações do sistema não estão disponíveis neste computador")

// Indireção sobre o runtime de janela, que exige o contexto real do Wails.
var bringWindowToFront = func(ctx context.Context) {
	runtime.WindowUnminimise(ctx)
	runtime.WindowShow(ctx)
}

// ReminderOpen é o payload de "reminder:open".
type ReminderOpen struct {
	Route string `json:"route"`
	Date  string `json:"date,omitempty"`
}

// TimerLogged é o payload de "timer:logged".
type TimerLogged struct {
	Logged bool             `json:"logged"`
	Error  string           `json:"error,omitempty"`
	Result timer.StopResult `json:"result"`
}

// NotificationStatus informa à tela de Configurações se as notificações do
// sistema funcionam.
type NotificationStatus struct {
	Available bool   `json:"available"`
	Error     string `json:"error,omitempty"`
}

// features agrupa o estado dos lembretes e do cronômetro dentro de App.
type features struct {
	once      sync.Once
	mu        sync.Mutex
	reminders *reminders.Scheduler
	timer     *timer.Service
	notifier  notify.Sender
	status    NotificationStatus
	cancel    context.CancelFunc
}

// initFeatures cria o agendador e o cronômetro (uma vez). Os testes podem
// preenchê-los antes, e então nada é criado.
func (a *App) initFeatures() {
	a.features.once.Do(func() {
		a.features.mu.Lock()
		defer a.features.mu.Unlock()

		var remindersPath, timerPath string
		if dir, err := fsutil.AppDir(); err == nil {
			remindersPath = filepath.Join(dir, "reminders.json")
			timerPath = filepath.Join(dir, "timer.json")
		}
		if a.features.notifier == nil {
			a.features.notifier = &unavailableNotifier{}
		}
		if a.features.reminders == nil {
			a.features.reminders = reminders.New(reminders.Options{
				Settings:   a.reminderSettings,
				Source:     a.reminderSource,
				Sender:     senderFunc(a.send),
				StatePath:  remindersPath,
				AuditCount: a.monthAuditPendingCount,
			})
		}
		if a.features.timer == nil {
			a.features.timer = timer.New(timer.Options{
				Path:     timerPath,
				Settings: a.timerSettings,
				OnChange: a.emitTimerChanged,
			})
		}
	})
}

func (a *App) reminderScheduler() *reminders.Scheduler {
	a.initFeatures()
	a.features.mu.Lock()
	defer a.features.mu.Unlock()
	return a.features.reminders
}

func (a *App) timerService() *timer.Service {
	a.initFeatures()
	a.features.mu.Lock()
	defer a.features.mu.Unlock()
	return a.features.timer
}

func (a *App) reminderSettings() config.ReminderSettings {
	if a.configManager == nil {
		return config.DefaultReminderSettings()
	}
	return a.configManager.GetReminderSettings()
}

func (a *App) timerSettings() config.TimerSettings {
	if a.configManager == nil {
		return config.DefaultTimerSettings()
	}
	return a.configManager.GetTimerSettings()
}

func (a *App) reminderSource() (reminders.Source, int, error) {
	c, err := a.client()
	if err != nil {
		return nil, 0, err
	}
	return c, c.Config.MinutosPorDia, nil
}

// send envia pela implementação atual (a do Wails depois do Startup).
func (a *App) send(n notify.Notification) error {
	a.features.mu.Lock()
	sender := a.features.notifier
	a.features.mu.Unlock()
	if sender == nil {
		return errNotificacoesIndisponiveis
	}
	return sender.Send(n)
}

type senderFunc func(notify.Notification) error

func (f senderFunc) Send(n notify.Notification) error { return f(n) }

type unavailableNotifier struct{}

func (unavailableNotifier) Send(notify.Notification) error { return errNotificacoesIndisponiveis }

// startFeatures liga as notificações e as verificações periódicas. Chamado no
// Startup; tudo para no Shutdown.
func (a *App) startFeatures(ctx context.Context) {
	a.initFeatures()
	bg, cancel := context.WithCancel(ctx)

	notifier, status := newWailsNotifier(ctx, a.handleNotificationResponse)

	a.features.mu.Lock()
	a.features.cancel = cancel
	a.features.notifier = notifier
	a.features.status = status
	scheduler, tm := a.features.reminders, a.features.timer
	a.features.mu.Unlock()

	go scheduler.Run(bg, checkInterval)
	go tm.Watch(bg, checkInterval, senderFunc(a.send))
}

// stopFeatures encerra as goroutines e libera as notificações.
func (a *App) stopFeatures(ctx context.Context) {
	a.features.mu.Lock()
	cancel := a.features.cancel
	available := a.features.status.Available
	a.features.cancel = nil
	a.features.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if available {
		runtime.CleanupNotifications(ctx)
	}
}

// handleNotificationResponse trata o clique numa notificação ou num botão.
func (a *App) handleNotificationResponse(resp notify.Response) {
	ctx := a.appContext()
	switch resp.Kind() {
	case reminders.Kind:
		route := resp.String("route")
		if route != reminders.RouteComplete && route != reminders.RouteMonthClose {
			route = reminders.RouteTimeLog
		}
		bringWindowToFront(ctx)
		emitEvent(ctx, EventReminderOpen, ReminderOpen{Route: route, Date: resp.String("date")})
	case timer.Kind:
		switch resp.ActionID {
		case timer.ActionStopLog:
			a.stopTimerFromNotification()
		case timer.ActionContinue:
			a.timerService().Snooze()
		default:
			bringWindowToFront(ctx)
		}
	default:
		bringWindowToFront(ctx)
	}
}

// stopTimerFromNotification para e lança o cronômetro com a descrição atual.
func (a *App) stopTimerFromNotification() {
	ctx := a.appContext()
	payload := TimerLogged{}
	client, err := a.client()
	if err == nil {
		svc := a.timerService()
		payload.Result, err = svc.Stop(true, svc.State().Description, nil, client)
	}
	if err != nil {
		slog.Warn("Não foi possível parar e lançar o cronômetro pela notificação", "err", err)
		payload.Error = err.Error()
		payload.Result.State = a.timerService().State()
		bringWindowToFront(ctx)
	}
	if payload.Result.Results == nil {
		payload.Result.Results = []api.TimeLogResult{}
	}
	payload.Logged = payload.Result.Logged
	emitEvent(ctx, EventTimerLogged, payload)
}

// emitTimerChanged avisa o frontend; antes do Startup não há janela.
func (a *App) emitTimerChanged(st timer.State) {
	a.apiMutex.RLock()
	ctx := a.ctx
	a.apiMutex.RUnlock()
	if ctx == nil {
		return
	}
	emitEvent(ctx, EventTimerChanged, st)
}

// GetNotificationStatus informa se as notificações do sistema estão
// disponíveis, para a tela de Configurações orientar o usuário.
func (a *App) GetNotificationStatus() NotificationStatus {
	a.features.mu.Lock()
	defer a.features.mu.Unlock()
	return a.features.status
}

// wailsNotifier envia pelo runtime do Wails.
type wailsNotifier struct {
	ctx context.Context
}

func (n *wailsNotifier) Send(notif notify.Notification) error {
	opts := runtime.NotificationOptions{
		ID:         notif.ID,
		Title:      notif.Title,
		Body:       notif.Body,
		CategoryID: notif.CategoryID,
		Data:       notif.Data,
	}
	if notif.CategoryID != "" {
		return runtime.SendNotificationWithActions(n.ctx, opts)
	}
	return runtime.SendNotification(n.ctx, opts)
}

// newWailsNotifier inicializa as notificações nativas e registra as
// categorias (botões). Sem suporte, devolve um Sender que sempre falha e o
// motivo no status — o app segue funcionando sem lembretes visíveis.
func newWailsNotifier(ctx context.Context, onResponse func(notify.Response)) (sender notify.Sender, status NotificationStatus) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("Panic ao inicializar notificações", "panic", r)
			sender = unavailableNotifier{}
			status = NotificationStatus{Error: "falha ao inicializar as notificações"}
		}
	}()

	if err := runtime.InitializeNotifications(ctx); err != nil {
		slog.Warn("Notificações do sistema indisponíveis", "err", err)
		return unavailableNotifier{}, NotificationStatus{Error: err.Error()}
	}
	if !runtime.IsNotificationAvailable(ctx) {
		slog.Warn("Notificações do sistema indisponíveis neste ambiente")
		return unavailableNotifier{}, NotificationStatus{Error: errNotificacoesIndisponiveis.Error()}
	}
	if ok, err := runtime.RequestNotificationAuthorization(ctx); err != nil || !ok {
		slog.Warn("Notificações não autorizadas pelo usuário", "err", err)
		return unavailableNotifier{}, NotificationStatus{Error: "as notificações não foram autorizadas nas preferências do sistema"}
	}

	categories := append(reminders.Categories(), timer.Categories()...)
	for _, c := range categories {
		actions := make([]runtime.NotificationAction, len(c.Actions))
		for i, act := range c.Actions {
			actions[i] = runtime.NotificationAction{ID: act.ID, Title: act.Title}
		}
		if err := runtime.RegisterNotificationCategory(ctx, runtime.NotificationCategory{ID: c.ID, Actions: actions}); err != nil {
			slog.Warn("Não foi possível registrar a categoria de notificação", "categoria", c.ID, "err", err)
		}
	}

	runtime.OnNotificationResponse(ctx, func(result runtime.NotificationResult) {
		if result.Error != nil {
			slog.Warn("Resposta de notificação inválida", "err", result.Error)
			return
		}
		onResponse(notify.Response{
			ActionID: result.Response.ActionIdentifier,
			Data:     result.Response.UserInfo,
		})
	})

	return &wailsNotifier{ctx: ctx}, NotificationStatus{Available: true}
}
