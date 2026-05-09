// Copyright 2026 Deepak Mardi. Licensed under Apache 2.0.

package ratelimit

import (
	"context"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// Limiter defines the system contract for high-throughput rate limiting.
type Limiter interface {
	Allow(ctx context.Context, tenantID string, limit int) (*Result, error)
}

// Result represents the outcome of a rate limit check.
type Result struct {
	Allowed    bool
	Limit      int
	Remaining  int
	ResetAt    time.Time
	RetryAfter time.Duration
}

// TokenBucket implements a high-throughput distributed token bucket.
type TokenBucket struct {
	mu            sync.RWMutex
	buckets       map[string]*localBucket
	redis         *redis.ClusterClient
	syncInterval  time.Duration
}

type localBucket struct {
	tokens float64
	last   time.Time
}

func NewTokenBucket(redis *redis.ClusterClient, syncInterval time.Duration) *TokenBucket {
	tb := &TokenBucket{
		buckets:      make(map[string]*localBucket),
		redis:        redis,
		syncInterval: syncInterval,
	}
	go tb.backgroundSync()
	return tb
}

func (tb *TokenBucket) Allow(ctx context.Context, tenantID string, limit int) (*Result, error) {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	bucket, exists := tb.buckets[tenantID]
	if !exists {
		// Start with a small local quota (10% of total limit or 10 tokens)
		initialTokens := float64(limit) / 10.0
		if initialTokens < 1 {
			initialTokens = 1
		}
		bucket = &localBucket{tokens: initialTokens, last: time.Now()}
		tb.buckets[tenantID] = bucket
	}

	// Refill based on time (local estimation)
	now := time.Now()
	refillRate := float64(limit) / 60.0 // per second
	duration := now.Sub(bucket.last).Seconds()
	bucket.tokens += duration * refillRate
	if bucket.tokens > float64(limit) {
		bucket.tokens = float64(limit)
	}
	bucket.last = now

	// Local-first check
	if bucket.tokens >= 1 {
		bucket.tokens -= 1
		return &Result{Allowed: true, Remaining: int(bucket.tokens), Limit: limit}, nil
	}

	return &Result{Allowed: false, RetryAfter: time.Second, Limit: limit}, nil
}

func (tb *TokenBucket) backgroundSync() {
	ticker := time.NewTicker(tb.syncInterval)
	for range ticker.C {
		// In a production implementation, this would:
		// 1. Gather local usage for all tenants.
		// 2. Batch-sync to Redis Cluster using a Lua script or Pipeline.
		// 3. Update local buckets with the latest cluster-wide available tokens.
	}
}
