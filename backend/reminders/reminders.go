// Package reminders agenda os lembretes de lançamento de horas: o diário (no
// horário configurado, se o dia ainda não fechou a jornada) e o de fim de mês
// (nos últimos dias úteis, listando os dias do mês abaixo da jornada).
//
// O agendador roda numa goroutine presa ao contexto do app e verifica o
// relógio a cada intervalo. Relógio, configuração, fonte de dados e envio são
// injetados para que os testes não dependam da hora real, do Teamwork nem do
// sistema de notificações.
package reminders

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	"logTime-go/backend/config"
	"logTime-go/backend/internal/fsutil"
	"logTime-go/backend/notify"
)

// Identificadores das notificações. Kind é o valor de Data["kind"] usado para
// rotear a resposta; as rotas são as do frontend.
const (
	Kind = "reminder"

	CategoryDaily    = "lembrete-diario"
	CategoryMonthEnd = "lembrete-fim-de-mes"
	// CategoryMonthClose é o lembrete de fim de mês quando a auditoria do
	// fechamento encontrou problemas: leva para /fechamento.
	CategoryMonthClose = "lembrete-fechamento"

	ActionLogNow        = "lancar-agora"
	ActionCompleteMonth = "completar-mes"
	ActionReviewMonth   = "revisar-mes"

	RouteTimeLog    = "/timelog"
	RouteComplete   = "/completar"
	RouteMonthClose = "/fechamento"
)

// retryDelay é a espera antes de tentar de novo quando o Teamwork falha, para
// não consultar a API a cada minuto enquanto ela estiver fora.
const retryDelay = 15 * time.Minute

// Categories devolve as categorias a registrar no sistema de notificações.
func Categories() []notify.Category {
	return []notify.Category{
		{ID: CategoryDaily, Actions: []notify.Action{{ID: ActionLogNow, Title: "Lançar agora"}}},
		{ID: CategoryMonthEnd, Actions: []notify.Action{{ID: ActionCompleteMonth, Title: "Completar o mês"}}},
		{ID: CategoryMonthClose, Actions: []notify.Action{{ID: ActionReviewMonth, Title: "Revisar o mês"}}},
	}
}

// Source é o que o agendador precisa do Teamwork (satisfeito por
// *api.TeamworkAPI).
type Source interface {
	DailyLoggedMinutes(start, end string) (map[string]int, error)
	IsWorkDay(day time.Time) bool
}

// Options configura o agendador.
type Options struct {
	// Settings lê a configuração atual a cada verificação.
	Settings func() config.ReminderSettings
	// Source devolve o cliente e a jornada diária em minutos; erro quando não
	// há conexão configurada (a verificação é pulada).
	Source func() (Source, int, error)
	Sender notify.Sender
	// Now é o relógio; nil usa time.Now.
	Now func() time.Time
	// StatePath é o arquivo que guarda o último lembrete enviado; "" mantém
	// só em memória.
	StatePath string
	// AuditCount (opcional) devolve quantos problemas pendentes a auditoria
	// do fechamento encontrou no mês. Com problemas, o lembrete de fim de mês
	// cita a contagem e leva para /fechamento. Erro é tratado como zero.
	AuditCount func(year, month int) (int, error)
}

// State registra o último dia (YYYY-MM-DD) em que cada lembrete foi avaliado,
// para não repetir no mesmo dia nem depois de reiniciar o app.
type State struct {
	LastDaily    string `json:"lastDaily"`
	LastMonthEnd string `json:"lastMonthEnd"`
}

// Scheduler é o agendador dos lembretes.
type Scheduler struct {
	opts Options

	mu         sync.Mutex
	state      State
	retryAfter time.Time
}

// New cria o agendador e carrega o estado salvo.
func New(opts Options) *Scheduler {
	if opts.Now == nil {
		opts.Now = time.Now
	}
	s := &Scheduler{opts: opts}
	s.state = loadState(opts.StatePath)
	return s
}

// State devolve o estado atual (para testes e diagnóstico).
func (s *Scheduler) State() State {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state
}

// Run verifica os lembretes a cada interval até ctx ser cancelado.
func (s *Scheduler) Run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.safeCheck()
		}
	}
}

