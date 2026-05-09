-- Data Plane (ClickHouse)
CREATE TABLE IF NOT EXISTS gateway_logs (
    request_id String,
    tenant_id String,
    provider String,
    model String,
    prompt_tokens Int32,
    completion_tokens Int32,
    cost_cents Float64,
    latency_ms Int32,
    status Int32,
    cached UInt8,
    timestamp DateTime64(9, 'UTC')
) ENGINE = MergeTree()
ORDER BY (tenant_id, timestamp);
