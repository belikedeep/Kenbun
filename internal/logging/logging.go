// Copyright 2026 Deepak Mardi. Licensed under Apache 2.0.

package logging

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
)

// LogEvent represents a single request/response log record.
type LogEvent struct {
	RequestID        string  `json:"request_id"`
	TenantID         string  `json:"tenant_id"`
	Provider         string  `json:"provider"`
	Model            string  `json:"model"`
	PromptTokens     int     `json:"prompt_tokens"`
	CompletionTokens int     `json:"completion_tokens"`
	CostCents        float64 `json:"cost_cents"`
	LatencyMs        int     `json:"latency_ms"`
	Status           int     `json:"status"`
	Cached           bool    `json:"cached"`
	Timestamp        int64   `json:"timestamp"` // Unix timestamp in nanos
}

// Logger defines the system contract for high-throughput log ingestion and real-time broadcasting.
type Logger interface {
	Log(ctx context.Context, event LogEvent) error
	Broadcast(ctx context.Context, event LogEvent) error
	Close() error
}

// MultiLogger handles both persistent storage (Kafka) and real-time streaming (Redis).
type MultiLogger struct {
	kafka *kafka.Writer
	redis *redis.ClusterClient
}

func NewMultiLogger(brokers []string, topic string, rdb *redis.ClusterClient) *MultiLogger {
	return &MultiLogger{
		kafka: &kafka.Writer{
			Addr:         kafka.TCP(brokers...),
			Topic:        topic,
			Balancer:     &kafka.LeastBytes{},
			Async:        true,
			BatchSize:    1000,
			BatchTimeout: 10 * time.Millisecond,
		},
		redis: rdb,
	}
}

func (l *MultiLogger) Log(ctx context.Context, event LogEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal log event: %w", err)
	}

	return l.kafka.WriteMessages(ctx, kafka.Message{
		Key:   []byte(event.TenantID),
		Value: payload,
	})
}

func (l *MultiLogger) Broadcast(ctx context.Context, event LogEvent) error {
	// 100k RPS Safety: Sampling Logic
	// 1. Always broadcast errors (4xx/5xx)
	// 2. Sample successes at 0.1% to prevent dashboard meltdown
	shouldBroadcast := event.Status >= 400 || (time.Now().UnixNano()%1000 == 0)

	if !shouldBroadcast {
		return nil
	}

	payload, _ := json.Marshal(event)
	return l.redis.Publish(ctx, "logs:live", payload).Err()
}

func (l *MultiLogger) Close() error {
	return l.kafka.Close()
}
