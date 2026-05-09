export const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080/admin";
export const ADMIN_TOKEN = process.env.NEXT_PUBLIC_ADMIN_TOKEN || "kb-master-key";

const headers = {
  "Content-Type": "application/json",
  "X-Admin-Token": ADMIN_TOKEN,
};

export async function getStats() {
  const res = await fetch(`${API_BASE}/stats`, { headers });
  if (!res.ok) throw new Error("Failed to fetch stats");
  return res.json();
}

export async function getTenants() {
  const res = await fetch(`${API_BASE}/tenants`, { headers });
  if (!res.ok) throw new Error("Failed to fetch tenants");
  return res.json();
}

export async function getCharts() {
  const res = await fetch(`${API_BASE}/charts`, { headers });
  if (!res.ok) throw new Error("Failed to fetch charts");
  return res.json();
}

export async function createTenant(data: { name: string; api_key: string; rate_limit_rpm: number; budget_cents: number }) {
  const res = await fetch(`${API_BASE}/tenants`, {
    method: "POST",
    headers,
    body: JSON.stringify(data),
  });
  if (!res.ok) throw new Error("Failed to create tenant");
  return res.json();
}
