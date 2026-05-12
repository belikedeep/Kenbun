"use client";

import { useEffect, useState } from "react";
import { 
  Table, 
  TableBody, 
  TableCell, 
  TableHead, 
  TableHeader, 
  TableRow 
} from "@/components/ui/table";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Plus, Key, ShieldCheck, Loader2 } from "lucide-react";
import { getTenants, toggleTenantStatus } from "@/lib/api";
import { CreateTenantModal } from "@/components/create-tenant-modal";

export default function TenantsPage() {
  const [tenants, setTenants] = useState<any[]>([]);
  const [mounted, setMounted] = useState(false);
  const [togglingId, setTogglingId] = useState<string | null>(null);

  async function fetchData() {
    try {
      const data = await getTenants();
      setTenants(data || []);
    } catch (err) {
      console.error(err);
    }
  }

  async function handleToggleStatus(tenant: any) {
    setTogglingId(tenant.id);
    console.log("Toggling tenant:", tenant.id, tenant.name);
    try {
      await toggleTenantStatus(tenant.id);
      await fetchData();
    } catch (err) {
      console.error("Failed to toggle status:", err);
    } finally {
      setTogglingId(null);
    }
  }

  useEffect(() => {
    setMounted(true);
    fetchData();
  }, []);

  if (!mounted) return null;

  return (
    <div className="space-y-8">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Tenants</h1>
          <p className="text-muted-foreground">Manage API keys and access quotas.</p>
        </div>
        <CreateTenantModal onCreated={fetchData} />
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Active Tenants</CardTitle>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Name</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Rate Limit</TableHead>
                <TableHead>Budget</TableHead>
                <TableHead>Spent</TableHead>
                <TableHead className="text-right">Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {tenants.map((tenant) => (
                <TableRow key={tenant.id}>
                  <TableCell className="font-medium">
                    <div className="flex items-center gap-2">
                      <div className="w-8 h-8 rounded-full bg-blue-500/10 flex items-center justify-center text-blue-500">
                        {tenant.name ? tenant.name[0].toUpperCase() : "?"}
                      </div>
                      {tenant.name || "Unknown"}
                    </div>
                  </TableCell>
                  <TableCell>
                    <Badge variant={tenant.is_active ? "default" : "secondary"}>
                      {tenant.is_active ? "Active" : "Inactive"}
                    </Badge>
                  </TableCell>
                  <TableCell>{tenant.rate_limit_rpm} RPM</TableCell>
                  <TableCell>${(tenant.budget_cents / 100).toFixed(2)}</TableCell>
                  <TableCell>${(tenant.spent_cents / 100).toFixed(2)}</TableCell>
                  <TableCell className="text-right">
                    <Button 
                      variant="ghost" 
                      size="icon"
                      onClick={() => console.log("Rotate key for:", tenant.id)}
                    >
                      <Key className="w-4 h-4" />
                    </Button>
                    <Button 
                      variant="ghost" 
                      size="icon"
                      disabled={togglingId === tenant.id}
                      onClick={() => handleToggleStatus(tenant)}
                    >
                      {togglingId === tenant.id ? (
                        <Loader2 className="w-4 h-4 animate-spin" />
                      ) : (
                        <ShieldCheck className="w-4 h-4" />
                      )}
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
              {tenants.length === 0 && (
                <TableRow>
                  <TableCell colSpan={6} className="text-center py-12 text-muted-foreground">
                    No tenants found. Create your first one to get started.
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </div>
  );
}
