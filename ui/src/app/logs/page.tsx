"use client";

import { useEffect, useState, useRef } from "react";
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
import { API_BASE, ADMIN_TOKEN } from "@/lib/api";
import { Pause, Play } from "lucide-react";

export default function LogsPage() {
  const [logs, setLogs] = useState<any[]>([]);
  const [isPaused, setIsPaused] = useState(false);
  const isPausedRef = useRef(isPaused);

  useEffect(() => {
    isPausedRef.current = isPaused;
  }, [isPaused]);

  useEffect(() => {
    const sse = new EventSource(`${API_BASE}/logs/stream?token=${ADMIN_TOKEN}`);

    sse.onmessage = (e) => {
      if (isPausedRef.current) return;
      const log = JSON.parse(e.data);
      setLogs((prev) => [log, ...prev].slice(0, 50)); // Circular Buffer: Keep last 50
    };

    sse.onerror = (e) => {
      console.error("SSE Error:", e);
      sse.close();
    };

    return () => sse.close();
  }, []);

  return (
    <div className="space-y-8">
      <div className="flex justify-between items-end">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Live Logs</h1>
          <p className="text-muted-foreground">Real-time "Observation Haki" stream of your data plane.</p>
        </div>
        <Button 
          variant={isPaused ? "default" : "secondary"} 
          size="sm" 
          onClick={() => setIsPaused(!isPaused)}
          className="gap-2"
        >
          {isPaused ? (
            <>
              <Play className="w-4 h-4" /> Resume Stream
            </>
          ) : (
            <>
              <Pause className="w-4 h-4" /> Pause Stream
            </>
          )}
        </Button>
      </div>

      <Card className="bg-card/50 backdrop-blur-sm border-border/50">
        <CardHeader className="flex flex-row items-center justify-between">
          <CardTitle>Event Stream</CardTitle>
          <div className="flex items-center gap-2">
            {!isPaused && <div className="w-2 h-2 rounded-full bg-emerald-500 animate-pulse" />}
            {isPaused && <div className="w-2 h-2 rounded-full bg-yellow-500" />}
            <span className="text-xs text-muted-foreground uppercase tracking-widest font-semibold">
              {isPaused ? "Paused" : "Live"}
            </span>
          </div>
        </CardHeader>
        <CardContent>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Time</TableHead>
                <TableHead>Tenant</TableHead>
                <TableHead>Provider</TableHead>
                <TableHead>Model</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Latency</TableHead>
                <TableHead className="text-right">Tokens</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {logs.map((log, i) => (
                <TableRow key={log.request_id || i} className="animate-in slide-in-from-top-2 duration-300">
                  <TableCell className="font-mono text-xs text-muted-foreground">
                    {new Date().toLocaleTimeString()}
                  </TableCell>
                  <TableCell className="font-medium">{log.tenant_id.slice(0, 8)}...</TableCell>
                  <TableCell>
                    <Badge variant="outline" className="capitalize">{log.provider}</Badge>
                  </TableCell>
                  <TableCell className="text-xs font-mono">{log.model}</TableCell>
                  <TableCell>
                    <Badge variant={log.status >= 400 ? "destructive" : "default"}>
                      {log.status}
                    </Badge>
                  </TableCell>
                  <TableCell className="font-mono">{log.latency_ms}ms</TableCell>
                  <TableCell className="text-right font-mono">
                    {log.prompt_tokens + log.completion_tokens}
                  </TableCell>
                </TableRow>
              ))}
              {logs.length === 0 && (
                <TableRow>
                  <TableCell colSpan={7} className="text-center py-20 text-muted-foreground italic">
                    Waiting for events... Hit the gateway to see the haki in action.
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
