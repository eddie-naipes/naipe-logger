package backend

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"logTime-go/backend/agenda"
	"logTime-go/backend/config"
	"logTime-go/backend/internal/fsutil"
	"logTime-go/backend/logging"
	"logTime-go/backend/security"
)

// Bindings de "Importar reuniões da agenda". Os links iCal privados são
// segredos: vão direto para o cofre do sistema, são registrados no
// mascaramento do log e nunca voltam ao frontend (só MaskedURL).

// Indireções para os testes não tocarem no cofre real nem abrirem janela.
var (
	storeAgendaURL  = security.StoreAgendaURL
	loadAgendaURL   = security.LoadAgendaURL
	deleteAgendaURL = security.DeleteAgendaURL

	openAgendaFileDialog = func(ctx context.Context, opts runtime.OpenDialogOptions) (string, error) {
		return runtime.OpenFileDialog(ctx, opts)
	}
)

// maxAgendaPeriodDays limita o período de PlanAgenda (cada agenda é baixada e
// expandida inteira a cada pedido).
const maxAgendaPeriodDays = 62

// agendaState guarda o arquivo .ics importado na sessão e o registro de
// ocorrências já lançadas. O valor zero funciona (App{} nos testes).
type agendaState struct {
	mu      sync.Mutex
	file    *agendaFile
	store   *agenda.ImportedStore
	fetcher agenda.Fetcher
	// storeDir troca ~/.teamwork-logger nos testes.
	storeDir string
}

type agendaFile struct {
	name   string
	events []agenda.Event
}

// AgendaFileInfo descreve o arquivo .ics carregado na sessão.
type AgendaFileInfo struct {
	Name   string `json:"name"`
	Events int    `json:"events"`
}

// AgendaPlan é o resultado de PlanAgenda.
type AgendaPlan struct {
	Items []agenda.PlanItem `json:"items"`
	// Warnings lista agendas que não puderam ser lidas (as demais seguem).
	Warnings []string `json:"warnings"`
	// Sources são os nomes das agendas lidas com sucesso.
	Sources []string `json:"sources"`
}

// AddAgendaCalendar valida o link (baixando e lendo a agenda) e o guarda no
// cofre; config.json recebe só o nome e o link mascarado.
func (a *App) AddAgendaCalendar(name, link string) (config.AgendaCalendar, error) {
	normalized, err := agenda.NormalizeURL(link)
	if err != nil {
		return config.AgendaCalendar{}, err
	}
	logging.AddSecret(normalized)
	logging.AddSecret(strings.TrimSpace(link))

	name = strings.TrimSpace(name)
	masked := agenda.MaskURL(normalized)
	if name == "" {
		name = masked
	}

	if _, err := a.fetchAgenda(normalized, name); err != nil {
		return config.AgendaCalendar{}, err
	}

	id, err := newAgendaID()
	if err != nil {
		return config.AgendaCalendar{}, err
	}
	if err := storeAgendaURL(id, normalized); err != nil {
		return config.AgendaCalendar{}, err
	}
	cal := config.AgendaCalendar{
		ID:        id,
		Name:      name,
		MaskedURL: masked,
		AddedAt:   time.Now().Format(time.RFC3339),
	}
	if err := a.configManager.AddAgendaCalendar(cal); err != nil {
		// Sem a entrada na config o link ficaria órfão no cofre.
		_ = deleteAgendaURL(id)
		return config.AgendaCalendar{}, err
	}
	slog.Info("Agenda adicionada", "id", id, "host", masked)
	return cal, nil
}

// ListAgendaCalendars devolve as agendas cadastradas (sem os links).
func (a *App) ListAgendaCalendars() []config.AgendaCalendar {
	return a.configManager.GetAgendaSettings().Calendars
}

// RemoveAgendaCalendar apaga o link do cofre e tira a agenda da lista.
func (a *App) RemoveAgendaCalendar(id string) error {
	if err := deleteAgendaURL(id); err != nil {
		return err
	}
	return a.configManager.RemoveAgendaCalendar(id)
}

