// Copyright 2026 Deepak Mardi. Licensed under Apache 2.0.

package router

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/belikedeep/kenbun/internal/db"
	"github.com/belikedeep/kenbun/internal/logging"
)

type AdminHandler struct {
	db          *db.Client
	ch          *logging.ClickHouseClient
	monitor     HealthMonitor
	adminSecret string
}

func NewAdminHandler(db *db.Client, ch *logging.ClickHouseClient, monitor HealthMonitor, secret string) *AdminHandler {
	return &AdminHandler{
		db:          db,
		ch:          ch,
		monitor:     monitor,
		adminSecret: secret,
	}
}

func (h *AdminHandler) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("X-Admin-Token")
		if token == "" {
			token = r.URL.Query().Get("token")
		}

		if h.adminSecret == "" || token != h.adminSecret {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (h *AdminHandler) RegisterRoutes(r chi.Router) {
	r.Use(h.AuthMiddleware)
	r.Get("/tenants", h.listTenants)
	r.Post("/tenants", h.createTenant)
	r.Get("/tenants/{id}/stats", h.getTenantStats)
	r.Get("/stats", h.getStats)
	r.Get("/charts", h.getCharts)
	r.Get("/logs/stream", h.streamLogs)
	r.Post("/providers/{name}/fail", h.failProvider)
}

func (h *AdminHandler) streamLogs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Subscribe to Redis
	pubsub := h.ch.GetRedis().Subscribe(r.Context(), "logs:live")
	defer pubsub.Close()

	ch := pubsub.Channel()

	for {
		select {
		case msg := <-ch:
			fmt.Fprintf(w, "data: %s\n\n", msg.Payload)
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

func (h *AdminHandler) jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func (h *AdminHandler) listTenants(w http.ResponseWriter, r *http.Request) {
	tenants, err := h.db.GetAllTenants(r.Context())
	if err != nil {
		h.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tenants)
}

func (h *AdminHandler) createTenant(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name              string   `json:"name"`
		RateLimitRPM      int      `json:"rate_limit_rpm"`
		BudgetCents       int      `json:"budget_cents"`
		APIKey            string   `json:"api_key"`
		ProviderAllowlist []string `json:"provider_allowlist"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	hash := sha256.Sum256([]byte(req.APIKey))
	keyHash := hex.EncodeToString(hash[:])

	tenant, err := h.db.CreateTenant(r.Context(), req.Name, keyHash, req.RateLimitRPM, req.BudgetCents, req.ProviderAllowlist)
	if err != nil {
		h.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tenant)
}

func (h *AdminHandler) getStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.ch.GetGlobalStats(r.Context())
	if err != nil {
		fmt.Printf("GetGlobalStats failed: %v\n", err)
		h.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Printf("Stats retrieved: %+v\n", stats)
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(stats); err != nil {
		fmt.Printf("Encoding failed: %v\n", err)
	}
}

func (h *AdminHandler) getCharts(w http.ResponseWriter, r *http.Request) {
	charts, err := h.ch.GetChartData(r.Context())
	if err != nil {
		h.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(charts)
}

func (h *AdminHandler) getTenantStats(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	stats, err := h.ch.GetTenantStats(r.Context(), id)
	if err != nil {
		h.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (h *AdminHandler) failProvider(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	h.monitor.ForceFailure(name)
	w.WriteHeader(http.StatusNoContent)
}
