import Link from "next/link";
import { Eye, LayoutDashboard, Users, Activity, Settings } from "lucide-react";

export function Sidebar() {
  return (
    <div className="w-64 border-r bg-card h-screen p-4 flex flex-col">
      <div className="flex items-center gap-2 mb-8 px-2">
        <Eye className="w-8 h-8 text-primary" />
        <span className="text-xl font-bold tracking-tight">Kenbun</span>
      </div>
      
      <nav className="flex-1 space-y-1">
        <Link href="/" className="flex items-center gap-3 px-3 py-2 text-sm font-medium rounded-md hover:bg-accent transition-colors">
          <LayoutDashboard className="w-4 h-4" />
          Overview
        </Link>
        <Link href="/tenants" className="flex items-center gap-3 px-3 py-2 text-sm font-medium rounded-md hover:bg-accent transition-colors">
          <Users className="w-4 h-4" />
          Tenants
        </Link>
        <Link href="/logs" className="flex items-center gap-3 px-3 py-2 text-sm font-medium rounded-md hover:bg-accent transition-colors">
          <Activity className="w-4 h-4" />
          Live Logs
        </Link>
      </nav>

      <div className="mt-auto">
        <Link href="/settings" className="flex items-center gap-3 px-3 py-2 text-sm font-medium rounded-md hover:bg-accent transition-colors">
          <Settings className="w-4 h-4" />
          Settings
        </Link>
      </div>
    </div>
  );
}
