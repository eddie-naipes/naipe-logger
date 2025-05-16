// api/reports.go

package api

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (t *TeamworkAPI) DownloadUtilizationReport(startDate, endDate, filePath string) error {
	if !t.IsConfigured() {
		return fmt.Errorf("API não configurada")
	}

	_, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return fmt.Errorf("data inicial inválida: %v", err)
	}

	_, err = time.Parse("2006-01-02", endDate)
	if err != nil {
		return fmt.Errorf("data final inválida: %v", err)
	}

	// Verificar se o relatório já existe e está atualizado
	if fileInfo, err := os.Stat(filePath); err == nil {
		modTime := fileInfo.ModTime()
		// Se o arquivo foi modificado nas últimas 24 horas, não baixar novamente
		if time.Since(modTime) < 24*time.Hour {
			return nil
		}
	}

	path := fmt.Sprintf("/projects/api/v3/reporting/precanned/utilization.pdf?startDate=%s&endDate=%s&userId=%d",
		startDate, endDate, t.Config.UserID)
	url := t.buildURL(path)

	t.logDebug("Baixando relatório de utilização de %s a %s...", startDate, endDate)

	req, err := t.createRequest("GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Accept", "application/pdf")

	resp, body, err := t.doRequest(req)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		return fmt.Errorf("erro ao baixar relatório: %d %s - %s",
			resp.StatusCode, resp.Status, string(body[:minValue(len(body), 100)]))
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/pdf") {
		return fmt.Errorf("tipo de conteúdo inesperado: %s", contentType)
	}

	dir := filepath.Dir(filePath)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("erro ao criar diretório: %v", err)
		}
	}

	// Usar buffer temporário para garantir que o arquivo só seja atualizado se o download for completo
	tempFile := filePath + ".tmp"
	if err := os.WriteFile(tempFile, body, 0644); err != nil {
		return fmt.Errorf("erro ao salvar arquivo temporário: %v", err)
	}

	if err := os.Rename(tempFile, filePath); err != nil {
		os.Remove(tempFile) // Limpar o arquivo temporário em caso de erro
		return fmt.Errorf("erro ao mover arquivo para destino final: %v", err)
	}

	t.logDebug("Relatório salvo em: %s", filePath)
	return nil
}

func (t *TeamworkAPI) GetDefaultReportPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("erro ao obter diretório do usuário: %v", err)
	}

	now := time.Now()
	monthYear := now.Format("2006-01")

	reportsDir := filepath.Join(homeDir, "TeamworkReports")
	if err := os.MkdirAll(reportsDir, 0755); err != nil {
		return "", fmt.Errorf("erro ao criar diretório de relatórios: %v", err)
	}

	fileName := fmt.Sprintf("TeamworkUtilization_%s.pdf", monthYear)
	return filepath.Join(reportsDir, fileName), nil
}

func (t *TeamworkAPI) DownloadCurrentMonthReport() (string, error) {
	now := time.Now()
	startDate := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	endDate := time.Date(now.Year(), now.Month()+1, 0, 0, 0, 0, 0, now.Location())

	startDateStr := startDate.Format("2006-01-02")
	endDateStr := endDate.Format("2006-01-02")

	filePath, err := t.GetDefaultReportPath()
	if err != nil {
		return "", err
	}

	err = t.DownloadUtilizationReport(startDateStr, endDateStr, filePath)
	if err != nil {
		return "", err
	}

	return filePath, nil
}
