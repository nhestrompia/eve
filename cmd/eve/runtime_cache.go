package main

import (
	"context"
	"sync"
	"time"
)

const runtimeDerivedCacheTTL = 2 * time.Second

type runtimeDerivedCacheEntry struct {
	generation uint64
	expiresAt  time.Time
	value      any
}

type runtimeDerivedCacheLoad struct {
	done chan struct{}
}

type runtimeDerivedCache struct {
	mu      sync.Mutex
	entries map[string]runtimeDerivedCacheEntry
	loads   map[string]*runtimeDerivedCacheLoad
}

type runtimeDerivedCacheStats struct {
	Entries  int `json:"entries"`
	InFlight int `json:"inFlight"`
}

func newRuntimeDerivedCache() *runtimeDerivedCache {
	return &runtimeDerivedCache{
		entries: make(map[string]runtimeDerivedCacheEntry),
		loads:   make(map[string]*runtimeDerivedCacheLoad),
	}
}

func (cache *runtimeDerivedCache) stats() runtimeDerivedCacheStats {
	if cache == nil {
		return runtimeDerivedCacheStats{}
	}
	cache.mu.Lock()
	defer cache.mu.Unlock()
	return runtimeDerivedCacheStats{
		Entries:  len(cache.entries),
		InFlight: len(cache.loads),
	}
}

func cachedRuntimeValue[T any](
	ctx context.Context,
	cache *runtimeDerivedCache,
	key string,
	generation uint64,
	ttl time.Duration,
	load func() (T, error),
) (T, error) {
	if cache == nil {
		return load()
	}
	if ttl <= 0 {
		ttl = runtimeDerivedCacheTTL
	}
	for {
		now := time.Now()
		cache.mu.Lock()
		if entry, ok := cache.entries[key]; ok &&
			entry.generation == generation &&
			now.Before(entry.expiresAt) {
			value := entry.value.(T)
			cache.mu.Unlock()
			return value, nil
		}
		if pending, ok := cache.loads[key]; ok {
			cache.mu.Unlock()
			select {
			case <-ctx.Done():
				var zero T
				return zero, ctx.Err()
			case <-pending.done:
				continue
			}
		}
		pending := &runtimeDerivedCacheLoad{done: make(chan struct{})}
		cache.loads[key] = pending
		cache.mu.Unlock()

		value, err := load()

		cache.mu.Lock()
		if err == nil {
			cache.entries[key] = runtimeDerivedCacheEntry{
				generation: generation,
				expiresAt:  time.Now().Add(ttl),
				value:      value,
			}
		}
		delete(cache.loads, key)
		close(pending.done)
		cache.mu.Unlock()
		return value, err
	}
}
