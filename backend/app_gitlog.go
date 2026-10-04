package backend

import (
	"context"
	"errors"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"logTime-go/backend/config"
	"logTime-go/backend/gitlog"
)

// Bindings da integração com Git: sugestão de descrição a partir dos commits
// do dia nos repositórios configurados. Nada aqui fala com o Teamwork.

// openDirectoryDialog é variável para que os testes não abram janela.
var openDirectoryDialog = func(ctx context.Context, opts runtime.OpenDialogOptions) (string, error) {
	return runtime.OpenDirectoryDialog(ctx, opts)
}

var errGitDesativado = errors.New("integração com Git desativada; ative-a em Configurações")

// GetGitSuggestion lista os commits do usuário no dia (AAAA-MM-DD) e devolve a
// sugestão de descrição montada com todos eles.
func (a *App) GetGitSuggestion(date string) (gitlog.Result, error) {
	cfg := a.configManager.GetGitIntegration()
	if !cfg.Enabled {
		return gitlog.Result{}, errGitDesativado
	}
	if len(cfg.Repositories) == 0 {
		return gitlog.Result{}, errors.New("nenhum repositório configurado; adicione um em Configurações > Integração com Git")
	}
	return gitlog.Suggest(a.appContext(), date, gitlog.Options{
		Repositories: cfg.Repositories,
		AuthorEmail:  cfg.AuthorEmail,
	})
}

// BuildGitSuggestion monta a descrição só com os commits que o usuário
// escolheu, com as mesmas regras de GetGitSuggestion.
func (a *App) BuildGitSuggestion(commits []gitlog.Commit) string {
	return gitlog.BuildSuggestion(commits, gitlog.DefaultMaxLen)
}

func (a *App) GetGitIntegration() config.GitIntegration {
	return a.configManager.GetGitIntegration()
}

func (a *App) SaveGitIntegration(cfg config.GitIntegration) error {
	return a.configManager.SetGitIntegration(cfg)
}

// SelectGitRepository abre o seletor de pasta e devolve o caminho escolhido
// depois de confirmar que é um repositório git. Devolve "" se o usuário
// cancelar. Não salva: o frontend acrescenta à lista e chama SaveGitIntegration.
func (a *App) SelectGitRepository() (string, error) {
	path, err := openDirectoryDialog(a.appContext(), runtime.OpenDialogOptions{
		Title: "Escolha a pasta de um repositório Git",
	})
	if err != nil || path == "" {
		return "", err
	}
	if err := gitlog.ValidateRepo(a.appContext(), path); err != nil {
		return "", err
	}
	return path, nil
}
