package backend

import (
	"fmt"

	"logTime-go/backend/api"
)

// Bindings de CRUD de apontamentos de tempo e seus totais.

func (a *App) GetTimeTotalsForPeriod(startDate, endDate string) (*api.TimeTotal, error) {
	if !a.api().IsConfigured() {
		return nil, fmt.Errorf("API não configurada")
	}

	timeTotal, err := a.api().GetTimeTotalsForPeriod(startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("erro ao obter totais de tempo: %v", err)
	}

	return timeTotal, nil
}

func (a *App) GetTimeEntriesForPeriod(startDate, endDate string) ([]api.TimeEntryReport, error) {
	if !a.api().IsConfigured() {
		return nil, fmt.Errorf("API não configurada")
	}

	return a.api().GetTimeEntriesForPeriod(startDate, endDate)
}

func (a *App) GetTimeEntriesWithDetails(startDate, endDate string) ([]api.TimeEntryReport, error) {
	if !a.api().IsConfigured() {
		return nil, fmt.Errorf("API não configurada")
	}

	return a.api().GetTimeEntriesWithDetails(startDate, endDate)
}

func (a *App) GetTimeEntriesForPeriodV2(startDate, endDate string, includeDeleted bool) ([]api.TimeEntryReport, error) {
	if !a.api().IsConfigured() {
		return nil, fmt.Errorf("API não configurada")
	}

	return a.api().GetTimeEntriesForPeriodV2(startDate, endDate, includeDeleted)
}

func (a *App) GetAllTimeEntriesForDay(date string) ([]api.TimeEntryReport, error) {
	if !a.api().IsConfigured() {
		return nil, fmt.Errorf("API não configurada")
	}

	return a.api().GetAllTimeEntriesForDay(date)
}

func (a *App) GetDeletedTimeEntries(startDate, endDate string) ([]api.TimeEntryReport, error) {
	if !a.api().IsConfigured() {
		return nil, fmt.Errorf("API não configurada")
	}

	return a.api().GetDeletedTimeEntries(startDate, endDate)
}

func (a *App) UpdateTimeEntry(entryID int, entry api.TimeEntry) (*api.TimeLogResult, error) {
	if !a.api().IsConfigured() {
		return nil, fmt.Errorf("API não configurada")
	}

	return a.api().UpdateTimeEntry(entryID, entry)
}

func (a *App) DeleteTimeEntry(entryID int) error {
	if !a.api().IsConfigured() {
		return fmt.Errorf("API não configurada")
	}

	return a.api().DeleteTimeEntry(entryID)
}

func (a *App) DeleteMultipleTimeEntries(entryIDs []int) ([]api.DeleteTimeEntryResult, error) {
	if !a.api().IsConfigured() {
		return nil, fmt.Errorf("API não configurada")
	}

	return a.api().DeleteMultipleTimeEntries(entryIDs)
}
