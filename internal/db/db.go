// Copyright 2026 Deepak Mardi. Licensed under Apache 2.0.

package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Tenant struct {
	ID                string   `json:"id"`
	Name              string   `json:"name"`
	APIKeyHash        string   `json:"api_key_hash"`
	RateLimitRPM      int      `json:"rate_limit_rpm"`
	BudgetCents       int      `json:"budget_cents"`
	SpentCents        int      `json:"spent_cents"`
	ProviderAllowlist []string `json:"provider_allowlist"`
	IsActive          bool     `json:"is_active"`
}

type TenantRepository interface {
	GetTenantByAPIKeyHash(ctx context.Context, hash string) (*Tenant, error)
	GetAllTenants(ctx context.Context) ([]Tenant, error)
	CreateTenant(ctx context.Context, name, keyHash string, rpm, budget int, allowlist []string) (*Tenant, error)
	UpdateTenantSpend(ctx context.Context, tenantID string, cents int) error
	Close()
}

type PostgresClient struct {
	Pool *pgxpool.Pool
}

func New(ctx context.Context, connString string) (TenantRepository, error) {
	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("unable to parse database url: %w", err)
	}

	// Optimized for High-Throughput Architecture
	config.MaxConns = 50
	config.MinConns = 10

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	return &PostgresClient{Pool: pool}, nil
}

func (c *PostgresClient) Close() {
	c.Pool.Close()
}

func (c *PostgresClient) UpdateTenantSpend(ctx context.Context, tenantID string, cents int) error {
	query := `UPDATE tenants SET spent_cents = spent_cents + $1 WHERE id = $2`
	_, err := c.Pool.Exec(ctx, query, cents, tenantID)
	return err
}

func (c *PostgresClient) GetTenantByAPIKeyHash(ctx context.Context, hash string) (*Tenant, error) {
	var t Tenant
	query := `SELECT id, name, api_key_hash, rate_limit_rpm, budget_cents, spent_cents, provider_allowlist, is_active 
              FROM tenants WHERE api_key_hash = $1 LIMIT 1`
	
	err := c.Pool.QueryRow(ctx, query, hash).Scan(
		&t.ID, &t.Name, &t.APIKeyHash, &t.RateLimitRPM, &t.BudgetCents, &t.SpentCents, &t.ProviderAllowlist, &t.IsActive,
	)
	if err != nil {
		return nil, err
	}

	return &t, nil
}

func (c *PostgresClient) GetAllTenants(ctx context.Context) ([]Tenant, error) {
	query := `SELECT id, name, api_key_hash, rate_limit_rpm, budget_cents, spent_cents, provider_allowlist, is_active FROM tenants ORDER BY created_at DESC`
	rows, err := c.Pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tenants []Tenant
	for rows.Next() {
		var t Tenant
		if err := rows.Scan(&t.ID, &t.Name, &t.APIKeyHash, &t.RateLimitRPM, &t.BudgetCents, &t.SpentCents, &t.ProviderAllowlist, &t.IsActive); err != nil {
			return nil, err
		}
		tenants = append(tenants, t)
	}
	return tenants, nil
}

func (c *PostgresClient) CreateTenant(ctx context.Context, name, keyHash string, rpm, budget int, allowlist []string) (*Tenant, error) {
	query := `INSERT INTO tenants (name, api_key_hash, rate_limit_rpm, budget_cents, provider_allowlist) 
              VALUES ($1, $2, $3, $4, $5) 
              RETURNING id, name, api_key_hash, rate_limit_rpm, budget_cents, spent_cents, provider_allowlist, is_active`
	
	var t Tenant
	err := c.Pool.QueryRow(ctx, query, name, keyHash, rpm, budget, allowlist).Scan(
		&t.ID, &t.Name, &t.APIKeyHash, &t.RateLimitRPM, &t.BudgetCents, &t.SpentCents, &t.ProviderAllowlist, &t.IsActive,
	)
	if err != nil {
		return nil, err
	}
	return &t, nil
}
