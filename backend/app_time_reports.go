package backend

import (
	"fmt"
	"path/filepath"

	"logTime-go/backend/api"
	"logTime-go/backend/internal/fsutil"
	"logTime-go/backend/reports"
)

// Bindings da página de Relatórios: resumo do período e exportação em CSV.
// Só leem lançamentos; nada aqui cria, altera ou apaga dados no Teamwork.

// csvFilePerm: o CSV é um documento do usuário, como o PDF, e não um arquivo
// interno do app; fica legível para outros programas do mesmo perfil.
const csvFilePerm = 0644

// GetTimeReportSummary agrega os lançamentos do usuário atual no período
// (AAAA-MM-DD, no máximo reports.MaxPeriodDays dias).
func (a *App) GetTimeReportSummary(startDate, endDate string) (reports.Report, error) {
	period, err := reports.ParsePeriod(startDate, endDate)
	if err != nil {
		return reports.Report{}, err
	}
	client, err := a.client()
	if err != nil {
		return reports.Report{}, err
	}
	_, report, err := buildTimeReport(client, period)
	return report, err
}

// ExportTimeReportCSV grava o CSV na mesma pasta do relatório em PDF e
// devolve o caminho. detailed=true lista os lançamentos; false resume por
// tarefa.
func (a *App) ExportTimeReportCSV(startDate, endDate string, detailed bool) (string, error) {
	// Valida antes de tudo: as datas entram no nome do arquivo.
	period, err := reports.ParsePeriod(startDate, endDate)
	if err != nil {
		return "", err
	}
	client, err := a.client()
	if err != nil {
		return "", err
	}

	entries, report, err := buildTimeReport(client, period)
	if err != nil {
		return "", err
	}

	kind := reports.CSVSummary
	data := reports.SummaryCSV(report)
	if detailed {
		kind = reports.CSVDetailed
		data = reports.DetailedCSV(entries)
	}

	defaultPath, err := client.GetDefaultReportPath()
	if err != nil {
		return "", fmt.Errorf("erro ao obter pasta de relatórios: %v", err)
	}
	filePath := filepath.Join(filepath.Dir(defaultPath), reports.FileName(kind, period))

	if err := fsutil.WriteFileAtomic(filePath, data, csvFilePerm); err != nil {
		return "", fmt.Errorf("erro ao gravar CSV: %v", err)
	}
	return filePath, nil
}

// buildTimeReport busca os lançamentos (só os não excluídos do usuário) e os
// dias úteis do período e monta o relatório. Devolve também os lançamentos
// filtrados, usados no CSV detalhado.
func buildTimeReport(client *api.TeamworkAPI, period reports.Period) ([]api.TimeEntryReport, reports.Report, error) {
	entries, err := client.GetTimeEntriesForPeriodV2(period.StartDate(), period.EndDate(), false)
	if err != nil {
		return nil, reports.Report{}, fmt.Errorf("erro ao buscar lançamentos: %v", err)
	}

	// GetWorkingDays devolve erro quando o período não tem nenhum dia útil
	// (ex.: só um fim de semana); para o relatório isso é zero dias úteis.
	workingDays, err := client.GetWorkingDays(period.StartDate(), period.EndDate())
	if err != nil {
		workingDays = nil
	}

	userID := client.Config.UserID
	filtered := reports.Filter(entries, period, userID)
	report := reports.Build(filtered, period, workingDays, client.Config.MinutosPorDia, userID)
	return filtered, report, nil
}
