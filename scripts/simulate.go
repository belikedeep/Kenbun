package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"time"
)

// This script simulates a high-throughput load on the Kenbun Gateway.
// It uses the pre-seeded "Test Tenant" key to prove the systems logic (Caching, RL, Logging).

func main() {
	gatewayURL := "http://localhost:8080/v1/chat/completions"
	apiKey := "sk-kenbun-test"
	
	models := []string{"gpt-4o", "claude-3-sonnet", "gemini-1.5-pro"}
	prompts := []string{
		"What is Observation Haki?",
		"Explain Kafka in one sentence.",
		"How do I scale a Go application?",
		"Tell me a joke about Redis.",
	}

	fmt.Println("👁️ Kenbun Haki Simulator Starting...")
	fmt.Printf("Target: %s\n\n", gatewayURL)

	for {
		go func() {
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
			req, _ := http.NewRequest("POST", gatewayURL, bytes.NewBuffer(body))
			req.Header.Set("X-API-Key", apiKey)
			req.Header.Set("Content-Type", "application/json")

			start := time.Now()
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				fmt.Printf("❌ Failed to sense: %v\n", err)
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

		// Sleep between 50ms and 500ms to simulate varied traffic
		time.Sleep(time.Duration(rand.Intn(450)+50) * time.Millisecond)
	}
}
