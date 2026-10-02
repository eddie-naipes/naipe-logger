package backend

import (
	"errors"
	"testing"

	"logTime-go/backend/legacy"
)

// trocaLegacy simula o registro; qualquer tentativa de executar falha o teste.
func trocaLegacy(t *testing.T, install legacy.Install, err error) {
	t.Helper()
	origDetect, origRun := detectLegacy, runLegacyUninstall
	detectLegacy = func() (legacy.Install, error) { return install, err }
	runLegacyUninstall = func(legacy.Install) error { t.Error("não deveria executar"); return nil }
	t.Cleanup(func() { detectLegacy, runLegacyUninstall = origDetect, origRun })
}

func TestRunLegacyUninstallerRedetectaAntesDeExecutar(t *testing.T) {
	antiga := legacy.Install{Found: true, DisplayName: "Naipe Logger", UninstallString: `C:\Program Files\Naipe Logger\uninst.exe`}
	var executado *legacy.Install
	origDetect, origRun := detectLegacy, runLegacyUninstall
	detectLegacy = func() (legacy.Install, error) { return antiga, nil }
	runLegacyUninstall = func(i legacy.Install) error { executado = &i; return nil }
	t.Cleanup(func() { detectLegacy, runLegacyUninstall = origDetect, origRun })

	a := &App{}
	if got := a.GetLegacyInstall(); !got.Found {
		t.Fatalf("GetLegacyInstall = %+v", got)
	}
	if err := a.RunLegacyUninstaller(); err != nil {
		t.Fatal(err)
	}
	if executado == nil || executado.UninstallString != antiga.UninstallString {
		t.Errorf("executou %+v", executado)
	}
}

func TestRunLegacyUninstallerSemInstalacao(t *testing.T) {
	trocaLegacy(t, legacy.Install{}, nil)
	if err := (&App{}).RunLegacyUninstaller(); !errors.Is(err, legacy.ErrNotFound) {
		t.Errorf("esperava ErrNotFound, veio %v", err)
	}
}

func TestGetLegacyInstallComErroDeRegistro(t *testing.T) {
	trocaLegacy(t, legacy.Install{Found: true}, errors.New("acesso negado"))
	if got := (&App{}).GetLegacyInstall(); got.Found {
		t.Errorf("com erro de leitura deveria devolver Found=false: %+v", got)
	}
}
