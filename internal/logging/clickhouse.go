// Copyright 2026 Deepak Mardi. Licensed under Apache 2.0.

package logging

import (
	"context"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/redis/go-redis/v9"
)

type GlobalStats struct {
	TotalRequests uint64  `json:"total_requests"`
	TotalTokens   uint64  `json:"total_tokens"`
	TotalCost     float64 `json:"total_cost"`
	AvgLatency    float64 `json:"avg_latency_ms"`
}

type ChartPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Requests  uint64    `json:"requests"`
	Cost      float64   `json:"cost"`
}

type TenantStats struct {
	TotalRequests uint64             `json:"total_requests"`
	TotalCost     float64            `json:"total_cost"`
	Models        map[string]uint64  `json:"models"`
}

type ClickHouseClient struct {
	conn  clickhouse.Conn
	redis *redis.ClusterClient
}

func NewClickHouseClient(addr string, rdb *redis.ClusterClient) (*ClickHouseClient, error) {
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{addr},
		Auth: clickhouse.Auth{
			Database: "default",
			Username: "gateway",
			Password: "gateway",
		},
		Settings: clickhouse.Settings{
			"max_execution_time": 60,
		},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return nil, err
	}
	return &ClickHouseClient{conn: conn, redis: rdb}, nil
}

func (c *ClickHouseClient) GetRedis() *redis.ClusterClient {
	return c.redis
}

func (c *ClickHouseClient) GetGlobalStats(ctx context.Context) (*GlobalStats, error) {
	var stats GlobalStats
	// Use ifNull and NaN checks to ensure valid JSON response
	query := `SELECT 
                toUInt64(count()), 
                toUInt64(ifNull(sum(prompt_tokens + completion_tokens), 0)), 
                toFloat64(ifNull(sum(cost_cents), 0) / 100), 
                toFloat64(if(isFinite(avg(latency_ms)), avg(latency_ms), 0)) 
              FROM gateway_logs`
	
	err := c.conn.QueryRow(ctx, query).Scan(
		&stats.TotalRequests, &stats.TotalTokens, &stats.TotalCost, &stats.AvgLatency,
	)
	if err != nil {
		return nil, err
	}
	return &stats, nil
}

func (c *ClickHouseClient) GetChartData(ctx context.Context) ([]ChartPoint, error) {
	query := `SELECT 
                toStartOfHour(timestamp) as ts, 
                toUInt64(count()), 
                toFloat64(ifNull(sum(cost_cents), 0) / 100)
              FROM gateway_logs 
              WHERE timestamp > now() - INTERVAL 24 HOUR 
              GROUP BY ts 
              ORDER BY ts`
	
	rows, err := c.conn.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var points []ChartPoint
	for rows.Next() {
		var p ChartPoint
		if err := rows.Scan(&p.Timestamp, &p.Requests, &p.Cost); err != nil {
			return nil, err
		}
		points = append(points, p)
	}
	return points, nil
}

func (c *ClickHouseClient) GetTenantStats(ctx context.Context, tenantID string) (*TenantStats, error) {
	stats := &TenantStats{Models: make(map[string]uint64)}
	
	query := `SELECT 
                toUInt64(count()), 
                toFloat64(ifNull(sum(cost_cents), 0) / 100)
              FROM gateway_logs 
              WHERE tenant_id = $1`
	
	err := c.conn.QueryRow(ctx, query, tenantID).Scan(&stats.TotalRequests, &stats.TotalCost)
	if err != nil {
		return nil, err
	}

	modelQuery := `SELECT model, toUInt64(count()) FROM gateway_logs WHERE tenant_id = $1 GROUP BY model`
	rows, err := c.conn.Query(ctx, modelQuery, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var model string
		var count uint64
		if err := rows.Scan(&model, &count); err != nil {
			return nil, err
		}
		stats.Models[model] = count
	}

	return stats, nil
}
