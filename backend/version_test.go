package backend

import (
	"os"
	"testing"
)

func TestParseProductVersion(t *testing.T) {
	casos := []struct {
		nome, json, esperado string
	}{
		{"versao normal", `{"info":{"productVersion":"1.4.2"}}`, "1.4.2"},
		{"prefixo v removido", `{"info":{"productVersion":"v2.0.0"}}`, "2.0.0"},
		{"espacos", `{"info":{"productVersion":" 1.0.0 "}}`, "1.0.0"},
		{"campo ausente", `{"info":{}}`, DevVersion},
		{"json invalido", `{`, DevVersion},
	}
	for _, c := range casos {
		if got := ParseProductVersion([]byte(c.json)); got != c.esperado {
			t.Errorf("%s: ParseProductVersion = %q, esperava %q", c.nome, got, c.esperado)
		}
	}
}

// O wails.json real do repositório precisa ter info.productVersion: é dele que
// main.go tira a versão embutida no binário.
func TestWailsJSONDoRepositorioTemVersao(t *testing.T) {
	data, err := os.ReadFile("../wails.json")
	if err != nil {
		t.Fatalf("lendo wails.json: %v", err)
	}
	if v := ParseProductVersion(data); v == DevVersion {
		t.Error("wails.json sem info.productVersion legível")
	}
}

func TestMarkDevVersion(t *testing.T) {
	if got := MarkDevVersion("1.2.3", true); got != "1.2.3-dev" {
		t.Errorf("build dev: %q", got)
	}
	if got := MarkDevVersion("1.2.3", false); got != "1.2.3" {
		t.Errorf("build de produção não deveria mudar: %q", got)
	}
	if got := MarkDevVersion(DevVersion, true); got != DevVersion {
		t.Errorf("já pré-release não ganha outro sufixo: %q", got)
	}
}

func TestGetAppVersion(t *testing.T) {
	if got := (&App{version: "1.5.0"}).GetAppVersion(); got != "1.5.0" {
		t.Errorf("GetAppVersion = %q", got)
	}
	if got := (&App{}).GetAppVersion(); got != DevVersion {
		t.Errorf("sem versão deveria devolver %q, veio %q", DevVersion, got)
	}
}
