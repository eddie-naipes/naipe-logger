// Package backend expõe ao frontend (via Wails) os bindings da aplicação. Os
// métodos de App estão divididos por domínio em arquivos app_*.go; este arquivo
// concentra o ciclo de vida, a conexão e a fronteira de segurança com o token.
package backend

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"

	"logTime-go/backend/api"
	"logTime-go/backend/config"
	"logTime-go/backend/internal/fsutil"
	"logTime-go/backend/logging"
	"logTime-go/backend/update"
)

// errAPINaoConfigurada é devolvido por todo binding que precisa falar com o
// Teamwork enquanto não há host e token válidos.
var errAPINaoConfigurada = errors.New("API não configurada. Configure sua conta na tela de Configurações")

type App struct {
	ctx           context.Context
	configManager *config.Manager

	// apiMutex protege teamworkAPI, que é substituído quando a conexão muda
	// enquanto outros bindings o leem em paralelo.
	apiMutex    sync.RWMutex
	teamworkAPI *api.TeamworkAPI

	// logsDir é a pasta dos logs configurada em main.go ("" se o arquivo
	// não pôde ser aberto).
	logsDir string

	// version é a versão embutida no binário (wails.json, injetada pelo CI).
	version string

	// updater consulta as GitHub Releases; nil nos testes que montam App{}.
	updater *update.Updater

	// features guarda lembretes, cronômetro e notificações (app_notifications.go).
	features features
}

// Options reúne o que main.go descobre antes de criar a App.
type Options struct {
	// LogsDir é a pasta onde backend/logging grava app.log.
	LogsDir string
	// Version é info.productVersion do wails.json embutido (ver ParseProductVersion).
	Version string
}

// api devolve o cliente atual sob lock de leitura.
func (a *App) api() *api.TeamworkAPI {
	a.apiMutex.RLock()
	defer a.apiMutex.RUnlock()
	return a.teamworkAPI
}

// client devolve o cliente atual se houver uma conexão configurada. Os
// bindings devem lê-lo uma única vez e usar a mesma instância até o fim: chamar
// a.api() várias vezes pode pegar clientes diferentes se a conexão mudar no
// meio da chamada.
func (a *App) client() (*api.TeamworkAPI, error) {
	c := a.api()
	if c == nil || !c.IsConfigured() {
		return nil, errAPINaoConfigurada
	}
	return c, nil
}

// setAPI substitui o cliente e lhe entrega o contexto da aplicação, para que as
// requisições do cliente novo também sejam abortadas no encerramento.
func (a *App) setAPI(client *api.TeamworkAPI) {
	a.apiMutex.Lock()
	defer a.apiMutex.Unlock()
	if a.ctx != nil {
		client.SetContext(a.ctx)
	}
	a.teamworkAPI = client
}

// setContext guarda o contexto sob o mesmo lock que protege o cliente, já que
// setAPI o lê para repassá-lo a cada cliente novo, e o entrega também ao
// cliente atual.
func (a *App) setContext(ctx context.Context) {
	a.apiMutex.Lock()
	defer a.apiMutex.Unlock()
	a.ctx = ctx
	if ctx != nil && a.teamworkAPI != nil {
		a.teamworkAPI.SetContext(ctx)
	}
}

func NewApp(ctx context.Context, opts Options) (*App, error) {
	configManager, err := config.NewManager()
	if err != nil {
		return nil, fmt.Errorf("erro ao inicializar gerenciador de configurações: %v", err)
	}

	app := &App{configManager: configManager, logsDir: opts.LogsDir, version: opts.Version}
	app.updater = update.New(app.GetAppVersion(), goos())
	app.setContext(ctx)
	app.setAPI(api.NewTeamworkAPI(configManager.GetTeamworkConfig()))

	setupHolidayDiskCache()

	return app, nil
}

// setupHolidayDiskCache liga o cache de feriados em ~/.teamwork-logger/cache e
// carrega o que já estava salvo. Sem disco o app segue só com a memória.
func setupHolidayDiskCache() {
	appDir, err := fsutil.AppDir()
	if err != nil {
		slog.Warn("Cache de feriados em disco desligado", "err", err)
		return
	}
	if err := api.SetHolidayCacheDir(filepath.Join(appDir, "cache")); err != nil {
		slog.Warn("Cache de feriados em disco desligado", "err", err)
		return
	}
	api.LoadHolidayCacheFromDisk()
}

