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
