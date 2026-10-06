// Package gitlog sugere a descrição de um lançamento a partir dos commits que
// o usuário fez num dia, lendo `git log` dos repositórios configurados.
//
// O git é sempre executado diretamente (exec.CommandContext com argumentos em
// array, nunca por um shell), a partir do PATH e com timeout por repositório.
// Nada é escrito nos repositórios.
package gitlog

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	// DefaultTimeout limita cada chamada ao git por repositório.
	DefaultTimeout = 5 * time.Second
	// DefaultMaxLen é o tamanho máximo, em caracteres, da sugestão.
	DefaultMaxLen = 250

	fieldSep  = "\x1f"
	recordSep = "\x1e"
)

// ErrGitNaoEncontrado indica que o executável git não está no PATH.
var ErrGitNaoEncontrado = errors.New("git não encontrado no PATH; instale o Git para usar a sugestão por commits")

// Commit é um commit do dia, já filtrado por autor.
type Commit struct {
	// Repo é o nome da pasta do repositório (exibição e agrupamento).
	Repo string `json:"repo"`
	// Hash é o hash curto.
	Hash string `json:"hash"`
	// Time é o horário do commit (data do autor) no fuso local, "HH:MM".
	Time    string `json:"time"`
	Subject string `json:"subject"`
}

// Result é a resposta de Suggest.
type Result struct {
	Suggestion string   `json:"suggestion"`
	Commits    []Commit `json:"commits"`
	// Warnings descreve repositórios ignorados (caminho inválido, sem e-mail
	// de autor, erro do git); não impedem a sugestão dos demais.
	Warnings []string `json:"warnings"`
}

// Options configura Suggest.
type Options struct {
	Repositories []string
	// AuthorEmail filtra os commits; vazio usa `git config user.email` de
	// cada repositório.
	AuthorEmail string
	Timeout     time.Duration
	MaxLen      int
}

// lookPath é variável para que os testes simulem o git ausente.
var lookPath = exec.LookPath

// CheckGit confirma que o git está disponível.
func CheckGit() (string, error) {
	path, err := lookPath("git")
	if err != nil {
		return "", ErrGitNaoEncontrado
	}
	return path, nil
}

// Suggest lista os commits do dia (YYYY-MM-DD, fuso local) em cada repositório
// e monta a sugestão de descrição.
func Suggest(ctx context.Context, day string, opts Options) (Result, error) {
	start, err := time.ParseInLocation("2006-01-02", day, time.Local)
	if err != nil {
		return Result{}, fmt.Errorf("data inválida %q: use AAAA-MM-DD", day)
	}
	gitPath, err := CheckGit()
	if err != nil {
		return Result{}, err
	}
	if opts.Timeout <= 0 {
		opts.Timeout = DefaultTimeout
	}
	if opts.MaxLen <= 0 {
		opts.MaxLen = DefaultMaxLen
	}

	result := Result{Commits: []Commit{}, Warnings: []string{}}
	end := start.AddDate(0, 0, 1)

	for _, repo := range opts.Repositories {
		repo = strings.TrimSpace(repo)
		if repo == "" {
			continue
		}
		commits, err := repoCommits(ctx, gitPath, repo, start, end, opts)
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("%s: %v", repo, err))
			continue
		}
		result.Commits = append(result.Commits, commits...)
	}

	sort.SliceStable(result.Commits, func(i, j int) bool {
		return result.Commits[i].Time < result.Commits[j].Time
	})
	result.Suggestion = BuildSuggestion(result.Commits, opts.MaxLen)
	return result, nil
}

// ValidateRepo confirma que path existe e está dentro de um repositório git.
func ValidateRepo(ctx context.Context, path string) error {
	gitPath, err := CheckGit()
	if err != nil {
		return err
	}
	return validateRepo(ctx, gitPath, path, DefaultTimeout)
}

func validateRepo(ctx context.Context, gitPath, path string, timeout time.Duration) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("pasta não encontrada")
	}
	if !info.IsDir() {
		return fmt.Errorf("não é uma pasta")
	}
	out, err := runGit(ctx, gitPath, path, timeout, "rev-parse", "--is-inside-work-tree")
	if err != nil || strings.TrimSpace(out) != "true" {
		return fmt.Errorf("não é um repositório git")
	}
	return nil
}

