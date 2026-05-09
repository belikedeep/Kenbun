// Copyright 2026 Deepak Mardi. Licensed under Apache 2.0.

package router

import (
	"sync"
	"time"
)

// ProviderState represents the health of an upstream provider node.
type ProviderState string

const (
	StateHealthy   ProviderState = "healthy"
	StateDegraded  ProviderState = "degraded"
	StateUnhealthy ProviderState = "unhealthy"
)

// HealthMonitor defines the system contract for local outlier detection.
type HealthMonitor interface {
	RecordSuccess(provider string, latency time.Duration)
	RecordError(provider string, statusCode int)
	GetState(provider string) ProviderState
}

type providerMetrics struct {
	mu           sync.RWMutex
	latencyEWMA  float64
	errorRate    float64
	lastUpdate   time.Time
	alpha        float64 // smoothing factor
}

// EWMAMonitor implements local outlier detection using Exponentially Weighted Moving Averages.
type EWMAMonitor struct {
	mu      sync.RWMutex
	metrics map[string]*providerMetrics
}

func NewEWMAMonitor() *EWMAMonitor {
	return &EWMAMonitor{
		metrics: make(map[string]*providerMetrics),
	}
}

func (m *EWMAMonitor) getMetrics(provider string) *providerMetrics {
	m.mu.Lock()
	defer m.mu.Unlock()

	if pm, ok := m.metrics[provider]; ok {
		return pm
	}

	pm := &providerMetrics{
		alpha:      0.1, // 10% weight to new samples
		lastUpdate: time.Now(),
	}
	m.metrics[provider] = pm
	return pm
}

func (m *EWMAMonitor) RecordSuccess(provider string, latency time.Duration) {
	pm := m.getMetrics(provider)
	pm.mu.Lock()
	defer pm.mu.Unlock()

	ms := float64(latency.Milliseconds())
	if pm.latencyEWMA == 0 {
		pm.latencyEWMA = ms
	} else {
		pm.latencyEWMA = (pm.alpha * ms) + ((1 - pm.alpha) * pm.latencyEWMA)
	}
	
	// Fade error rate on success
	pm.errorRate = (1 - pm.alpha) * pm.errorRate
}

func (m *EWMAMonitor) RecordError(provider string, statusCode int) {
	pm := m.getMetrics(provider)
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Update error rate (1.0 = failure, 0.0 = success)
	pm.errorRate = (pm.alpha * 1.0) + ((1 - pm.alpha) * pm.errorRate)
}

func (m *EWMAMonitor) GetState(provider string) ProviderState {
	pm := m.getMetrics(provider)
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if pm.errorRate > 0.5 {
		return StateUnhealthy
	}
	if pm.errorRate > 0.2 || pm.latencyEWMA > 5000 {
		return StateDegraded
	}
	return StateHealthy
}
