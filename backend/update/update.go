// Package update verifica e baixa novas versões publicadas nas GitHub
// Releases do projeto. Não há servidor próprio nem custo: a API pública do
// GitHub informa a última release e os assets são baixados de lá.
//
// Regras de segurança (ver CLAUDE.md):
//   - toda URL usada (API, download e cada redirecionamento) precisa ser https
//     e de um host da lista permitida;
//   - o instalador só é aceito se o SHA-256 bater com o SHA256SUMS.txt da
//     MESMA release;
//   - downloads têm tamanho máximo, para um asset adulterado não encher o disco;
//   - versões de pré-release em execução (ex.: "1.2.0-dev", builds de `wails
//     dev`) podem checar, mas nunca instalar.
package update

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	// DefaultAPIBaseURL e Repo apontam para as releases públicas do projeto.
	DefaultAPIBaseURL = "https://api.github.com"
	Repo              = "eddie-naipes/naipe-logger"

	// ChecksumsAsset é publicado pelo job create-release do build.yml com o
	// sha256sum de todos os artefatos.
	ChecksumsAsset = "SHA256SUMS.txt"
	// InstallerSuffix identifica o instalador NSIS por usuário
	// (TeamworkLogger-amd64-installer.exe).
	InstallerSuffix = "-installer.exe"

	checkCacheTTL     = time.Hour
	maxAPIResponse    = 2 << 20   // 2 MB
	maxChecksumsSize  = 64 << 10  // 64 KB
	maxInstallerSize  = 300 << 20 // 300 MB
	progressEveryByte = 256 << 10 // emite progresso a cada 256 KB
	userAgent         = "naipe-logger-updater"
)

// DefaultAllowedHosts são os únicos hosts de onde a API e os assets podem vir.
// github.com serve o browser_download_url, que redireciona para
// objects.githubusercontent.com ou (desde 2025) release-assets.githubusercontent.com.
var DefaultAllowedHosts = []string{
	"api.github.com",
	"github.com",
	"objects.githubusercontent.com",
	"release-assets.githubusercontent.com",
}

var (
	// ErrInstallNotSupported é devolvido fora do Windows: lá o usuário baixa a
	// versão nova pela página da release.
	ErrInstallNotSupported = errors.New("a instalação automática só está disponível no Windows; abra a página da release para baixar a nova versão")
	// ErrNoUpdate indica que não há versão mais nova para instalar.
	ErrNoUpdate = errors.New("nenhuma atualização disponível")
	// ErrDevBuild impede que um build de desenvolvimento seja substituído.
	ErrDevBuild = errors.New("esta é uma versão de desenvolvimento; a instalação automática está desativada")
)

// Info é o resultado de uma verificação, no formato que o frontend consome.
type Info struct {
	Available      bool   `json:"available"`
	CurrentVersion string `json:"currentVersion"`
	LatestVersion  string `json:"latestVersion"`
	ReleaseNotes   string `json:"releaseNotes"`
	ReleaseURL     string `json:"releaseUrl"`
	PublishedAt    string `json:"publishedAt"`
	// CanInstall diz se DownloadAndInstallUpdate funcionaria aqui: Windows,
	// versão atual final (não -dev) e instalador presente na release. Quando
	// false, o frontend deve oferecer só "abrir página da release".
	CanInstall bool `json:"canInstall"`
}

// Progress é o payload do evento update:progress.
type Progress struct {
	Received int64 `json:"received"`
	Total    int64 `json:"total"`
}

type release struct {
	TagName     string  `json:"tag_name"`
	Body        string  `json:"body"`
	HTMLURL     string  `json:"html_url"`
	Draft       bool    `json:"draft"`
	Prerelease  bool    `json:"prerelease"`
	PublishedAt string  `json:"published_at"`
	Assets      []asset `json:"assets"`
}

type asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

// Updater consulta as releases. Seguro para uso concorrente.
type Updater struct {
	currentVersion string
	apiBaseURL     string
	allowedHosts   map[string]bool
	client         *http.Client
	goos           string
	tempDir        string // vazio = os.TempDir()
	now            func() time.Time

	mu        sync.Mutex
	cached    *release
	cachedErr error
	cachedAt  time.Time
}

