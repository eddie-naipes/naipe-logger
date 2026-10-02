package update

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const nomeInstalador = "TeamworkLogger-amd64-installer.exe"

// fakeGitHub simula a API de releases e o download dos assets num servidor
// TLS local. Os campos podem ser ajustados antes de cada teste.
type fakeGitHub struct {
	server        *httptest.Server
	tag           string
	prerelease    bool
	draft         bool
	installer     []byte
	sums          string // vazio = calculado do instalador
	semInstalador bool
	semSums       bool
	assetURLBase  string // vazio = o próprio servidor
	redirectTo    string // se definido, o download do instalador redireciona para cá
	status        int    // status da API (0 = 200)
	apiCalls      int32
}

func novoFakeGitHub(t *testing.T) *fakeGitHub {
	t.Helper()
	f := &fakeGitHub{tag: "v1.2.0", installer: bytes.Repeat([]byte("MZ-instalador-"), 50000)}
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/"+Repo+"/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&f.apiCalls, 1)
		if r.Header.Get("User-Agent") == "" {
			t.Error("a API do GitHub exige User-Agent")
		}
		if f.status != 0 {
			w.WriteHeader(f.status)
			return
		}
		base := f.assetURLBase
		if base == "" {
			base = f.server.URL
		}
		var assets []asset
		if !f.semInstalador {
			assets = append(assets, asset{Name: nomeInstalador, BrowserDownloadURL: base + "/download/" + nomeInstalador, Size: int64(len(f.installer))})
		}
		if !f.semSums {
			assets = append(assets, asset{Name: ChecksumsAsset, BrowserDownloadURL: base + "/download/" + ChecksumsAsset})
		}
		_ = json.NewEncoder(w).Encode(release{
			TagName: f.tag, Body: "## Novidades", HTMLURL: f.server.URL + "/releases/tag/" + f.tag,
			Draft: f.draft, Prerelease: f.prerelease, PublishedAt: "2026-09-30T12:00:00Z", Assets: assets,
		})
	})
	mux.HandleFunc("/download/"+nomeInstalador, func(w http.ResponseWriter, r *http.Request) {
		if f.redirectTo != "" {
			http.Redirect(w, r, f.redirectTo, http.StatusFound)
			return
		}
		_, _ = w.Write(f.installer)
	})
	mux.HandleFunc("/download/"+ChecksumsAsset, func(w http.ResponseWriter, r *http.Request) {
		sums := f.sums
		if sums == "" {
			sum := sha256.Sum256(f.installer)
			// Mesmo formato do build.yml: caminho com o diretório do job.
			sums = fmt.Sprintf("%s  windows/teamwork-logger.exe\n%s  windows/%s\n",
				strings.Repeat("0", 64), hex.EncodeToString(sum[:]), nomeInstalador)
		}
		_, _ = w.Write([]byte(sums))
	})
	f.server = httptest.NewTLSServer(mux)
	t.Cleanup(f.server.Close)
	return f
}

func (f *fakeGitHub) updater(t *testing.T, current, goos string) *Updater {
	t.Helper()
	return New(current, goos,
		WithAPIBaseURL(f.server.URL),
		WithHTTPClient(f.server.Client()),
		WithAllowedHosts("127.0.0.1"),
		WithTempDir(t.TempDir()),
	)
}

