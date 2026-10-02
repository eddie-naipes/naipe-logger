package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"logTime-go/backend/internal/fsutil"
)

// Persistência do cache de feriados em ~/.teamwork-logger/cache/holidays-<ano>.json.
// Sem ela, cada abertura do app consultava a BrasilAPI de novo e, se ela
// estivesse fora, o app passava a usar só o calendário local (sem feriados
// avulsos) até a API voltar. Só dados vindos da BrasilAPI vão para o disco: o
// calendário local é recalculado na hora e não precisa de cópia.

var (
	holidayDiskMu sync.RWMutex
	// holidayDiskDir vazio desliga a persistência (é o padrão nos testes).
	holidayDiskDir string
)

const holidayFilePrefix = "holidays-"

// SetHolidayCacheDir liga a persistência do cache de feriados em dir,
// criando a pasta com permissão 0700. Vazio desliga.
func SetHolidayCacheDir(dir string) error {
	if dir != "" {
		if err := os.MkdirAll(dir, fsutil.DirPerm); err != nil {
			return fmt.Errorf("erro ao criar pasta de cache: %v", err)
		}
	}
	holidayDiskMu.Lock()
	holidayDiskDir = dir
	holidayDiskMu.Unlock()
	return nil
}

func currentHolidayDiskDir() string {
	holidayDiskMu.RLock()
	defer holidayDiskMu.RUnlock()
	return holidayDiskDir
}

func holidayCachePath(dir string, year int) string {
	return filepath.Join(dir, fmt.Sprintf("%s%d.json", holidayFilePrefix, year))
}

// persistHolidayCache grava o ano no disco de forma atômica. Falhar aqui só
// custa uma consulta à rede na próxima abertura, então apenas registra.
func persistHolidayCache(cache *HolidayCache) {
	dir := currentHolidayDiskDir()
	if dir == "" || cache == nil {
		return
	}
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		slog.Warn("Erro ao serializar cache de feriados", "ano", cache.Year, "err", err)
		return
	}
	if err := fsutil.WriteFileAtomic(holidayCachePath(dir, cache.Year), data, fsutil.FilePerm); err != nil {
		slog.Warn("Erro ao gravar cache de feriados em disco", "ano", cache.Year, "err", err)
	}
}

// readHolidayCacheFile lê e valida o arquivo de um ano. Arquivo corrompido,
// de outro ano ou sem feriados é ignorado (e apagado, para não ser relido a
// cada abertura).
func readHolidayCacheFile(dir string, year int) (*HolidayCache, bool) {
	path := holidayCachePath(dir, year)
	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			slog.Warn("Erro ao ler cache de feriados", "arquivo", path, "err", err)
		}
		return nil, false
	}

	var cache HolidayCache
	if err := json.Unmarshal(data, &cache); err != nil || cache.Year != year || len(cache.Holidays) == 0 || !cache.authoritative() {
		slog.Warn("Cache de feriados em disco inválido; descartado", "arquivo", path, "err", err)
		_ = os.Remove(path)
		return nil, false
	}
	return &cache, true
}

// loadHolidayYearFromDisk traz o ano do disco para a memória, se a memória
// ainda não o tiver. Devolve o que ficou valendo na memória.
func loadHolidayYearFromDisk(year int) (*HolidayCache, bool) {
	dir := currentHolidayDiskDir()
	if dir == "" {
		return nil, false
	}
	cache, ok := readHolidayCacheFile(dir, year)
	if !ok {
		return nil, false
	}

	holidayCacheLock.Lock()
	defer holidayCacheLock.Unlock()
	// Outra goroutine pode ter buscado o ano na rede enquanto o arquivo era
	// lido; o dado mais novo vence.
	if current, exists := holidayCache[year]; exists {
		return current, true
	}
	holidayCache[year] = cache
	return cache, true
}

// LoadHolidayCacheFromDisk carrega para a memória todos os anos persistidos.
// Chamado na inicialização; devolve quantos anos foram carregados.
func LoadHolidayCacheFromDisk() int {
	dir := currentHolidayDiskDir()
	if dir == "" {
		return 0
	}
	loaded := 0
	for _, year := range persistedHolidayYears(dir) {
		if _, ok := loadHolidayYearFromDisk(year); ok {
			loaded++
		}
	}
	if loaded > 0 {
		slog.Debug("Cache de feriados carregado do disco", "anos", loaded)
	}
	return loaded
}

// persistedHolidayYears lista os anos com arquivo holidays-<ano>.json.
func persistedHolidayYears(dir string) []int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var years []int
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasPrefix(name, holidayFilePrefix) || !strings.HasSuffix(name, ".json") {
			continue
		}
		year, err := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(name, holidayFilePrefix), ".json"))
		if err != nil {
			continue
		}
		years = append(years, year)
	}
	return years
}

func removeHolidayCacheFile(year int) {
	dir := currentHolidayDiskDir()
	if dir == "" {
		return
	}
	if err := os.Remove(holidayCachePath(dir, year)); err != nil && !os.IsNotExist(err) {
		slog.Warn("Erro ao apagar cache de feriados em disco", "ano", year, "err", err)
	}
}

func removeAllHolidayCacheFiles() {
	dir := currentHolidayDiskDir()
	if dir == "" {
		return
	}
	for _, year := range persistedHolidayYears(dir) {
		removeHolidayCacheFile(year)
	}
}
