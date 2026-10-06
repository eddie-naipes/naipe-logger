package autostart

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Nenhum teste toca no registro real, em ~/Library ou em ~/.config: o
// registro é um mapa e as pastas são t.TempDir().

type fakeStore struct {
	values map[string]string
	sets   int
}

func newFakeStore() *fakeStore { return &fakeStore{values: map[string]string{}} }

func (f *fakeStore) GetString(name string) (string, bool, error) {
	v, ok := f.values[name]
	return v, ok, nil
}

func (f *fakeStore) SetString(name, value string) error {
	f.sets++
	f.values[name] = value
	return nil
}

func (f *fakeStore) Delete(name string) error {
	delete(f.values, name)
	return nil
}

func exeFixo(path string) func() (string, error) {
	return func() (string, error) { return path, nil }
}

func TestHasMinimizedFlag(t *testing.T) {
	if !HasMinimizedFlag([]string{"-x", "--minimized"}) || HasMinimizedFlag([]string{"--minimizedx"}) || HasMinimizedFlag(nil) {
		t.Error("HasMinimizedFlag errado")
	}
}

func TestRunKeyHabilitaComCaminhoComEspacosEDesabilita(t *testing.T) {
	store := newFakeStore()
	m := Manager{
		Launcher:   RunKey{Store: store},
		Executable: exeFixo(`C:\Users\Fulano de Tal\AppData\Local\Programs\Teamwork Logger\teamwork-logger.exe`),
		Version:    "3.2.0",
	}
	st, err := m.SetEnabled(true)
	if err != nil || !st.Enabled || !st.Supported {
		t.Fatalf("SetEnabled(true) = %+v, %v", st, err)
	}
	want := `"C:\Users\Fulano de Tal\AppData\Local\Programs\Teamwork Logger\teamwork-logger.exe" --minimized`
	if got := store.values[RunValueName]; got != want {
		t.Errorf("valor = %s\nesperado %s", got, want)
	}

	st, err = m.SetEnabled(false)
	if err != nil || st.Enabled {
		t.Fatalf("SetEnabled(false) = %+v, %v", st, err)
	}
	if _, ok := store.values[RunValueName]; ok {
		t.Error("valor não removido")
	}
	// Desabilitar de novo não é erro.
	if _, err := m.SetEnabled(false); err != nil {
		t.Errorf("desabilitar duas vezes: %v", err)
	}
}

func TestRunKeyIdempotente(t *testing.T) {
	store := newFakeStore()
	m := Manager{Launcher: RunKey{Store: store}, Executable: exeFixo(`C:\app.exe`), Version: "3.2.0"}
	for i := 0; i < 3; i++ {
		if _, err := m.SetEnabled(true); err != nil {
			t.Fatal(err)
		}
	}
	if store.sets != 1 || len(store.values) != 1 {
		t.Errorf("gravações = %d, valores = %v", store.sets, store.values)
	}
	// Executável mudou de lugar: reescreve.
	m.Executable = exeFixo(`D:\novo\app.exe`)
	_, _ = m.SetEnabled(true)
	if store.sets != 2 || !strings.HasPrefix(store.values[RunValueName], `"D:\novo\app.exe"`) {
		t.Errorf("valor = %v", store.values)
	}
}

func TestVersaoDevNaoHabilita(t *testing.T) {
	store := newFakeStore()
	for _, m := range []Manager{
		{Launcher: RunKey{Store: store}, Executable: exeFixo(`C:\tmp\app.exe`), Version: "3.2.0-dev"},
		{Launcher: RunKey{Store: store}, Executable: exeFixo(`C:\tmp\app.exe`), Version: "0.0.0-dev"},
		{Launcher: RunKey{Store: store}, Executable: exeFixo(`C:\tmp\app.exe`), Version: "3.2.0", Dev: true},
	} {
		st := m.Status()
		if st.Supported || !strings.Contains(st.Reason, "desenvolvimento") {
			t.Errorf("Status(%q) = %+v", m.Version, st)
		}
		if _, err := m.SetEnabled(true); err == nil {
			t.Errorf("SetEnabled em %q deveria falhar", m.Version)
		}
	}
	if len(store.values) != 0 {
		t.Errorf("registro alterado: %v", store.values)
	}
}

