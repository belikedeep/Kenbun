# 👁️ Kenbun Gateway

**The Observation Haki for your AI Infrastructure.**

**Kenbun** (Observation Haki) allows you to sense the presence and health of your LLM providers, predicting failures and optimizing traffic in real-time. Built in Go for high-throughput, low-latency performance, and distributed reliability.

---

## Key Features

- **Unified API:** Single interface for **OpenAI, Anthropic, and Google Gemini**.
- **Distributed Resilience:** Local-first token buckets for fast rate limiting with background sync to Redis.
- **Health Monitoring:** Local outlier detection using EWMA to sense and route around provider latency shifts.
- **High-Ingestion Logging:** Asynchronous event streaming via **Kafka (Redpanda)** to a **ClickHouse** analytics sink.
- **Two-Tier Caching:** Low-latency L1 (In-Memory) + L2 (Redis Cluster) caching.

---

## Quick Start

### 1. Provision Infrastructure
Kenbun requires PostgreSQL (Control Plane), Redis (State), and Redpanda (Data Ingestion).

```bash
docker compose up -d
```

### 2. Configure Environment
Set the following environment variables:
```bash
DATABASE_URL=postgresql://gateway:gateway@localhost:5432/gateway
REDIS_ADDRS=localhost:6379
KAFKA_BROKERS=localhost:19092
CLICKHOUSE_ADDR=localhost:9000
OPENAI_API_KEY=your_key
ANTHROPIC_API_KEY=your_key
GEMINI_API_KEY=your_key
```

### 3. Run the Gateway
```bash
go run cmd/gateway/main.go
```

---

## License

Apache 2.0 - See [LICENSE](LICENSE) for details.
