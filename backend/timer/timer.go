// Package timer implementa o cronômetro por tarefa: um único cronômetro por
// vez, que pode ser pausado e retomado, sobrevive a reiniciar o app (estado em
// ~/.teamwork-logger/timer.json, gravado de forma atômica) e, ao parar, vira
// um ou mais lançamentos no Teamwork — um por dia, se atravessou a meia-noite.
//
// O relógio é injetável para os testes.
package timer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"logTime-go/backend/api"
	"logTime-go/backend/config"
	"logTime-go/backend/internal/fsutil"
	"logTime-go/backend/notify"
)

// Erros devolvidos pelas operações.
var (
	ErrAlreadyActive = errors.New("já existe um cronômetro ativo; pare-o antes de iniciar outro")
	ErrNotActive     = errors.New("nenhum cronômetro ativo")
	ErrNotRunning    = errors.New("o cronômetro não está rodando")
	ErrNotPaused     = errors.New("o cronômetro não está pausado")
	ErrNothingToLog  = errors.New("o cronômetro não tem tempo a lançar")
)

// Identificadores da notificação de cronômetro esquecido.
const (
	Kind            = "timer"
	CategoryLongRun = "cronometro-longo"
	ActionStopLog   = "parar-e-lancar"
	ActionContinue  = "continuar"
)

// Categories devolve as categorias a registrar no sistema de notificações.
func Categories() []notify.Category {
	return []notify.Category{{
		ID: CategoryLongRun,
		Actions: []notify.Action{
			{ID: ActionStopLog, Title: "Parar e lançar"},
			{ID: ActionContinue, Title: "Continuar"},
		},
	}}
}

// TaskRef identifica a tarefa cronometrada.
type TaskRef struct {
	TaskID      int    `json:"taskId"`
	TaskName    string `json:"taskName"`
	ProjectName string `json:"projectName"`
}

// Segment é um trecho em que o cronômetro rodou (horários RFC 3339).
type Segment struct {
	Start string `json:"start"`
	End   string `json:"end"`
}

// State é o estado exposto ao frontend. Com Running e Paused falsos não há
// cronômetro ativo. O tempo decorrido é AccumulatedSeconds (trechos fechados)
// mais o intervalo desde StartedAt, quando rodando.
type State struct {
	Running            bool      `json:"running"`
	Paused             bool      `json:"paused"`
	TaskID             int       `json:"taskId"`
	TaskName           string    `json:"taskName"`
	ProjectName        string    `json:"projectName"`
	Description        string    `json:"description"`
	Billable           bool      `json:"billable"`
	StartedAt          string    `json:"startedAt"`
	FirstStartedAt     string    `json:"firstStartedAt"`
	AccumulatedSeconds int64     `json:"accumulatedSeconds"`
	ElapsedSeconds     int64     `json:"elapsedSeconds"`
	Segments           []Segment `json:"segments"`
	// Date é o dia (YYYY-MM-DD) em que o cronômetro começou.
	Date string `json:"date"`
	// LoggedDates são os dias já lançados num Stop que falhou no meio; não são
	// lançados de novo.
	LoggedDates []string `json:"loggedDates"`
}

// Entry é um lançamento previsto: um por dia.
type Entry struct {
	Date    string `json:"date"`
	Time    string `json:"time"`
	Minutes int    `json:"minutes"`
	Seconds int64  `json:"seconds"`
}

// StopResult é o resultado de Stop.
type StopResult struct {
	Logged  bool                `json:"logged"`
	Results []api.TimeLogResult `json:"results"`
	State   State               `json:"state"`
}

// LogClient lança uma entrada de tempo (satisfeito por *api.TeamworkAPI).
type LogClient interface {
	LogTime(taskID int, entry api.TimeEntry) (*api.TimeLogResult, error)
}