// Startup recebe o contexto da aplicação. O cliente criado em NewApp é mantido
// (a configuração não mudou desde então); só passa a usar esse contexto.
// A configuração não precisa de OnShutdown (toda mutação grava o disco na
// hora); Shutdown só encerra lembretes e cronômetro.
func (a *App) Startup(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("Erro crítico durante a inicialização", "panic", r)
		}
	}()

	a.setContext(ctx)

	// O cache de feriados não é mais podado aqui: um ano vencido vindo da
	// BrasilAPI continua valendo e é revalidado em segundo plano.
	client := a.api()

	go func() {
		// Um panic numa goroutine derruba o processo inteiro; o recover do
		// Startup não a alcança.
		defer func() {
			if r := recover(); r != nil {
				slog.Error("Panic ao pré-carregar feriados", "panic", r)
			}
		}()
		if err := client.PreloadUpcomingHolidays(); err != nil {
			slog.Warn("Erro ao pré-carregar feriados", "err", err)
		}
	}()

	go a.checkUpdatesOnStartup()

	a.startFeatures(ctx)
}

// Shutdown encerra as verificações periódicas (lembretes, cronômetro).
func (a *App) Shutdown(ctx context.Context) {
	a.stopFeatures(ctx)
}

// GetPublicConfig devolve ao frontend apenas o que ele precisa saber. O token
// de API nunca atravessa a fronteira Go -> JavaScript.
func (a *App) GetPublicConfig() api.PublicConfig {
	cfg := a.configManager.GetTeamworkConfig()
	return api.PublicConfig{
		Configured:    a.api().IsConfigured(),
		ApiHost:       cfg.ApiHost,
		UserID:        cfg.UserID,
		MinutosPorDia: cfg.MinutosPorDia,
	}
}

// IsConfigured informa se há uma conexão utilizável configurada.
func (a *App) IsConfigured() bool {
	return a.api().IsConfigured()
}

// LegacyCredentialPurged informa que uma credencial antiga (email:senha) foi
// encontrada e apagada, para que a UI oriente o usuário a trocar a senha.
func (a *App) LegacyCredentialPurged() bool {
	return a.configManager.LegacyCredentialPurged()
}

// CorruptedConfigBackups lista os arquivos de configuração que estavam
// corrompidos na inicialização e foram renomeados (<nome>.corrompido-<data-hora>).
// Vazio quando nada foi recuperado; caso contrário a UI deve avisar que a
// configuração padrão está em uso e onde está o backup.
func (a *App) CorruptedConfigBackups() []string {
	return a.configManager.CorruptedConfigBackups()
}

// ConnectWithToken valida um token de API do Teamwork e, em caso de sucesso,
// grava-o no cofre de credenciais do sistema. O token nunca é devolvido ao
// frontend nem gravado em config.json.
func (a *App) ConnectWithToken(token, host string) (*api.LoginResponse, error) {
	// Aparado aqui para que o token validado seja exatamente o que vai para o
	// cofre e para a memória.
	token = strings.TrimSpace(token)

	loginResponse, err := api.ValidateToken(token, host)
	if err != nil {
		return nil, err
	}

	if !loginResponse.Success {
		return loginResponse, nil
	}

	if err := a.configManager.SetConnection(loginResponse.InstanceID, loginResponse.UserID, token); err != nil {
		return nil, fmt.Errorf("erro ao salvar configuração: %v", err)
	}

	a.setAPI(api.NewTeamworkAPI(a.configManager.GetTeamworkConfig()))

	return loginResponse, nil
}

// Logout remove o token do cofre do sistema e limpa a conexão.
func (a *App) Logout() error {
	if err := a.configManager.ClearConnection(); err != nil {
		return err
	}
	// O token antigo não precisa mais ficar na lista de mascaramento.
	logging.ClearSecrets()
	a.setAPI(api.NewTeamworkAPI(a.configManager.GetTeamworkConfig()))
	return nil
}
