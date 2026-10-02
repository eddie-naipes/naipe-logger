package backend

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	goruntime "runtime"
	"sync"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"logTime-go/backend/update"
)

// Bindings de atualização automática via GitHub Releases.
//
// Eventos emitidos para o frontend:
//   - "update:available" (update.Info): na inicialização, se
//     AppSettings.CheckUpdatesOnStartup estiver ligado e houver versão nova;
//   - "update:progress" (update.Progress {received, total}): durante o
//     download do instalador em DownloadAndInstallUpdate.

const (
	EventUpdateAvailable = "update:available"
	EventUpdateProgress  = "update:progress"
)

// Indireções sobre o runtime do Wails e o sistema: o runtime exige o contexto
// real da janela, inexistente nos testes, e os testes não podem executar
// instaladores nem fechar o processo.
var (
	emitEvent = func(ctx context.Context, name string, data ...interface{}) {
		runtime.EventsEmit(ctx, name, data...)
	}
	openBrowserURL = func(ctx context.Context, url string) { runtime.BrowserOpenURL(ctx, url) }
	quitApp        = func(ctx context.Context) { runtime.Quit(ctx) }
	startInstaller = func(path string) error {
		cmd := exec.Command(path)
		if err := cmd.Start(); err != nil {
			return err
		}
		// Não espera: o app vai fechar para o instalador poder substituir o
		// executável. Release evita manter o handle do processo.
		return cmd.Process.Release()
	}
)

// goos é variável para que os testes simulem outro sistema.
var goos = func() string { return goruntime.GOOS }

// installMu impede dois downloads simultâneos (clique duplo no botão).
var installMu sync.Mutex

// appContext devolve o contexto da aplicação (Background antes do Startup).
func (a *App) appContext() context.Context {
	a.apiMutex.RLock()
	defer a.apiMutex.RUnlock()
	if a.ctx == nil {
		return context.Background()
	}
	return a.ctx
}

func (a *App) getUpdater() *update.Updater {
	if a.updater == nil {
		return update.New(a.GetAppVersion(), goos())
	}
	return a.updater
}

// CheckForUpdate consulta a última release publicada (resultado em cache por
// 1h). Available indica versão mais nova; CanInstall indica se
// DownloadAndInstallUpdate funciona aqui (só Windows e fora de builds -dev).
func (a *App) CheckForUpdate() (update.Info, error) {
	return a.getUpdater().Check(a.appContext())
}

// CheckForUpdateNow ignora o cache de 1h (botão "verificar agora").
func (a *App) CheckForUpdateNow() (update.Info, error) {
	return a.getUpdater().Refresh(a.appContext())
}

// DownloadAndInstallUpdate baixa o instalador da última release, confere o
// SHA-256 contra o SHA256SUMS.txt da mesma release, executa o instalador e
// fecha o app. Emite "update:progress" durante o download. Fora do Windows
// devolve erro orientando a usar OpenReleasePage.
func (a *App) DownloadAndInstallUpdate() error {
	if !installMu.TryLock() {
		return fmt.Errorf("a atualização já está em andamento")
	}
	defer installMu.Unlock()

	ctx := a.appContext()
	path, err := a.getUpdater().DownloadInstaller(ctx, func(p update.Progress) {
		emitEvent(ctx, EventUpdateProgress, p)
	})
	if err != nil {
		slog.Warn("Atualização não instalada", "err", err)
		return err
	}

	slog.Info("Executando instalador da atualização", "arquivo", path)
	if err := startInstaller(path); err != nil {
		return fmt.Errorf("não foi possível executar o instalador: %v", err)
	}
	quitApp(ctx)
	return nil
}

// OpenReleasePage abre no navegador a página da última release (ou a lista de
// releases, se ainda não houve verificação).
func (a *App) OpenReleasePage() error {
	u := a.getUpdater()
	target := update.ReleasesPageURL()
	if info, err := u.Check(a.appContext()); err == nil && info.ReleaseURL != "" {
		target = info.ReleaseURL
	}
	safe, err := u.SafeReleaseURL(target)
	if err != nil {
		return err
	}
	openBrowserURL(a.appContext(), safe)
	return nil
}

// checkUpdatesOnStartup avisa o frontend, por evento, de que há versão nova.
// Roda em goroutine a partir do Startup; falhas só vão para o log.
func (a *App) checkUpdatesOnStartup() {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("Panic ao verificar atualizações", "panic", r)
		}
	}()
	if a.configManager == nil || !a.configManager.GetAppSettings().CheckUpdatesOnStartup {
		return
	}
	a.notifyIfUpdateAvailable()
}

// notifyIfUpdateAvailable emite "update:available" se houver versão nova.
func (a *App) notifyIfUpdateAvailable() {
	info, err := a.CheckForUpdate()
	if err != nil {
		slog.Info("Não foi possível verificar atualizações", "err", err)
		return
	}
	if info.Available {
		slog.Info("Nova versão disponível", "atual", info.CurrentVersion, "nova", info.LatestVersion)
		emitEvent(a.appContext(), EventUpdateAvailable, info)
	}
}
