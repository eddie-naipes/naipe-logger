package backend

import (
	"path/filepath"
	"testing"
)

func TestOpenLogsFolderAbreAPastaConfigurada(t *testing.T) {
	dir := t.TempDir()
	var aberto string
	original := openInFileManager
	openInFileManager = func(path string) error { aberto = path; return nil }
	t.Cleanup(func() { openInFileManager = original })

	a := &App{logsDir: dir}
	if a.GetLogsPath() != dir {
		t.Errorf("GetLogsPath = %q, esperava %q", a.GetLogsPath(), dir)
	}
	if err := a.OpenLogsFolder(); err != nil {
		t.Fatalf("OpenLogsFolder: %v", err)
	}
	if aberto != dir {
		t.Errorf("abriu %q, esperava %q", aberto, dir)
	}
}

func TestOpenLogsFolderRecusaPastaAusente(t *testing.T) {
	original := openInFileManager
	openInFileManager = func(string) error { t.Fatal("não deveria abrir nada"); return nil }
	t.Cleanup(func() { openInFileManager = original })

	if err := (&App{}).OpenLogsFolder(); err == nil {
		t.Error("esperava erro sem pasta de logs")
	}
	if err := (&App{logsDir: filepath.Join(t.TempDir(), "nao-existe")}).OpenLogsFolder(); err == nil {
		t.Error("esperava erro com pasta inexistente")
	}
}