// ImportAgendaFile abre o seletor de arquivo e carrega um .ics local para a
// sessão (não é salvo). Devolve nil se o usuário cancelar.
func (a *App) ImportAgendaFile() (*AgendaFileInfo, error) {
	path, err := openAgendaFileDialog(a.appContext(), runtime.OpenDialogOptions{
		Title: "Escolha um arquivo de agenda (.ics)",
		Filters: []runtime.FileFilter{
			{DisplayName: "Agenda iCalendar (*.ics)", Pattern: "*.ics"},
		},
	})
	if err != nil || path == "" {
		return nil, err
	}
	return a.loadAgendaFile(path)
}

func (a *App) loadAgendaFile(path string) (*AgendaFileInfo, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("erro ao abrir o arquivo: %v", err)
	}
	defer func() { _ = f.Close() }()
	data, err := io.ReadAll(io.LimitReader(f, agenda.MaxICSBytes+1))
	if err != nil {
		return nil, fmt.Errorf("erro ao ler o arquivo: %v", err)
	}
	if len(data) > agenda.MaxICSBytes {
		return nil, fmt.Errorf("o arquivo passa do limite de %d MB", agenda.MaxICSBytes>>20)
	}
	events, err := agenda.Parse(bytes.NewReader(data), time.Local)
	if err != nil {
		return nil, err
	}
	name := filepath.Base(path)

	a.agenda.mu.Lock()
	a.agenda.file = &agendaFile{name: name, events: events}
	a.agenda.mu.Unlock()
	return &AgendaFileInfo{Name: name, Events: len(events)}, nil
}

// GetAgendaFile informa o arquivo .ics carregado na sessão (nil se nenhum).
func (a *App) GetAgendaFile() *AgendaFileInfo {
	a.agenda.mu.Lock()
	defer a.agenda.mu.Unlock()
	if a.agenda.file == nil {
		return nil
	}
	return &AgendaFileInfo{Name: a.agenda.file.name, Events: len(a.agenda.file.events)}
}

// ClearAgendaFile descarta o arquivo .ics da sessão.
func (a *App) ClearAgendaFile() {
	a.agenda.mu.Lock()
	a.agenda.file = nil
	a.agenda.mu.Unlock()
}

// GetAgendaSettings devolve regras e preferências da importação.
func (a *App) GetAgendaSettings() config.AgendaSettings {
	return a.configManager.GetAgendaSettings()
}

// SaveAgendaSettings grava regras e preferências (a lista de agendas é mantida).
func (a *App) SaveAgendaSettings(s config.AgendaSettings) error {
	return a.configManager.SetAgendaSettings(s)
}

// PlanAgenda lê todas as agendas (e o arquivo da sessão), expande os eventos
// do período (AAAA-MM-DD, inclusive) e aplica as regras. Uma agenda que falha
// vira aviso; só é erro quando nenhuma fonte pôde ser lida.
func (a *App) PlanAgenda(start, end string) (*AgendaPlan, error) {
	from, err := time.ParseInLocation("2006-01-02", start, time.Local)
	if err != nil {
		return nil, fmt.Errorf("data inicial inválida: %q", start)
	}
	until, err := time.ParseInLocation("2006-01-02", end, time.Local)
	if err != nil {
		return nil, fmt.Errorf("data final inválida: %q", end)
	}
	if until.Before(from) {
		return nil, errors.New("a data final é anterior à inicial")
	}
	if until.Sub(from) > maxAgendaPeriodDays*24*time.Hour {
		return nil, fmt.Errorf("o período pode ter no máximo %d dias", maxAgendaPeriodDays)
	}
	to := until.AddDate(0, 0, 1)

	settings := a.configManager.GetAgendaSettings()
	plan := &AgendaPlan{Items: []agenda.PlanItem{}, Warnings: []string{}, Sources: []string{}}
	occs := []agenda.Occurrence{}
	fontes := 0

	for _, cal := range settings.Calendars {
		fontes++
		link, err := loadAgendaURL(cal.ID)
		if err != nil {
			plan.Warnings = append(plan.Warnings, fmt.Sprintf("%s: %v", cal.Name, err))
			continue
		}
		logging.AddSecret(link)
		events, err := a.fetchAgenda(link, cal.Name)
		if err != nil {
			slog.Warn("Agenda não pôde ser lida", "agenda", cal.Name, "err", err)
			plan.Warnings = append(plan.Warnings, fmt.Sprintf("%s: %v", cal.Name, err))
			continue
		}
		occs = append(occs, agenda.Expand(events, cal.Name, from, to, time.Local)...)
		plan.Sources = append(plan.Sources, cal.Name)
	}

	a.agenda.mu.Lock()
	file := a.agenda.file
	a.agenda.mu.Unlock()
	if file != nil {
		fontes++
		occs = append(occs, agenda.Expand(file.events, file.name, from, to, time.Local)...)
		plan.Sources = append(plan.Sources, file.name)
	}

	if fontes == 0 {
		return nil, errors.New("nenhuma agenda configurada: adicione um link iCal ou importe um arquivo .ics")
	}
	if len(plan.Sources) == 0 {
		return nil, fmt.Errorf("nenhuma agenda pôde ser lida: %s", strings.Join(plan.Warnings, "; "))
	}

	store, err := a.importedStore()
	if err != nil {
		return nil, err
	}

	items, err := agenda.BuildPlan(agenda.Dedupe(occs), planOptions(settings, store.Has))
	if err != nil {
		return nil, err
	}
	plan.Items = items
	return plan, nil
}

