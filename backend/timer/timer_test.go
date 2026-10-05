package timer

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"logTime-go/backend/api"
	"logTime-go/backend/config"
	"logTime-go/backend/notify"
)

type relogio struct {
	mu    sync.Mutex
	agora time.Time
}

func (r *relogio) Now() time.Time {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.agora
}

func (r *relogio) avancar(d time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.agora = r.agora.Add(d)
}

func hora(s string) time.Time {
	t, err := time.ParseInLocation("2006-01-02 15:04:05", s, time.Local)
	if err != nil {
		panic(err)
	}
	return t
}

var tarefa = TaskRef{TaskID: 77, TaskName: "Revisar PR", ProjectName: "Logger"}

func novoServico(t *testing.T, inicio string, cfg config.TimerSettings) (*Service, *relogio, string, *[]State) {
	t.Helper()
	rel := &relogio{agora: hora(inicio)}
	path := filepath.Join(t.TempDir(), "timer.json")
	var mudancas []State
	s := New(Options{
		Path:     path,
		Now:      rel.Now,
		Settings: func() config.TimerSettings { return cfg },
		OnChange: func(st State) { mudancas = append(mudancas, st) },
	})
	return s, rel, path, &mudancas
}

// clienteFalso registra os lançamentos sem rede.
type clienteFalso struct {
	lancados []api.TimeEntry
	falharEm int // 1-based; 0 = nunca
}

func (c *clienteFalso) LogTime(taskID int, e api.TimeEntry) (*api.TimeLogResult, error) {
	c.lancados = append(c.lancados, e)
	if c.falharEm == len(c.lancados) {
		return &api.TimeLogResult{Success: false, Date: e.Date, TaskID: taskID}, errors.New("erro 500")
	}
	return &api.TimeLogResult{Success: true, Date: e.Date, TaskID: taskID, EntryID: 1000 + len(c.lancados)}, nil
}

func TestCronometroIniciaPausaRetomaEAcumula(t *testing.T) {
	s, rel, _, mudancas := novoServico(t, "2026-09-09 09:00:00", config.DefaultTimerSettings())

	st, err := s.Start(tarefa, "codando", true)
	if err != nil {
		t.Fatal(err)
	}
	if !st.Running || st.Paused || st.TaskID != 77 || st.StartedAt == "" || st.Date != "2026-09-09" {
		t.Fatalf("estado após iniciar = %+v", st)
	}
	if _, err := s.Start(tarefa, "", false); !errors.Is(err, ErrAlreadyActive) {
		t.Errorf("segundo Start = %v, esperava ErrAlreadyActive", err)
	}

	rel.avancar(30 * time.Minute)
	st, err = s.Pause()
	if err != nil {
		t.Fatal(err)
	}
	if st.Running || !st.Paused || st.AccumulatedSeconds != 1800 || st.StartedAt != "" {
		t.Fatalf("estado pausado = %+v", st)
	}
	if _, err := s.Pause(); !errors.Is(err, ErrNotRunning) {
		t.Errorf("pausar de novo = %v", err)
	}

	rel.avancar(time.Hour) // pausado não conta
	if _, err := s.Resume(); err != nil {
		t.Fatal(err)
	}
	rel.avancar(10 * time.Minute)
	st = s.State()
	if st.ElapsedSeconds != 2400 || st.AccumulatedSeconds != 1800 {
		t.Errorf("decorrido = %d / acumulado = %d", st.ElapsedSeconds, st.AccumulatedSeconds)
	}
	if len(*mudancas) != 3 {
		t.Errorf("OnChange chamado %d vezes, esperava 3", len(*mudancas))
	}
}