// Options configura o serviço.
type Options struct {
	// Path é o arquivo de estado; "" mantém só em memória.
	Path string
	// Now é o relógio; nil usa time.Now.
	Now func() time.Time
	// Settings lê a configuração atual; nil usa os padrões.
	Settings func() config.TimerSettings
	// OnChange é chamado (fora do lock) após cada mudança de estado.
	OnChange func(State)
}

// segment e persisted são a forma interna, com time.Time.
type segment struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

type persisted struct {
	Task              TaskRef   `json:"task"`
	Description       string    `json:"description"`
	Billable          bool      `json:"billable"`
	Segments          []segment `json:"segments"`
	Current           time.Time `json:"current"` // zero quando pausado
	LoggedDates       []string  `json:"loggedDates,omitempty"`
	LongRunNotifiedAt time.Time `json:"longRunNotifiedAt"`
}

// Service guarda o cronômetro.
type Service struct {
	opts Options
	mu   sync.Mutex
	st   *persisted
	// stopping marca um Stop com lançamentos em andamento: outro Stop (ou
	// um Discard/Resume) no meio duplicaria ou perderia lançamentos.
	stopping bool
}

// ErrStopping indica que um lançamento do cronômetro está em andamento.
var ErrStopping = errors.New("o lançamento do cronômetro está em andamento")

// New cria o serviço e restaura o cronômetro salvo, se houver.
func New(opts Options) *Service {
	if opts.Now == nil {
		opts.Now = time.Now
	}
	if opts.Settings == nil {
		opts.Settings = config.DefaultTimerSettings
	}
	s := &Service{opts: opts}
	s.st = load(opts.Path)
	return s
}

// State devolve o estado atual.
func (s *Service) State() State {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.stateLocked()
}

// Start inicia um cronômetro. Só um pode estar ativo.
func (s *Service) Start(task TaskRef, description string, billable bool) (State, error) {
	if task.TaskID <= 0 {
		return State{}, fmt.Errorf("tarefa inválida: %d", task.TaskID)
	}
	return s.mutate(func(now time.Time) error {
		if s.st != nil {
			return ErrAlreadyActive
		}
		s.st = &persisted{Task: task, Description: description, Billable: billable, Current: now}
		return nil
	})
}

// Pause fecha o trecho atual.
func (s *Service) Pause() (State, error) {
	return s.mutate(func(now time.Time) error {
		if s.st == nil {
			return ErrNotActive
		}
		if s.st.Current.IsZero() {
			return ErrNotRunning
		}
		s.closeSegmentLocked(now)
		return nil
	})
}

// Resume abre um novo trecho.
func (s *Service) Resume() (State, error) {
	return s.mutate(func(now time.Time) error {
		if s.st == nil {
			return ErrNotActive
		}
		if !s.st.Current.IsZero() {
			return ErrNotPaused
		}
		s.st.Current = now
		s.st.LongRunNotifiedAt = time.Time{}
		return nil
	})
}

// Discard apaga o cronômetro sem lançar.
func (s *Service) Discard() (State, error) {
	return s.mutate(func(time.Time) error {
		if s.st == nil {
			return ErrNotActive
		}
		s.st = nil
		return nil
	})
}

// Snooze adia o aviso de cronômetro esquecido por mais um período.
func (s *Service) Snooze() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.st != nil {
		s.st.LongRunNotifiedAt = s.opts.Now()
		s.saveLocked()
	}
}

// Preview calcula os lançamentos que Stop faria agora.
func (s *Service) Preview() ([]Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.st == nil {
		return nil, ErrNotActive
	}
	return s.entriesLocked(s.opts.Now()), nil
}

