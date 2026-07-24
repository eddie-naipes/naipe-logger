package backend

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Bindings de relatórios em PDF e abertura da pasta de destino.

func (a *App) DownloadCurrentMonthReport() (string, error) {
	if !a.api().IsConfigured() {
		return "", fmt.Errorf("API não configurada. Configure sua conta antes de exportar relatórios")
	}

	filePath, err := a.api().DownloadCurrentMonthTimeReport()
	if err != nil {
		return "", fmt.Errorf("erro ao baixar relatório: %v", err)
	}

	return filePath, nil
}

func (a *App) DownloadTimeReport(startDate, endDate string) (string, error) {
	if !a.api().IsConfigured() {
		return "", fmt.Errorf("API não configurada. Configure sua conta antes de exportar relatórios")
	}

	filePath, err := a.api().GetDefaultReportPath()
	if err != nil {
		return "", fmt.Errorf("erro ao obter caminho padrão de relatório: %v", err)
	}

	filePath = strings.TrimSuffix(filePath, filepath.Ext(filePath)) + "_" + startDate + "_" + endDate + ".pdf"

	err = a.api().DownloadTimeReportPDF(startDate, endDate, filePath)
	if err != nil {
		return "", fmt.Errorf("erro ao baixar relatório: %v", err)
	}

	return filePath, nil
}

func (a *App) OpenDirectoryPath(filePath string) error {
	dirPath := filepath.Dir(filePath)

	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", dirPath)
	case "darwin":
		cmd = exec.Command("open", dirPath)
	case "linux":
		cmd = exec.Command("xdg-open", dirPath)
	default:
		return fmt.Errorf("sistema operacional não suportado: %s", runtime.GOOS)
	}

	return cmd.Start()
}