func TestSemLauncherNaoESuportado(t *testing.T) {
	st := Manager{Version: "3.2.0"}.Status()
	if st.Supported || st.Reason == "" {
		t.Errorf("Status = %+v", st)
	}
}

func TestErroAoDescobrirExecutavel(t *testing.T) {
	m := Manager{Launcher: RunKey{Store: newFakeStore()}, Version: "3.2.0",
		Executable: func() (string, error) { return "", errors.New("sem executável") }}
	if _, err := m.SetEnabled(true); err == nil {
		t.Error("esperava erro")
	}
}

func TestLaunchAgentGravaPlistComRunAtLoad(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "LaunchAgents")
	m := Manager{
		Launcher:   NewLaunchAgent(dir, BundleID),
		Executable: exeFixo("/Applications/Teamwork Logger.app/Contents/MacOS/teamwork-logger"),
		Version:    "3.2.0",
	}
	if st, err := m.SetEnabled(true); err != nil || !st.Enabled {
		t.Fatalf("SetEnabled = %+v, %v", st, err)
	}
	data, err := os.ReadFile(filepath.Join(dir, BundleID+".plist"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	for _, trecho := range []string{
		"<string>" + BundleID + "</string>",
		"<string>/Applications/Teamwork Logger.app/Contents/MacOS/teamwork-logger</string>",
		"<string>--minimized</string>",
		"<key>RunAtLoad</key>\n\t<true/>",
	} {
		if !strings.Contains(s, trecho) {
			t.Errorf("plist sem %q:\n%s", trecho, s)
		}
	}
	// Idempotente e desabilitável.
	if _, err := m.SetEnabled(true); err != nil {
		t.Fatal(err)
	}
	if st, err := m.SetEnabled(false); err != nil || st.Enabled {
		t.Fatalf("SetEnabled(false) = %+v, %v", st, err)
	}
	if _, err := os.Stat(filepath.Join(dir, BundleID+".plist")); !errors.Is(err, os.ErrNotExist) {
		t.Error("plist não removido")
	}
}

func TestLaunchAgentEscapaXML(t *testing.T) {
	s := string(launchAgentPlist("x", "/tmp/a&b<c>/app", nil))
	if !strings.Contains(s, "/tmp/a&amp;b&lt;c&gt;/app") {
		t.Errorf("caminho não escapado:\n%s", s)
	}
}

func TestDesktopEntryGravaEDesabilita(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "autostart")
	m := Manager{
		Launcher:   NewDesktopEntry(dir),
		Executable: exeFixo(`/home/fulano/Meus Apps/teamwork-logger`),
		Version:    "3.2.0",
	}
	if st, err := m.SetEnabled(true); err != nil || !st.Enabled {
		t.Fatalf("SetEnabled = %+v, %v", st, err)
	}
	data, err := os.ReadFile(filepath.Join(dir, DesktopFileName))
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if !strings.Contains(s, `Exec="/home/fulano/Meus Apps/teamwork-logger" --minimized`+"\n") ||
		!strings.Contains(s, "X-GNOME-Autostart-enabled=true") || !strings.HasPrefix(s, "[Desktop Entry]\n") {
		t.Errorf("entrada:\n%s", s)
	}
	if st, err := m.SetEnabled(false); err != nil || st.Enabled {
		t.Fatalf("SetEnabled(false) = %+v, %v", st, err)
	}
}

func TestDesktopExecArgEscapa(t *testing.T) {
	casos := map[string]string{
		"/usr/bin/app":         "/usr/bin/app",
		"/opt/meu app":         `"/opt/meu app"`,
		`/opt/a"b $c`:          `"/opt/a\"b \$c"`,
		"/opt/100%":            "/opt/100%%",
		`/opt/back\slash x`:    `"/opt/back\\slash x"`,
		"/opt/crase`x` espaço": "\"/opt/crase\\`x\\` espaço\"",
	}
	for in, want := range casos {
		if got := desktopExecArg(in); got != want {
			t.Errorf("desktopExecArg(%q) = %s; esperava %s", in, got, want)
		}
	}
}

func TestWindowsCommandLineCitaArgumentosComEspaco(t *testing.T) {
	got := WindowsCommandLine(`C:\a b\app.exe`, []string{"--minimized", "x y"})
	if got != `"C:\a b\app.exe" --minimized "x y"` {
		t.Errorf("linha = %s", got)
	}
}