// Stop para o cronômetro. Com log=false apenas descarta. Com log=true lança
// um registro por dia via client; entries, se não vazio, substitui os minutos
// calculados (só para dias que o cronômetro cobriu). Se um lançamento falhar,
// o cronômetro fica pausado com os dias já lançados marcados, para que tentar
// de novo não os duplique.
func (s *Service) Stop(log bool, description string, entries []Entry, client LogClient) (StopResult, error) {
	if !log {
		st, err := s.Discard()
		return StopResult{State: st}, err
	}
	if client == nil {
		return StopResult{}, errors.New("cliente da API indisponível")
	}

	s.mu.Lock()
	if s.stopping {
		s.mu.Unlock()
		return StopResult{State: s.State()}, ErrStopping
	}
	if s.st == nil {
		s.mu.Unlock()
		return StopResult{}, ErrNotActive
	}
	now := s.opts.Now()
	if !s.st.Current.IsZero() {
		s.closeSegmentLocked(now)
	}
	s.st.Description = description
	s.saveLocked()

	planned := s.entriesLocked(now)
	if len(entries) > 0 {
		var err error
		if planned, err = applyOverrides(planned, entries); err != nil {
			s.mu.Unlock()
			s.notify()
			return StopResult{State: s.State()}, err
		}
	}
	if len(planned) == 0 {
		s.mu.Unlock()
		s.notify()
		return StopResult{State: s.State()}, ErrNothingToLog
	}
	task, billable := s.st.Task, s.st.Billable
	s.stopping = true
	s.mu.Unlock()

	// As chamadas à API ficam fora do lock: o widget continua respondendo.
	results := make([]api.TimeLogResult, 0, len(planned))
	var stopErr error
	for _, e := range planned {
		res, err := client.LogTime(task.TaskID, api.TimeEntry{
			Minutes:     e.Minutes,
			Time:        e.Time,
			Description: description,
			IsBillable:  billable,
			Date:        e.Date,
		})
		if res != nil {
			results = append(results, *res)
		}
		if err != nil || res == nil || !res.Success {
			if err == nil {
				err = errors.New("falha ao lançar")
			}
			stopErr = fmt.Errorf("erro ao lançar %s: %w", e.Date, err)
			break
		}
		s.mu.Lock()
		if s.st != nil {
			s.st.LoggedDates = append(s.st.LoggedDates, e.Date)
			s.saveLocked()
		}
		s.mu.Unlock()
	}

	s.mu.Lock()
	s.stopping = false
	if stopErr == nil {
		s.st = nil
		s.saveLocked()
	}
	st := s.stateLocked()
	s.mu.Unlock()
	s.notify()

	return StopResult{Logged: stopErr == nil, Results: results, State: st}, stopErr
}

// CheckLongRunning informa se o cronômetro está rodando há mais que o limite
// configurado desde o início do trecho atual (ou desde o último aviso) e, se
// sim, marca o aviso como dado.
func (s *Service) CheckLongRunning() bool {
	limit := s.opts.Settings().LongRunningHours
	if limit <= 0 {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.st == nil || s.st.Current.IsZero() {
		return false
	}
	ref := s.st.Current
	if s.st.LongRunNotifiedAt.After(ref) {
		ref = s.st.LongRunNotifiedAt
	}
	now := s.opts.Now()
	if now.Sub(ref) < time.Duration(limit)*time.Hour {
		return false
	}
	s.st.LongRunNotifiedAt = now
	s.saveLocked()
	return true
}

// Watch verifica o cronômetro esquecido a cada interval até ctx terminar.
func (s *Service) Watch(ctx context.Context, interval time.Duration, sender notify.Sender) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.checkAndNotify(sender)
		}
	}
}

func (s *Service) checkAndNotify(sender notify.Sender) {
	if !s.CheckLongRunning() {
		return
	}
	st := s.State()
	hours := s.opts.Settings().LongRunningHours
	body := fmt.Sprintf("O cronômetro de \"%s\" está rodando há mais de %dh.", st.TaskName, hours)
	err := sender.Send(notify.Notification{
		ID:         fmt.Sprintf("cronometro-longo-%d", s.opts.Now().Unix()),
		Title:      "Esqueceu o cronômetro ligado?",
		Body:       body,
		CategoryID: CategoryLongRun,
		Data:       map[string]any{"kind": Kind},
	})
	if err != nil {
		slog.Warn("Não foi possível avisar do cronômetro longo", "err", err)
	}
}

