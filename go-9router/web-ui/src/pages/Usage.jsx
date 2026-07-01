import { useState, useEffect, useMemo } from "react";
import {
  AreaChart, Area, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer,
} from "recharts";
import Card from "../components/Card";
import SegmentedControl from "../components/SegmentedControl";
import { cn } from "../lib/cn";

const PERIODS = [
  { value: "today", label: "Today" },
  { value: "24h", label: "24h" },
  { value: "7d", label: "7D" },
  { value: "30d", label: "30D" },
  { value: "60d", label: "60D" },
];

function fmt(n) {
  if (n == null) return "0";
  return new Intl.NumberFormat().format(Math.round(n));
}

function fmtCost(n) {
  if (n == null || n === 0) return "$0.00";
  if (n < 0.01) return `~$${n.toFixed(4)}`;
  return `$${n.toFixed(2)}`;
}

// --- OverviewCards ---
function OverviewCards({ stats }) {
  const cards = [
    { label: "Total Requests", value: fmt(stats.totalRequests), icon: "send", color: "text-primary" },
    { label: "Input Tokens", value: fmt(stats.totalPromptTokens), icon: "input", color: "text-blue-500" },
    { label: "Output Tokens", value: fmt(stats.totalCompletionTokens), icon: "output", color: "text-green-500" },
    { label: "Est. Cost", value: `~${fmtCost(stats.totalCost)}`, icon: "payments", color: "text-amber-500" },
  ];

  return (
    <div className="grid grid-cols-2 lg:grid-cols-4 gap-3">
      {cards.map((card) => (
        <div
          key={card.label}
          className="bg-surface border border-border-subtle rounded-[12px] shadow-[var(--shadow-soft)] p-4"
        >
          <div className="flex items-center gap-2 mb-2">
            <span className={cn("material-symbols-outlined text-[18px]", card.color)}>{card.icon}</span>
            <span className="text-xs text-text-muted">{card.label}</span>
          </div>
          <p className="text-2xl font-bold text-text-main">{card.value}</p>
        </div>
      ))}
    </div>
  );
}

// --- UsageChart ---
function UsageChart({ period }) {
  const [chartData, setChartData] = useState([]);
  const [loading, setLoading] = useState(true);
  const [viewMode, setViewMode] = useState("tokens");

  useEffect(() => {
    setLoading(true);
    fetch(`/api/usage/chart?period=${period}`)
      .then((r) => r.json())
      .then((data) => {
        setChartData(Array.isArray(data) ? data : []);
      })
      .catch(() => setChartData([]))
      .finally(() => setLoading(false));
  }, [period]);

  const hasData = chartData.some((d) => d.tokens > 0 || d.cost > 0);
  const dataKey = viewMode === "tokens" ? "tokens" : "cost";
  const fillColor = viewMode === "tokens" ? "#E56A4A" : "#F59E0B";
  const strokeColor = viewMode === "tokens" ? "#cc5236" : "#D97706";

  return (
    <Card>
      <div className="flex items-center justify-between mb-4">
        <h3 className="text-sm font-semibold text-text-main flex items-center gap-2">
          <span className="material-symbols-outlined text-primary text-[18px]">show_chart</span>
          Usage Over Time
        </h3>
        <SegmentedControl
          options={[
            { value: "tokens", label: "Tokens" },
            { value: "cost", label: "Cost" },
          ]}
          value={viewMode}
          onChange={setViewMode}
          size="sm"
        />
      </div>

      {loading ? (
        <div className="h-[240px] flex items-center justify-center">
          <span className="material-symbols-outlined animate-spin text-text-muted">progress_activity</span>
        </div>
      ) : !hasData ? (
        <div className="h-[240px] flex flex-col items-center justify-center text-text-muted">
          <span className="material-symbols-outlined text-[32px] mb-2">bar_chart</span>
          <p className="text-sm">No usage data for this period</p>
        </div>
      ) : (
        <ResponsiveContainer width="100%" height={240}>
          <AreaChart data={chartData} margin={{ top: 5, right: 10, left: 0, bottom: 0 }}>
            <defs>
              <linearGradient id="colorFill" x1="0" y1="0" x2="0" y2="1">
                <stop offset="5%" stopColor={fillColor} stopOpacity={0.3} />
                <stop offset="95%" stopColor={fillColor} stopOpacity={0.02} />
              </linearGradient>
            </defs>
            <CartesianGrid strokeDasharray="3 3" stroke="var(--color-border-subtle)" />
            <XAxis
              dataKey="label"
              tick={{ fontSize: 11, fill: "var(--color-text-muted)" }}
              axisLine={{ stroke: "var(--color-border-subtle)" }}
              tickLine={false}
            />
            <YAxis
              tick={{ fontSize: 11, fill: "var(--color-text-muted)" }}
              axisLine={false}
              tickLine={false}
              tickFormatter={(v) => (viewMode === "cost" ? `$${v}` : fmt(v))}
              width={60}
            />
            <Tooltip
              contentStyle={{
                backgroundColor: "var(--color-surface)",
                border: "1px solid var(--color-border)",
                borderRadius: "8px",
                fontSize: "12px",
              }}
              formatter={(v) => [viewMode === "cost" ? fmtCost(v) : fmt(v), viewMode === "cost" ? "Cost" : "Tokens"]}
            />
            <Area
              type="monotone"
              dataKey={dataKey}
              stroke={strokeColor}
              strokeWidth={2}
              fill="url(#colorFill)"
            />
          </AreaChart>
        </ResponsiveContainer>
      )}
    </Card>
  );
}

