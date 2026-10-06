package backend

import (
	"errors"

	"logTime-go/backend/config"
	"logTime-go/backend/timer"
)

// Bindings do cronômetro por tarefa (backend/timer). Toda mudança também
// chega ao frontend pelo evento "timer:changed".

func (a *App) GetTimerState() timer.State {
	return a.timerService().State()
}

func (a *App) StartTimer(task timer.TaskRef, description string, billable bool) (timer.State, error) {
	return a.timerService().Start(task, description, billable)
}

func (a *App) PauseTimer() (timer.State, error) {
	return a.timerService().Pause()
}

func (a *App) ResumeTimer() (timer.State, error) {
	return a.timerService().Resume()
}

func (a *App) DiscardTimer() (timer.State, error) {
	return a.timerService().Discard()
}

// PreviewTimerStop devolve os lançamentos (um por dia) que StopTimer faria
// agora, para o usuário revisar antes de confirmar.
func (a *App) PreviewTimerStop() ([]timer.Entry, error) {
	return a.timerService().Preview()
}

// StopTimer para o cronômetro. Com log=false descarta; com log=true lança no
// Teamwork com a descrição e os minutos revisados (entries vazio usa os
// calculados).
func (a *App) StopTimer(log bool, description string, entries []timer.Entry) (timer.StopResult, error) {
	if !log {
		return a.timerService().Stop(false, "", nil, nil)
	}
	client, err := a.client()
	if err != nil {
		return timer.StopResult{State: a.timerService().State()}, err
	}
	return a.timerService().Stop(true, description, entries, client)
}

func (a *App) GetTimerSettings() config.TimerSettings {
	return a.timerSettings()
}

func (a *App) SaveTimerSettings(settings config.TimerSettings) error {
	if a.configManager == nil {
		return errors.New("configuração indisponível")
	}
	return a.configManager.SetTimerSettings(settings)
}
