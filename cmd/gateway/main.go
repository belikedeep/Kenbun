// Copyright 2026 Deepak Mardi. Licensed under Apache 2.0.

package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/redis/go-redis/v9"
	"github.com/belikedeep/kenbun/internal/cache"
	"github.com/belikedeep/kenbun/internal/config"
	"github.com/belikedeep/kenbun/internal/db"
	"github.com/belikedeep/kenbun/internal/logging"
	"github.com/belikedeep/kenbun/internal/provider"
	"github.com/belikedeep/kenbun/internal/ratelimit"
	"github.com/belikedeep/kenbun/internal/router"
)

func main() {
	cfg := config.Load()
	ctx := context.Background()

	// 1. Control Plane (PostgreSQL)
	database, err := db.New(ctx, cfg.DatabaseURL)
	if err != nil {
		fmt.Printf("Failed to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer database.Close()

	// 2. State Plane (Redis Cluster)
	redisCluster := redis.NewClusterClient(&redis.ClusterOptions{
		Addrs: cfg.RedisAddrs,
	})
	defer redisCluster.Close()

	// 3. Data Plane Ingestion (Kafka)
	logger := logging.NewKafkaLogger(cfg.KafkaBrokers, "gateway_logs")
	defer logger.Close()

	// 4. Systems Components
	limiter := ratelimit.NewTokenBucket(redisCluster, cfg.RateLimitSyncFreq)
	twoTierCache, _ := cache.NewTwoTierCache(redisCluster)
	monitor := router.NewEWMAMonitor()

	// 5. Providers
	geminiProv, _ := provider.NewGeminiProvider(ctx, cfg.GeminiAPIKey)

	providers := map[string]provider.Provider{
		"openai": provider.NewOpenAIProvider(os.Getenv("OPENAI_API_KEY")),
		"anthropic": provider.NewAnthropicProvider(os.Getenv("ANTHROPIC_API_KEY"), &http.Client{
			Transport: &http.Transport{
				MaxIdleConns:        1000,
				MaxIdleConnsPerHost: 100,
				IdleConnTimeout:     90 * time.Second,
			},
		}),
		"gemini": geminiProv,
	}

	// 6. Router & Handler
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(cfg.RequestTimeout))

	handler := router.NewGatewayHandler(database, limiter, twoTierCache, monitor, logger, providers)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	r.Post("/v1/chat/completions", handler.ServeHTTP)

	fmt.Printf("👁️ Kenbun Gateway starting on port %s...\n", cfg.Port)
	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      r,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil {
		fmt.Printf("Server failed: %v\n", err)
	}
}
