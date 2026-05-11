# 👁️ Kenbun Gateway

**The Observation Haki for your AI Infrastructure.**

**Kenbun** (Observation Haki) allows you to sense the presence and health of your LLM providers, predicting failures and optimizing traffic in real-time. Built in Go for high-throughput, low-latency performance, and distributed reliability.

[![Go Report Card](https://goreportcard.com/badge/github.com/belikedeep/Kenbun)](https://goreportcard.com/report/github.com/belikedeep/Kenbun)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

---

## 🏗 System Architecture

```mermaid
graph TD
    Client[Client Application] -->|HTTP/REST| GW[Kenbun Gateway]
    
    subgraph "Kenbun Gateway (Go)"
        Auth[Auth & Rate Limiting] --> Router[LLM Router]
        Router --> Balancer[Latency-Aware Load Balancer]
        Balancer --> Monitor[Health Monitor / EWMA]
        Monitor --> Cache[Two-Tier Cache]
    end
    
    subgraph "Data Tier"
        Cache --> L1[In-Memory Cache]
        Cache --> L2[Redis Cluster]
        Monitor --> DB[(PostgreSQL - Config)]
        Monitor --> Sync[Redis - Token Buckets]
    end
    
    subgraph "Observability"
        GW -->|Async Events| Ingest[Redpanda / Kafka]
        Ingest --> Sink[ClickHouse]
        Sink --> Dashboard[Analytics API]
    end
    
    subgraph "Providers"
        Balancer --> OpenAI[OpenAI]
        Balancer --> Anthropic[Anthropic]
        Balancer --> Gemini[Google Gemini]
        Balancer --> Local[Ollama / Local]
    end
```

---

## 🚀 Key Engineering Pillars

### 1. Distributed Resilience
*   **Token Bucket Rate Limiting:** Implements local-first token buckets for near-zero latency rate limiting, with periodic background synchronization to a Redis Cluster to ensure global consistency across multiple gateway instances.
*   **Graceful Degradation:** When a provider (e.g., OpenAI) exceeds latency thresholds or returns 5xx errors, Kenbun automatically reroutes traffic to the next best provider based on a `LatencyAwareSelector`.

### 2. High-Performance Monitoring
*   **Outlier Detection:** Uses **Exponentially Weighted Moving Average (EWMA)** to calculate real-time latency shifts. This allows the system to sense "brownouts" before they become full outages.
*   **Health Checks:** Active and passive health monitoring keeps the provider registry up-to-date without blocking the critical path of user requests.

### 3. High-Ingestion Logging & Analytics
*   **Async Event Streaming:** Every request metadata (tokens, latency, cost) is streamed asynchronously via **Redpanda** (Kafka-compatible) to prevent logging from impacting request latency.
*   **OLAP Storage:** Events are ingested into **ClickHouse**, enabling sub-second analytical queries across millions of requests for cost tracking and usage auditing.

---

## 🛠 Why this project exists?

As AI becomes a core component of production systems, the "direct-to-provider" approach becomes a single point of failure. Kenbun provides an abstraction layer that treats LLMs as interchangeable infrastructure components, prioritizing **uptime, observability, and cost control**.

### Production Considerations:
- **Zero-Downtime Config:** All provider configurations and routing rules are reloaded from PostgreSQL without restarting the gateway.
- **Circuit Breaking:** Prevents cascading failures when a specific model or provider is under heavy load.
- **Audit Logging:** Built-in support for PII masking and structured logging for compliance.

---

## 📖 Quick Start

### 1. Provision Infrastructure
Kenbun requires PostgreSQL (Control Plane), Redis (State), and Redpanda (Data Ingestion).

```bash
docker compose up -d
```

### 2. Configure Environment
```bash
DATABASE_URL=postgresql://gateway:gateway@localhost:5432/gateway
REDIS_ADDRS=localhost:6379
KAFKA_BROKERS=localhost:19092
OPENAI_API_KEY=your_key
```

### 3. Run the Gateway
```bash
go run cmd/gateway/main.go
```

---

## ⚖️ License

Apache 2.0 - See [LICENSE](LICENSE) for details.
