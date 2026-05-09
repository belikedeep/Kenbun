// Copyright 2026 Deepak Mardi. Licensed under Apache 2.0.

package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Tenant struct {
	ID           string
	Name         string
	APIKeyHash   string
	RateLimitRPM int
	BudgetCents  int
	SpentCents   int
	IsActive     bool
}

type Client struct {
	Pool *pgxpool.Pool
}

func New(ctx context.Context, connString string) (*Client, error) {
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

	return &Client{Pool: pool}, nil
}

func (c *Client) Close() {
	c.Pool.Close()
}

func (c *Client) GetTenantByAPIKeyHash(ctx context.Context, hash string) (*Tenant, error) {
	var t Tenant
	query := `SELECT id, name, api_key_hash, rate_limit_rpm, budget_cents, spent_cents, is_active 
              FROM tenants WHERE api_key_hash = $1 LIMIT 1`
	
	err := c.Pool.QueryRow(ctx, query, hash).Scan(
		&t.ID, &t.Name, &t.APIKeyHash, &t.RateLimitRPM, &t.BudgetCents, &t.SpentCents, &t.IsActive,
	)
	if err != nil {
		return nil, err
	}

	return &t, nil
}

func (c *Client) GetAllTenants(ctx context.Context) ([]Tenant, error) {
	query := `SELECT id, name, api_key_hash, rate_limit_rpm, budget_cents, spent_cents, is_active FROM tenants ORDER BY created_at DESC`
	rows, err := c.Pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tenants []Tenant
	for rows.Next() {
		var t Tenant
		if err := rows.Scan(&t.ID, &t.Name, &t.APIKeyHash, &t.RateLimitRPM, &t.BudgetCents, &t.SpentCents, &t.IsActive); err != nil {
			return nil, err
		}
		tenants = append(tenants, t)
	}
	return tenants, nil
}

func (c *Client) CreateTenant(ctx context.Context, name, keyHash string, rpm, budget int) (*Tenant, error) {
	query := `INSERT INTO tenants (name, api_key_hash, rate_limit_rpm, budget_cents) 
              VALUES ($1, $2, $3, $4) 
              RETURNING id, name, api_key_hash, rate_limit_rpm, budget_cents, spent_cents, is_active`
	
	var t Tenant
	err := c.Pool.QueryRow(ctx, query, name, keyHash, rpm, budget).Scan(
		&t.ID, &t.Name, &t.APIKeyHash, &t.RateLimitRPM, &t.BudgetCents, &t.SpentCents, &t.IsActive,
	)
	if err != nil {
		return nil, err
	}
	return &t, nil
}
