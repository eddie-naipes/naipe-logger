package config

import "fmt"

// Modos de arredondamento do cronômetro ao lançar.
const (
	// TimerRoundingExact arredonda para o minuto mais próximo (mínimo 1 min).
	TimerRoundingExact = "exact"
	// TimerRounding15 arredonda para cima, em múltiplos de 15 minutos.
	TimerRounding15 = "15min"
)

// TimerSettings configura o cronômetro por tarefa (seção "timer" de
// config.json; padrões aplicados a arquivos antigos como nos lembretes).
type TimerSettings struct {
	// Rounding é TimerRoundingExact (padrão) ou TimerRounding15.
	Rounding string `json:"rounding"`
	// LongRunningHours é depois de quantas horas seguidas rodando o app
	// pergunta se o cronômetro foi esquecido ligado. Padrão 4; 0 desliga.
	LongRunningHours int `json:"longRunningHours"`
}

// DefaultTimerSettings devolve os padrões do cronômetro.
func DefaultTimerSettings() TimerSettings {
	return TimerSettings{Rounding: TimerRoundingExact, LongRunningHours: 4}
}

func (t TimerSettings) normalized() TimerSettings {
	if t.Rounding != TimerRounding15 {
		t.Rounding = TimerRoundingExact
	}
	if t.LongRunningHours < 0 {
		t.LongRunningHours = 0
	}
	return t
}

// GetTimerSettings devolve a configuração do cronômetro.
func (m *Manager) GetTimerSettings() TimerSettings {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.appConfig.Timer.normalized()
}

// SetTimerSettings valida e grava a configuração do cronômetro.
func (m *Manager) SetTimerSettings(settings TimerSettings) error {
	if settings.Rounding != TimerRoundingExact && settings.Rounding != TimerRounding15 {
		return fmt.Errorf("arredondamento inválido: %q", settings.Rounding)
	}
	if settings.LongRunningHours < 0 || settings.LongRunningHours > 24 {
		return fmt.Errorf("limite de horas inválido: %d (use de 0 a 24)", settings.LongRunningHours)
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.appConfig.Timer = settings
	return m.saveLocked()
}
