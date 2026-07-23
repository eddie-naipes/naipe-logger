package api

import (
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
	defer c.mutex.RUnlock()

	entry, exists := c.data[key]
	if !exists {
		return nil, false
	}

	if time.Now().After(entry.ExpiresAt) {
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

func (c *Cache) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.data = make(map[string]CacheEntry)
}
