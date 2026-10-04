package gitlog

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

const (
	euEmail    = "eu@exemplo.invalid"
	outroEmail = "outra@exemplo.invalid"
	diaTeste   = "2026-09-15"
)

// isolarGit garante que o git dos testes não leia a configuração do usuário
// (assinatura de commits, hooks globais, user.email).
func isolarGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git não está no PATH")
	}
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
}

func git(t *testing.T, dir string, env []string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(), env...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func novoRepo(t *testing.T, nome string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), nome)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	git(t, dir, nil, "init", "-q", "-b", "main")
	git(t, dir, nil, "config", "user.email", euEmail)
	git(t, dir, nil, "config", "user.name", "Eu")
	git(t, dir, nil, "config", "commit.gpgsign", "false")
	return dir
}

// commitEm cria um commit vazio com autor e datas controlados (hora local).
func commitEm(t *testing.T, dir, email, quando, assunto string) {
	t.Helper()
	ts, err := time.ParseInLocation("2006-01-02 15:04", quando, time.Local)
	if err != nil {
		t.Fatal(err)
	}
	data := ts.Format(time.RFC3339)
	git(t, dir, []string{
		"GIT_AUTHOR_DATE=" + data, "GIT_COMMITTER_DATE=" + data,
		"GIT_AUTHOR_EMAIL=" + email, "GIT_COMMITTER_EMAIL=" + email,
		"GIT_AUTHOR_NAME=Pessoa", "GIT_COMMITTER_NAME=Pessoa",
	}, "commit", "-q", "--allow-empty", "-m", assunto)
}

func assuntos(commits []Commit) []string {
	out := make([]string, len(commits))
	for i, c := range commits {
		out[i] = c.Subject
	}
	return out
}

func TestSuggestFiltraPorDiaEAutor(t *testing.T) {
	isolarGit(t)
	repo := novoRepo(t, "app")
	commitEm(t, repo, euEmail, "2026-09-14 23:50", "feat: vespera")
	commitEm(t, repo, euEmail, "2026-09-15 09:00", "feat(api): corrige login")
	commitEm(t, repo, outroEmail, "2026-09-15 10:00", "fix: commit de outra pessoa")
	commitEm(t, repo, euEmail, "2026-09-15 23:30", "fix: ajusta relatorio")
	commitEm(t, repo, euEmail, "2026-09-16 00:10", "chore: dia seguinte")

	res, err := Suggest(context.Background(), diaTeste, Options{Repositories: []string{repo}})
	if err != nil {
		t.Fatalf("Suggest: %v", err)
	}
	got := strings.Join(assuntos(res.Commits), " | ")
	want := "feat(api): corrige login | fix: ajusta relatorio"
	if got != want {
		t.Errorf("commits = %q, esperava %q", got, want)
	}
	if res.Suggestion != "corrige login; ajusta relatorio" {
		t.Errorf("sugestão = %q", res.Suggestion)
	}
	if len(res.Warnings) != 0 {
		t.Errorf("avisos inesperados: %v", res.Warnings)
	}
	c := res.Commits[0]
	if c.Repo != "app" || c.Time != "09:00" || len(c.Hash) < 7 {
		t.Errorf("commit = %+v", c)
	}
}

func TestSuggestUsaEmailConfiguradoNaIntegracao(t *testing.T) {
	isolarGit(t)
	repo := novoRepo(t, "app")
	commitEm(t, repo, euEmail, "2026-09-15 09:00", "meu commit")
	commitEm(t, repo, outroEmail, "2026-09-15 10:00", "commit dela")

	res, err := Suggest(context.Background(), diaTeste, Options{
		Repositories: []string{repo}, AuthorEmail: strings.ToUpper(outroEmail),
	})
	if err != nil {
		t.Fatalf("Suggest: %v", err)
	}
	if got := assuntos(res.Commits); len(got) != 1 || got[0] != "commit dela" {
		t.Errorf("commits = %v", got)
	}
}

func TestSuggestIgnoraMerges(t *testing.T) {
	isolarGit(t)
	repo := novoRepo(t, "app")
	commitEm(t, repo, euEmail, "2026-09-15 08:00", "base")
	git(t, repo, nil, "checkout", "-q", "-b", "outra")
	commitEm(t, repo, euEmail, "2026-09-15 09:00", "no ramo")
	git(t, repo, nil, "checkout", "-q", "main")
	commitEm(t, repo, euEmail, "2026-09-15 10:00", "na main")
	data := time.Date(2026, 9, 15, 11, 0, 0, 0, time.Local).Format(time.RFC3339)
	git(t, repo, []string{"GIT_AUTHOR_DATE=" + data, "GIT_COMMITTER_DATE=" + data},
		"merge", "-q", "--no-ff", "-m", "Merge branch outra", "outra")

	res, err := Suggest(context.Background(), diaTeste, Options{Repositories: []string{repo}})
	if err != nil {
		t.Fatalf("Suggest: %v", err)
	}
	got := strings.Join(assuntos(res.Commits), ", ")
	if got != "base, no ramo, na main" {
		t.Errorf("commits = %q (merge não deveria aparecer)", got)
	}
}

