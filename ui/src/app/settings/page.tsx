"use client";

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { API_BASE } from "@/lib/api";
import { Settings as SettingsIcon, Globe } from "lucide-react";

export default function SettingsPage() {
  return (
    <div className="space-y-8 animate-in fade-in duration-500">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Settings</h1>
        <p className="text-muted-foreground">Manage your Kenbun Gateway configuration.</p>
      </div>

      <div className="grid gap-4 md:grid-cols-2">
        <Card className="bg-card/50 backdrop-blur-sm border-border/50">
          <CardHeader className="flex flex-row items-center gap-4">
            <div className="p-2 bg-primary/10 rounded-lg">
              <Globe className="h-6 w-6 text-primary" />
            </div>
            <div>
              <CardTitle>Backend Connection</CardTitle>
              <CardDescription>The current Gateway API endpoint.</CardDescription>
            </div>
          </CardHeader>
          <CardContent>
            <div className="space-y-2">
              <label className="text-sm font-medium text-muted-foreground">Gateway URL</label>
              <div className="p-3 rounded-md bg-muted font-mono text-sm break-all">
                {API_BASE}
              </div>
            </div>
          </CardContent>
        </Card>

        <Card className="bg-card/50 backdrop-blur-sm border-border/50">
          <CardHeader className="flex flex-row items-center gap-4">
            <div className="p-2 bg-primary/10 rounded-lg">
              <SettingsIcon className="h-6 w-6 text-primary" />
            </div>
            <div>
              <CardTitle>System Information</CardTitle>
              <CardDescription>Frontend environment details.</CardDescription>
            </div>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              <div className="flex justify-between items-center">
                <span className="text-sm font-medium text-muted-foreground">Status</span>
                <span className="flex items-center gap-2 text-sm font-semibold text-green-500">
                  <span className="h-2 w-2 rounded-full bg-green-500 animate-pulse" />
                  Connected
                </span>
              </div>
              <div className="flex justify-between items-center">
                <span className="text-sm font-medium text-muted-foreground">Version</span>
                <span className="text-sm font-mono">v0.1.0-alpha</span>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
