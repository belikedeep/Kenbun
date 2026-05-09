// Copyright 2026 Deepak Mardi. Licensed under Apache 2.0.

package router

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/belikedeep/kenbun/internal/cache"
	"github.com/belikedeep/kenbun/internal/db"
	"github.com/belikedeep/kenbun/internal/logging"
	"github.com/belikedeep/kenbun/internal/provider"
	"github.com/belikedeep/kenbun/internal/ratelimit"
)

type GatewayHandler struct {
	db        *db.Client
	limiter   ratelimit.Limiter
	cache     cache.Cache
	monitor   HealthMonitor
	selector  ProviderSelector
	logger    logging.Logger
	providers []provider.Provider
}

func NewGatewayHandler(
	db *db.Client,
	limiter ratelimit.Limiter,
	cache cache.Cache,
	monitor HealthMonitor,
	selector ProviderSelector,
	logger logging.Logger,
	providers []provider.Provider,
) *GatewayHandler {
	return &GatewayHandler{
		db:        db,
		limiter:   limiter,
		cache:     cache,
		monitor:   monitor,
		selector:  selector,
		logger:    logger,
		providers: providers,
	}
}

func (h *GatewayHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	start := time.Now()

	// 1. Auth & Tenant Lookup (Cached)
	apiKey := r.Header.Get("X-API-Key")
	if apiKey == "" {
		http.Error(w, "missing_api_key", http.StatusUnauthorized)
		return
	}
	hash := sha256.Sum256([]byte(apiKey))
	keyHash := hex.EncodeToString(hash[:])

	var tenant *db.Tenant
	tenantKey := "tenant:key:" + keyHash
	if cached, _ := h.cache.Get(ctx, tenantKey); cached != nil {
		json.Unmarshal(cached, &tenant)
	} else {
		var err error
		tenant, err = h.db.GetTenantByAPIKeyHash(ctx, keyHash)
		if err != nil {
			http.Error(w, "invalid_api_key", http.StatusUnauthorized)
			return
		}
		tenantData, _ := json.Marshal(tenant)
		h.cache.Set(ctx, tenantKey, tenantData, 5*time.Minute)
	}

	// 2. Budget Enforcement (Local-first with Redis Sync)
	if tenant.BudgetCents > 0 && tenant.SpentCents >= tenant.BudgetCents {
		http.Error(w, "budget_exhausted", http.StatusPaymentRequired)
		return
	}

	// 3. Rate Limit (Local-first)
	rlResult, err := h.limiter.Allow(ctx, tenant.ID, tenant.RateLimitRPM)
	if err != nil || !rlResult.Allowed {
		w.Header().Set("Retry-After", "1")
		http.Error(w, "rate_limited", http.StatusTooManyRequests)
		return
	}

	// 4. Parse Request
	var chatReq provider.ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&chatReq); err != nil {
		http.Error(w, "invalid_request", http.StatusBadRequest)
		return
	}

	// 5. Cache Check (L1/L2)
	cacheKey := fmt.Sprintf("cache:%s:%s:%x", tenant.ID, chatReq.Model, sha256.Sum256([]byte(fmt.Sprintf("%v", chatReq.Messages))))
	if !chatReq.Stream {
		if cached, _ := h.cache.Get(ctx, cacheKey); cached != nil {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-Cache", "HIT")
			w.Write(cached)
			return
		}
	}

	// 6. Routing & Dispatch (with Allowlists & Resilience)
	p := h.selector.Select(h.providers, tenant.ProviderAllowlist)

	if p == nil {
		http.Error(w, "no_available_providers", http.StatusServiceUnavailable)
		return
	}

	if chatReq.Stream {
		h.handleStream(w, r, p, chatReq, tenant, start)
	} else {
		h.handleSync(w, r, p, chatReq, tenant, start, cacheKey)
	}
}

func (h *GatewayHandler) handleSync(w http.ResponseWriter, r *http.Request, p provider.Provider, req provider.ChatRequest, tenant *db.Tenant, start time.Time, cacheKey string) {
	resp, err := p.Chat(r.Context(), req)
	latency := time.Since(start)

	if err != nil {
		h.monitor.RecordError(p.Name(), http.StatusInternalServerError)
		http.Error(w, "provider_error", http.StatusBadGateway)
		return
	}

	h.monitor.RecordSuccess(p.Name(), latency)

	logEvent := logging.LogEvent{
		TenantID:         tenant.ID,
		Provider:         p.Name(),
		Model:            resp.Model,
		PromptTokens:     resp.PromptTokens,
		CompletionTokens: resp.CompletionTokens,
		CostCents:        provider.EstimateCost(resp.Model, resp.PromptTokens, resp.CompletionTokens),
		LatencyMs:        int(latency.Milliseconds()),
		Status:           200,
		Timestamp:        time.Now().UnixNano(),
	}

	// Async Logging (Kafka)
	h.logger.Log(r.Context(), logEvent)
	// Real-time Broadcast (Redis Sampled)
	h.logger.Broadcast(r.Context(), logEvent)

	// Set Cache
	respBody, _ := json.Marshal(resp)
	h.cache.Set(r.Context(), cacheKey, respBody, time.Hour)

	w.Header().Set("Content-Type", "application/json")
	w.Write(respBody)
}

func (h *GatewayHandler) handleStream(w http.ResponseWriter, r *http.Request, p provider.Provider, req provider.ChatRequest, tenant *db.Tenant, start time.Time) {
	chunks, errs := p.ChatStream(r.Context(), req)

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, _ := w.(http.Flusher)

	var totalPromptTokens, totalCompletionTokens int

	for {
		select {
		case chunk, ok := <-chunks:
			if !ok {
				// Final log
				logEvent := logging.LogEvent{
					TenantID:         tenant.ID,
					Provider:         p.Name(),
					Model:            req.Model,
					PromptTokens:     totalPromptTokens,
					CompletionTokens: totalCompletionTokens,
					LatencyMs:        int(time.Since(start).Milliseconds()),
					Status:           200,
					Timestamp:        time.Now().UnixNano(),
				}
				h.logger.Log(r.Context(), logEvent)
				h.logger.Broadcast(r.Context(), logEvent)
				return
			}
			totalPromptTokens += chunk.PromptTokens
			totalCompletionTokens += chunk.CompletionTokens

			fmt.Fprintf(w, "data: %s\n\n", chunk.Content)
			flusher.Flush()
		case err := <-errs:
			if err != nil {
				h.monitor.RecordError(p.Name(), 502)
				// In a real SSE stream, we'd send an error event
				return
			}
		case <-r.Context().Done():
			return
		}
	}
}
