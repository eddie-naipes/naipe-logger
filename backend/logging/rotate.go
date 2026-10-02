package logging

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"logTime-go/backend/internal/fsutil"
)

// RotatingFile é um io.Writer que grava num arquivo e o gira quando ele passa
// de maxSize: app.log vira app.log.1, app.log.1 vira app.log.2 e assim por
// diante, descartando o que passar de maxBackups. Feito à mão para não trazer
// uma dependência só para isso; o volume de log do app é pequeno.
type RotatingFile struct {
	mu         sync.Mutex
	path       string
	maxSize    int64
	maxBackups int
	file       *os.File
	size       int64
}

// OpenRotatingFile abre (ou cria) path para acréscimo com permissão 0600.
func OpenRotatingFile(path string, maxSize int64, maxBackups int) (*RotatingFile, error) {
	if maxSize <= 0 {
		return nil, fmt.Errorf("tamanho máximo do log inválido: %d", maxSize)
	}
	if maxBackups < 0 {
		maxBackups = 0
	}
	r := &RotatingFile{path: path, maxSize: maxSize, maxBackups: maxBackups}
	if err := r.openLocked(); err != nil {
		return nil, err
	}
	return r, nil
}

// Path devolve o caminho do arquivo de log atual.
func (r *RotatingFile) Path() string { return r.path }

func (r *RotatingFile) openLocked() error {
	if err := os.MkdirAll(filepath.Dir(r.path), fsutil.DirPerm); err != nil {
		return err
	}
	f, err := os.OpenFile(r.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, fsutil.FilePerm)
	if err != nil {
		return err
	}
	// OpenFile não muda a permissão de um arquivo que já existia; um log
	// criado por outra ferramenta com permissões abertas é corrigido aqui.
	_ = os.Chmod(r.path, fsutil.FilePerm)
	info, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return err
	}
	r.file = f
	r.size = info.Size()
	return nil
}

// Write grava p inteiro no arquivo atual, girando antes se ele estourar o
// limite. Uma linha nunca é dividida entre dois arquivos.
func (r *RotatingFile) Write(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.file == nil {
		if err := r.openLocked(); err != nil {
			return 0, err
		}
	}

	// Com o arquivo vazio não adianta girar: uma entrada maior que o limite
	// entraria em laço; ela é gravada mesmo assim.
	if r.size > 0 && r.size+int64(len(p)) > r.maxSize {
		if err := r.rotateLocked(); err != nil {
			// Falhar ao girar não pode fazer o log sumir: segue gravando no
			// arquivo atual (reaberto se preciso).
			if r.file == nil {
				if openErr := r.openLocked(); openErr != nil {
					return 0, openErr
				}
			}
		}
	}

	n, err := r.file.Write(p)
	r.size += int64(n)
	return n, err
}

// rotateLocked fecha o arquivo atual antes de renomeá-lo: no Windows não é
// possível renomear um arquivo aberto.
func (r *RotatingFile) rotateLocked() error {
	if r.file != nil {
		_ = r.file.Close()
		r.file = nil
	}

	if r.maxBackups == 0 {
		if err := os.Remove(r.path); err != nil && !os.IsNotExist(err) {
			return err
		}
		return r.openLocked()
	}

	// O mais antigo sai; os demais andam uma posição.
	_ = os.Remove(backupName(r.path, r.maxBackups))
	for i := r.maxBackups - 1; i >= 1; i-- {
		src := backupName(r.path, i)
		if _, err := os.Stat(src); err == nil {
			_ = os.Rename(src, backupName(r.path, i+1))
		}
	}
	if err := os.Rename(r.path, backupName(r.path, 1)); err != nil && !os.IsNotExist(err) {
		_ = r.openLocked()
		return err
	}
	return r.openLocked()
}

func backupName(path string, n int) string {
	return fmt.Sprintf("%s.%d", path, n)
}

// Close fecha o arquivo atual. Escritas posteriores o reabrem.
func (r *RotatingFile) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.file == nil {
		return nil
	}
	err := r.file.Close()
	r.file = nil
	return err
}
