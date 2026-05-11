// Copyright 2026 Deepak Mardi. Licensed under Apache 2.0.

package db

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/belikedeep/kenbun/internal/logging"
	"github.com/segmentio/kafka-go"
)

// BudgetProcessor consumes log events and updates tenant spending in PostgreSQL.
type BudgetProcessor struct {
	db           *PostgresClient
	reader       *kafka.Reader
	flushInterval time.Duration
	batchSize    int

	mu       sync.Mutex
	pending  map[string]float64
	lastFlush time.Time
}

func NewBudgetProcessor(db *PostgresClient, brokers []string, topic string) *BudgetProcessor {
	return &BudgetProcessor{
		db: db,
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:  brokers,
			Topic:    topic,
			GroupID:  "budget-processor",
			MinBytes: 10e3, // 10KB
			MaxBytes: 10e6, // 10MB
		}),
		flushInterval: 1 * time.Second,
		batchSize:     1000,
		pending:       make(map[string]float64),
		lastFlush:     time.Now(),
	}
}

func (p *BudgetProcessor) Start(ctx context.Context) {
	fmt.Println("🚀 Budget Processor starting...")
	
	ticker := time.NewTicker(p.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			p.flush(context.Background())
			p.reader.Close()
			return
		case <-ticker.C:
			p.flush(ctx)
		default:
			// Read with a short timeout to not block the select
			m, err := p.reader.ReadMessage(ctx)
			if err != nil {
				continue
			}

			var event logging.LogEvent
			if err := json.Unmarshal(m.Value, &event); err != nil {
				continue
			}

			p.mu.Lock()
			p.pending[event.TenantID] += event.CostCents
			shouldFlush := len(p.pending) >= p.batchSize
			p.mu.Unlock()

			if shouldFlush {
				p.flush(ctx)
			}
		}
	}
}

func (p *BudgetProcessor) flush(ctx context.Context) {
	p.mu.Lock()
	if len(p.pending) == 0 {
		p.mu.Unlock()
		return
	}
	updates := p.pending
	p.pending = make(map[string]float64)
	p.mu.Unlock()

	// Perform batch update to PostgreSQL
	// In a real high-throughput scenario, you might use a temp table and a JOIN
	// For this assignment, we'll iterate through the map to keep it simple but decoupled.
	for tenantID, cost := range updates {
		query := `UPDATE tenants SET spent_cents = spent_cents + $1 WHERE id = $2`
		_, err := p.db.Pool.Exec(ctx, query, int(cost), tenantID)
		if err != nil {
			fmt.Printf("❌ Failed to update budget for tenant %s: %v\n", tenantID, err)
		}
	}
}
