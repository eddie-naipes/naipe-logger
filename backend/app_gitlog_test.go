package backend

import (
	"context"
	"os/exec"
	"testing"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"logTime-go/backend/gitlog"
)

func trocarSeletorDePasta(t *testing.T, escolha string) {
	t.Helper()
	original := openDirectoryDialog
	openDirectoryDialog = func(context.Context, runtime.OpenDialogOptions) (string, error) {
		return escolha, nil
	}
	t.Cleanup(func() { openDirectoryDialog = original })
}

func TestSelectGitRepositoryCanceladoDevolveVazio(t *testing.T) {
	trocarSeletorDePasta(t, "")
	path, err := (&App{}).SelectGitRepository()
	if err != nil || path != "" {
		t.Errorf("SelectGitRepository = %q, %v; esperava vazio sem erro", path, err)
	}
}

func TestSelectGitRepositoryRecusaPastaQueNaoERepositorio(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git não está no PATH")
	}
	trocarSeletorDePasta(t, t.TempDir())
	if _, err := (&App{}).SelectGitRepository(); err == nil {
		t.Error("pasta comum aceita como repositório")
	}
}

func TestBuildGitSuggestionUsaAsRegrasDoPacote(t *testing.T) {
	got := (&App{}).BuildGitSuggestion([]gitlog.Commit{
		{Repo: "r", Subject: "feat: tela nova"},
		{Repo: "r", Subject: "fix: tela nova"},
	})
	if got != "tela nova" {
		t.Errorf("BuildGitSuggestion = %q", got)
	}
}
