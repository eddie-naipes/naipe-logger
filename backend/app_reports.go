package backend

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Bindings de relatórios em PDF e abertura da pasta de destino.

func (a *App) DownloadCurrentMonthReport() (string, error) {
	client, err := a.client()
	if err != nil {
		return "", err
	}

	filePath, err := client.DownloadCurrentMonthTimeReport()
	if err != nil {
		return "", fmt.Errorf("erro ao baixar relatório: %v", err)
	}

	return filePath, nil
}

func (a *App) DownloadTimeReport(startDate, endDate string) (string, error) {
	client, err := a.client()
	if err != nil {
		return "", err
	}

	// Valida antes de pedir o caminho padrão, que cria a pasta de relatórios.
	if _, err := reportFilePath("relatorio.pdf", startDate, endDate); err != nil {
		return "", err
	}

	defaultPath, err := client.GetDefaultReportPath()
	if err != nil {
		return "", fmt.Errorf("erro ao obter caminho padrão de relatório: %v", err)
	}

	filePath, err := reportFilePath(defaultPath, startDate, endDate)
	if err != nil {
		return "", err
	}

	err = client.DownloadTimeReportPDF(startDate, endDate, filePath)
	if err != nil {
		return "", fmt.Errorf("erro ao baixar relatório: %v", err)
	}

	return filePath, nil
}

// reportFilePath monta <dir>/<nome>_<início>_<fim>.pdf a partir do caminho
// padrão. As datas entram no nome do arquivo, então só são aceitas no formato
// AAAA-MM-DD: um valor como "../../x" escaparia da pasta de relatórios.
func reportFilePath(defaultPath, startDate, endDate string) (string, error) {
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return "", fmt.Errorf("data inicial inválida (use AAAA-MM-DD): %q", startDate)
	}
	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return "", fmt.Errorf("data final inválida (use AAAA-MM-DD): %q", endDate)
	}
	if end.Before(start) {
		return "", fmt.Errorf("a data final (%s) é anterior à inicial (%s)", endDate, startDate)
	}

	base := filepath.Base(defaultPath)
	base = strings.TrimSuffix(base, filepath.Ext(base))
	name := fmt.Sprintf("%s_%s_%s.pdf", base, start.Format("2006-01-02"), end.Format("2006-01-02"))

	return filepath.Join(filepath.Dir(defaultPath), name), nil
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
