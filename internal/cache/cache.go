// Copyright 2026 Deepak Mardi. Licensed under Apache 2.0.

package cache

import (
	"context"
	"time"

	"github.com/dgraph-io/ristretto"
	"github.com/redis/go-redis/v9"
)

// Cache defines the system contract for a two-tier high-throughput cache.
type Cache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
}

// TwoTierCache implements the Cache interface using an in-memory LRU and Redis Cluster.
type TwoTierCache struct {
	l1    *ristretto.Cache
	redis *redis.ClusterClient
}

func NewTwoTierCache(redisClient *redis.ClusterClient) (Cache, error) {
	l1, err := ristretto.NewCache(&ristretto.Config{
		NumCounters: 1e7,     // number of keys to track frequency of (10M).
		MaxCost:     1 << 30, // maximum cost of cache (1GB).
		BufferItems: 64,      // number of keys per Get buffer.
	})
	if err != nil {
		return nil, err
	}

	return &TwoTierCache{
		l1:    l1,
		redis: redisClient,
	}, nil
}

func (c *TwoTierCache) Get(ctx context.Context, key string) ([]byte, error) {
	// 1. L1 (In-Memory) Lookup
	if val, ok := c.l1.Get(key); ok {
		return val.([]byte), nil
	}

	// 2. L2 (Redis Cluster) Fallback
	val, err := c.redis.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	// 3. Backfill L1
	c.l1.SetWithTTL(key, val, 1, time.Minute*10)

	return val, nil
}

func (c *TwoTierCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	// Set L1
	c.l1.SetWithTTL(key, value, 1, ttl)

	// Set L2
	return c.redis.Set(ctx, key, value, ttl).Err()
}
