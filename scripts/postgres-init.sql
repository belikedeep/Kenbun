-- Control Plane (PostgreSQL)
CREATE TABLE IF NOT EXISTS tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    api_key_hash TEXT UNIQUE NOT NULL,
    rate_limit_rpm INTEGER NOT NULL DEFAULT 60,
    budget_cents INTEGER NOT NULL DEFAULT 1000,
    spent_cents INTEGER NOT NULL DEFAULT 0,
    provider_allowlist TEXT[] DEFAULT '{}',
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT now()
);

-- Seed a test tenant (Key: sk-kenbun-test)
-- SHA256 of "sk-kenbun-test" is 2c76897a806f1188dcd9b074b958e6ba88c82fe3eed21d3db25977059ce29417
INSERT INTO tenants (name, api_key_hash, rate_limit_rpm, budget_cents)
VALUES ('Test Tenant', '2c76897a806f1188dcd9b074b958e6ba88c82fe3eed21d3db25977059ce29417', 100, 5000)
ON CONFLICT (api_key_hash) DO NOTHING;
