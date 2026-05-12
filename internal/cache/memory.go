// Copyright 2026 Deepak Mardi. Licensed under Apache 2.0.

package cache

import (
	"context"
	"sync"
	"time"
)

// MemoryCache implements the Cache interface using a standard map and a mutex.
type MemoryCache struct {
	mu   sync.RWMutex
	data map[string][]byte
}

func NewMemoryCache() Cache {
	return &MemoryCache{
		data: make(map[string][]byte),
	}
}

func (m *MemoryCache) Get(ctx context.Context, key string) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	val, ok := m.data[key]
	if !ok {
		return nil, nil // Return nil, nil when not found, matching TwoTierCache behavior
	}

	return val, nil
}

func (m *MemoryCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.data[key] = value
	return nil
}