func TestCronometroSobreviveAReiniciar(t *testing.T) {
	s, rel, path, _ := novoServico(t, "2026-09-09 09:00:00", config.DefaultTimerSettings())
	if _, err := s.Start(tarefa, "persistir", false); err != nil {
		t.Fatal(err)
	}
	rel.avancar(20 * time.Minute)
	if _, err := s.Pause(); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Resume(); err != nil {
		t.Fatal(err)
	}

	rel.avancar(5 * time.Minute)
	reaberto := New(Options{Path: path, Now: rel.Now})
	st := reaberto.State()
	if !st.Running || st.Description != "persistir" || st.ElapsedSeconds != 25*60 || len(st.Segments) != 1 {
		t.Errorf("estado restaurado = %+v", st)
	}

	if _, err := reaberto.Discard(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("descartar deveria apagar %s (err=%v)", path, err)
	}
	if st := New(Options{Path: path, Now: rel.Now}).State(); st.Running || st.Paused {
		t.Errorf("depois de descartar o estado deveria estar vazio: %+v", st)
	}
}

func TestCronometroArquivoCorrompidoEIgnorado(t *testing.T) {
	path := filepath.Join(t.TempDir(), "timer.json")
	if err := os.WriteFile(path, []byte("{lixo"), 0600); err != nil {
		t.Fatal(err)
	}
	if st := New(Options{Path: path}).State(); st.Running || st.Paused {
		t.Errorf("estado = %+v", st)
	}
}

func TestRoundMinutes(t *testing.T) {
	casos := []struct {
		seg  int64
		modo string
		want int
	}{
		{10, config.TimerRoundingExact, 1},
		{89, config.TimerRoundingExact, 1},
		{90, config.TimerRoundingExact, 2},
		{3600, config.TimerRoundingExact, 60},
		{60, config.TimerRounding15, 15},
		{15 * 60, config.TimerRounding15, 15},
		{16 * 60, config.TimerRounding15, 30},
		{0, config.TimerRoundingExact, 0},
	}
	for _, c := range casos {
		if got := RoundMinutes(c.seg, c.modo); got != c.want {
			t.Errorf("RoundMinutes(%d, %s) = %d, esperava %d", c.seg, c.modo, got, c.want)
		}
	}
}

func TestSplitByDayDivideNaMeiaNoite(t *testing.T) {
	segs := []segment{
		{Start: hora("2026-09-09 22:30:00"), End: hora("2026-09-10 01:15:00")},
		{Start: hora("2026-09-10 08:00:00"), End: hora("2026-09-10 08:45:00")},
	}
	got := splitByDay(segs, config.TimerRoundingExact)
	if len(got) != 2 {
		t.Fatalf("entradas = %+v", got)
	}
	if got[0].Date != "2026-09-09" || got[0].Time != "22:30:00" || got[0].Minutes != 90 {
		t.Errorf("primeiro dia = %+v", got[0])
	}
	if got[1].Date != "2026-09-10" || got[1].Time != "00:00:00" || got[1].Minutes != 120 {
		t.Errorf("segundo dia = %+v", got[1])
	}
}

func TestSplitByDayIgnoraSobraDeSegundosAposMeiaNoite(t *testing.T) {
	segs := []segment{{Start: hora("2026-09-09 23:00:00"), End: hora("2026-09-10 00:00:20")}}
	got := splitByDay(segs, config.TimerRounding15)
	if len(got) != 1 || got[0].Date != "2026-09-09" || got[0].Minutes != 60 {
		t.Errorf("entradas = %+v", got)
	}

	curto := splitByDay([]segment{{Start: hora("2026-09-09 10:00:00"), End: hora("2026-09-09 10:00:20")}}, config.TimerRoundingExact)
	if len(curto) != 1 || curto[0].Minutes != 1 {
		t.Errorf("tempo curto deveria virar 1 minuto: %+v", curto)
	}
}

