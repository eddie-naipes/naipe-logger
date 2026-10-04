package config

import (
	"fmt"
	"path/filepath"
	"strings"
)

// GitIntegration configura a sugestão de descrição a partir dos commits do
// dia (pacote backend/gitlog).
type GitIntegration struct {
	// Enabled liga o botão de sugestão nos campos de descrição.
	Enabled bool `json:"enabled"`
	// Repositories são caminhos de pastas de repositórios git.
	Repositories []string `json:"repositories"`
	// AuthorEmail filtra os commits; vazio usa `git config user.email` de
	// cada repositório.
	AuthorEmail string `json:"authorEmail"`
}

// defaultGitIntegration vale para config.json antigos, sem o campo: ligada,
// mas sem repositórios, então nada acontece até o usuário adicionar um.
func defaultGitIntegration() GitIntegration {
	return GitIntegration{Enabled: true, Repositories: []string{}}
}

// normalize apara os valores, descarta caminhos vazios ou repetidos e garante
// slice não-nulo (o frontend recebe [] e não null).
func (g GitIntegration) normalize() GitIntegration {
	out := GitIntegration{
		Enabled:      g.Enabled,
		AuthorEmail:  strings.TrimSpace(g.AuthorEmail),
		Repositories: make([]string, 0, len(g.Repositories)),
	}
	seen := map[string]bool{}
	for _, repo := range g.Repositories {
		repo = strings.TrimSpace(repo)
		if repo == "" {
			continue
		}
		repo = filepath.Clean(repo)
		key := strings.ToLower(repo)
		if seen[key] {
			continue
		}
		seen[key] = true
		out.Repositories = append(out.Repositories, repo)
	}
	return out
}

// GetGitIntegration devolve uma cópia da configuração da integração com Git.
func (m *Manager) GetGitIntegration() GitIntegration {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.appConfig.GitIntegration.normalize()
}

// SetGitIntegration valida e persiste a configuração da integração com Git.
func (m *Manager) SetGitIntegration(g GitIntegration) error {
	g = g.normalize()
	if g.AuthorEmail != "" && (!strings.Contains(g.AuthorEmail, "@") || strings.ContainsAny(g.AuthorEmail, " <>")) {
		return fmt.Errorf("e-mail de autor inválido: %q", g.AuthorEmail)
	}
	for _, repo := range g.Repositories {
		if !filepath.IsAbs(repo) {
			return fmt.Errorf("o caminho do repositório precisa ser absoluto: %q", repo)
		}
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.appConfig.GitIntegration = g
	return m.saveLocked()
}
