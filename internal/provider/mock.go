// Copyright 2026 Deepak Mardi. Licensed under Apache 2.0.

package provider

import (
	"context"
	"fmt"
	"time"
)

type MockProvider struct {
	name string
}

func NewMockProvider(name string) *MockProvider {
	return &MockProvider{name: name}
}

func (p *MockProvider) Name() string {
	return p.name
}

func (p *MockProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	// Simulate some latency
	time.Sleep(100 * time.Millisecond)

	return &ChatResponse{
		Content:          fmt.Sprintf("[MOCK RESPONSE from %s] You said: %s", p.name, req.Messages[len(req.Messages)-1].Content),
		Model:            req.Model,
		Provider:         p.name,
		PromptTokens:     10,
		CompletionTokens: 20,
	}, nil
}

func (p *MockProvider) ChatStream(ctx context.Context, req ChatRequest) (<-chan ChatChunk, <-chan error) {
	chunkChan := make(chan ChatChunk)
	errChan := make(chan error, 1)

	go func() {
		defer close(chunkChan)
		defer close(errChan)

		response := fmt.Sprintf("[MOCK STREAM from %s] You said: %s", p.name, req.Messages[len(req.Messages)-1].Content)
		
		// Send prompt tokens in first chunk
		chunkChan <- ChatChunk{PromptTokens: 10}

		// Stream the response character by character
		for _, char := range response {
			select {
			case <-ctx.Done():
				return
			default:
				chunkChan <- ChatChunk{Content: string(char)}
				time.Sleep(20 * time.Millisecond)
			}
		}

		// Send completion tokens in last chunk
		chunkChan <- ChatChunk{CompletionTokens: 20, FinishReason: "stop"}
	}()

	return chunkChan, errChan
}