// mutate aplica fn sob lock, persiste e avisa OnChange.
func (s *Service) mutate(fn func(now time.Time) error) (State, error) {
	s.mu.Lock()
	if s.stopping {
		st := s.stateLocked()
		s.mu.Unlock()
		return st, ErrStopping
	}
	if err := fn(s.opts.Now()); err != nil {
		st := s.stateLocked()
		s.mu.Unlock()
		return st, err
	}
	s.saveLocked()
	st := s.stateLocked()
	s.mu.Unlock()
	s.notify()
	return st, nil
}

func (s *Service) notify() {
	if s.opts.OnChange != nil {
		s.opts.OnChange(s.State())
	}
}

func (s *Service) closeSegmentLocked(now time.Time) {
	if now.After(s.st.Current) {
		s.st.Segments = append(s.st.Segments, segment{Start: s.st.Current, End: now})
	}
	s.st.Current = time.Time{}
}

func (s *Service) stateLocked() State {
	if s.st == nil {
		return State{Segments: []Segment{}, LoggedDates: []string{}}
	}
	now := s.opts.Now()
	var acc time.Duration
	segs := make([]Segment, 0, len(s.st.Segments))
	for _, seg := range s.st.Segments {
		acc += seg.End.Sub(seg.Start)
		segs = append(segs, Segment{Start: seg.Start.Format(time.RFC3339), End: seg.End.Format(time.RFC3339)})
	}
	elapsed := acc
	st := State{
		Running:            !s.st.Current.IsZero(),
		Paused:             s.st.Current.IsZero(),
		TaskID:             s.st.Task.TaskID,
		TaskName:           s.st.Task.TaskName,
		ProjectName:        s.st.Task.ProjectName,
		Description:        s.st.Description,
		Billable:           s.st.Billable,
		AccumulatedSeconds: int64(acc / time.Second),
		Segments:           segs,
		LoggedDates:        append([]string{}, s.st.LoggedDates...),
	}
	first := s.st.Current
	if len(s.st.Segments) > 0 {
		first = s.st.Segments[0].Start
	}
	st.FirstStartedAt = first.Format(time.RFC3339)
	st.Date = first.Format("2006-01-02")
	if st.Running {
		st.StartedAt = s.st.Current.Format(time.RFC3339)
		if now.After(s.st.Current) {
			elapsed += now.Sub(s.st.Current)
		}
	}
	st.ElapsedSeconds = int64(elapsed / time.Second)
	return st
}

// entriesLocked calcula os lançamentos (um por dia) considerando o trecho em
// andamento até now e pulando os dias já lançados.
func (s *Service) entriesLocked(now time.Time) []Entry {
	segs := append([]segment(nil), s.st.Segments...)
	if !s.st.Current.IsZero() && now.After(s.st.Current) {
		segs = append(segs, segment{Start: s.st.Current, End: now})
	}
	entries := splitByDay(segs, s.opts.Settings().Rounding)
	if len(s.st.LoggedDates) == 0 {
		return entries
	}
	out := entries[:0]
	for _, e := range entries {
		if !containsString(s.st.LoggedDates, e.Date) {
			out = append(out, e)
		}
	}
	return out
}

