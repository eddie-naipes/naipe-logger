package legacy

import "testing"

// fakeRegistry implementa Registry com entradas fixas.
type fakeRegistry []Entry

func (f fakeRegistry) UninstallEntries() ([]Entry, error) { return f, nil }

// Entradas exatamente como os instaladores antigos gravavam.
var (
	naipeLoggerAntigo = Entry{
		KeyName:         "Naipe Logger",
		DisplayName:     "Naipe Logger",
		UninstallString: `C:\Program Files\Naipe Logger\uninst.exe`,
	}
	wailsAdminAntigo = Entry{
		KeyName:         "Naipe Sync SolutionsTeamwork Logger",
		DisplayName:     "Teamwork Logger",
		UninstallString: `"C:\Program Files\Naipe Sync Solutions\Teamwork Logger\uninstall.exe"`,
	}
	outroPrograma = Entry{
		KeyName:         "7-Zip",
		DisplayName:     "7-Zip 23.01",
		InstallLocation: `C:\Program Files\7-Zip\`,
		UninstallString: `"C:\Program Files\7-Zip\Uninstall.exe"`,
	}
)

func entradas(t *testing.T, r Registry) []Entry {
	t.Helper()
	e, err := r.UninstallEntries()
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func TestSelectEncontraInstaladorAntigoSemInstallLocation(t *testing.T) {
	got := Select(entradas(t, fakeRegistry{outroPrograma, naipeLoggerAntigo}), `C:\Users\ana\AppData\Local\Programs\Teamwork Logger`)
	if !got.Found || got.DisplayName != "Naipe Logger" {
		t.Fatalf("esperava a instalação Naipe Logger, veio %+v", got)
	}
	if got.InstallLocation != `C:\Program Files\Naipe Logger` {
		t.Errorf("pasta deduzida do desinstalador = %q", got.InstallLocation)
	}
}

func TestSelectEncontraInstalacaoAdminDoWails(t *testing.T) {
	got := Select(fakeRegistry{wailsAdminAntigo}, `C:\Users\ana\AppData\Local\Programs\Teamwork Logger`)
	if !got.Found || got.InstallLocation != `C:\Program Files\Naipe Sync Solutions\Teamwork Logger` {
		t.Fatalf("resultado: %+v", got)
	}
}

func TestSelectIgnoraAPropriaPastaDoExecutavel(t *testing.T) {
	// O app em execução é o próprio de Program Files: não é "antigo".
	got := Select(fakeRegistry{wailsAdminAntigo}, `c:\program files\naipe sync solutions\teamwork logger\`)
	if got.Found {
		t.Errorf("não deveria oferecer remover a instalação em execução: %+v", got)
	}

	comLocation := naipeLoggerAntigo
	comLocation.InstallLocation = `C:\Program Files\Naipe Logger\`
	if got := Select(fakeRegistry{comLocation}, `C:\Program Files\Naipe Logger`); got.Found {
		t.Errorf("InstallLocation igual à pasta atual deveria ser ignorado: %+v", got)
	}
}

func TestSelectIgnoraOutrosProgramasESemDesinstalador(t *testing.T) {
	semDesinstalador := naipeLoggerAntigo
	semDesinstalador.UninstallString = ""
	if got := Select(fakeRegistry{outroPrograma, semDesinstalador}, `C:\x`); got.Found {
		t.Errorf("nada deveria ser selecionado: %+v", got)
	}
	if got := Select(nil, `C:\x`); got.Found {
		t.Error("registro vazio não tem instalação antiga")
	}
}

func TestSplitCommand(t *testing.T) {
	casos := []struct{ in, exe, args string }{
		{`"C:\Program Files\X\uninstall.exe" /S`, `C:\Program Files\X\uninstall.exe`, "/S"},
		{`"C:\Program Files\X\uninstall.exe"`, `C:\Program Files\X\uninstall.exe`, ""},
		{`C:\Program Files\Naipe Logger\uninst.exe`, `C:\Program Files\Naipe Logger\uninst.exe`, ""},
		{`C:\Program Files\Naipe Logger\uninst.EXE /S`, `C:\Program Files\Naipe Logger\uninst.EXE`, "/S"},
		{`"C:\sem fim`, "", ""},
		{`MsiExec.exe /X{GUID}`, "MsiExec.exe", "/X{GUID}"},
	}
	for _, c := range casos {
		exe, args := SplitCommand(c.in)
		if exe != c.exe || args != c.args {
			t.Errorf("SplitCommand(%q) = (%q, %q), esperava (%q, %q)", c.in, exe, args, c.exe, c.args)
		}
	}
}

func TestValidateUninstallerExigeCaminhoAbsolutoDeExe(t *testing.T) {
	if _, _, err := validateUninstaller(`"C:\Program Files\X\uninstall.exe" /S`); err != nil {
		t.Errorf("caminho válido recusado: %v", err)
	}
	for _, ruim := range []string{`MsiExec.exe /X{GUID}`, `uninstall.exe`, `"C:\x\script.bat"`, ""} {
		if _, _, err := validateUninstaller(ruim); err == nil {
			t.Errorf("%q deveria ser recusado", ruim)
		}
	}
}

func TestRunUninstallerSemInstalacao(t *testing.T) {
	if err := RunUninstaller(Install{}); err == nil {
		t.Error("sem instalação antiga deveria devolver erro")
	}
}
