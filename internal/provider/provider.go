// Copyright 2026 Deepak Mardi. Licensed under Apache 2.0.

package provider

import (
	"context"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
}

type ChatResponse struct {
	Content          string `json:"content"`
	Model            string `json:"model"`
	Provider         string `json:"provider"`
	PromptTokens     int    `json:"prompt_tokens"`
	CompletionTokens int    `json:"completion_tokens"`
}

type ChatChunk struct {
	Content          string `json:"content"`
	FinishReason     string `json:"finish_reason"`
	PromptTokens     int    `json:"prompt_tokens"`     // Usually only in first or last chunk
	CompletionTokens int    `json:"completion_tokens"` // Usually cumulative or in last chunk
}

// Provider defines the interface for an upstream LLM backend.
type Provider interface {
	// Name returns the provider identifier (e.g., "openai", "anthropic").
	Name() string

	// Chat dispatches a non-streaming request.
	Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)

	// ChatStream dispatches a streaming request.
	// It returns a channel of chunks and a channel for errors.
	ChatStream(ctx context.Context, req ChatRequest) (<-chan ChatChunk, <-chan error)
}

// EstimateCost calculates the approximate cost in cents for a request.
func EstimateCost(model string, promptTokens, completionTokens int) float64 {
	// Simplified pricing (cents per 1k tokens)
	// GPT-4o: 0.5c / 1.5c
	// Claude 3 Sonnet: 0.3c / 1.5c
	// Default: 0.1c / 0.1c
	var promptRate, completionRate float64
	
	switch {
	case model == "gpt-4o":
		promptRate, completionRate = 0.5, 1.5
	case model == "claude-3-sonnet":
		promptRate, completionRate = 0.3, 1.5
	default:
		promptRate, completionRate = 0.1, 0.2
	}

	return (float64(promptTokens) * promptRate / 1000.0) + (float64(completionTokens) * completionRate / 1000.0)
}