// MarkAgendaImported registra ocorrências lançadas, para não importá-las de novo.
func (a *App) MarkAgendaImported(records []agenda.ImportedRecord) error {
	store, err := a.importedStore()
	if err != nil {
		return err
	}
	return store.Mark(records)
}

// UnmarkAgendaImported desfaz a marcação (ex.: o lote foi desfeito).
func (a *App) UnmarkAgendaImported(keys []string) error {
	store, err := a.importedStore()
	if err != nil {
		return err
	}
	return store.Unmark(keys)
}

func (a *App) fetchAgenda(link, name string) ([]agenda.Event, error) {
	data, err := a.agenda.fetcher.Fetch(a.appContext(), link)
	if err != nil {
		return nil, err
	}
	events, err := agenda.Parse(bytes.NewReader(data), time.Local)
	if err != nil {
		return nil, fmt.Errorf("a agenda %q não é um iCalendar válido", name)
	}
	return events, nil
}

func (a *App) importedStore() (*agenda.ImportedStore, error) {
	a.agenda.mu.Lock()
	defer a.agenda.mu.Unlock()
	if a.agenda.store != nil {
		return a.agenda.store, nil
	}
	dir := a.agenda.storeDir
	if dir == "" {
		appDir, err := fsutil.AppDir()
		if err != nil {
			return nil, err
		}
		dir = appDir
	}
	store, err := agenda.OpenImportedStore(filepath.Join(dir, "agenda-importados.json"))
	if err != nil {
		return nil, err
	}
	a.agenda.store = store
	return store, nil
}

func planOptions(s config.AgendaSettings, imported func(string) bool) agenda.PlanOptions {
	toRef := func(t config.AgendaTask) agenda.TaskRef {
		return agenda.TaskRef{TaskID: t.TaskID, TaskName: t.TaskName, ProjectID: t.ProjectID, ProjectName: t.ProjectName}
	}
	rules := make([]agenda.Rule, len(s.Rules))
	for i, r := range s.Rules {
		rules[i] = agenda.Rule{Match: r.Match, Regex: r.IsRegex, Task: toRef(r.Task), Description: r.Description}
	}
	round := 0
	if s.Rounding == config.AgendaRounding15 {
		round = 15
	}
	return agenda.PlanOptions{
		Rules:              rules,
		DefaultTask:        toRef(s.DefaultTask),
		IgnoreWords:        s.IgnoreWords,
		MinMinutes:         s.MinMinutes,
		RoundTo:            round,
		UserEmail:          s.UserEmail,
		IncludeTransparent: s.IncludeTransparent,
		Billable:           s.Billable,
		Imported:           imported,
	}
}

func newAgendaID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