func TestSuggestAvisaRepoInvalidoESegue(t *testing.T) {
	isolarGit(t)
	repo := novoRepo(t, "bom")
	commitEm(t, repo, euEmail, "2026-09-15 09:00", "feito")
	naoRepo := t.TempDir()
	inexistente := filepath.Join(t.TempDir(), "nao-existe")

	res, err := Suggest(context.Background(), diaTeste, Options{
		Repositories: []string{naoRepo, inexistente, "  ", repo},
	})
	if err != nil {
		t.Fatalf("Suggest: %v", err)
	}
	if len(res.Commits) != 1 {
		t.Errorf("commits = %v", assuntos(res.Commits))
	}
	if len(res.Warnings) != 2 {
		t.Fatalf("avisos = %v, esperava 2", res.Warnings)
	}
	if !strings.Contains(res.Warnings[0], "não é um repositório git") ||
		!strings.Contains(res.Warnings[1], "pasta não encontrada") {
		t.Errorf("avisos = %v", res.Warnings)
	}
}

func TestSuggestRecusaDataInvalida(t *testing.T) {
	if _, err := Suggest(context.Background(), "15/09/2026", Options{}); err == nil {
		t.Error("data fora de AAAA-MM-DD deveria falhar")
	}
}

func TestSuggestSemGit(t *testing.T) {
	orig := lookPath
	lookPath = func(string) (string, error) { return "", exec.ErrNotFound }
	t.Cleanup(func() { lookPath = orig })

	if _, err := Suggest(context.Background(), diaTeste, Options{}); !errors.Is(err, ErrGitNaoEncontrado) {
		t.Errorf("erro = %v, esperava ErrGitNaoEncontrado", err)
	}
}

func TestValidateRepo(t *testing.T) {
	isolarGit(t)
	if err := ValidateRepo(context.Background(), novoRepo(t, "r")); err != nil {
		t.Errorf("repo válido recusado: %v", err)
	}
	if err := ValidateRepo(context.Background(), t.TempDir()); err == nil {
		t.Error("pasta comum aceita como repositório")
	}
}

func TestBuildSuggestionDeduplicaEAgrupa(t *testing.T) {
	commits := []Commit{
		{Repo: "api", Subject: "feat(api): corrige login"},
		{Repo: "api", Subject: "fix: Corrige login"},
		{Repo: "web", Subject: "tela nova"},
		{Repo: "api", Subject: "refactor!: limpa codigo"},
		{Repo: "web", Subject: "   "},
	}
	got := BuildSuggestion(commits, 0)
	want := "api: corrige login; limpa codigo | web: tela nova"
	if got != want {
		t.Errorf("BuildSuggestion = %q, esperava %q", got, want)
	}
	if BuildSuggestion(nil, 0) != "" {
		t.Error("sem commits a sugestão deveria ser vazia")
	}
}

func TestBuildSuggestionLimitaTamanho(t *testing.T) {
	var commits []Commit
	for i := 0; i < 40; i++ {
		commits = append(commits, Commit{Repo: "r", Subject: "ação número " + strings.Repeat("x", i)})
	}
	got := BuildSuggestion(commits, 0)
	if n := utf8.RuneCountInString(got); n > DefaultMaxLen {
		t.Errorf("sugestão com %d caracteres, limite %d", n, DefaultMaxLen)
	}
	if !strings.HasSuffix(got, "…") {
		t.Errorf("sugestão cortada deveria terminar em reticências: %q", got)
	}
	if got := BuildSuggestion(commits[:1], 0); got != "ação número" {
		t.Errorf("sugestão curta não deveria ser cortada: %q", got)
	}
}

func TestCleanSubject(t *testing.T) {
	casos := map[string]string{
		"feat: x":                "x",
		"fix(calendario): y":     "y",
		"chore(deps)!: z":        "z",
		"FIX: maiusculas":        "maiusculas",
		"Pagamentos: tela nova":  "Pagamentos: tela nova",
		"sem prefixo nenhum":     "sem prefixo nenhum",
		"v1.2: versao com ponto": "v1.2: versao com ponto",
	}
	for in, want := range casos {
		if got := CleanSubject(in); got != want {
			t.Errorf("CleanSubject(%q) = %q, esperava %q", in, got, want)
		}
	}
}
