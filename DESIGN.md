# Design Document: Kenbun Gateway

**Kenbun** (Observation Haki) is a high-throughput, multi-tenant LLM gateway designed to provide distributed resilience, real-time observability, and cost-optimized routing for production AI applications.

---

## 1. Problem Framing

### The Core Problem
Scaling AI products involves more than just calling an API. Modern applications face:
- **Provider Volatility:** Upstream providers fail, rate limit, or experience latency spikes without warning.
- **Cost Management:** Predicting and controlling spend across multiple teams/tenants is difficult.
- **Operational Blindness:** Understanding token usage and cost per tenant in real-time is often a post-hoc manual task.
- **Infrastructure Overhead:** Every app team building their own retry logic, caching, and rate limiting is redundant and error-prone.

### Responsibility Boundary
Kenbun acts as the **Data Plane** for LLM traffic. 
- **It handles:** Routing, Resilience, Rate Limiting, Caching, and Observability.
- **It does NOT handle:** Prompt engineering, long-term conversation storage (memory), or fine-tuning management.

---

## 2. Architecture

Kenbun is built with a **decoupled, multi-plane architecture** to support a target throughput of **100k RPS**.

### Components
1. **Control Plane (PostgreSQL):** Stores tenant configuration, API keys, budgets, and provider allowlists.
2. **State Plane (Redis Cluster):** Handles low-latency state sharing for rate limiting (sync), L2 caching, and real-time event broadcasting.
3. **Data Plane (Go Gateway):** The high-performance engine that routes requests.
4. **Ingestion Plane (Redpanda/Kafka):** Decouples request logging from the request path.
5. **Analytics Plane (ClickHouse):** Provides sub-second queries for high-cardinality token and cost accounting.

### Request Flow
1. **Auth & Validation:** API key is hashed and checked against a local/L1 cache (Redis) for tenant info.
2. **Rate Limiting:** A **Local-first Token Bucket** is checked. It syncs usage to Redis in the background (every 100ms) to ensure distributed consistency without the RTT penalty on every request.
3. **Routing:** The **Latency-Aware Selector** picks the best provider using **EWMA (Exponentially Weighted Moving Average)** latency data.
4. **Resilience Decorators:** The request is wrapped in a decorator that handles retries with exponential backoff and timeouts.
5. **Async Logging:** Once the response is sent, a `LogEvent` is pushed to Kafka.
6. **Budget Processing:** A background worker consumes Kafka events and updates the PostgreSQL `spent_cents` field in batches.

---

## 3. Key Decisions and Tradeoffs

### 1. Local-first Rate Limiting (Hybrid Approach)
- **Decision:** Use local token buckets with background Redis sync.
- **Alternative:** Pure Redis-based rate limiting (e.g., `INCR` or Lua).
- **Tradeoff:** Pure Redis is too slow for 100k RPS due to network RTT. Local-first allows sub-millisecond checks but introduces a small "leakage" window where a tenant might slightly exceed their limit across multiple nodes before sync occurs. We accepted this for the sake of 100x performance.

### 2. Kafka -> ClickHouse for Logging
- **Decision:** Stream logs asynchronously via Kafka to ClickHouse.
- **Alternative:** Direct writes to PostgreSQL.
- **Tradeoff:** Direct writes would kill DB performance at 10k+ RPS. Kafka provides backpressure and durability, while ClickHouse allows us to answer complex queries (e.g., "spend by model per tenant") across billions of rows in milliseconds.

### 3. EWMA for Health Monitoring
- **Decision:** Use Exponentially Weighted Moving Averages for latency tracking.
- **Alternative:** Simple moving average or fixed threshold.
- **Tradeoff:** EWMA reacts faster to sudden latency shifts (important for LLMs) while ignoring one-off outliers. It's more complex to implement but provides much better routing stability.

---

## 4. Failure Modes

| Failure | Impact | Mitigation |
|---------|--------|------------|
| **Redis Down** | L2 Cache and RL sync fail. | Gateway falls back to local-only rate limiting (safe mode) and skips L2 cache. |
| **Kafka Down** | Logs are lost; budgets don't update. | Local buffers in the Gateway will try to retry; eventually, logging is skipped to preserve request flow (availability over observability). |
| **Provider Latency Spike** | Slow responses for all users. | **EWMA** senses the spike and the **Selector** automatically routes traffic to a faster provider. |
| **DB Partition** | New tenants can't be added. | Existing tenants continue to function as their config is cached in Redis/Local RAM. |

---

## 5. What I Didn't Build (v2 Roadmap)

1. **Streaming Cache:** Currently only non-streaming responses are cached. Caching SSE streams requires a more complex "chunk-buffer" logic to avoid partial delivery.
2. **Dynamic Budget Sync:** Budgets are currently updated in Postgres every 1s. For 100k RPS, this should be synced to Redis to allow millisecond-accurate budget blocking.
3. **Model-Class Routing:** "Small" vs "Large" model abstraction (e.g., routing a generic "gpt-3.5" request to the cheapest available equivalent).
4. **Hardened Auth:** Using JWTs or OIDC instead of simple API keys.
5. **Dashboard UI:** A full-featured UI for managing tenants (started as a prototype in `ui/`).

---

## 6. Production Gap Analysis

1. **Secrets Management:** Use HashiCorp Vault or AWS Secrets Manager instead of `.env` files.
2. **Circuit Breaker Persistence:** Health state (EWMA) is currently in-memory. In production, this should be periodically synced to Redis so all gateway nodes "know" a provider is down simultaneously.
3. **Load Testing:** Need a real benchmark suite using `k6` or `ghz` to verify the 100k RPS claim on real hardware.
4. **Autoscaling:** K8s HPA based on Kafka lag and CPU/Latency metrics.
5. **Compliance:** PII filtering in logs before they hit ClickHouse.

---

## 7. Scaling Story

- **10 RPS:** Overkill architecture. A single SQLite DB would suffice.
- **1,000 RPS:** PostgreSQL starts to sweat on logs; the Kafka/ClickHouse decoupling becomes necessary.
- **100k RPS:** The local-first rate limiter and Redis Cluster are the heroes. Multiple Gateway nodes are deployed behind a Load Balancer (NLB). The bottleneck shifts to Kafka ingestion bandwidth and ClickHouse disk I/O.
