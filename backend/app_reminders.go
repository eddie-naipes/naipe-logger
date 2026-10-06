package backend

import (
	"errors"

	"logTime-go/backend/config"
)

// Bindings dos lembretes de lançamento de horas. O agendador em si fica em
// backend/reminders; a infraestrutura de notificações em app_notifications.go.

func (a *App) GetReminderSettings() config.ReminderSettings {
	return a.reminderSettings()
}

func (a *App) SaveReminderSettings(settings config.ReminderSettings) error {
	if a.configManager == nil {
		return errors.New("configuração indisponível")
	}
	return a.configManager.SetReminderSettings(settings)
}

// SendTestReminder envia uma notificação de teste agora. Só deve ser chamado
// por ação explícita do usuário (botão "Testar lembrete").
func (a *App) SendTestReminder() error {
	return a.reminderScheduler().SendTest()
}
