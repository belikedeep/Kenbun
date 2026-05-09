// Copyright 2026 Deepak Mardi. Licensed under Apache 2.0.

package router

import (
	"math"
	"sort"

	"github.com/belikedeep/kenbun/internal/provider"
)

// ProviderSelector defines the interface for selecting an upstream provider.
type ProviderSelector interface {
	Select(providers []provider.Provider, allowlist []string) provider.Provider
}

// LatencyAwareSelector selects the best provider based on EWMA latency and health.
type LatencyAwareSelector struct {
	monitor HealthMonitor
}

func NewLatencyAwareSelector(monitor HealthMonitor) *LatencyAwareSelector {
	return &LatencyAwareSelector{monitor: monitor}
}

func (s *LatencyAwareSelector) Select(providers []provider.Provider, allowlist []string) provider.Provider {
	var candidates []provider.Provider

	// 1. Filter by Allowlist and State
	for _, p := range providers {
		allowed := false
		if len(allowlist) == 0 {
			allowed = true
		} else {
			for _, a := range allowlist {
				if a == p.Name() {
					allowed = true
					break
				}
			}
		}

		if allowed && s.monitor.GetState(p.Name()) != StateUnhealthy {
			candidates = append(candidates, p)
		}
	}

	if len(candidates) == 0 {
		return nil
	}

	// 2. Sort by Latency (EWMA)
	sort.Slice(candidates, func(i, j int) bool {
		li := s.getLatency(candidates[i].Name())
		lj := s.getLatency(candidates[j].Name())
		return li < lj
	})

	return candidates[0]
}

func (s *LatencyAwareSelector) getLatency(name string) float64 {
	m, ok := s.monitor.(*EWMAMonitor)
	if !ok {
		return 0
	}
	
	pm := m.getMetrics(name)
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	if pm.latencyEWMA == 0 {
		return math.MaxFloat64 // Penalize unknown latency to favor explored paths
	}
	return pm.latencyEWMA
}
