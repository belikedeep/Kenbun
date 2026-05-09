package provider

import (
	"context"
	"errors"
	"io"

	"github.com/sashabaranov/go-openai"
)

type OpenAIProvider struct {
	client *openai.Client
}

func NewOpenAIProvider(apiKey string) *OpenAIProvider {
	return &OpenAIProvider{
		client: openai.NewClient(apiKey),
	}
}

func (p *OpenAIProvider) Name() string {
	return "openai"
}

func (p *OpenAIProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	messages := make([]openai.ChatCompletionMessage, len(req.Messages))
	for i, m := range req.Messages {
		messages[i] = openai.ChatCompletionMessage{
			Role:    m.Role,
			Content: m.Content,
		}
	}

	resp, err := p.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:    req.Model,
		Messages: messages,
		Stream:   false,
	})
	if err != nil {
		return nil, err
	}

	return &ChatResponse{
		Content:          resp.Choices[0].Message.Content,
		Model:            resp.Model,
		Provider:         "openai",
		PromptTokens:     resp.Usage.PromptTokens,
		CompletionTokens: resp.Usage.CompletionTokens,
	}, nil
}

func (p *OpenAIProvider) ChatStream(ctx context.Context, req ChatRequest) (<-chan ChatChunk, <-chan error) {
	chunkChan := make(chan ChatChunk)
	errChan := make(chan error, 1)

	messages := make([]openai.ChatCompletionMessage, len(req.Messages))
	for i, m := range req.Messages {
		messages[i] = openai.ChatCompletionMessage{
			Role:    m.Role,
			Content: m.Content,
		}
	}

	stream, err := p.client.CreateChatCompletionStream(ctx, openai.ChatCompletionRequest{
		Model:    req.Model,
		Messages: messages,
		Stream:   true,
	})
	if err != nil {
		errChan <- err
		close(chunkChan)
		close(errChan)
		return chunkChan, errChan
	}

	go func() {
		defer stream.Close()
		defer close(chunkChan)
		defer close(errChan)

		for {
			response, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				return
			}
			if err != nil {
				errChan <- err
				return
			}

			if len(response.Choices) > 0 {
				chunkChan <- ChatChunk{
					Content:      response.Choices[0].Delta.Content,
					FinishReason: string(response.Choices[0].FinishReason),
					// OpenAI streaming usage is usually only in the last chunk or requires specific flags
				}
			}
		}
	}()

	return chunkChan, errChan
}
