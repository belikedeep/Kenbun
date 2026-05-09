// Copyright 2026 Deepak Mardi. Licensed under Apache 2.0.

package router

import (
	"testing"
	"time"
)

func TestEWMAMonitor(t *testing.T) {
	m := NewEWMAMonitor()

	// 1. Initial State
	if m.GetState("p1") != StateHealthy {
		t.Error("expected initial state to be healthy")
	}

	// 2. Record Latency
	m.RecordSuccess("p1", 100*time.Millisecond)
	pm := m.getMetrics("p1")
	if pm.latencyEWMA != 100 {
		t.Errorf("expected latency EWMA to be 100, got %f", pm.latencyEWMA)
	}

	// 3. Record Error (Trigger Unhealthy)
	for i := 0; i < 10; i++ {
		m.RecordError("p1", 500)
	}
	if m.GetState("p1") != StateUnhealthy {
		t.Errorf("expected state to be unhealthy, got %s", m.GetState("p1"))
	}

	// 4. Recover on Success
	for i := 0; i < 20; i++ {
		m.RecordSuccess("p1", 50*time.Millisecond)
	}
	if m.GetState("p1") != StateHealthy {
		t.Errorf("expected state to recover to healthy, got %s", m.GetState("p1"))
	}
}
