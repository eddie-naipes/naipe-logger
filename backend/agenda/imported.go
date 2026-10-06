package agenda

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"logTime-go/backend/internal/fsutil"
)

// ImportedRecord registra uma ocorrência já lançada, para não ser importada
// de novo.
type ImportedRecord struct {
	Key     string `json:"key"`
	Date    string `json:"date"`
	TaskID  int    `json:"taskId"`
	EntryID int    `json:"entryId"`
	// ImportedAt é preenchido ao marcar (RFC 3339).
	ImportedAt string `json:"importedAt"`
}

// importedRetention é por quanto tempo uma marcação é guardada: depois disso a
// ocorrência já está longe de qualquer período que se importe.
const importedRetention = 400 * 24 * time.Hour

// ImportedStore guarda as marcações num JSON (escrita atômica, 0600).
type ImportedStore struct {
	path    string
	mu      sync.Mutex
	records map[string]ImportedRecord
	now     func() time.Time
}

// OpenImportedStore lê o arquivo; ausente vira um registro vazio e um arquivo
// corrompido é ignorado (com aviso no log) em vez de travar a importação.
func OpenImportedStore(path string) (*ImportedStore, error) {
	s := &ImportedStore{path: path, records: map[string]ImportedRecord{}, now: time.Now}
	data, err := os.ReadFile(path)
	switch {
	case errors.Is(err, os.ErrNotExist):
		return s, nil
	case err != nil:
		return nil, fmt.Errorf("erro ao ler os eventos já importados: %v", err)
	}
	var lista []ImportedRecord
	if err := json.Unmarshal(data, &lista); err != nil {
		slog.Warn("Registro de eventos importados corrompido; começando vazio", "caminho", path, "err", err)
		return s, nil
	}
	for _, r := range lista {
		if r.Key != "" {
			s.records[r.Key] = r
		}
	}
	return s, nil
}

// Has informa se a chave já foi lançada.
func (s *ImportedStore) Has(key string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.records[key]
	return ok
}

// Mark grava as marcações (uma chave repetida é atualizada).
func (s *ImportedStore) Mark(recs []ImportedRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	agora := s.now()
	for _, r := range recs {
		if r.Key == "" {
			continue
		}
		r.ImportedAt = agora.Format(time.RFC3339)
		s.records[r.Key] = r
	}
	return s.saveLocked(agora)
}

// Unmark remove marcações (ex.: o lote foi desfeito).
func (s *ImportedStore) Unmark(keys []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, k := range keys {
		delete(s.records, k)
	}
	return s.saveLocked(s.now())
}

func (s *ImportedStore) saveLocked(agora time.Time) error {
	lista := make([]ImportedRecord, 0, len(s.records))
	for k, r := range s.records {
		if t, err := time.Parse(time.RFC3339, r.ImportedAt); err == nil && agora.Sub(t) > importedRetention {
			delete(s.records, k)
			continue
		}
		lista = append(lista, r)
	}
	data, err := json.MarshalIndent(lista, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.path), fsutil.DirPerm); err != nil {
		return err
	}
	if err := fsutil.WriteFileAtomic(s.path, data, fsutil.FilePerm); err != nil {
		return fmt.Errorf("erro ao salvar os eventos importados: %v", err)
	}
	return nil
}
