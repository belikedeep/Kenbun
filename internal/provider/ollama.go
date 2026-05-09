// Copyright 2026 Deepak Mardi. Licensed under Apache 2.0.

package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type OllamaProvider struct {
	host   string
	client *http.Client
}

func NewOllamaProvider(host string) *OllamaProvider {
	if host == "" {
		host = "http://localhost:11434"
	}
	return &OllamaProvider{
		host:   host,
		client: &http.Client{},
	}
}

func (p *OllamaProvider) Name() string {
	return "ollama"
}

func (p *OllamaProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	payload := map[string]interface{}{
		"model":    req.Model,
		"messages": req.Messages,
		"stream":   false,
	}

	body, _ := json.Marshal(payload)
	httpReq, _ := http.NewRequestWithContext(ctx, "POST", p.host+"/api/chat", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("ollama error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama returned status: %d", resp.StatusCode)
	}

	var result struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		PromptEvalCount int `json:"prompt_eval_count"`
		EvalCount       int `json:"eval_count"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &ChatResponse{
		Content:          result.Message.Content,
		Model:            req.Model,
		Provider:         "ollama",
		PromptTokens:     result.PromptEvalCount,
		CompletionTokens: result.EvalCount,
	}, nil
}

func (p *OllamaProvider) ChatStream(ctx context.Context, req ChatRequest) (<-chan ChatChunk, <-chan error) {
	chunkChan := make(chan ChatChunk)
	errChan := make(chan error, 1)

	payload := map[string]interface{}{
		"model":    req.Model,
		"messages": req.Messages,
		"stream":   true,
	}

	body, _ := json.Marshal(payload)
	httpReq, _ := http.NewRequestWithContext(ctx, "POST", p.host+"/api/chat", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")

	go func() {
		defer close(chunkChan)
		defer close(errChan)

		resp, err := p.client.Do(httpReq)
		if err != nil {
			errChan <- err
			return
		}
		defer resp.Body.Close()

		decoder := json.NewDecoder(resp.Body)
		for {
			var line struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
				Done            bool   `json:"done"`
				PromptEvalCount int    `json:"prompt_eval_count"`
				EvalCount       int    `json:"eval_count"`
			}

			if err := decoder.Decode(&line); err != nil {
				if err == io.EOF {
					break
				}
				errChan <- err
				return
			}

			chunkChan <- ChatChunk{
				Content:          line.Message.Content,
				PromptTokens:     line.PromptEvalCount,
				CompletionTokens: line.EvalCount,
			}

			if line.Done {
				break
			}
		}
	}()

	return chunkChan, errChan
}
