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
	"github.com/rs/cors"
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

	// 3. Data Plane Ingestion (Kafka + Redis Broadcast)
	logger := logging.NewMultiLogger(cfg.KafkaBrokers, "gateway_logs", redisCluster)
	defer logger.Close()

	// 3.1 Data Plane Query (ClickHouse)
	chClient, err := logging.NewClickHouseClient(cfg.ClickHouseAddr, redisCluster)
	if err != nil {
		fmt.Printf("Failed to connect to ClickHouse: %v\n", err)
	}

	// 4. Systems Components
	limiter := ratelimit.NewTokenBucket(redisCluster, cfg.RateLimitSyncFreq)
	twoTierCache, _ := cache.NewTwoTierCache(redisCluster)
	monitor := router.NewEWMAMonitor()

	// 5. Providers with Resilience
	// Standard timeouts: 30s for non-stream, 60s for stream start
	maxRetries := 2
	timeout := 30 * time.Second

	openaiKey := os.Getenv("OPENAI_API_KEY")
	anthropicKey := os.Getenv("ANTHROPIC_API_KEY")

	var openaiProv, anthropicProv provider.Provider
	if openaiKey == "" {
		openaiProv = provider.NewMockProvider("openai")
	} else {
		openaiProv = provider.NewOpenAIProvider(openaiKey)
	}

	if anthropicKey == "" {
		anthropicProv = provider.NewMockProvider("anthropic")
	} else {
		anthropicProv = provider.NewAnthropicProvider(anthropicKey, &http.Client{
			Transport: &http.Transport{
				MaxIdleConns:        1000,
				MaxIdleConnsPerHost: 100,
				IdleConnTimeout:     90 * time.Second,
			},
		})
	}

	var geminiProv provider.Provider
	geminiProv, _ = provider.NewGeminiProvider(ctx, cfg.GeminiAPIKey)
	if geminiProv == nil {
		geminiProv = provider.NewMockProvider("gemini")
	}

	// Wrap in Resilience Decorators
	providers := []provider.Provider{
		provider.NewResilientProvider(openaiProv, maxRetries, timeout),
		provider.NewResilientProvider(anthropicProv, maxRetries, timeout),
		provider.NewResilientProvider(geminiProv, maxRetries, timeout),
		provider.NewResilientProvider(provider.NewOllamaProvider(cfg.OllamaHost), maxRetries, timeout),
	}

	// 6. Router & Selector
	selector := router.NewLatencyAwareSelector(monitor)
	handler := router.NewGatewayHandler(database, limiter, twoTierCache, monitor, selector, logger, providers)
	adminHandler := router.NewAdminHandler(database, chClient, cfg.AdminSecret)

	// 7. Background Workers
	budgetProcessor := db.NewBudgetProcessor(database, cfg.KafkaBrokers, "gateway_logs")
	go budgetProcessor.Start(ctx)

	r := chi.NewRouter()

	// CORS
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "X-API-Key", "X-Admin-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	})
	r.Use(c.Handler)

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	// Chat completions (Streaming can last longer than standard timeout)
	r.Post("/v1/chat/completions", handler.ServeHTTP)

	// Admin API
	r.Route("/admin", func(r chi.Router) {
		adminHandler.RegisterRoutes(r)
	})

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
