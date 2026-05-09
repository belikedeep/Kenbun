"use client";

import { useEffect, useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { 
  BarChart, 
  Bar, 
  XAxis, 
  YAxis, 
  CartesianGrid, 
  Tooltip, 
  ResponsiveContainer,
  LineChart,
  Line
} from "recharts";
import { Zap, DollarSign, Cpu, Clock, AlertCircle } from "lucide-react";
import { getStats, getCharts } from "@/lib/api";
import { ChartContainer, ChartConfig } from "@/components/ui/chart";

const chartConfig = {
  requests: {
    label: "Requests",
    color: "hsl(var(--primary))",
  },
  cost: {
    label: "Cost",
    color: "#10b981",
  },
} satisfies ChartConfig;

export default function OverviewPage() {
  const [stats, setStats] = useState<any>(null);
  const [chartData, setChartData] = useState<any[]>([]);
  const [mounted, setMounted] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    setMounted(true);
    async function fetchData() {
      try {
        const [s, c] = await Promise.all([getStats(), getCharts()]);
        setStats(s);
        setChartData(c || []);
        setError(null);
      } catch (err) {
        console.error(err);
        setError("Sense restricted. Check if the Kenbun backend is active (port 8080).");
      }
    }
    fetchData();
  }, []);

  if (!mounted) return null;

  if (error) {
    return (
      <div className="flex flex-col items-center justify-center h-[80vh] gap-4">
        <AlertCircle className="w-12 h-12 text-destructive" />
        <div className="text-xl font-semibold">{error}</div>
        <p className="text-muted-foreground text-sm">Failed to connect to http://localhost:8080/admin</p>
      </div>
    );
  }

  const dataCards = [
    {
      title: "Total Requests",
      value: stats?.total_requests?.toLocaleString() || "0",
      icon: Zap,
      description: "Lifetime requests handled",
    },
    {
      title: "Total Tokens",
      value: stats?.total_tokens?.toLocaleString() || "0",
      icon: Cpu,
      description: "Cumulative token count",
    },
    {
      title: "Total Cost",
      value: `$${stats?.total_cost?.toFixed(2) || "0.00"}`,
      icon: DollarSign, 
      description: "Total infrastructure spend",
    },
    {
      title: "Avg Latency",
      value: `${stats?.avg_latency_ms?.toFixed(0) || "0"}ms`,
      icon: Clock,
      description: "End-to-end response time",
    },
  ];

  return (
    <div className="space-y-8 animate-in fade-in duration-500">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Observation Haki</h1>
        <p className="text-muted-foreground">Real-time awareness of your LLM infrastructure.</p>
      </div>

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        {dataCards.map((card, index) => (
          <Card key={index} className="bg-card/50 backdrop-blur-sm border-border/50">
            <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
              <CardTitle className="text-sm font-medium">{card.title}</CardTitle>
              <card.icon className="h-4 w-4 text-muted-foreground" />
            </CardHeader>
            <CardContent>
              <div className="text-2xl font-bold">{card.value}</div>
              <p className="text-xs text-muted-foreground">{card.description}</p>
            </CardContent>
          </Card>
        ))}
      </div>

      <div className="grid gap-4 md:grid-cols-1 lg:grid-cols-7">
        <Card className="col-span-4 bg-card/50 backdrop-blur-sm border-border/50">
          <CardHeader>
            <CardTitle>Request Volume (24h)</CardTitle>
          </CardHeader>
          <CardContent className="pl-2">
            <div className="h-[350px] w-full">
              <ChartContainer config={chartConfig}>
                <BarChart data={chartData}>
                  <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="hsl(var(--border))" />
                  <XAxis 
                    dataKey="timestamp" 
                    stroke="hsl(var(--muted-foreground))" 
                    fontSize={12} 
                    tickLine={false} 
                    axisLine={false}
                    tickFormatter={(value) => new Date(value).getHours() + ":00"}
                  />
                  <YAxis stroke="hsl(var(--muted-foreground))" fontSize={12} tickLine={false} axisLine={false} />
                  <Tooltip 
                    contentStyle={{ backgroundColor: "hsl(var(--card))", border: "1px solid hsl(var(--border))" }}
                    labelFormatter={(label) => new Date(label).toLocaleString()}
                    cursor={{ fill: "hsl(var(--muted)/0.2)" }}
                  />
                  <Bar dataKey="requests" fill="var(--color-requests)" radius={[4, 4, 0, 0]} />
                </BarChart>
              </ChartContainer>
            </div>
          </CardContent>
        </Card>

        <Card className="col-span-3 bg-card/50 backdrop-blur-sm border-border/50">
          <CardHeader>
            <CardTitle>Infrastructure Cost</CardTitle>
          </CardHeader>
          <CardContent className="pl-2">
            <div className="h-[350px] w-full">
              <ChartContainer config={chartConfig}>
                <LineChart data={chartData}>
                  <CartesianGrid strokeDasharray="3 3" vertical={false} stroke="hsl(var(--border))" />
                  <XAxis 
                    dataKey="timestamp" 
                    stroke="hsl(var(--muted-foreground))" 
                    fontSize={12} 
                    tickLine={false} 
                    axisLine={false}
                    tickFormatter={(value) => new Date(value).getHours() + ":00"}
                  />
                  <YAxis stroke="hsl(var(--muted-foreground))" fontSize={12} tickLine={false} axisLine={false} tickFormatter={(v) => `$${v}`} />
                  <Tooltip 
                    contentStyle={{ backgroundColor: "hsl(var(--card))", border: "1px solid hsl(var(--border))" }}
                    labelFormatter={(label) => new Date(label).toLocaleString()}
                  />
                  <Line type="monotone" dataKey="cost" stroke="var(--color-cost)" strokeWidth={2} dot={false} />
                </LineChart>
              </ChartContainer>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
