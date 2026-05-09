package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// This script simulates a high-throughput load on the Kenbun Gateway.
// It uses the pre-seeded "Test Tenant" key to prove the systems logic (Caching, RL, Logging).

func main() {
	gatewayURL := "http://localhost:8080/v1/chat/completions"
	apiKey := "sk-kb-iux3n2bgcjm"

	models := []string{"gpt-4o", "claude-3-sonnet", "gemini-1.5-pro"}
	prompts := []string{
		"What is Observation Haki?",
		"Explain Kafka in one sentence.",
		"How do I scale a Go application?",
		"Tell me a joke about Redis.",
	}

	fmt.Println("👁️ Kenbun Haki Simulator Starting...")
	fmt.Printf("Target: %s\n\n", gatewayURL)

	// Create a context that is cancelled when an interrupt signal is received
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var wg sync.WaitGroup

	// Initial delay
	nextDelay := func() time.Duration {
		return time.Duration(rand.Intn(450)+50) * time.Millisecond
	}

	timer := time.NewTimer(nextDelay())

	for {
		select {
		case <-ctx.Done():
			fmt.Println("\n🛑 Stopping simulator...")
			goto done
		case <-timer.C:
			wg.Add(1)
			go func() {
				defer wg.Done()
				model := models[rand.Intn(len(models))]
				prompt := prompts[rand.Intn(len(prompts))]

				// Occasionally use the same prompt to trigger Caching
				if rand.Float64() < 0.3 {
					prompt = prompts[0]
				}

				payload := map[string]interface{}{
					"model": model,
					"messages": []map[string]string{
						{"role": "user", "content": prompt},
					},
					"stream": false,
				}

				body, _ := json.Marshal(payload)
				req, _ := http.NewRequestWithContext(ctx, "POST", gatewayURL, bytes.NewBuffer(body))
				req.Header.Set("X-API-Key", apiKey)
				req.Header.Set("Content-Type", "application/json")

				start := time.Now()
				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					if ctx.Err() == nil {
						fmt.Printf("❌ Failed to sense: %v\n", err)
					}
					return
				}
				defer resp.Body.Close()

				latency := time.Since(start)
				cacheHit := resp.Header.Get("X-Cache") == "HIT"

				statusIcon := "✅"
				if resp.StatusCode == 429 {
					statusIcon = "⏳ (Rate Limited)"
				} else if resp.StatusCode >= 500 {
					statusIcon = "💥 (Provider Error)"
				} else if cacheHit {
					statusIcon = "⚡ (Cache HIT)"
				}

				fmt.Printf("[%s] %s | Model: %-15s | Latency: %4dms | Status: %d %s\n",
					time.Now().Format("15:04:05"),
					statusIcon,
					model,
					latency.Milliseconds(),
					resp.StatusCode,
					statusIcon)
			}()

			// Reset timer for next request
			timer.Reset(nextDelay())
		}
	}

done:
	fmt.Println("⏳ Waiting for pending requests to finish...")
	wg.Wait()
	fmt.Println("👋 Simulator stopped.")
}