func TestStopLancaComDataEHorarioReais(t *testing.T) {
	s, rel, path, _ := novoServico(t, "2026-09-09 14:10:00", config.DefaultTimerSettings())
	if _, err := s.Start(tarefa, "inicial", true); err != nil {
		t.Fatal(err)
	}
	rel.avancar(47*time.Minute + 40*time.Second)

	prev, err := s.Preview()
	if err != nil || len(prev) != 1 || prev[0].Minutes != 48 {
		t.Fatalf("prévia = %+v, %v", prev, err)
	}

	cli := &clienteFalso{}
	res, err := s.Stop(true, "revisado", nil, cli)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Logged || len(res.Results) != 1 || res.State.Running || res.State.Paused {
		t.Errorf("resultado = %+v", res)
	}
	e := cli.lancados[0]
	if e.Date != "2026-09-09" || e.Time != "14:10:00" || e.Minutes != 48 || e.Description != "revisado" || !e.IsBillable {
		t.Errorf("lançado = %+v", e)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("após lançar o arquivo de estado deveria sumir")
	}
}

func TestStopComMinutosRevisados(t *testing.T) {
	s, rel, _, _ := novoServico(t, "2026-09-09 14:00:00", config.DefaultTimerSettings())
	if _, err := s.Start(tarefa, "", false); err != nil {
		t.Fatal(err)
	}
	rel.avancar(time.Hour)

	if _, err := s.Stop(true, "x", []Entry{{Date: "2026-09-01", Minutes: 30}}, &clienteFalso{}); err == nil {
		t.Error("esperava erro para dia fora do cronômetro")
	}
	if st := s.State(); !st.Paused {
		t.Errorf("após erro de validação o cronômetro deveria ficar pausado: %+v", st)
	}

	cli := &clienteFalso{}
	if _, err := s.Stop(true, "x", []Entry{{Date: "2026-09-09", Minutes: 45}}, cli); err != nil {
		t.Fatal(err)
	}
	if cli.lancados[0].Minutes != 45 {
		t.Errorf("minutos lançados = %d, esperava 45", cli.lancados[0].Minutes)
	}
}

func TestStopFalhaNoMeioNaoDuplicaDiasJaLancados(t *testing.T) {
	s, rel, _, _ := novoServico(t, "2026-09-09 23:00:00", config.DefaultTimerSettings())
	if _, err := s.Start(tarefa, "", false); err != nil {
		t.Fatal(err)
	}
	rel.avancar(2 * time.Hour)

	cli := &clienteFalso{falharEm: 2}
	res, err := s.Stop(true, "noite", nil, cli)
	if err == nil || res.Logged {
		t.Fatalf("esperava falha parcial: %+v, %v", res, err)
	}
	if !res.State.Paused || len(res.State.LoggedDates) != 1 || res.State.LoggedDates[0] != "2026-09-09" {
		t.Fatalf("estado após falha = %+v", res.State)
	}

	cli2 := &clienteFalso{}
	if _, err := s.Stop(true, "noite", nil, cli2); err != nil {
		t.Fatal(err)
	}
	if len(cli2.lancados) != 1 || cli2.lancados[0].Date != "2026-09-10" {
		t.Errorf("segunda tentativa lançou %+v, esperava só 2026-09-10", cli2.lancados)
	}
}

func TestStopSemLancarDescarta(t *testing.T) {
	s, _, _, _ := novoServico(t, "2026-09-09 10:00:00", config.DefaultTimerSettings())
	if _, err := s.Start(tarefa, "", false); err != nil {
		t.Fatal(err)
	}
	res, err := s.Stop(false, "", nil, nil)
	if err != nil || res.Logged || res.State.Running {
		t.Errorf("Stop(false) = %+v, %v", res, err)
	}
	if _, err := s.Stop(false, "", nil, nil); !errors.Is(err, ErrNotActive) {
		t.Errorf("sem cronômetro = %v", err)
	}
}

