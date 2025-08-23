package cache

import (
	"sync"
	"time"
)

type entry[V any] struct {
	v   V
	exp time.Time
}

type TTLCache[K comparable, V any] struct {
	mu   sync.RWMutex
	data map[K]entry[V]
	ttl  time.Duration
}

func NewTTL[K comparable, V any](ttl time.Duration) *TTLCache[K, V] {
	return &TTLCache[K, V]{data: make(map[K]entry[V]), ttl: ttl}
}

func (c *TTLCache[K, V]) Get(k K) (V, bool) {
	c.mu.RLock()
	e, ok := c.data[k]
	c.mu.RUnlock()
	if !ok {
		var zero V
		return zero, false
	}
	if time.Now().After(e.exp) {
		c.mu.Lock()
		delete(c.data, k)
		c.mu.Unlock()
		var zero V
		return zero, false
	}
	return e.v, true
}

func (c *TTLCache[K, V]) Set(k K, v V) {
	c.mu.Lock()
	c.data[k] = entry[V]{v: v, exp: time.Now().Add(c.ttl)}
	c.mu.Unlock()
}

func (c *TTLCache[K, V]) Delete(k K) {
	c.mu.Lock()
	delete(c.data, k)
	c.mu.Unlock()
}

func (c *TTLCache[K, V]) Clear() {
	c.mu.Lock()
	c.data = make(map[K]entry[V])
	c.mu.Unlock()
}
