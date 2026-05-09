// Copyright 2026 Deepak Mardi. Licensed under Apache 2.0.

package router

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/belikedeep/kenbun/internal/db"
	"github.com/belikedeep/kenbun/internal/logging"
)

type AdminHandler struct {
	db      *db.Client
	ch      *logging.ClickHouseClient
}

func NewAdminHandler(db *db.Client, ch *logging.ClickHouseClient) *AdminHandler {
	return &AdminHandler{
		db: db,
		ch: ch,
	}
}

func (h *AdminHandler) RegisterRoutes(r chi.Router) {
	r.Get("/tenants", h.listTenants)
	r.Post("/tenants", h.createTenant)
	r.Get("/stats", h.getStats)
	r.Get("/charts", h.getCharts)
}

func (h *AdminHandler) listTenants(w http.ResponseWriter, r *http.Request) {
	tenants, err := h.db.GetAllTenants(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tenants)
}

func (h *AdminHandler) createTenant(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name         string `json:"name"`
		RateLimitRPM int    `json:"rate_limit_rpm"`
		BudgetCents  int    `json:"budget_cents"`
		APIKey       string `json:"api_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	hash := sha256.Sum256([]byte(req.APIKey))
	keyHash := hex.EncodeToString(hash[:])

	tenant, err := h.db.CreateTenant(r.Context(), req.Name, keyHash, req.RateLimitRPM, req.BudgetCents)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tenant)
}

func (h *AdminHandler) getStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.ch.GetGlobalStats(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (h *AdminHandler) getCharts(w http.ResponseWriter, r *http.Request) {
	charts, err := h.ch.GetChartData(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(charts)
}