func TestStopLancaViaServidorHTTP(t *testing.T) {
	var mu sync.Mutex
	var recebidos []api.TimelogRequest
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/projects/api/v3/tasks/77/time.json" {
			t.Errorf("requisição inesperada %s %s", r.Method, r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var req api.TimelogRequest
		if err := json.Unmarshal(body, &req); err != nil {
			t.Errorf("corpo inválido: %s", body)
		}
		mu.Lock()
		recebidos = append(recebidos, req)
		n := len(recebidos)
		mu.Unlock()
		w.WriteHeader(http.StatusCreated)
		fmt.Fprintf(w, `{"timelog":{"id":%d}}`, 500+n)
	}))
	t.Cleanup(server.Close)

	cliente := api.NewTeamworkAPI(api.Config{AuthToken: "token-de-teste", UserID: 42, ApiHost: server.URL})
	cliente.SetHTTPClient(server.Client())

	s, rel, _, _ := novoServico(t, "2026-09-09 23:30:00", config.TimerSettings{Rounding: config.TimerRounding15, LongRunningHours: 4})
	if _, err := s.Start(tarefa, "", true); err != nil {
		t.Fatal(err)
	}
	rel.avancar(50 * time.Minute) // 30min no dia 9, 20min no dia 10

	res, err := s.Stop(true, "virada", nil, cliente)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Logged || len(res.Results) != 2 || res.Results[0].EntryID != 501 {
		t.Errorf("resultado = %+v", res)
	}
	if len(recebidos) != 2 {
		t.Fatalf("servidor recebeu %d lançamentos", len(recebidos))
	}
	a, b := recebidos[0].Timelog, recebidos[1].Timelog
	if a.Date != "2026-09-09" || a.Time != "23:30:00" || a.Minutes != 30 || a.UserID != 42 || !a.IsBillable {
		t.Errorf("primeiro lançamento = %+v", a)
	}
	if b.Date != "2026-09-10" || b.Time != "00:00:00" || b.Minutes != 30 || b.Description != "virada" {
		t.Errorf("segundo lançamento = %+v", b)
	}
}

func TestAvisoDeCronometroEsquecido(t *testing.T) {
	s, rel, _, _ := novoServico(t, "2026-09-09 08:00:00", config.TimerSettings{Rounding: config.TimerRoundingExact, LongRunningHours: 4})
	rec := &notify.Recorder{}
	s.checkAndNotify(rec)
	if len(rec.Sent()) != 0 {
		t.Fatal("avisou sem cronômetro")
	}
	if _, err := s.Start(tarefa, "", false); err != nil {
		t.Fatal(err)
	}

	rel.avancar(3*time.Hour + 59*time.Minute)
	s.checkAndNotify(rec)
	if len(rec.Sent()) != 0 {
		t.Fatal("avisou antes do limite")
	}

	rel.avancar(time.Minute)
	s.checkAndNotify(rec)
	if len(rec.Sent()) != 1 || rec.Sent()[0].CategoryID != CategoryLongRun || rec.Sent()[0].Data["kind"] != Kind {
		t.Fatalf("enviado = %+v", rec.Sent())
	}

	// Não repete logo em seguida; "Continuar" adia por mais 4h.
	rel.avancar(time.Hour)
	s.Snooze()
	rel.avancar(3 * time.Hour)
	s.checkAndNotify(rec)
	if len(rec.Sent()) != 1 {
		t.Fatalf("repetiu cedo demais: %d avisos", len(rec.Sent()))
	}
	rel.avancar(time.Hour)
	s.checkAndNotify(rec)
	if len(rec.Sent()) != 2 {
		t.Errorf("esperava o segundo aviso, houve %d", len(rec.Sent()))
	}

	// Pausado não avisa.
	if _, err := s.Pause(); err != nil {
		t.Fatal(err)
	}
	rel.avancar(10 * time.Hour)
	if s.CheckLongRunning() {
		t.Error("cronômetro pausado não deveria gerar aviso")
	}
}

func TestAvisoDesligadoComZeroHoras(t *testing.T) {
	s, rel, _, _ := novoServico(t, "2026-09-09 08:00:00", config.TimerSettings{Rounding: config.TimerRoundingExact})
	if _, err := s.Start(tarefa, "", false); err != nil {
		t.Fatal(err)
	}
	rel.avancar(24 * time.Hour)
	if s.CheckLongRunning() {
		t.Error("com LongRunningHours=0 não deveria avisar")
	}
}