// Option ajusta um Updater; usado pelos testes para apontar para um
// httptest.Server.
type Option func(*Updater)

// WithAPIBaseURL troca https://api.github.com.
func WithAPIBaseURL(base string) Option {
	return func(u *Updater) { u.apiBaseURL = strings.TrimRight(base, "/") }
}

// WithHTTPClient troca o cliente HTTP (o CheckRedirect é sempre reinstalado).
func WithHTTPClient(c *http.Client) Option {
	return func(u *Updater) {
		clone := *c
		u.client = &clone
	}
}

// WithAllowedHosts troca a lista de hosts permitidos.
func WithAllowedHosts(hosts ...string) Option {
	return func(u *Updater) {
		u.allowedHosts = make(map[string]bool, len(hosts))
		for _, h := range hosts {
			u.allowedHosts[strings.ToLower(h)] = true
		}
	}
}

// WithGOOS simula outro sistema operacional.
func WithGOOS(goos string) Option { return func(u *Updater) { u.goos = goos } }

// WithTempDir define onde o instalador é salvo.
func WithTempDir(dir string) Option { return func(u *Updater) { u.tempDir = dir } }

// New cria um Updater para a versão em execução.
func New(currentVersion, goos string, opts ...Option) *Updater {
	u := &Updater{
		currentVersion: currentVersion,
		apiBaseURL:     DefaultAPIBaseURL,
		client:         &http.Client{Timeout: 10 * time.Minute},
		goos:           goos,
		now:            time.Now,
	}
	WithAllowedHosts(DefaultAllowedHosts...)(u)
	for _, opt := range opts {
		opt(u)
	}
	// Cada salto de redirecionamento passa pela mesma verificação de host: o
	// browser_download_url é github.com, mas o arquivo vem de outro domínio.
	u.client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return errors.New("redirecionamentos demais")
		}
		return u.checkURL(req.URL)
	}
	return u
}

// CurrentVersion devolve a versão em execução.
func (u *Updater) CurrentVersion() string { return u.currentVersion }

func (u *Updater) checkURL(target *url.URL) error {
	if target == nil || target.Scheme != "https" {
		return fmt.Errorf("download bloqueado: só https é permitido (%v)", target)
	}
	if !u.allowedHosts[strings.ToLower(target.Hostname())] {
		return fmt.Errorf("download bloqueado: host %q não é permitido", target.Hostname())
	}
	return nil
}

func (u *Updater) checkRawURL(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("URL inválida %q: %v", raw, err)
	}
	return u.checkURL(parsed)
}

// Check consulta a última release (com cache de 1h, inclusive de falhas, para
// não estourar o limite de 60 requisições/h da API anônima do GitHub).
func (u *Updater) Check(ctx context.Context) (Info, error) {
	rel, err := u.latestRelease(ctx, false)
	if err != nil {
		return Info{CurrentVersion: u.currentVersion}, err
	}
	return u.infoFor(rel)
}

// Refresh ignora o cache (botão "verificar agora").
func (u *Updater) Refresh(ctx context.Context) (Info, error) {
	rel, err := u.latestRelease(ctx, true)
	if err != nil {
		return Info{CurrentVersion: u.currentVersion}, err
	}
	return u.infoFor(rel)
}

func (u *Updater) infoFor(rel *release) (Info, error) {
	info := Info{CurrentVersion: u.currentVersion}
	// Sem release publicada (404), rascunho ou pré-release: nada a oferecer.
	if rel == nil || rel.Draft || rel.Prerelease {
		return info, nil
	}

	latest, err := ParseVersion(rel.TagName)
	if err != nil {
		return info, fmt.Errorf("release com tag inesperada: %v", err)
	}
	current, err := ParseVersion(u.currentVersion)
	if err != nil {
		return info, fmt.Errorf("versão atual inválida: %v", err)
	}

	info.LatestVersion = latest.String()
	info.ReleaseNotes = rel.Body
	info.PublishedAt = rel.PublishedAt
	if u.checkRawURL(rel.HTMLURL) == nil {
		info.ReleaseURL = rel.HTMLURL
	}
	info.Available = latest.Compare(current) > 0
	_, installerErr := findInstaller(rel)
	info.CanInstall = info.Available && u.goos == "windows" && !current.IsPrerelease() && installerErr == nil
	return info, nil
}

