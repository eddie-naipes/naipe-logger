package backend

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"logTime-go/backend/update"
)

const instaladorTeste = "TeamworkLogger-amd64-installer.exe"

// servidorReleases publica a release v9.9.9 com instalador e SHA256SUMS.txt.
func servidorReleases(t *testing.T) *httptest.Server {
	t.Helper()
	conteudo := []byte(strings.Repeat("instalador", 1000))
	soma := sha256.Sum256(conteudo)

	var server *httptest.Server
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/"+update.Repo+"/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"tag_name":     "v9.9.9",
			"html_url":     server.URL + "/releases/tag/v9.9.9",
			"published_at": "2026-10-01T00:00:00Z",
			"assets": []map[string]any{
				{"name": instaladorTeste, "browser_download_url": server.URL + "/dl/inst", "size": len(conteudo)},
				{"name": update.ChecksumsAsset, "browser_download_url": server.URL + "/dl/sums"},
			},
		})
	})
	mux.HandleFunc("/dl/inst", func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write(conteudo) })
	mux.HandleFunc("/dl/sums", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(hex.EncodeToString(soma[:]) + "  windows/" + instaladorTeste + "\n"))
	})
	server = httptest.NewTLSServer(mux)
	t.Cleanup(server.Close)
	return server
}

// stubsDeRuntime troca as chamadas ao runtime do Wails e ao sistema por
// gravadores, restaurando-as ao final.
type stubs struct {
	mu        sync.Mutex
	eventos   []string
	executado string
	fechou    bool
	aberto    string
}

func instalaStubs(t *testing.T, sistema string) *stubs {
	t.Helper()
	s := &stubs{}
	origEmit, origOpen, origQuit, origStart, origGOOS := emitEvent, openBrowserURL, quitApp, startInstaller, goos
	emitEvent = func(_ context.Context, name string, _ ...interface{}) {
		s.mu.Lock()
		s.eventos = append(s.eventos, name)
		s.mu.Unlock()
	}
	openBrowserURL = func(_ context.Context, url string) { s.aberto = url }
	quitApp = func(context.Context) { s.fechou = true }
	startInstaller = func(path string) error { s.executado = path; return nil }
	goos = func() string { return sistema }
	t.Cleanup(func() {
		emitEvent, openBrowserURL, quitApp, startInstaller, goos = origEmit, origOpen, origQuit, origStart, origGOOS
	})
	return s
}

func appComUpdater(t *testing.T, server *httptest.Server, versao, sistema string) *App {
	t.Helper()
	return &App{version: versao, updater: update.New(versao, sistema,
		update.WithAPIBaseURL(server.URL),
		update.WithHTTPClient(server.Client()),
		update.WithAllowedHosts("127.0.0.1"),
		update.WithTempDir(t.TempDir()),
	)}
}

func TestDownloadAndInstallUpdateExecutaInstaladorEFecha(t *testing.T) {
	server := servidorReleases(t)
	s := instalaStubs(t, "windows")
	a := appComUpdater(t, server, "1.0.0", "windows")

	if err := a.DownloadAndInstallUpdate(); err != nil {
		t.Fatalf("DownloadAndInstallUpdate: %v", err)
	}
	if !strings.HasSuffix(s.executado, instaladorTeste) {
		t.Errorf("instalador executado: %q", s.executado)
	}
	if _, err := os.Stat(s.executado); err != nil {
		t.Errorf("instalador deveria existir no disco: %v", err)
	}
	if !s.fechou {
		t.Error("o app deveria fechar para o instalador substituir o executável")
	}
	if len(s.eventos) == 0 || s.eventos[len(s.eventos)-1] != EventUpdateProgress {
		t.Errorf("esperava eventos %s, vieram %v", EventUpdateProgress, s.eventos)
	}
}

func TestDownloadAndInstallUpdateForaDoWindows(t *testing.T) {
	server := servidorReleases(t)
	s := instalaStubs(t, "darwin")
	a := appComUpdater(t, server, "1.0.0", "darwin")

	err := a.DownloadAndInstallUpdate()
	if !errors.Is(err, update.ErrInstallNotSupported) {
		t.Fatalf("esperava ErrInstallNotSupported, veio %v", err)
	}
	if s.executado != "" || s.fechou {
		t.Error("fora do Windows nada pode ser executado nem fechado")
	}
}

func TestOpenReleasePageAbreURLDaRelease(t *testing.T) {
	server := servidorReleases(t)
	s := instalaStubs(t, "linux")
	a := appComUpdater(t, server, "1.0.0", "linux")

	if err := a.OpenReleasePage(); err != nil {
		t.Fatal(err)
	}
	if s.aberto != server.URL+"/releases/tag/v9.9.9" {
		t.Errorf("abriu %q", s.aberto)
	}
}

func TestNotifyIfUpdateAvailableEmiteEvento(t *testing.T) {
	server := servidorReleases(t)
	s := instalaStubs(t, "windows")

	appComUpdater(t, server, "1.0.0", "windows").notifyIfUpdateAvailable()
	if len(s.eventos) != 1 || s.eventos[0] != EventUpdateAvailable {
		t.Errorf("esperava %s, vieram %v", EventUpdateAvailable, s.eventos)
	}

	s.eventos = nil
	appComUpdater(t, server, "9.9.9", "windows").notifyIfUpdateAvailable()
	if len(s.eventos) != 0 {
		t.Errorf("sem versão nova não deveria emitir: %v", s.eventos)
	}
}

func TestCheckUpdatesOnStartupSemConfigNaoConsulta(t *testing.T) {
	s := instalaStubs(t, "windows")
	(&App{}).checkUpdatesOnStartup()
	if len(s.eventos) != 0 {
		t.Error("sem configuração não deveria consultar nem emitir")
	}
}
