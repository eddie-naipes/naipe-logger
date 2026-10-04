package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGitIntegrationPadraoParaConfigAntigo(t *testing.T) {
	fakeKeyring(t)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(legacyConfigJSON), 0600); err != nil {
		t.Fatal(err)
	}
	m, err := newManagerAt(dir)
	if err != nil {
		t.Fatalf("newManagerAt: %v", err)
	}
	g := m.GetGitIntegration()
	if !g.Enabled || g.Repositories == nil || len(g.Repositories) != 0 || g.AuthorEmail != "" {
		t.Errorf("padrão = %+v", g)
	}
}

func TestSetGitIntegrationNormalizaEPersiste(t *testing.T) {
	fakeKeyring(t)
	dir := t.TempDir()
	m, err := newManagerAt(dir)
	if err != nil {
		t.Fatalf("newManagerAt: %v", err)
	}
	repo := t.TempDir()
	err = m.SetGitIntegration(GitIntegration{
		Enabled:      true,
		Repositories: []string{"  " + repo + "  ", "", repo + string(filepath.Separator)},
		AuthorEmail:  " eu@exemplo.invalid ",
	})
	if err != nil {
		t.Fatalf("SetGitIntegration: %v", err)
	}

	recarregado, err := newManagerAt(dir)
	if err != nil {
		t.Fatalf("recarga: %v", err)
	}
	g := recarregado.GetGitIntegration()
	if len(g.Repositories) != 1 || g.Repositories[0] != filepath.Clean(repo) {
		t.Errorf("repositórios = %v", g.Repositories)
	}
	if g.AuthorEmail != "eu@exemplo.invalid" {
		t.Errorf("e-mail = %q", g.AuthorEmail)
	}
}

func TestSetGitIntegrationRecusaValoresInvalidos(t *testing.T) {
	fakeKeyring(t)
	m, err := newManagerAt(t.TempDir())
	if err != nil {
		t.Fatalf("newManagerAt: %v", err)
	}
	if err := m.SetGitIntegration(GitIntegration{AuthorEmail: "sem-arroba"}); err == nil {
		t.Error("e-mail inválido aceito")
	}
	if err := m.SetGitIntegration(GitIntegration{Repositories: []string{"relativo/repo"}}); err == nil {
		t.Error("caminho relativo aceito")
	}
}