func (u *Updater) latestRelease(ctx context.Context, force bool) (*release, error) {
	u.mu.Lock()
	defer u.mu.Unlock()

	if !force && !u.cachedAt.IsZero() && u.now().Sub(u.cachedAt) < checkCacheTTL {
		return u.cached, u.cachedErr
	}

	rel, err := u.fetchLatest(ctx)
	// Cancelamento não é resultado: não deve ficar uma hora em cache.
	if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
		u.cached, u.cachedErr, u.cachedAt = rel, err, u.now()
	}
	return rel, err
}

func (u *Updater) fetchLatest(ctx context.Context) (*release, error) {
	endpoint := fmt.Sprintf("%s/repos/%s/releases/latest", u.apiBaseURL, Repo)
	if err := u.checkRawURL(endpoint); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := u.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("erro ao consultar atualizações: %v", err)
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusNotFound:
		return nil, nil
	case resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests:
		return nil, errors.New("limite de consultas ao GitHub atingido; tente novamente mais tarde")
	case resp.StatusCode != http.StatusOK:
		return nil, fmt.Errorf("GitHub respondeu %d ao consultar atualizações", resp.StatusCode)
	}

	var rel release
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxAPIResponse)).Decode(&rel); err != nil {
		return nil, fmt.Errorf("resposta inválida do GitHub: %v", err)
	}
	return &rel, nil
}

func findInstaller(rel *release) (asset, error) {
	for _, a := range rel.Assets {
		if strings.HasSuffix(strings.ToLower(a.Name), InstallerSuffix) {
			return a, nil
		}
	}
	return asset{}, errors.New("a release não tem instalador para Windows")
}

func findAsset(rel *release, name string) (asset, error) {
	for _, a := range rel.Assets {
		if a.Name == name {
			return a, nil
		}
	}
	return asset{}, fmt.Errorf("a release não tem %s", name)
}

// DownloadInstaller baixa e verifica o instalador da última release,
// devolvendo o caminho do arquivo já conferido. Quem chama o executa.
func (u *Updater) DownloadInstaller(ctx context.Context, progress func(Progress)) (string, error) {
	if u.goos != "windows" {
		return "", ErrInstallNotSupported
	}
	current, err := ParseVersion(u.currentVersion)
	if err != nil {
		return "", fmt.Errorf("versão atual inválida: %v", err)
	}
	if current.IsPrerelease() {
		return "", ErrDevBuild
	}

	rel, err := u.latestRelease(ctx, false)
	if err != nil {
		return "", err
	}
	info, err := u.infoFor(rel)
	if err != nil {
		return "", err
	}
	if !info.Available {
		return "", ErrNoUpdate
	}

	installer, err := findInstaller(rel)
	if err != nil {
		return "", err
	}
	sums, err := findAsset(rel, ChecksumsAsset)
	if err != nil {
		return "", fmt.Errorf("%v; sem checksum o instalador não é aceito", err)
	}
	// O nome vira nome de arquivo local: nada de caminhos.
	if installer.Name != path.Base(installer.Name) || strings.ContainsAny(installer.Name, `/\:`) {
		return "", fmt.Errorf("nome de instalador inválido %q", installer.Name)
	}

	sumsData, err := u.download(ctx, sums.BrowserDownloadURL, maxChecksumsSize, io.Discard, nil, 0)
	if err != nil {
		return "", fmt.Errorf("erro ao baixar %s: %v", ChecksumsAsset, err)
	}
	expected, err := checksumFor(string(sumsData), installer.Name)
	if err != nil {
		return "", err
	}

	dir, err := os.MkdirTemp(u.tempDir, "teamwork-logger-update-")
	if err != nil {
		return "", fmt.Errorf("erro ao criar pasta temporária: %v", err)
	}
	dest := filepath.Join(dir, installer.Name)
	f, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0700)
	if err != nil {
		_ = os.RemoveAll(dir)
		return "", err
	}

	hasher := sha256.New()
	_, err = u.download(ctx, installer.BrowserDownloadURL, maxInstallerSize, io.MultiWriter(f, hasher), progress, installer.Size)
	closeErr := f.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		_ = os.RemoveAll(dir)
		return "", fmt.Errorf("erro ao baixar o instalador: %v", err)
	}

	if got := hex.EncodeToString(hasher.Sum(nil)); !strings.EqualFold(got, expected) {
		_ = os.RemoveAll(dir)
		return "", fmt.Errorf("checksum do instalador não confere (esperado %s, obtido %s); o arquivo foi descartado", expected, got)
	}
	return dest, nil
}