// --- UsageTable ---
function UsageTable({ stats, viewMode }) {
  const entries = useMemo(() => {
    if (!stats?.byModel) return [];
    return Object.entries(stats.byModel)
      .map(([key, val]) => ({
        key,
        rawModel: val.rawModel || key,
        provider: val.provider || "unknown",
        requests: val.requests || 0,
        promptTokens: val.promptTokens || 0,
        completionTokens: val.completionTokens || 0,
        totalTokens: (val.promptTokens || 0) + (val.completionTokens || 0),
        cost: val.cost || 0,
      }))
      .sort((a, b) => b.requests - a.requests);
  }, [stats]);

  if (entries.length === 0) {
    return (
      <Card>
        <h3 className="text-sm font-semibold text-text-main mb-4 flex items-center gap-2">
          <span className="material-symbols-outlined text-primary text-[18px]">table_chart</span>
          Usage by Model
        </h3>
        <div className="text-center py-8 text-text-muted">
          <span className="material-symbols-outlined text-[32px] mb-2">table_rows</span>
          <p className="text-sm">No usage data yet</p>
        </div>
      </Card>
    );
  }

  return (
    <Card padding="none">
      <div className="p-4 pb-0">
        <h3 className="text-sm font-semibold text-text-main mb-3 flex items-center gap-2">
          <span className="material-symbols-outlined text-primary text-[18px]">table_chart</span>
          Usage by Model
        </h3>
      </div>
      <div className="overflow-x-auto">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-border-subtle">
              <th className="text-left px-4 py-2 text-xs font-semibold text-text-muted">Model</th>
              <th className="text-left px-4 py-2 text-xs font-semibold text-text-muted">Provider</th>
              <th className="text-right px-4 py-2 text-xs font-semibold text-text-muted">Requests</th>
              {viewMode === "tokens" ? (
                <>
                  <th className="text-right px-4 py-2 text-xs font-semibold text-text-muted">Input</th>
                  <th className="text-right px-4 py-2 text-xs font-semibold text-text-muted">Output</th>
                  <th className="text-right px-4 py-2 text-xs font-semibold text-text-muted">Total</th>
                </>
              ) : (
                <th className="text-right px-4 py-2 text-xs font-semibold text-text-muted">Cost</th>
              )}
            </tr>
          </thead>
          <tbody>
            {entries.map((entry) => (
              <tr
                key={entry.key}
                className="border-b border-border-subtle/50 last:border-b-0 hover:bg-surface-2/50 transition-colors"
              >
                <td className="px-4 py-2.5 font-medium text-text-main">{entry.rawModel}</td>
                <td className="px-4 py-2.5 text-text-muted">{entry.provider}</td>
                <td className="px-4 py-2.5 text-right text-text-main">{fmt(entry.requests)}</td>
                {viewMode === "tokens" ? (
                  <>
                    <td className="px-4 py-2.5 text-right text-text-muted">{fmt(entry.promptTokens)}</td>
                    <td className="px-4 py-2.5 text-right text-text-muted">{fmt(entry.completionTokens)}</td>
                    <td className="px-4 py-2.5 text-right font-medium text-text-main">{fmt(entry.totalTokens)}</td>
                  </>
                ) : (
                  <td className="px-4 py-2.5 text-right font-medium text-text-main">{fmtCost(entry.cost)}</td>
                )}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </Card>
  );
}

// --- Main Usage Page ---
export default function Usage() {
  const [period, setPeriod] = useState("today");
  const [activeTab, setActiveTab] = useState("overview");
  const [stats, setStats] = useState(null);
  const [loading, setLoading] = useState(true);
  const [viewMode, setViewMode] = useState("tokens");

  useEffect(() => {
    setLoading(true);
    fetch(`/api/usage/stats?period=${period}`)
      .then((r) => r.json())
      .then((data) => setStats(data))
      .catch(() => setStats(null))
      .finally(() => setLoading(false));
  }, [period]);

  return (
    <div className="flex min-w-0 flex-col gap-6">
      {/* Tabs + period selector */}
      <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
        <SegmentedControl
          options={[
            { value: "overview", label: "Overview" },
            { value: "details", label: "Details" },
          ]}
          value={activeTab}
          onChange={setActiveTab}
          className="w-full sm:w-auto"
        />
        {activeTab === "overview" && (
          <SegmentedControl
            options={PERIODS}
            value={period}
            onChange={setPeriod}
            size="sm"
            className="w-full sm:w-auto"
          />
        )}
      </div>

      {activeTab === "overview" && (
        <>
          {loading ? (
            <div className="grid grid-cols-2 lg:grid-cols-4 gap-3">
              {[1, 2, 3, 4].map((i) => (
                <div key={i} className="bg-surface border border-border-subtle rounded-[12px] shadow-[var(--shadow-soft)] p-4 animate-pulse h-24" />
              ))}
            </div>
          ) : stats ? (
            <>
              <OverviewCards stats={stats} />

              {/* Chart + Recent side by side on desktop */}
              <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
                <UsageChart period={period} />
                <RecentRequests requests={stats.recentRequests || []} />
              </div>

              {/* Table view toggle */}
              <div className="flex items-center justify-end gap-2">
                <SegmentedControl
                  options={[
                    { value: "tokens", label: "Tokens" },
                    { value: "costs", label: "Costs" },
                  ]}
                  value={viewMode}
                  onChange={setViewMode}
                  size="sm"
                />
              </div>

              <UsageTable stats={stats} viewMode={viewMode === "costs" ? "costs" : "tokens"} />
            </>
          ) : (
            <div className="text-center py-20 text-text-muted">
              <span className="material-symbols-outlined text-[48px] mb-4">error</span>
              <p>Failed to load usage data</p>
            </div>
          )}
        </>
      )}

      {activeTab === "details" && (
        <Card>
          <div className="text-center py-12">
            <span className="material-symbols-outlined text-[40px] text-text-muted mb-3">construction</span>
            <h3 className="text-lg font-semibold mb-2">Request Details</h3>
            <p className="text-text-muted text-sm">
              Detailed per-request logs with latency, tokens, and error traces — coming in a future phase.
            </p>
          </div>
        </Card>
      )}
    </div>
  );
}

// --- RecentRequests (inline component) ---
function RecentRequests({ requests }) {
  if (!requests || requests.length === 0) {
    return (
      <Card>
        <h3 className="text-sm font-semibold text-text-main mb-4 flex items-center gap-2">
          <span className="material-symbols-outlined text-primary text-[18px]">schedule</span>
          Recent Requests
        </h3>
        <div className="text-center py-8 text-text-muted">
          <span className="material-symbols-outlined text-[32px] mb-2">hourglass_empty</span>
          <p className="text-sm">No recent requests</p>
        </div>
      </Card>
    );
  }

  return (
    <Card padding="none">
      <div className="p-4 pb-0">
        <h3 className="text-sm font-semibold text-text-main mb-3 flex items-center gap-2">
          <span className="material-symbols-outlined text-primary text-[18px]">schedule</span>
          Recent Requests
        </h3>
      </div>
      <div className="overflow-y-auto max-h-[240px] custom-scrollbar">
        <table className="w-full text-xs">
          <thead>
            <tr className="border-b border-border-subtle">
              <th className="text-left px-4 py-1.5 text-text-muted font-medium"></th>
              <th className="text-left px-4 py-1.5 text-text-muted font-medium">Model</th>
              <th className="text-right px-4 py-1.5 text-text-muted font-medium">In</th>
              <th className="text-right px-4 py-1.5 text-text-muted font-medium">Out</th>
            </tr>
          </thead>
          <tbody>
            {requests.map((req, i) => (
              <tr key={i} className="border-b border-border-subtle/30 last:border-b-0">
                <td className="px-4 py-1.5">
                  <span className={cn(
                    "inline-block size-2 rounded-full",
                    req.status === "ok" || req.status === "success" ? "bg-green-500" : "bg-red-500"
                  )} />
                </td>
                <td className="px-4 py-1.5 text-text-main truncate max-w-[150px]">{req.model}</td>
                <td className="px-4 py-1.5 text-right text-text-muted">{fmt(req.promptTokens)}</td>
                <td className="px-4 py-1.5 text-right text-text-muted">{fmt(req.completionTokens)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </Card>
  );
}