// splitByDay divide os trechos na meia-noite (horário local) e devolve um
// lançamento por dia, com o horário de início real do primeiro trecho do dia
// e os minutos arredondados. Partes de dia com menos de 1 minuto são
// ignoradas, exceto se forem o único tempo registrado.
func splitByDay(segs []segment, rounding string) []Entry {
	type dia struct {
		start   time.Time
		seconds int64
	}
	dias := map[string]*dia{}
	var total int64
	for _, seg := range segs {
		start := seg.Start
		for start.Before(seg.End) {
			y, m, d := start.Date()
			midnight := time.Date(y, m, d+1, 0, 0, 0, 0, start.Location())
			end := seg.End
			if midnight.Before(end) {
				end = midnight
			}
			key := start.Format("2006-01-02")
			if dias[key] == nil {
				dias[key] = &dia{start: start}
			} else if start.Before(dias[key].start) {
				dias[key].start = start
			}
			secs := int64(end.Sub(start) / time.Second)
			dias[key].seconds += secs
			total += secs
			start = end
		}
	}

	keys := make([]string, 0, len(dias))
	for k := range dias {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	entries := make([]Entry, 0, len(keys))
	for _, k := range keys {
		d := dias[k]
		if d.seconds < 60 {
			continue
		}
		entries = append(entries, Entry{
			Date:    k,
			Time:    d.start.Format("15:04:05"),
			Minutes: RoundMinutes(d.seconds, rounding),
			Seconds: d.seconds,
		})
	}
	if len(entries) == 0 && total > 0 && len(keys) > 0 {
		d := dias[keys[0]]
		entries = append(entries, Entry{
			Date:    keys[0],
			Time:    d.start.Format("15:04:05"),
			Minutes: RoundMinutes(total, rounding),
			Seconds: total,
		})
	}
	return entries
}

// RoundMinutes converte segundos em minutos a lançar: no modo exato, para o
// minuto mais próximo; no modo de 15, para cima em múltiplos de 15. O mínimo é
// 1 minuto (15 no modo de 15) quando há algum tempo.
func RoundMinutes(seconds int64, rounding string) int {
	if seconds <= 0 {
		return 0
	}
	if rounding == config.TimerRounding15 {
		return int(math.Ceil(float64(seconds)/(15*60))) * 15
	}
	m := int(math.Round(float64(seconds) / 60))
	if m < 1 {
		m = 1
	}
	return m
}

// applyOverrides troca os minutos calculados pelos revisados pelo usuário.
func applyOverrides(planned, overrides []Entry) ([]Entry, error) {
	byDate := map[string]int{}
	for _, o := range overrides {
		byDate[o.Date] = o.Minutes
	}
	var known []string
	out := make([]Entry, 0, len(planned))
	for _, p := range planned {
		known = append(known, p.Date)
		m, ok := byDate[p.Date]
		if !ok {
			continue
		}
		if m <= 0 || m > 24*60 {
			return nil, fmt.Errorf("minutos inválidos para %s: %d", p.Date, m)
		}
		p.Minutes = m
		out = append(out, p)
	}
	for date := range byDate {
		if !containsString(known, date) {
			return nil, fmt.Errorf("o cronômetro não cobre o dia %s (dias: %s)", date, strings.Join(known, ", "))
		}
	}
	return out, nil
}

func containsString(list []string, v string) bool {
	for _, x := range list {
		if x == v {
			return true
		}
	}
	return false
}

func load(path string) *persisted {
	if path == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			slog.Warn("Não foi possível ler o cronômetro salvo", "err", err)
		}
		return nil
	}
	var p persisted
	if err := json.Unmarshal(data, &p); err != nil || p.Task.TaskID <= 0 {
		slog.Warn("Cronômetro salvo inválido; ignorado", "err", err)
		return nil
	}
	return &p
}

// saveLocked grava o estado (ou apaga o arquivo sem cronômetro ativo).
func (s *Service) saveLocked() {
	if s.opts.Path == "" {
		return
	}
	if s.st == nil {
		if err := os.Remove(s.opts.Path); err != nil && !errors.Is(err, os.ErrNotExist) {
			slog.Warn("Não foi possível apagar o cronômetro salvo", "err", err)
		}
		return
	}
	data, err := json.MarshalIndent(s.st, "", "  ")
	if err != nil {
		return
	}
	if err := fsutil.WriteFileAtomic(s.opts.Path, data, fsutil.FilePerm); err != nil {
		slog.Warn("Não foi possível salvar o cronômetro", "err", err)
	}
}
