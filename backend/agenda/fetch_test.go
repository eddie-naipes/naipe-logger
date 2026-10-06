package agenda

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNormalizeURLExigeHttpsEConverteWebcal(t *testing.T) {
	got, err := NormalizeURL("  webcal://agenda.exemplo.com/privado/abc/basic.ics ")
	if err != nil || got != "https://agenda.exemplo.com/privado/abc/basic.ics" {
		t.Errorf("webcal = %q, %v", got, err)
	}
	for _, ruim := range []string{"http://agenda.exemplo.com/a.ics", "ftp://x/a.ics", "agenda.ics", "", "https://"} {
		if _, err := NormalizeURL(ruim); err == nil {
			t.Errorf("%q aceito", ruim)
		}
	}
}

func TestMaskURLMostraSoHostEFinal(t *testing.T) {
	got := MaskURL("https://agenda.exemplo.com/calendar/ical/segredo123456/private-abcdef/basic.ics")
	if got != "agenda.exemplo.com…ic.ics" {
		t.Errorf("MaskURL = %q", got)
	}
	if strings.Contains(got, "segredo") {
		t.Error("máscara vazou o segredo")
	}
}

const icsMinimo = "BEGIN:VCALENDAR\r\nVERSION:2.0\r\nEND:VCALENDAR\r\n"

func TestFetchBaixaAgenda(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(icsMinimo))
	}))
	defer srv.Close()

	data, err := Fetcher{Client: srv.Client()}.Fetch(context.Background(), srv.URL+"/privado/segredo.ics")
	if err != nil || string(data) != icsMinimo {
		t.Fatalf("Fetch = %q, %v", data, err)
	}
}

func TestFetchErroNaoExpoeAURL(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	_, err := Fetcher{Client: srv.Client()}.Fetch(context.Background(), srv.URL+"/privado/segredoXYZ/a.ics")
	if err == nil || strings.Contains(err.Error(), "segredoXYZ") || !strings.Contains(err.Error(), "404") {
		t.Errorf("erro = %v", err)
	}

	// Erro de rede (servidor fechado) também não pode citar a URL.
	srv.Close()
	_, err = Fetcher{Client: srv.Client()}.Fetch(context.Background(), srv.URL+"/privado/segredoXYZ/a.ics")
	if err == nil || strings.Contains(err.Error(), "segredoXYZ") {
		t.Errorf("erro de rede = %v", err)
	}
}

func TestFetchRecusaRedirecionamentoParaHttp(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "http://exemplo.invalid/a.ics", http.StatusFound)
	}))
	defer srv.Close()
	if _, err := (Fetcher{Client: srv.Client()}).Fetch(context.Background(), srv.URL+"/a.ics"); err == nil {
		t.Error("redirecionamento para http aceito")
	}
}

func TestFetchLimitaRedirecionamentos(t *testing.T) {
	var srv *httptest.Server
	srv = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, srv.URL+r.URL.Path+"x", http.StatusFound)
	}))
	defer srv.Close()
	if _, err := (Fetcher{Client: srv.Client()}).Fetch(context.Background(), srv.URL+"/a"); err == nil {
		t.Error("redirecionamentos infinitos aceitos")
	}
}

func TestFetchLimitaTamanho(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(strings.Repeat("x", 2048)))
	}))
	defer srv.Close()
	if _, err := (Fetcher{Client: srv.Client(), MaxBytes: 1024}).Fetch(context.Background(), srv.URL+"/a.ics"); err == nil {
		t.Error("download acima do limite aceito")
	}
}

func TestFetchRecusaHttp(t *testing.T) {
	if _, err := (Fetcher{}).Fetch(context.Background(), "http://127.0.0.1:1/a.ics"); err == nil {
		t.Error("http aceito")
	}
}

func TestImportedStoreMarcaDesmarcaEPersiste(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "agenda-importados.json")
	s, err := OpenImportedStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Mark([]ImportedRecord{{Key: "a", Date: "2026-10-05", TaskID: 1, EntryID: 7}, {Key: "b"}, {Key: ""}}); err != nil {
		t.Fatal(err)
	}
	if err := s.Unmark([]string{"b"}); err != nil {
		t.Fatal(err)
	}

	s2, err := OpenImportedStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if !s2.Has("a") || s2.Has("b") {
		t.Errorf("registro reaberto: a=%v b=%v", s2.Has("a"), s2.Has("b"))
	}
}

func TestImportedStoreDescartaAntigosEToleraArquivoCorrompido(t *testing.T) {
	path := filepath.Join(t.TempDir(), "importados.json")
	if err := os.WriteFile(path, []byte("{lixo"), 0o600); err != nil {
		t.Fatal(err)
	}
	s, err := OpenImportedStore(path)
	if err != nil {
		t.Fatal(err)
	}
	agora := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return agora.AddDate(-2, 0, 0) }
	_ = s.Mark([]ImportedRecord{{Key: "velho"}})
	s.now = func() time.Time { return agora }
	_ = s.Mark([]ImportedRecord{{Key: "novo"}})
	if s.Has("velho") || !s.Has("novo") {
		t.Error("marcação antiga deveria ter sido descartada")
	}
}
