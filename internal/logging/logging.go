package logging

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

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

// Logger defines the system contract for high-throughput log ingestion.
type Logger interface {
	Log(ctx context.Context, event LogEvent) error
	Close() error
}

// KafkaLogger implements the Logger interface using Kafka.
type KafkaLogger struct {
	writer *kafka.Writer
}

func NewKafkaLogger(brokers []string, topic string) *KafkaLogger {
	return &KafkaLogger{
		writer: &kafka.Writer{
			Addr:     kafka.TCP(brokers...),
			Topic:    topic,
			Balancer: &kafka.LeastBytes{},
			// Optimized for high-throughput
			Async:        true,
			BatchSize:    1000,
			BatchTimeout: 10 * time.Millisecond,
		},
	}
}

func (l *KafkaLogger) Log(ctx context.Context, event LogEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal log event: %w", err)
	}

	err = l.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(event.TenantID),
		Value: payload,
	})

	if err != nil {
		return fmt.Errorf("failed to write message to kafka: %w", err)
	}

	return nil
}

func (l *KafkaLogger) Close() error {
	return l.writer.Close()
}
