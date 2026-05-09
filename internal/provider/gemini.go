package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

type GeminiProvider struct {
	client *genai.Client
}

func NewGeminiProvider(ctx context.Context, apiKey string) (*GeminiProvider, error) {
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return nil, err
	}
	return &GeminiProvider{client: client}, nil
}

func (p *GeminiProvider) Name() string {
	return "gemini"
}

func (p *GeminiProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	model := p.client.GenerativeModel(req.Model)
	cs := model.StartChat()

	// Convert messages to history
	// Note: Gemini history excludes the last message which is sent via SendMessage
	var history []*genai.Content
	for i := 0; i < len(req.Messages)-1; i++ {
		m := req.Messages[i]
		role := "user"
		if m.Role == "assistant" || m.Role == "model" {
			role = "model"
		}
		history = append(history, &genai.Content{
			Role:  role,
			Parts: []genai.Part{genai.Text(m.Content)},
		})
	}
	cs.History = history

	lastMsg := req.Messages[len(req.Messages)-1].Content
	resp, err := cs.SendMessage(ctx, genai.Text(lastMsg))
	if err != nil {
		return nil, err
	}

	var content strings.Builder
	for _, cand := range resp.Candidates {
		for _, part := range cand.Content.Parts {
			fmt.Fprintf(&content, "%v", part)
		}
	}

	return &ChatResponse{
		Content:          content.String(),
		Model:            req.Model,
		Provider:         "gemini",
		PromptTokens:     int(resp.UsageMetadata.PromptTokenCount),
		CompletionTokens: int(resp.UsageMetadata.CandidatesTokenCount),
	}, nil
}

func (p *GeminiProvider) ChatStream(ctx context.Context, req ChatRequest) (<-chan ChatChunk, <-chan error) {
	chunkChan := make(chan ChatChunk)
	errChan := make(chan error, 1)

	model := p.client.GenerativeModel(req.Model)
	cs := model.StartChat()

	var history []*genai.Content
	for i := 0; i < len(req.Messages)-1; i++ {
		m := req.Messages[i]
		role := "user"
		if m.Role == "assistant" || m.Role == "model" {
			role = "model"
		}
		history = append(history, &genai.Content{
			Role:  role,
			Parts: []genai.Part{genai.Text(m.Content)},
		})
	}
	cs.History = history

	lastMsg := req.Messages[len(req.Messages)-1].Content
	iter := cs.SendMessageStream(ctx, genai.Text(lastMsg))

	go func() {
		defer close(chunkChan)
		defer close(errChan)

		for {
			resp, err := iter.Next()
			if err == iterator.Done {
				return
			}
			if err != nil {
				errChan <- err
				return
			}

			var content strings.Builder
			for _, cand := range resp.Candidates {
				for _, part := range cand.Content.Parts {
					fmt.Fprintf(&content, "%v", part)
				}
			}

			chunkChan <- ChatChunk{
				Content:      content.String(),
				PromptTokens: int(resp.UsageMetadata.PromptTokenCount),
				// Token counts are often cumulative in Gemini stream responses
				CompletionTokens: int(resp.UsageMetadata.CandidatesTokenCount),
			}
		}
	}()

	return chunkChan, errChan
}
