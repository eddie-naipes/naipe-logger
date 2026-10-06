package backend

import (
	"errors"
	"fmt"
	"time"

	"logTime-go/backend/api"
	"logTime-go/backend/audit"
	"logTime-go/backend/config"
)

// Bindings do "Fechamento do mês": audita os lançamentos do usuário num mês
// (ver backend/audit). Só leem lançamentos; as correções usam os bindings de
// edição/exclusão já existentes (UpdateTimeEntry, DeleteMultipleTimeEntries).

// auditSource é o que a auditoria precisa do Teamwork (satisfeito por
// *api.TeamworkAPI).
type auditSource interface {
	GetTimeEntriesForPeriodV2(startDate, endDate string, includeDeleted bool) ([]api.TimeEntryReport, error)
	GetWorkingDays(inicio, fim string) ([]string, error)
	ListNonWorkingDays(year, month int) ([]api.NonWorkingDay, error)
}

// auditNow é o relógio da auditoria (substituível nos testes).
var auditNow = time.Now

// RunMonthAudit audita o mês (1-12) do ano e devolve os problemas (inclusive
// os ignorados, marcados), a contagem por tipo e o resumo lançado × esperado.
func (a *App) RunMonthAudit(year, month int) (audit.Result, error) {
	if _, _, err := audit.MonthRange(year, month); err != nil {
		return audit.Result{}, err
	}
	client, err := a.client()
	if err != nil {
		return audit.Result{}, err
	}
	return runMonthAudit(client, client.Config.UserID, client.Config.MinutosPorDia, a.auditSettings(), year, month, auditNow())
}

func (a *App) GetAuditSettings() config.AuditSettings {
	return a.auditSettings()
}

// SaveAuditSettings grava limite diário e descrições genéricas; a lista de
// ignorados só muda por IgnoreAuditIssue/UnignoreAuditIssue.
func (a *App) SaveAuditSettings(settings config.AuditSettings) error {
	if a.configManager == nil {
		return errors.New("configuração indisponível")
	}
	return a.configManager.SetAuditSettings(settings)
}

// IgnoreAuditIssue esconde um problema (pela chave audit.Issue.Key) nas
// próximas auditorias.
func (a *App) IgnoreAuditIssue(key string) error {
	if a.configManager == nil {
		return errors.New("configuração indisponível")
	}
	return a.configManager.IgnoreAuditIssue(key)
}

// UnignoreAuditIssue volta a exibir um problema ignorado.
func (a *App) UnignoreAuditIssue(key string) error {
	if a.configManager == nil {
		return errors.New("configuração indisponível")
	}
	return a.configManager.UnignoreAuditIssue(key)
}

func (a *App) auditSettings() config.AuditSettings {
	if a.configManager == nil {
		return config.DefaultAuditSettings()
	}
	return a.configManager.GetAuditSettings()
}

// monthAuditPendingCount alimenta o lembrete de fim de mês: problemas não
// ignorados do mês.
func (a *App) monthAuditPendingCount(year, month int) (int, error) {
	result, err := a.RunMonthAudit(year, month)
	if err != nil {
		return 0, err
	}
	return result.PendingCount(), nil
}

// runMonthAudit busca os dados do mês e roda a auditoria pura.
func runMonthAudit(src auditSource, userID, minutesPerDay int, settings config.AuditSettings, year, month int, now time.Time) (audit.Result, error) {
	start, end, err := audit.MonthRange(year, month)
	if err != nil {
		return audit.Result{}, err
	}

	entries, err := src.GetTimeEntriesForPeriodV2(start, end, false)
	if err != nil {
		return audit.Result{}, fmt.Errorf("erro ao buscar lançamentos: %v", err)
	}

	// GetWorkingDays devolve erro quando o mês não tem nenhum dia útil (ex.:
	// férias o mês inteiro); para a auditoria isso é zero dias úteis.
	workingDays, err := src.GetWorkingDays(start, end)
	if err != nil {
		workingDays = nil
	}

	nonWorking, err := src.ListNonWorkingDays(year, month)
	if err != nil {
		// Sem a lista, o motivo do dia não útil fica genérico; a verificação
		// em si usa os dias úteis.
		nonWorking = nil
	}

	if minutesPerDay <= 0 {
		minutesPerDay = 8 * 60
	}

	return audit.Run(audit.Input{
		Year:                year,
		Month:               month,
		Today:               now.Format("2006-01-02"),
		Entries:             entries,
		UserID:              userID,
		WorkingDays:         workingDays,
		NonWorkingDays:      nonWorking,
		MinutesPerDay:       minutesPerDay,
		DailyLimitMinutes:   settings.DailyLimitMinutes,
		GenericDescriptions: settings.GenericDescriptions,
		Ignored:             settings.IgnoredIssues,
	}), nil
}
