package backend

import (
	"testing"

	"logTime-go/backend/autostart"
)

type registroFalso map[string]string

func (r registroFalso) GetString(n string) (string, bool, error) { v, ok := r[n]; return v, ok, nil }
func (r registroFalso) SetString(n, v string) error              { r[n] = v; return nil }
func (r registroFalso) Delete(n string) error                    { delete(r, n); return nil }

func trocarAutostart(t *testing.T, reg registroFalso) {
	t.Helper()
	original := newAutostart
	newAutostart = func(version string, dev bool) autostart.Manager {
		return autostart.Manager{
			Launcher:   autostart.RunKey{Store: reg},
			Executable: func() (string, error) { return `C:\Programas\Teamwork Logger\teamwork-logger.exe`, nil },
			Version:    version,
			Dev:        dev,
		}
	}
	t.Cleanup(func() { newAutostart = original })
}

func TestSetAutostartLigaEDesligaComRegistroFalso(t *testing.T) {
	reg := registroFalso{}
	trocarAutostart(t, reg)
	app := &App{version: "3.2.0"}

	st, err := app.SetAutostart(true)
	if err != nil || !st.Enabled || reg[autostart.RunValueName] != `"C:\Programas\Teamwork Logger\teamwork-logger.exe" --minimized` {
		t.Fatalf("SetAutostart(true) = %+v, %v; registro %v", st, err, reg)
	}
	if !app.GetAutostart().Enabled {
		t.Error("GetAutostart deveria informar ligado")
	}
	if st, err := app.SetAutostart(false); err != nil || st.Enabled || len(reg) != 0 {
		t.Fatalf("SetAutostart(false) = %+v, %v; registro %v", st, err, reg)
	}
}

func TestAutostartIndisponivelNaVersaoDev(t *testing.T) {
	reg := registroFalso{}
	trocarAutostart(t, reg)
	app := &App{} // sem versão = 0.0.0-dev
	if st := app.GetAutostart(); st.Supported {
		t.Errorf("GetAutostart = %+v", st)
	}
	if _, err := app.SetAutostart(true); err == nil || len(reg) != 0 {
		t.Errorf("SetAutostart na versão dev: %v; registro %v", err, reg)
	}
}
