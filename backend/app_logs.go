package backend

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

// Bindings de diagnóstico: onde ficam os logs e atalho para abri-los, para que
// o usuário consiga anexá-los a um pedido de suporte.

// GetLogsPath devolve a pasta dos logs (~/.teamwork-logger/logs), ou "" se o
// arquivo de log não pôde ser aberto nesta execução.
func (a *App) GetLogsPath() string {
	return a.logsDir
}

// OpenLogsFolder abre a pasta de logs no gerenciador de arquivos do sistema.
func (a *App) OpenLogsFolder() error {
	if a.logsDir == "" {
		return errors.New("a pasta de logs não está disponível nesta execução")
	}
	if _, err := os.Stat(a.logsDir); err != nil {
		return fmt.Errorf("pasta de logs não encontrada: %v", err)
	}
	return openInFileManager(a.logsDir)
}

// openInFileManager é variável para que os testes não abram janelas.
var openInFileManager = func(path string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("explorer", path)
	case "darwin":
		cmd = exec.Command("open", path)
	default:
		cmd = exec.Command("xdg-open", path)
	}
	// Só Start: o explorer devolve código de saída 1 mesmo quando abre a
	// pasta, e esperar o processo não acrescenta nada.
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("não foi possível abrir a pasta: %v", err)
	}
	go func() { _ = cmd.Wait() }()
	return nil
}
