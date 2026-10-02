// Package fsutil reúne utilitários de disco compartilhados pelos pacotes do
// backend (configuração, cache de feriados, logs). Fica em internal para que
// não vire API pública do módulo.
package fsutil

import (
	"os"
	"path/filepath"
)

// Permissões usadas para tudo que o app grava no perfil do usuário: só o dono
// acessa. Nenhum desses arquivos guarda o token, mas revelam host, ID de
// usuário, tarefas e o histórico de uso.
const (
	DirPerm  os.FileMode = 0700
	FilePerm os.FileMode = 0600
)

// WriteFileAtomic grava data em path sem nunca deixar um arquivo pela metade:
// escreve num temporário do mesmo diretório, força para o disco e o renomeia
// por cima do destino. Uma queda de energia ou um crash no meio da escrita
// deixa o arquivo antigo intacto em vez de um JSON truncado.
func WriteFileAtomic(path string, data []byte, perm os.FileMode) (err error) {
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	closed := false
	defer func() {
		if err != nil {
			if !closed {
				_ = tmp.Close()
			}
			_ = os.Remove(tmpName)
		}
	}()

	if _, err = tmp.Write(data); err != nil {
		return err
	}
	if err = tmp.Sync(); err != nil {
		return err
	}
	closed = true
	if err = tmp.Close(); err != nil {
		return err
	}
	// O rename leva junto as permissões do temporário, de modo que um destino
	// antigo com permissões abertas passa a ter perm.
	if err = os.Chmod(tmpName, perm); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

// AppDir devolve ~/.teamwork-logger, raiz de tudo que o app grava no perfil.
func AppDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, ".teamwork-logger"), nil
}