// download baixa rawURL para w. Se w for io.Discard, devolve o corpo (só para
// arquivos pequenos, como o SHA256SUMS.txt).
func (u *Updater) download(ctx context.Context, rawURL string, limit int64, w io.Writer, progress func(Progress), size int64) ([]byte, error) {
	if err := u.checkRawURL(rawURL); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/octet-stream")

	resp, err := u.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}

	total := resp.ContentLength
	if total <= 0 {
		total = size
	}
	if total > limit {
		return nil, fmt.Errorf("arquivo grande demais (%d bytes)", total)
	}

	// Lê um byte além do limite para detectar corpo maior que o anunciado.
	body := io.LimitReader(resp.Body, limit+1)
	if w == io.Discard {
		data, err := io.ReadAll(body)
		if err != nil {
			return nil, err
		}
		if int64(len(data)) > limit {
			return nil, errors.New("arquivo grande demais")
		}
		return data, nil
	}

	var received, lastEmit int64
	buf := make([]byte, 32<<10)
	for {
		n, readErr := body.Read(buf)
		if n > 0 {
			received += int64(n)
			if received > limit {
				return nil, errors.New("arquivo grande demais")
			}
			if _, err := w.Write(buf[:n]); err != nil {
				return nil, err
			}
			if progress != nil && received-lastEmit >= progressEveryByte {
				lastEmit = received
				progress(Progress{Received: received, Total: total})
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return nil, readErr
		}
	}
	if total > 0 && received != total {
		return nil, fmt.Errorf("download incompleto (%d de %d bytes)", received, total)
	}
	if progress != nil {
		progress(Progress{Received: received, Total: total})
	}
	return nil, nil
}

// checksumFor procura no SHA256SUMS.txt (formato do sha256sum: "<hex>  <arq>",
// com "*" opcional para modo binário) a linha do instalador. O build.yml grava
// caminhos como "windows/<nome>", então compara pelo nome base.
func checksumFor(sums, name string) (string, error) {
	for _, line := range strings.Split(sums, "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) != 2 {
			continue
		}
		file := strings.TrimPrefix(fields[1], "*")
		if path.Base(strings.ReplaceAll(file, `\`, "/")) != name {
			continue
		}
		sum := strings.ToLower(fields[0])
		if len(sum) != sha256.Size*2 {
			return "", fmt.Errorf("checksum malformado para %s", name)
		}
		if _, err := hex.DecodeString(sum); err != nil {
			return "", fmt.Errorf("checksum malformado para %s", name)
		}
		return sum, nil
	}
	return "", fmt.Errorf("%s não lista %s; o instalador não é aceito sem checksum", ChecksumsAsset, name)
}

// SafeReleaseURL devolve a URL da página da release se ela for de um host
// permitido, para abri-la no navegador sem confiar cegamente na resposta.
func (u *Updater) SafeReleaseURL(raw string) (string, error) {
	if err := u.checkRawURL(raw); err != nil {
		return "", err
	}
	return raw, nil
}

// ReleasesPageURL é a página de releases, usada quando não há URL específica.
func ReleasesPageURL() string {
	return "https://github.com/" + Repo + "/releases/latest"
}
