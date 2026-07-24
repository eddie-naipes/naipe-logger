// Package backend expõe ao frontend (via Wails) os bindings da aplicação. Os
// métodos de App estão divididos por domínio em arquivos app_*.go; este arquivo
// concentra o ciclo de vida, a conexão e a fronteira de segurança com o token.
package backend

import (
	"context"
	"fmt"
	"sync"

	"logTime-go/backend/api"
	"logTime-go/backend/config"
)

type App struct {
	ctx           context.Context
	configManager *config.Manager

	// apiMutex protege teamworkAPI, que é substituído quando a conexão muda
	// enquanto outros bindings o leem em paralelo.
	apiMutex    sync.RWMutex
	teamworkAPI *api.TeamworkAPI
}

// api devolve o cliente atual sob lock de leitura.
func (a *App) api() *api.TeamworkAPI {
	a.apiMutex.RLock()
	defer a.apiMutex.RUnlock()
	return a.teamworkAPI
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
// setAPI o lê para repassá-lo a cada cliente novo.
func (a *App) setContext(ctx context.Context) {
	a.apiMutex.Lock()
	defer a.apiMutex.Unlock()
	a.ctx = ctx
}

func NewApp(ctx context.Context) (*App, error) {
	configManager, err := config.NewManager()
	if err != nil {
		return nil, fmt.Errorf("erro ao inicializar gerenciador de configurações: %v", err)
	}

	app := &App{configManager: configManager}
	app.setContext(ctx)
	app.setAPI(api.NewTeamworkAPI(configManager.GetTeamworkConfig()))

	return app, nil
}

func (a *App) Startup(ctx context.Context) {
	a.setContext(ctx)
	a.setAPI(api.NewTeamworkAPI(a.configManager.GetTeamworkConfig()))

	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("Erro crítico durante a inicialização: %v\n", r)
		}
	}()

	a.api().ClearExpiredHolidayCache()

	go func() {
		if err := a.api().PreloadUpcomingHolidays(); err != nil {
			fmt.Printf("Aviso: erro ao pré-carregar feriados: %v\n", err)
		}
	}()
}

func (a *App) Shutdown(ctx context.Context) {
	_ = a.configManager.Save()
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

func (a *App) TestConnection() ([]interface{}, error) {
	success, message := a.api().TestConnection()
	return []interface{}{success, message}, nil
}

// ConnectWithToken valida um token de API do Teamwork e, em caso de sucesso,
// grava-o no cofre de credenciais do sistema. O token nunca é devolvido ao
// frontend nem gravado em config.json.
func (a *App) ConnectWithToken(token, host string) (*api.LoginResponse, error) {
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
	a.setAPI(api.NewTeamworkAPI(a.configManager.GetTeamworkConfig()))
	return nil
}
