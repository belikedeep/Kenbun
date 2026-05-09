// Copyright 2026 Deepak Mardi. Licensed under Apache 2.0.

package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type AnthropicProvider struct {
	apiKey string
	client *http.Client
}

func NewAnthropicProvider(apiKey string, client *http.Client) *AnthropicProvider {
	return &AnthropicProvider{
		apiKey: apiKey,
		client: client,
	}
}

func (p *AnthropicProvider) Name() string {
	return "anthropic"
}

func (p *AnthropicProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	// Anthropic API uses a different schema. 
	// This is a simplified implementation for the initial version.
	payload := map[string]interface{}{
		"model":      req.Model,
		"messages":   req.Messages,
		"max_tokens": 1024,
	}

	body, _ := json.Marshal(payload)
	httpReq, _ := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(body))
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	httpReq.Header.Set("content-type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("anthropic error: status %d", resp.StatusCode)
	}

	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &ChatResponse{
		Content:          result.Content[0].Text,
		Model:            req.Model,
		Provider:         "anthropic",
		PromptTokens:     result.Usage.InputTokens,
		CompletionTokens: result.Usage.OutputTokens,
	}, nil
}

func (p *AnthropicProvider) ChatStream(ctx context.Context, req ChatRequest) (<-chan ChatChunk, <-chan error) {
	// Streaming implementation for Anthropic would involve parsing SSE events manually.
	// For this demo turn, I'll return an error or a dummy stream.
	chunkChan := make(chan ChatChunk)
	errChan := make(chan error, 1)
	errChan <- fmt.Errorf("anthropic streaming not implemented in this re-arch turn")
	close(chunkChan)
	close(errChan)
	return chunkChan, errChan
}
