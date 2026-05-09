"use client";

import { useState } from "react";
import { 
  Dialog, 
  DialogContent, 
  DialogDescription, 
  DialogFooter, 
  DialogHeader, 
  DialogTitle, 
  DialogTrigger 
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Plus, Key, Copy, Check } from "lucide-react";
import { createTenant } from "@/lib/api";

export function CreateTenantModal({ onCreated }: { onCreated: () => void }) {
  const [open, setOpen] = useState(false);
  const [loading, setLoading] = useState(false);
  const [newKey, setNewKey] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);

  async function onSubmit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setLoading(true);
    const formData = new FormData(e.currentTarget);
    
    const apiKey = `sk-kb-${Math.random().toString(36).substring(2, 15)}`;
    
    try {
      await createTenant({
        name: formData.get("name") as string,
        rate_limit_rpm: parseInt(formData.get("rpm") as string),
        budget_cents: parseInt(formData.get("budget") as string) * 100,
        api_key: apiKey,
      });
      setNewKey(apiKey);
      onCreated();
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  }

  const copyToClipboard = () => {
    if (newKey) {
      navigator.clipboard.writeText(newKey);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  };

  return (
    <Dialog open={open} onOpenChange={(val) => {
        setOpen(val);
        if (!val) setNewKey(null);
    }}>
      <DialogTrigger render={
        <Button className="gap-2">
          <Plus className="w-4 h-4" />
          Create Tenant
        </Button>
      } />
      <DialogContent className="sm:max-w-[425px] bg-card border-border">
        {newKey ? (
          <div className="space-y-4 py-4 text-center">
            <div className="w-12 h-12 bg-emerald-500/10 text-emerald-500 rounded-full flex items-center justify-center mx-auto mb-2">
              <Key className="w-6 h-6" />
            </div>
            <DialogHeader>
              <DialogTitle>Tenant Created Successfully</DialogTitle>
              <DialogDescription>
                Copy this API key now. For security, you won't be able to see it again.
              </DialogDescription>
            </DialogHeader>
            <div className="flex items-center gap-2 p-3 bg-muted rounded-md font-mono text-sm break-all relative group">
              {newKey}
              <Button 
                size="icon" 
                variant="ghost" 
                className="shrink-0 ml-auto h-8 w-8" 
                onClick={copyToClipboard}
              >
                {copied ? <Check className="w-4 h-4 text-emerald-500" /> : <Copy className="w-4 h-4" />}
              </Button>
            </div>
            <Button className="w-full" onClick={() => setOpen(false)}>Done</Button>
          </div>
        ) : (
          <form onSubmit={onSubmit}>
            <DialogHeader>
              <DialogTitle>New Tenant</DialogTitle>
              <DialogDescription>
                Create a new API identity with dedicated rate limits and budget.
              </DialogDescription>
            </DialogHeader>
            <div className="grid gap-4 py-4">
              <div className="grid gap-2">
                <Label htmlFor="name">Display Name</Label>
                <Input id="name" name="name" placeholder="e.g. Maritime AI Team" required />
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div className="grid gap-2">
                  <Label htmlFor="rpm">Rate Limit (RPM)</Label>
                  <Input id="rpm" name="rpm" type="number" defaultValue="60" required />
                </div>
                <div className="grid gap-2">
                  <Label htmlFor="budget">Monthly Budget ($)</Label>
                  <Input id="budget" name="budget" type="number" defaultValue="50" required />
                </div>
              </div>
            </div>
            <DialogFooter>
              <Button type="submit" disabled={loading}>
                {loading ? "Sensing..." : "Generate API Key"}
              </Button>
            </DialogFooter>
          </form>
        )}
      </DialogContent>
    </Dialog>
  );
}