func (s *Scheduler) safeCheck() {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("Panic ao verificar lembretes", "panic", r)
		}
	}()
	s.Check()
}

// Check avalia os lembretes no instante atual e devolve as categorias das
// notificações enviadas.
func (s *Scheduler) Check() []string {
	settings := s.opts.Settings()
	if !settings.Enabled {
		return nil
	}

	now := s.opts.Now()
	hour, minute, err := config.ParseDailyTime(settings.DailyTime)
	if err != nil {
		return nil
	}
	due := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())
	if now.Before(due) {
		return nil
	}

	today := ymd(now)

	s.mu.Lock()
	defer s.mu.Unlock()

	needDaily := s.state.LastDaily != today
	needMonthEnd := settings.MonthEndEnabled && s.state.LastMonthEnd != today
	if (!needDaily && !needMonthEnd) || now.Before(s.retryAfter) {
		return nil
	}

	source, jornada, err := s.opts.Source()
	if err != nil {
		// Sem conexão não há o que comparar; tenta de novo no próximo ciclo.
		return nil
	}
	if jornada <= 0 {
		jornada = 8 * 60
	}

	if settings.WorkDaysOnly && !source.IsWorkDay(now) {
		s.markLocked(today, true, true)
		return nil
	}

	inMonthEnd := needMonthEnd && isInLastWorkDays(now, settings.MonthEndDays, source.IsWorkDay)
	if !inMonthEnd && !needDaily {
		s.markLocked(today, false, true)
		return nil
	}

	start := today
	if inMonthEnd {
		start = ymd(time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()))
	}
	minutes, err := source.DailyLoggedMinutes(start, today)
	if err != nil {
		slog.Warn("Lembretes: não foi possível consultar as horas lançadas", "err", err)
		s.retryAfter = now.Add(retryDelay)
		return nil
	}

	var sent []string
	monthEndCoversToday := false
	if inMonthEnd {
		pending := pendingDays(now, minutes, jornada, source.IsWorkDay)
		problems := s.auditCount(now)
		if len(pending) > 0 || problems > 0 {
			monthEndCoversToday = len(pending) > 0 && pending[len(pending)-1] == today
			n := monthEndNotification(today, pending, problems)
			s.send(n)
			sent = append(sent, n.CategoryID)
		}
	}

	// O lembrete de fim de mês já lista o dia de hoje quando ele está
	// incompleto; o diário seria repetido.
	if needDaily && !monthEndCoversToday {
		if logged := minutes[today]; logged < jornada {
			s.send(dailyNotification(today, jornada-logged))
			sent = append(sent, CategoryDaily)
		}
	}

	s.markLocked(today, needDaily, needMonthEnd)
	return sent
}

// SendTest envia uma notificação de teste (só por ação do usuário).
func (s *Scheduler) SendTest() error {
	today := ymd(s.opts.Now())
	return s.opts.Sender.Send(notify.Notification{
		ID:         "lembrete-teste-" + today,
		Title:      "Teste de lembrete",
		Body:       "Os lembretes estão funcionando. Clique em \"Lançar agora\" para abrir o lançamento de horas.",
		CategoryID: CategoryDaily,
		Data:       map[string]any{"kind": Kind, "route": RouteTimeLog, "date": today},
	})
}

func (s *Scheduler) send(n notify.Notification) {
	if err := s.opts.Sender.Send(n); err != nil {
		// Marcado como enviado mesmo assim: sem notificações disponíveis, a
		// tentativa se repetiria a cada minuto.
		slog.Warn("Não foi possível enviar o lembrete", "categoria", n.CategoryID, "err", err)
	}
}

// markLocked grava o dia avaliado. Exige s.mu travado.
func (s *Scheduler) markLocked(today string, daily, monthEnd bool) {
	if daily {
		s.state.LastDaily = today
	}
	if monthEnd {
		s.state.LastMonthEnd = today
	}
	saveState(s.opts.StatePath, s.state)
}

func dailyNotification(today string, missing int) notify.Notification {
	return notify.Notification{
		ID:         "lembrete-diario-" + today,
		Title:      "Lançamento de horas",
		Body:       fmt.Sprintf("Faltam %s para fechar o dia", FormatMinutes(missing)),
		CategoryID: CategoryDaily,
		Data:       map[string]any{"kind": Kind, "route": RouteTimeLog, "date": today},
	}
}

