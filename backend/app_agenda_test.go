package backend

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"logTime-go/backend/agenda"
)

const icsDeTeste = "BEGIN:VCALENDAR\r\nVERSION:2.0\r\n" +
	"BEGIN:VEVENT\r\nUID:a@exemplo\r\nSUMMARY:Daily\r\nDTSTART:20261005T120000Z\r\nDTEND:20261005T121500Z\r\nEND:VEVENT\r\n" +
	"END:VCALENDAR\r\n"

func trocarSeletorDeArquivo(t *testing.T, escolha string) {
	t.Helper()
	original := openAgendaFileDialog
	openAgendaFileDialog = func(context.Context, runtime.OpenDialogOptions) (string, error) {
		return escolha, nil
	}
	t.Cleanup(func() { openAgendaFileDialog = original })
}

func TestImportAgendaFileCanceladoDevolveNil(t *testing.T) {
	trocarSeletorDeArquivo(t, "")
	info, err := (&App{}).ImportAgendaFile()
	if info != nil || err != nil {
		t.Errorf("ImportAgendaFile = %+v, %v", info, err)
	}
}

func TestImportAgendaFileCarregaParaASessao(t *testing.T) {
	path := filepath.Join(t.TempDir(), "minha agenda.ics")
	if err := os.WriteFile(path, []byte(icsDeTeste), 0o600); err != nil {
		t.Fatal(err)
	}
	trocarSeletorDeArquivo(t, path)
	app := &App{}
	info, err := app.ImportAgendaFile()
	if err != nil || info == nil || info.Name != "minha agenda.ics" || info.Events != 1 {
		t.Fatalf("ImportAgendaFile = %+v, %v", info, err)
	}
	if got := app.GetAgendaFile(); got == nil || got.Events != 1 {
		t.Errorf("GetAgendaFile = %+v", got)
	}
	app.ClearAgendaFile()
	if app.GetAgendaFile() != nil {
		t.Error("arquivo não descartado")
	}
}

func TestImportAgendaFileRecusaArquivoQueNaoEAgenda(t *testing.T) {
	path := filepath.Join(t.TempDir(), "x.ics")
	if err := os.WriteFile(path, []byte("não sou agenda"), 0o600); err != nil {
		t.Fatal(err)
	}
	trocarSeletorDeArquivo(t, path)
	if _, err := (&App{}).ImportAgendaFile(); err == nil {
		t.Error("arquivo inválido aceito")
	}
}

func TestMarkAgendaImportedUsaPastaInjetada(t *testing.T) {
	dir := t.TempDir()
	app := &App{agenda: agendaState{storeDir: dir}}
	if err := app.MarkAgendaImported([]agenda.ImportedRecord{{Key: "a@exemplo", TaskID: 1}}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "agenda-importados.json")); err != nil {
		t.Errorf("registro não gravado: %v", err)
	}
	if err := app.UnmarkAgendaImported([]string{"a@exemplo"}); err != nil {
		t.Fatal(err)
	}
	store, _ := app.importedStore()
	if store.Has("a@exemplo") {
		t.Error("marcação não removida")
	}
}

func TestAddAgendaCalendarRecusaHttpSemTocarNoCofre(t *testing.T) {
	original := storeAgendaURL
	storeAgendaURL = func(string, string) error {
		t.Error("o cofre não deveria ser usado")
		return nil
	}
	t.Cleanup(func() { storeAgendaURL = original })

	_, err := (&App{}).AddAgendaCalendar("Trabalho", "http://agenda.exemplo.com/privado-segredo/basic.ics")
	if err == nil || strings.Contains(err.Error(), "privado-segredo") {
		t.Errorf("erro = %v", err)
	}
}
