package backend

import (
	"fmt"

	"logTime-go/backend/api"
)

// Bindings de CRUD de apontamentos de tempo e seus totais.

func (a *App) GetTimeTotalsForPeriod(startDate, endDate string) (*api.TimeTotal, error) {
	client, err := a.client()
	if err != nil {
		return nil, err
	}

	timeTotal, err := client.GetTimeTotalsForPeriod(startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("erro ao obter totais de tempo: %v", err)
	}

	return timeTotal, nil
}

func (a *App) GetTimeEntriesForPeriod(startDate, endDate string) ([]api.TimeEntryReport, error) {
	client, err := a.client()
	if err != nil {
		return nil, err
	}
	return client.GetTimeEntriesForPeriod(startDate, endDate)
}

func (a *App) GetTimeEntriesForPeriodV2(startDate, endDate string, includeDeleted bool) ([]api.TimeEntryReport, error) {
	client, err := a.client()
	if err != nil {
		return nil, err
	}
	return client.GetTimeEntriesForPeriodV2(startDate, endDate, includeDeleted)
}

func (a *App) UpdateTimeEntry(entryID int, entry api.TimeEntry) (*api.TimeLogResult, error) {
	client, err := a.client()
	if err != nil {
		return nil, err
	}
	return client.UpdateTimeEntry(entryID, entry)
}

func (a *App) DeleteMultipleTimeEntries(entryIDs []int) ([]api.DeleteTimeEntryResult, error) {
	client, err := a.client()
	if err != nil {
		return nil, err
	}
	return client.DeleteMultipleTimeEntries(entryIDs)
}
