package api

import (
	"strings"
	"sync"
	"time"
)

type CacheEntry struct {
	Data      interface{}
	ExpiresAt time.Time
}

type Cache struct {
	data  map[string]CacheEntry
	mutex sync.RWMutex
}

func NewCache() *Cache {
	return &Cache{
		data: make(map[string]CacheEntry),
	}
}

// getCached lê um valor do cache verificando o tipo. Um valor gravado com tipo
// inesperado é descartado e tratado como ausente, em vez de derrubar a
// aplicação com um panic de type assertion.
func getCached[T any](c *Cache, key string) (T, bool) {
	var zero T

	raw, found := c.Get(key)
	if !found {
		return zero, false
	}

	typed, ok := raw.(T)
	if !ok {
		c.Delete(key)
		return zero, false
	}

	return typed, true
}

func (c *Cache) Get(key string) (interface{}, bool) {
	c.mutex.RLock()
	entry, exists := c.data[key]
	c.mutex.RUnlock()

	if !exists {
		return nil, false
	}

	if time.Now().After(entry.ExpiresAt) {
		// Remove a entrada vencida na leitura: sem isso o mapa só crescia,
		// já que nada mais apagava chaves expiradas.
		c.mutex.Lock()
		if current, ok := c.data[key]; ok && time.Now().After(current.ExpiresAt) {
			delete(c.data, key)
		}
		c.mutex.Unlock()
		return nil, false
	}

	return entry.Data, true
}

func (c *Cache) Set(key string, value interface{}, ttl time.Duration) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.data[key] = CacheEntry{
		Data:      value,
		ExpiresAt: time.Now().Add(ttl),
	}
}

func (c *Cache) Delete(key string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	delete(c.data, key)
}

// DeletePrefix remove todas as chaves que começam com o prefixo. Serve para
// invalidar famílias de chaves que embutem IDs (ex.: dashboard_stats_<user>).
func (c *Cache) DeletePrefix(prefix string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	for key := range c.data {
		if strings.HasPrefix(key, prefix) {
			delete(c.data, key)
		}
	}
}

func (c *Cache) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.data = make(map[string]CacheEntry)
}

// Chaves de cache derivadas dos lançamentos de tempo.
const (
	cacheKeyDashboardStatsPrefix = "dashboard_stats_"
	cacheKeyRecentActivities     = "recent_activities"
)

// invalidateTimeEntryCaches descarta o que foi calculado a partir dos
// lançamentos. Chamado depois de criar, editar ou apagar lançamentos: sem
// isso o dashboard mostrava as horas e atividades antigas por até uma hora.
func (t *TeamworkAPI) invalidateTimeEntryCaches() {
	if t.cache == nil {
		return
	}
	t.cache.DeletePrefix(cacheKeyDashboardStatsPrefix)
	t.cache.Delete(cacheKeyRecentActivities)
}
