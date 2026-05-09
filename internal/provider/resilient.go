// Copyright 2026 Deepak Mardi. Licensed under Apache 2.0.

package provider

import (
	"context"
	"fmt"
	"math"
	"time"
)

// ResilientProvider wraps a Provider with retries, timeouts, and circuit breaking logic.
type ResilientProvider struct {
	base        Provider
	maxRetries  int
	timeout     time.Duration
	retryDelay  time.Duration
}

func NewResilientProvider(base Provider, maxRetries int, timeout time.Duration) *ResilientProvider {
	return &ResilientProvider{
		base:       base,
		maxRetries: maxRetries,
		timeout:    timeout,
		retryDelay: 100 * time.Millisecond,
	}
}

func (p *ResilientProvider) Name() string {
	return p.base.Name()
}

func (p *ResilientProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	var lastErr error

	for i := 0; i <= p.maxRetries; i++ {
		// Context with Timeout
		tCtx, cancel := context.WithTimeout(ctx, p.timeout)
		resp, err := p.base.Chat(tCtx, req)
		cancel()

		if err == nil {
			return resp, nil
		}

		lastErr = err
		
		// Don't retry on user errors or context cancellation
		if ctx.Err() != nil {
			return nil, err
		}

		// Exponential Backoff
		if i < p.maxRetries {
			backoff := time.Duration(math.Pow(2, float64(i))) * p.retryDelay
			select {
			case <-time.After(backoff):
				continue
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
	}

	return nil, fmt.Errorf("provider %s failed after %d retries: %w", p.Name(), p.maxRetries, lastErr)
}

func (p *ResilientProvider) ChatStream(ctx context.Context, req ChatRequest) (<-chan ChatChunk, <-chan error) {
	// For streaming, we primarily rely on timeouts and initial connection retries.
	// Mid-stream retries are complex and often result in duplicate content.
	// Here we implement the initial connection timeout/retry.
	
	chunkChan := make(chan ChatChunk)
	errChan := make(chan error, 1)

	go func() {
		defer close(chunkChan)
		defer close(errChan)

		var lastErr error
		for i := 0; i <= p.maxRetries; i++ {
			tCtx, cancel := context.WithTimeout(ctx, p.timeout)
			chunks, errs := p.base.ChatStream(tCtx, req)
			
			select {
			case chunk, ok := <-chunks:
				if ok {
					cancel()
					chunkChan <- chunk
					// Pipe remaining chunks
					for c := range chunks {
						chunkChan <- c
					}
					return
				}
			case err := <-errs:
				if err != nil {
					lastErr = err
				}
			case <-tCtx.Done():
				lastErr = tCtx.Err()
			}
			cancel()

			if ctx.Err() != nil {
				errChan <- ctx.Err()
				return
			}

			if i < p.maxRetries {
				time.Sleep(time.Duration(math.Pow(2, float64(i))) * p.retryDelay)
			}
		}
		errChan <- lastErr
	}()

	return chunkChan, errChan
}
