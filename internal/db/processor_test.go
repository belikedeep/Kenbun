// Copyright 2026 Deepak Mardi. Licensed under Apache 2.0.

package db

import (
	"context"
	"testing"
)

type mockTenantRepo struct {
	updates map[string]int
}

func (m *mockTenantRepo) GetTenantByAPIKeyHash(ctx context.Context, hash string) (*Tenant, error) {
	return nil, nil
}

func (m *mockTenantRepo) GetAllTenants(ctx context.Context) ([]Tenant, error) {
	return nil, nil
}

func (m *mockTenantRepo) CreateTenant(ctx context.Context, name, keyHash string, rpm, budget int, allowlist []string) (*Tenant, error) {
	return nil, nil
}

func (m *mockTenantRepo) UpdateTenantSpend(ctx context.Context, tenantID string, cents int) error {
	if m.updates == nil {
		m.updates = make(map[string]int)
	}
	m.updates[tenantID] += cents
	return nil
}

func (m *mockTenantRepo) Close() {}

func TestBudgetProcessor_Flush(t *testing.T) {
	repo := &mockTenantRepo{}
	
	processor := &BudgetProcessor{
		db:      repo,
		pending: make(map[string]float64),
	}

	processor.pending["tenant-1"] = 100.5
	processor.pending["tenant-2"] = 200.0

	processor.flush(context.Background())

	if len(processor.pending) != 0 {
		t.Errorf("Expected pending to be empty, got %d", len(processor.pending))
	}

	if repo.updates["tenant-1"] != 100 {
		t.Errorf("Expected tenant-1 to have 100 updates, got %d", repo.updates["tenant-1"])
	}
	if repo.updates["tenant-2"] != 200 {
		t.Errorf("Expected tenant-2 to have 200 updates, got %d", repo.updates["tenant-2"])
	}
}