func TestCheckReleaseMaisNova(t *testing.T) {
	f := novoFakeGitHub(t)
	info, err := f.updater(t, "1.1.0", "windows").Check(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if !info.Available || info.LatestVersion != "1.2.0" || info.CurrentVersion != "1.1.0" {
		t.Errorf("info inesperada: %+v", info)
	}
	if !info.CanInstall {
		t.Error("no Windows, com instalador e checksum, deveria poder instalar")
	}
	if info.ReleaseNotes == "" || info.ReleaseURL == "" || info.PublishedAt == "" {
		t.Errorf("faltam notas/URL/data: %+v", info)
	}
}

func TestCheckReleaseIgualOuMaisVelha(t *testing.T) {
	f := novoFakeGitHub(t)
	for _, atual := range []string{"1.2.0", "v1.2.0", "1.3.0"} {
		info, err := f.updater(t, atual, "windows").Check(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if info.Available || info.CanInstall {
			t.Errorf("versão atual %s não deveria ver atualização: %+v", atual, info)
		}
	}
}

func TestCheckIgnoraPrereleaseERascunho(t *testing.T) {
	f := novoFakeGitHub(t)
	f.prerelease = true
	if info, _ := f.updater(t, "1.0.0", "windows").Check(t.Context()); info.Available {
		t.Error("pré-release não deveria ser oferecida")
	}
	f.prerelease, f.draft = false, true
	if info, _ := f.updater(t, "1.0.0", "windows").Check(t.Context()); info.Available {
		t.Error("rascunho não deveria ser oferecido")
	}
}

func TestCheckSemReleasesNaoEhErro(t *testing.T) {
	f := novoFakeGitHub(t)
	f.status = http.StatusNotFound
	info, err := f.updater(t, "1.0.0", "windows").Check(t.Context())
	if err != nil || info.Available {
		t.Errorf("sem releases: info=%+v err=%v", info, err)
	}
}

func TestCheckUsaCacheDeUmaHora(t *testing.T) {
	f := novoFakeGitHub(t)
	u := f.updater(t, "1.0.0", "windows")
	agora := time.Now()
	u.now = func() time.Time { return agora }

	_, _ = u.Check(t.Context())
	_, _ = u.Check(t.Context())
	if n := atomic.LoadInt32(&f.apiCalls); n != 1 {
		t.Errorf("esperava 1 consulta com cache, houve %d", n)
	}
	agora = agora.Add(checkCacheTTL + time.Minute)
	_, _ = u.Check(t.Context())
	if n := atomic.LoadInt32(&f.apiCalls); n != 2 {
		t.Errorf("cache vencido deveria consultar de novo (%d consultas)", n)
	}
	_, _ = u.Refresh(t.Context())
	if n := atomic.LoadInt32(&f.apiCalls); n != 3 {
		t.Errorf("Refresh deveria ignorar o cache (%d consultas)", n)
	}
}

func TestCheckLimiteDoGitHub(t *testing.T) {
	f := novoFakeGitHub(t)
	f.status = http.StatusForbidden
	if _, err := f.updater(t, "1.0.0", "windows").Check(t.Context()); err == nil {
		t.Error("403 deveria virar erro de limite")
	}
}

func TestVersaoDevChecaMasNaoInstala(t *testing.T) {
	f := novoFakeGitHub(t)
	u := f.updater(t, "1.0.0-dev", "windows")
	info, err := u.Check(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if !info.Available || info.CanInstall {
		t.Errorf("dev deveria ver a versão nova mas sem instalar: %+v", info)
	}
	if _, err := u.DownloadInstaller(t.Context(), nil); !errors.Is(err, ErrDevBuild) {
		t.Errorf("esperava ErrDevBuild, veio %v", err)
	}
}

func TestDownloadInstaladorVerificaChecksumEEmiteProgresso(t *testing.T) {
	f := novoFakeGitHub(t)
	var ultimos []Progress
	path, err := f.updater(t, "1.1.0", "windows").DownloadInstaller(t.Context(), func(p Progress) {
		ultimos = append(ultimos, p)
	})
	if err != nil {
		t.Fatalf("DownloadInstaller: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(data, f.installer) {
		t.Fatalf("instalador salvo difere do publicado (err=%v)", err)
	}
	if !strings.HasSuffix(path, nomeInstalador) {
		t.Errorf("nome do arquivo salvo: %s", path)
	}
	if len(ultimos) < 2 {
		t.Fatalf("esperava vários eventos de progresso, vieram %d", len(ultimos))
	}
	fim := ultimos[len(ultimos)-1]
	if fim.Received != int64(len(f.installer)) || fim.Total != int64(len(f.installer)) {
		t.Errorf("último progresso = %+v", fim)
	}
}

func TestDownloadRejeitaChecksumErrado(t *testing.T) {
	f := novoFakeGitHub(t)
	f.sums = strings.Repeat("a", 64) + "  windows/" + nomeInstalador + "\n"
	tmp := t.TempDir()
	u := New("1.1.0", "windows", WithAPIBaseURL(f.server.URL), WithHTTPClient(f.server.Client()),
		WithAllowedHosts("127.0.0.1"), WithTempDir(tmp))

	if _, err := u.DownloadInstaller(t.Context(), nil); err == nil || !strings.Contains(err.Error(), "checksum") {
		t.Fatalf("esperava erro de checksum, veio %v", err)
	}
	if entries, _ := os.ReadDir(tmp); len(entries) != 0 {
		t.Errorf("instalador rejeitado deveria ser apagado, sobrou %v", entries)
	}
}

func TestDownloadRejeitaSemChecksumDoInstalador(t *testing.T) {
	f := novoFakeGitHub(t)
	f.sums = strings.Repeat("a", 64) + "  windows/outro.exe\n"
	if _, err := f.updater(t, "1.1.0", "windows").DownloadInstaller(t.Context(), nil); err == nil {
		t.Fatal("instalador fora do SHA256SUMS.txt deveria ser recusado")
	}
}

func TestDownloadAssetsAusentes(t *testing.T) {
	f := novoFakeGitHub(t)
	f.semInstalador = true
	u := f.updater(t, "1.1.0", "windows")
	if info, _ := u.Check(t.Context()); info.CanInstall {
		t.Error("sem instalador não pode oferecer instalação")
	}
	if _, err := u.DownloadInstaller(t.Context(), nil); err == nil {
		t.Error("sem instalador deveria falhar")
	}

	f2 := novoFakeGitHub(t)
	f2.semSums = true
	if _, err := f2.updater(t, "1.1.0", "windows").DownloadInstaller(t.Context(), nil); err == nil {
		t.Error("sem SHA256SUMS.txt deveria falhar")
	}
}

func TestDownloadRejeitaHostNaoPermitido(t *testing.T) {
	f := novoFakeGitHub(t)
	f.assetURLBase = "https://malicioso.example.com"
	_, err := f.updater(t, "1.1.0", "windows").DownloadInstaller(t.Context(), nil)
	if err == nil || !strings.Contains(err.Error(), "não é permitido") {
		t.Fatalf("esperava bloqueio de host, veio %v", err)
	}
}

func TestDownloadRejeitaRedirecionamentoParaHostNaoPermitido(t *testing.T) {
	f := novoFakeGitHub(t)
	// Mesmo servidor, mas por "localhost": fora da lista permitida.
	f.redirectTo = strings.Replace(f.server.URL, "127.0.0.1", "localhost", 1) + "/download/" + nomeInstalador
	_, err := f.updater(t, "1.1.0", "windows").DownloadInstaller(t.Context(), nil)
	if err == nil || !strings.Contains(err.Error(), "não é permitido") {
		t.Fatalf("redirecionamento para host fora da lista deveria ser bloqueado, veio %v", err)
	}
}

func TestDownloadRejeitaHTTP(t *testing.T) {
	u := New("1.1.0", "windows", WithAPIBaseURL("http://127.0.0.1:1"), WithAllowedHosts("127.0.0.1"))
	if _, err := u.Check(t.Context()); err == nil || !strings.Contains(err.Error(), "https") {
		t.Fatalf("API em http deveria ser recusada, veio %v", err)
	}
}

func TestInstalacaoForaDoWindows(t *testing.T) {
	f := novoFakeGitHub(t)
	for _, goos := range []string{"darwin", "linux"} {
		u := f.updater(t, "1.1.0", goos)
		info, err := u.Check(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		if !info.Available || info.CanInstall {
			t.Errorf("%s: deveria avisar sem oferecer instalação: %+v", goos, info)
		}
		if _, err := u.DownloadInstaller(t.Context(), nil); !errors.Is(err, ErrInstallNotSupported) {
			t.Errorf("%s: esperava ErrInstallNotSupported, veio %v", goos, err)
		}
	}
}

func TestDownloadSemAtualizacao(t *testing.T) {
	f := novoFakeGitHub(t)
	if _, err := f.updater(t, "1.2.0", "windows").DownloadInstaller(t.Context(), nil); !errors.Is(err, ErrNoUpdate) {
		t.Errorf("esperava ErrNoUpdate, veio %v", err)
	}
}

func TestChecksumFor(t *testing.T) {
	sum := strings.Repeat("ab", 32)
	sums := "lixo\n" + sum + " *windows\\" + nomeInstalador + "\r\n"
	got, err := checksumFor(sums, nomeInstalador)
	if err != nil || got != sum {
		t.Errorf("checksumFor = %q, %v", got, err)
	}
	if _, err := checksumFor("zz  "+nomeInstalador, nomeInstalador); err == nil {
		t.Error("checksum malformado deveria falhar")
	}
}

func TestSafeReleaseURL(t *testing.T) {
	u := New("1.0.0", "windows")
	if _, err := u.SafeReleaseURL("https://github.com/" + Repo + "/releases/tag/v1.0.0"); err != nil {
		t.Errorf("URL do GitHub deveria passar: %v", err)
	}
	for _, ruim := range []string{"http://github.com/x", "https://evil.com/x", "javascript:alert(1)", "file:///C:/x"} {
		if _, err := u.SafeReleaseURL(ruim); err == nil {
			t.Errorf("%q deveria ser recusada", ruim)
		}
	}
}
