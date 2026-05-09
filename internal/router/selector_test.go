// Copyright 2026 Deepak Mardi. Licensed under Apache 2.0.

package router

import (
	"testing"
	"time"

	"github.com/belikedeep/kenbun/internal/provider"
)

func TestLatencyAwareSelector(t *testing.T) {
	monitor := NewEWMAMonitor()
	selector := NewLatencyAwareSelector(monitor)

	p1 := provider.NewMockProvider("p1")
	p2 := provider.NewMockProvider("p2")
	providers := []provider.Provider{p1, p2}

	// 1. Both healthy, prefer p1 (alphabetical if no latency data)
	// Actually alphabetical depends on iteration, but let's set latency.
	monitor.RecordSuccess("p1", 200*time.Millisecond)
	monitor.RecordSuccess("p2", 100*time.Millisecond)

	selected := selector.Select(providers, nil)
	if selected.Name() != "p2" {
		t.Errorf("expected p2 (faster), got %s", selected.Name())
	}

	// 2. p2 goes unhealthy, must pick p1
	for i := 0; i < 10; i++ {
		monitor.RecordError("p2", 500)
	}
	selected = selector.Select(providers, nil)
	if selected.Name() != "p1" {
		t.Errorf("expected p1 (only healthy), got %s", selected.Name())
	}

	// 3. Allowlist restriction
	selected = selector.Select(providers, []string{"p2"})
	if selected != nil {
		t.Error("expected nil selection because p2 is unhealthy and is the only allowed")
	}
}