func repoCommits(ctx context.Context, gitPath, repo string, start, end time.Time, opts Options) ([]Commit, error) {
	if err := validateRepo(ctx, gitPath, repo, opts.Timeout); err != nil {
		return nil, err
	}

	email := strings.TrimSpace(opts.AuthorEmail)
	if email == "" {
		out, err := runGit(ctx, gitPath, repo, opts.Timeout, "config", "user.email")
		email = strings.TrimSpace(out)
		if err != nil || email == "" {
			return nil, fmt.Errorf("sem e-mail de autor (configure git user.email ou o e-mail na integração)")
		}
	}

	out, err := runGit(ctx, gitPath, repo, opts.Timeout,
		"log", "--branches", "--no-merges", "--fixed-strings", "--regexp-ignore-case",
		"--author="+email,
		"--since="+start.Format(time.RFC3339),
		"--until="+end.Add(-time.Second).Format(time.RFC3339),
		"--pretty=format:%h"+fieldSep+"%aI"+fieldSep+"%ae"+fieldSep+"%s"+recordSep,
	)
	if err != nil {
		return nil, err
	}

	name := filepath.Base(filepath.Clean(repo))
	commits := []Commit{}
	for _, record := range strings.Split(out, recordSep) {
		fields := strings.Split(strings.TrimLeft(record, "\r\n"), fieldSep)
		if len(fields) != 4 {
			continue
		}
		// --author casa por substring em "Nome <email>"; confirma o e-mail exato.
		if !strings.EqualFold(strings.TrimSpace(fields[2]), email) {
			continue
		}
		when, err := time.Parse(time.RFC3339, fields[1])
		if err != nil {
			continue
		}
		when = when.In(time.Local)
		if when.Before(start) || !when.Before(end) {
			// Data do autor fora do dia (commit reescrito depois, p. ex.).
			continue
		}
		commits = append(commits, Commit{
			Repo:    name,
			Hash:    fields[0],
			Time:    when.Format("15:04"),
			Subject: strings.TrimSpace(fields[3]),
		})
	}
	return commits, nil
}

func runGit(ctx context.Context, gitPath, dir string, timeout time.Duration, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	full := append([]string{"-C", dir}, args...)
	cmd := exec.CommandContext(ctx, gitPath, full...)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "GIT_OPTIONAL_LOCKS=0")
	hideWindow(cmd)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("git demorou mais de %s", timeout)
		}
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("git %s: %s", args[0], firstLine(msg))
	}
	return stdout.String(), nil
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	return s
}

// reConventional casa o prefixo de Conventional Commits com os tipos usuais:
// "feat: ", "fix(api)!: ", "chore(deps): ". Outras palavras seguidas de dois
// pontos ("Pagamentos: ...") fazem parte do texto e ficam.
var reConventional = regexp.MustCompile(`(?i)^(feat|fix|chore|docs|style|refactor|perf|test|tests|build|ci|revert)(\([^)]*\))?!?:\s+`)

// CleanSubject remove o prefixo de Conventional Commits do assunto. O tipo e
// o escopo ("feat(api):") servem ao histórico do repositório, não à descrição
// do lançamento de horas, que deve ler como frase.
func CleanSubject(subject string) string {
	return strings.TrimSpace(reConventional.ReplaceAllString(strings.TrimSpace(subject), ""))
}

// BuildSuggestion junta os assuntos (sem prefixo Conventional Commits e sem
// repetições) com "; ". Com commits de mais de um repositório, agrupa por
// repositório: "repo-a: x; y | repo-b: z". O resultado tem no máximo maxLen
// caracteres (DefaultMaxLen se maxLen <= 0), com reticências quando cortado.
func BuildSuggestion(commits []Commit, maxLen int) string {
	if maxLen <= 0 {
		maxLen = DefaultMaxLen
	}

	var repos []string
	byRepo := map[string][]string{}
	seen := map[string]bool{}
	for _, c := range commits {
		subject := CleanSubject(c.Subject)
		key := strings.ToLower(subject)
		if subject == "" || seen[key] {
			continue
		}
		seen[key] = true
		if _, ok := byRepo[c.Repo]; !ok {
			repos = append(repos, c.Repo)
		}
		byRepo[c.Repo] = append(byRepo[c.Repo], subject)
	}

	var text string
	switch len(repos) {
	case 0:
		return ""
	case 1:
		text = strings.Join(byRepo[repos[0]], "; ")
	default:
		parts := make([]string, 0, len(repos))
		for _, r := range repos {
			parts = append(parts, r+": "+strings.Join(byRepo[r], "; "))
		}
		text = strings.Join(parts, " | ")
	}
	return truncate(text, maxLen)
}

func truncate(s string, maxLen int) string {
	if utf8.RuneCountInString(s) <= maxLen {
		return s
	}
	runes := []rune(s)
	return strings.TrimRight(string(runes[:maxLen-1]), " ;|") + "…"
}
