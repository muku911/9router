import { useState, useEffect } from "react";
import Card from "../components/Card";
import Badge from "../components/Badge";
import Button from "../components/Button";
import ProviderIcon from "../components/ProviderIcon";
import { AI_PROVIDERS } from "../constants/providers";

export default function Quota() {
  const [connections, setConnections] = useState([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchConnections();
  }, []);

  const fetchConnections = async () => {
    try {
      const res = await fetch("/api/providers");
      if (res.ok) {
        const data = await res.json();
        setConnections(data.connections || []);
      }
    } catch { /* ignore */ }
    setLoading(false);
  };

  // Group connections by provider
  const grouped = {};
  connections.forEach((c) => {
    if (!grouped[c.provider]) grouped[c.provider] = [];
    grouped[c.provider].push(c);
  });

  if (loading) {
    return (
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        {[1, 2, 3, 4].map((i) => (
          <div key={i} className="bg-surface border border-border-subtle rounded-[12px] shadow-[var(--shadow-soft)] p-6 animate-pulse h-40" />
        ))}
      </div>
    );
  }

  const providers = Object.entries(grouped);

  if (providers.length === 0) {
    return (
      <div className="text-center py-20">
        <div className="inline-flex items-center justify-center w-16 h-16 rounded-full bg-primary/10 text-primary mb-4">
          <span className="material-symbols-outlined text-[32px]">data_usage</span>
        </div>
        <h2 className="text-xl font-semibold mb-2">No Provider Connections</h2>
        <p className="text-text-muted mb-4">Add provider connections to track quota usage</p>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-6">
      <div className="flex items-center justify-between">
        <p className="text-sm text-text-muted">
          {connections.length} connection{connections.length !== 1 ? "s" : ""} across {providers.length} provider{providers.length !== 1 ? "s" : ""}
        </p>
        <Button variant="ghost" size="sm" icon="refresh" onClick={fetchConnections}>
          Refresh
        </Button>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        {providers.map(([providerId, conns]) => {
          const info = AI_PROVIDERS[providerId];
          const activeCount = conns.filter((c) => c.isActive).length;
          const errorCount = conns.filter((c) => c.testStatus === "error" || c.testStatus === "expired").length;

          return (
            <Card key={providerId} hover>
              <div className="flex items-start gap-3">
                <ProviderIcon
                  src={`/providers/${providerId}.png`}
                  alt={info?.name || providerId}
                  size={40}
                  className="rounded-lg object-contain shrink-0"
                  fallbackText={info?.textIcon || providerId.slice(0, 2).toUpperCase()}
                  fallbackColor={info?.color}
                />
                <div className="flex-1 min-w-0">
                  <h3 className="font-semibold text-text-main">{info?.name || providerId}</h3>
                  <div className="flex items-center gap-2 mt-1">
                    <Badge variant="success" size="sm" dot>{activeCount} active</Badge>
                    {errorCount > 0 && <Badge variant="error" size="sm" dot>{errorCount} error</Badge>}
                  </div>
                </div>
              </div>

              <div className="mt-3 space-y-2">
                {conns.map((conn) => (
                  <div key={conn.id} className="flex items-center justify-between py-1.5 text-sm">
                    <div className="flex items-center gap-2 min-w-0">
                      <span className={`size-2 rounded-full shrink-0 ${conn.isActive ? (conn.testStatus === "error" ? "bg-red-500" : "bg-green-500") : "bg-gray-400"}`} />
                      <span className="text-text-main truncate">{conn.name || conn.email || "Connection"}</span>
                    </div>
                    <span className="text-xs text-text-muted shrink-0">
                      {conn.authType === "oauth" ? "OAuth" : conn.authType === "cookie" ? "Cookie" : "API Key"}
                    </span>
                  </div>
                ))}
              </div>
            </Card>
          );
        })}
      </div>
    </div>
  );
}
