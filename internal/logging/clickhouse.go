// Copyright 2026 Deepak Mardi. Licensed under Apache 2.0.

package logging

import (
	"context"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
)

type GlobalStats struct {
	TotalRequests int     `json:"total_requests"`
	TotalTokens   int     `json:"total_tokens"`
	TotalCost     float64 `json:"total_cost"`
	AvgLatency    float64 `json:"avg_latency_ms"`
}

type ChartPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Requests  int       `json:"requests"`
	Cost      float64   `json:"cost"`
}

type ClickHouseClient struct {
	conn clickhouse.Conn
}

func NewClickHouseClient(addr string) (*ClickHouseClient, error) {
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{addr},
		Auth: clickhouse.Auth{
			Database: "default",
			Username: "default",
			Password: "",
		},
	})
	if err != nil {
		return nil, err
	}
	return &ClickHouseClient{conn: conn}, nil
}

func (c *ClickHouseClient) GetGlobalStats(ctx context.Context) (*GlobalStats, error) {
	var stats GlobalStats
	query := `SELECT 
                count(), 
                sum(prompt_tokens + completion_tokens), 
                sum(cost_cents) / 100, 
                avg(latency_ms) 
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
                count(), 
                sum(cost_cents) / 100 
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