// auditCount consulta a auditoria do fechamento, se configurada.
func (s *Scheduler) auditCount(now time.Time) int {
	if s.opts.AuditCount == nil {
		return 0
	}
	n, err := s.opts.AuditCount(now.Year(), int(now.Month()))
	if err != nil {
		slog.Warn("Lembretes: não foi possível auditar o mês", "err", err)
		return 0
	}
	return n
}

// monthEndNotification monta o lembrete de fim de mês. Sem problemas de
// auditoria leva a "Completar período"; com problemas cita a contagem e leva
// ao fechamento do mês (que também aponta os dias incompletos).
func monthEndNotification(today string, pending []string, problems int) notify.Notification {
	var parts []string
	if len(pending) > 0 {
		dias := make([]string, len(pending))
		for i, d := range pending {
			dias[i] = d[len(d)-2:]
		}
		label := "dias pendentes"
		if len(pending) == 1 {
			label = "dia pendente"
		}
		parts = append(parts, fmt.Sprintf("%d %s: %s", len(pending), label, strings.Join(dias, ", ")))
	}

	n := notify.Notification{
		ID:         "lembrete-fim-de-mes-" + today,
		Title:      "Fim de mês: complete suas horas",
		CategoryID: CategoryMonthEnd,
		Data:       map[string]any{"kind": Kind, "route": RouteComplete, "date": today},
	}
	if problems > 0 {
		label := "problemas"
		if problems == 1 {
			label = "problema"
		}
		parts = append(parts, fmt.Sprintf("%d %s no fechamento do mês", problems, label))
		n.Title = "Fim de mês: revise o fechamento"
		n.CategoryID = CategoryMonthClose
		n.Data["route"] = RouteMonthClose
	}
	n.Body = strings.Join(parts, " · ")
	return n
}

// pendingDays lista os dias úteis do mês até hoje (inclusive) com menos
// minutos que a jornada, em ordem.
func pendingDays(now time.Time, minutes map[string]int, jornada int, isWorkDay func(time.Time) bool) []string {
	var pending []string
	for d := time.Date(now.Year(), now.Month(), 1, 12, 0, 0, 0, now.Location()); d.Day() <= now.Day() && d.Month() == now.Month(); d = d.AddDate(0, 0, 1) {
		if !isWorkDay(d) {
			continue
		}
		if key := ymd(d); minutes[key] < jornada {
			pending = append(pending, key)
		}
	}
	return pending
}

// isInLastWorkDays informa se now cai a partir do n-ésimo último dia útil do
// mês.
func isInLastWorkDays(now time.Time, n int, isWorkDay func(time.Time) bool) bool {
	if n <= 0 {
		return false
	}
	last := time.Date(now.Year(), now.Month()+1, 0, 12, 0, 0, 0, now.Location())
	found := 0
	for d := last; d.Month() == now.Month(); d = d.AddDate(0, 0, -1) {
		if !isWorkDay(d) {
			continue
		}
		found++
		if found == n {
			return now.Day() >= d.Day()
		}
	}
	// Mês com menos de n dias úteis: vale o mês todo.
	return true
}

// FormatMinutes escreve uma duração como "2h 30min", "45min" ou "3h".
func FormatMinutes(total int) string {
	h, m := total/60, total%60
	switch {
	case h == 0:
		return fmt.Sprintf("%dmin", m)
	case m == 0:
		return fmt.Sprintf("%dh", h)
	default:
		return fmt.Sprintf("%dh %dmin", h, m)
	}
}

func ymd(t time.Time) string { return t.Format("2006-01-02") }

func loadState(path string) State {
	var st State
	if path == "" {
		return st
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			slog.Warn("Não foi possível ler o estado dos lembretes", "err", err)
		}
		return st
	}
	if err := json.Unmarshal(data, &st); err != nil {
		slog.Warn("Estado dos lembretes corrompido; recomeçando", "err", err)
		return State{}
	}
	return st
}

func saveState(path string, st State) {
	if path == "" {
		return
	}
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return
	}
	if err := fsutil.WriteFileAtomic(path, data, fsutil.FilePerm); err != nil {
		slog.Warn("Não foi possível salvar o estado dos lembretes", "err", err)
	}
}
