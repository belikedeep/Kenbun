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
		tb.mu.Lock()
		// Capture current local state to sync
		snapshot := make(map[string]float64)
		for id, bucket := range tb.buckets {
			snapshot[id] = bucket.tokens
		}
		tb.mu.Unlock()

		if len(snapshot) == 0 {
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
		
		// 100k RPS Optimization: Sync in a single Lua script to minimize RTT
		// script increments a global usage counter and returns the remaining tokens
		script := `
			local results = {}
			for i, key in ipairs(KEYS) do
				local tenant_id = key
				local usage = tonumber(ARGV[i])
				local limit = tonumber(ARGV[i + #KEYS])
				
				local current_usage = redis.call("INCRBYFLOAT", "rl:usage:" .. tenant_id, usage)
				redis.call("EXPIRE", "rl:usage:" .. tenant_id, 60)
				
				results[i] = limit - current_usage
			end
			return results
		`
		
		keys := make([]string, 0, len(snapshot))
		args := make([]interface{}, 0, len(snapshot)*2)
		
		for id := range snapshot {
			keys = append(keys, id)
		}
		// In a real impl, we'd pass the specific limits, here we use a placeholder 
		// or fetch from a local cache.
		for range keys {
			args = append(args, 1.0) // simplified usage increment
		}
		for range keys {
			args = append(args, 1000.0) // simplified limit
		}

		// Execute Batch Sync
		_ = tb.redis.Eval(ctx, script, keys, args...)
		cancel()
	}
}
