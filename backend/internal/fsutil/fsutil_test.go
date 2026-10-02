package fsutil

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestWriteFileAtomicSubstituiConteudoSemDeixarTemporario(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "dados.json")

	if err := os.WriteFile(path, []byte("antigo"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := WriteFileAtomic(path, []byte("novo"), FilePerm); err != nil {
		t.Fatalf("WriteFileAtomic: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil || string(got) != "novo" {
		t.Fatalf("conteúdo = %q, %v; esperava \"novo\"", got, err)
	}

	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if strings.Contains(e.Name(), ".tmp-") {
			t.Errorf("temporário %s ficou para trás", e.Name())
		}
	}

	// No Windows o Chmod só controla o atributo somente leitura.
	if runtime.GOOS != "windows" {
		info, _ := os.Stat(path)
		if info.Mode().Perm() != FilePerm {
			t.Errorf("permissão = %v, esperava %v", info.Mode().Perm(), FilePerm)
		}
	}
}

func TestWriteFileAtomicFalhaSemDiretorio(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nao-existe", "x.json")
	if err := WriteFileAtomic(path, []byte("x"), FilePerm); err == nil {
		t.Fatal("esperava erro ao gravar em diretório inexistente")
	}
}
