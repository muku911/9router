import { useState, useEffect, useCallback } from "react";
import Card from "../components/Card";
import Button from "../components/Button";
import Badge from "../components/Badge";
import Input from "../components/Input";
import Toggle from "../components/Toggle";
import Modal from "../components/Modal";
import { cn } from "../lib/cn";
import { useNotificationStore } from "../store/notificationStore";

const MITM_TOOLS = [
  {
    id: "antigravity",
    name: "Antigravity (Google Cloud Code)",
    icon: "rocket_launch",
    color: "#F59E0B",
    hosts: ["daily-cloudcode-pa.googleapis.com", "cloudcode-pa.googleapis.com"],
    defaultModels: [
      "gemini-2.5-pro", "gemini-2.5-flash", "gemini-2.0-flash",
      "gemini-2.0-flash-lite", "gemini-1.5-pro", "gemini-1.5-flash",
      "claude-3-5-sonnet", "claude-sonnet-4", "claude-3-5-haiku",
    ],
  },
  {
    id: "copilot",
    name: "GitHub Copilot",
    icon: "code",
    color: "#333333",
    hosts: ["api.individual.githubcopilot.com"],
    defaultModels: ["gpt-4o", "claude-3.5-sonnet", "o3-mini", "gemini-2.0-flash"],
  },
  {
    id: "kiro",
    name: "Kiro (AWS Q)",
    icon: "psychology_alt",
    color: "#FF6B35",
    hosts: ["q.us-east-1.amazonaws.com", "codewhisperer.us-east-1.amazonaws.com"],
    defaultModels: [
      "claude-sonnet-4", "claude-3.7-sonnet", "claude-3.5-sonnet",
      "claude-3.5-haiku", "amazon.nova-pro", "amazon.nova-micro",
      "deepseek-r1", "llama-4-maverick", "mistral-large",
    ],
  },
];

export default function Mitm() {
  const [status, setStatus] = useState({ running: false, dns: {} });
  const [sudoPassword, setSudoPassword] = useState("");
  const [showSudoModal, setShowSudoModal] = useState(false);
  const [pendingAction, setPendingAction] = useState(null);
  const [loading, setLoading] = useState(false);
  const [expandedTool, setExpandedTool] = useState(null);
  const [aliases, setAliases] = useState({});
  const [routerBaseUrl, setRouterBaseUrl] = useState("");
  const notify = useNotificationStore;

  useEffect(() => {
    if (typeof window !== "undefined") {
      setRouterBaseUrl(window.location.origin);
    }
    fetchStatus();
    const interval = setInterval(fetchStatus, 5000);
    return () => clearInterval(interval);
  }, []);

  const fetchStatus = async () => {
    try {
      const res = await fetch("/api/cli-tools/antigravity-mitm");
      if (res.ok) {
        const data = await res.json();
        setStatus(data);
      }
    } catch { /* ignore */ }
  };

  const fetchAliases = async (toolId) => {
    try {
      const res = await fetch(`/api/cli-tools/antigravity-mitm/alias?tool=${toolId}`);
      if (res.ok) {
        const data = await res.json();
        setAliases((prev) => ({ ...prev, [toolId]: data.mappings || data }));
      }
    } catch { /* ignore */ }
  };

  const saveAliases = async (toolId, mappings) => {
    try {
      await fetch("/api/cli-tools/antigravity-mitm/alias", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ tool: toolId, mappings }),
      });
    } catch { /* ignore */ }
  };

  const requireSudo = (action) => {
    if (sudoPassword) {
      executeAction(action, sudoPassword);
    } else {
      setPendingAction(action);
      setShowSudoModal(true);
    }
  };

  const handleSudoSubmit = () => {
    if (!sudoPassword.trim()) return;
    setShowSudoModal(false);
    if (pendingAction) {
      executeAction(pendingAction, sudoPassword);
      setPendingAction(null);
    }
  };

  const executeAction = async (action, password) => {
    setLoading(true);
    try {
      const res = await fetch("/api/cli-tools/antigravity-mitm", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          action,
          password,
          mitmRouterBaseUrl: routerBaseUrl,
        }),
      });
      if (res.ok) {
        await fetchStatus();
      } else {
        const err = await res.json().catch(() => ({}));
        console.log("MITM action failed:", err.error || "Unknown error");
      }
    } catch (e) {
      console.log("MITM action error:", e);
    } finally {
      setLoading(false);
    }
  };

  const handleToggleTool = (toolId) => {
    if (expandedTool === toolId) {
      setExpandedTool(null);
    } else {
      setExpandedTool(toolId);
      fetchAliases(toolId);
    }
  };

  return (
    <div className="flex flex-col gap-6">
      {/* Warning banner */}
      <div className="flex items-start gap-3 p-4 rounded-lg bg-amber-500/10 border border-amber-500/20 text-amber-700 dark:text-amber-400 text-sm">
        <span className="material-symbols-outlined text-[20px] shrink-0 mt-0.5">warning</span>
        <div>
          <p className="font-medium mb-1">MITM Proxy — Advanced Feature</p>
          <p className="text-xs opacity-80">
            This intercepts HTTPS traffic from IDE plugins by installing a custom CA certificate
            and redirecting DNS. Use only for personal development and testing.
          </p>
        </div>
      </div>

      {/* Server Status Card */}
      <Card>
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-semibold flex items-center gap-2">
            <span className="material-symbols-outlined text-primary">security</span>
            MITM Server
          </h2>
          <Badge
            variant={status.running ? "success" : "default"}
            size="sm"
            dot
          >
            {status.running ? "Running" : "Stopped"}
          </Badge>
        </div>

        <div className="grid grid-cols-3 gap-3 mb-4">
          <StatusIndicator
            label="Server"
            active={status.running}
            icon="dns"
          />
          <StatusIndicator
            label="Certificate"
            active={!!status.ca_path}
            icon="verified_user"
          />
          <StatusIndicator
            label="DNS"
            active={Object.values(status.dns || {}).some((v) => v)}
            icon="language"
          />
        </div>

        <div className="flex flex-col gap-3">
          <Input
            label="9Router Base URL"
            value={routerBaseUrl}
            onChange={(e) => setRouterBaseUrl(e.target.value)}
            placeholder="http://localhost:20128"
            hint="The MITM proxy will forward traffic to this address"
          />

          <div className="flex gap-2">
            {!status.running ? (
              <Button
                icon="play_arrow"
                onClick={() => requireSudo("start")}
                loading={loading}
                fullWidth
              >
                Start MITM Server
              </Button>
            ) : (
              <Button
                icon="stop"
                variant="danger"
                onClick={() => requireSudo("stop")}
                loading={loading}
                fullWidth
              >
                Stop MITM Server
              </Button>
            )}
          </div>
        </div>
      </Card>

      {/* Tool Cards */}
      {MITM_TOOLS.map((tool) => (
        <MitmToolCard
          key={tool.id}
          tool={tool}
          isExpanded={expandedTool === tool.id}
          onToggle={() => handleToggleTool(tool.id)}
          serverRunning={status.running}
          dnsActive={status.dns?.[tool.id]}
          aliases={aliases[tool.id] || {}}
          onAliasChange={(model, value) => {
            setAliases((prev) => ({
              ...prev,
              [tool.id]: { ...(prev[tool.id] || {}), [model]: value },
            }));
          }}
          onSaveAliases={() => saveAliases(tool.id, aliases[tool.id] || {})}
          onDnsToggle={(enable) =>
            requireSudo(enable ? "dns_enable" : "dns_disable")
          }
          loading={loading}
        />
      ))}

      {/* Sudo Password Modal */}
      <Modal
        isOpen={showSudoModal}
        title="Enter Sudo Password"
        onClose={() => {
          setShowSudoModal(false);
          setPendingAction(null);
        }}
        size="sm"
      >
        <div className="flex flex-col gap-4">
          <p className="text-sm text-text-muted">
            MITM operations require root access to modify /etc/hosts and bind port 443.
          </p>
          <Input
            type="password"
            label="Sudo Password"
            value={sudoPassword}
            onChange={(e) => setSudoPassword(e.target.value)}
            placeholder="Enter your password"
            onKeyDown={(e) => e.key === "Enter" && handleSudoSubmit()}
          />
          <div className="flex gap-2">
            <Button onClick={handleSudoSubmit} fullWidth disabled={!sudoPassword.trim()}>
              Continue
            </Button>
            <Button
              variant="ghost"
              fullWidth
              onClick={() => {
                setShowSudoModal(false);
                setPendingAction(null);
              }}
            >
              Cancel
            </Button>
          </div>
        </div>
      </Modal>
    </div>
  );
}

function StatusIndicator({ label, active, icon }) {
  return (
    <div className={cn(
      "flex flex-col items-center gap-1.5 p-3 rounded-lg border transition-colors",
      active
        ? "bg-green-500/5 border-green-500/20 text-green-600 dark:text-green-400"
        : "bg-surface-2/50 border-border-subtle text-text-muted"
    )}>
      <span className="material-symbols-outlined text-[20px]">{icon}</span>
      <span className="text-xs font-medium">{label}</span>
      <span className={cn("size-2 rounded-full", active ? "bg-green-500" : "bg-gray-400")} />
    </div>
  );
}

function MitmToolCard({
  tool, isExpanded, onToggle, serverRunning, dnsActive,
  aliases, onAliasChange, onSaveAliases, onDnsToggle, loading,
}) {
  return (
    <Card padding="none">
      {/* Header — always visible */}
      <button
        onClick={onToggle}
        className="w-full flex items-center justify-between p-4 hover:bg-surface-2/30 transition-colors"
      >
        <div className="flex items-center gap-3">
          <div
            className="flex items-center justify-center size-10 rounded-lg"
            style={{ backgroundColor: `${tool.color}20`, color: tool.color }}
          >
            <span className="material-symbols-outlined text-[20px]">{tool.icon}</span>
          </div>
          <div className="text-left">
            <h3 className="text-sm font-semibold text-text-main">{tool.name}</h3>
            <p className="text-xs text-text-muted">{tool.hosts[0]}</p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          {dnsActive && (
            <Badge variant="success" size="sm" dot>DNS Active</Badge>
          )}
          <span
            className="material-symbols-outlined text-text-muted text-[20px] transition-transform"
            style={{ transform: isExpanded ? "rotate(180deg)" : "rotate(0deg)" }}
          >
            expand_more
          </span>
        </div>
      </button>

      {/* Expanded content */}
      {isExpanded && (
        <div className="border-t border-border-subtle p-4 space-y-4">
          {/* Hosts */}
          <div>
            <p className="text-xs font-semibold text-text-muted mb-2">DNS Entries (/etc/hosts)</p>
            <div className="space-y-1">
              {tool.hosts.map((host) => (
                <code
                  key={host}
                  className="block text-xs font-mono px-3 py-1.5 rounded bg-surface-2 text-text-muted"
                >
                  127.0.0.1 {host}
                </code>
              ))}
            </div>
          </div>

          {/* Model Aliases */}
          <div>
            <p className="text-xs font-semibold text-text-muted mb-2">Model Mappings</p>
            <div className="space-y-2">
              {tool.defaultModels.map((model) => (
                <div key={model} className="flex items-center gap-2">
                  <span className="text-xs text-text-muted font-mono min-w-[160px] truncate">
                    {model}
                  </span>
                  <span className="text-text-muted text-xs">→</span>
                  <input
                    type="text"
                    value={aliases[model] || ""}
                    onChange={(e) => onAliasChange(model, e.target.value)}
                    placeholder={model}
                    className="flex-1 px-2 py-1 text-xs font-mono rounded bg-surface-2 border border-transparent focus:border-brand-500/30 focus:outline-none text-text-main placeholder:text-text-muted/50"
                  />
                </div>
              ))}
            </div>
            <Button
              variant="secondary"
              size="sm"
              icon="save"
              onClick={onSaveAliases}
              className="mt-2"
            >
              Save Mappings
            </Button>
          </div>

          {/* DNS Toggle */}
          <div className="flex items-center justify-between pt-2 border-t border-border-subtle">
            <div>
              <p className="text-sm font-medium">DNS Redirect</p>
              <p className="text-xs text-text-muted">
                {dnsActive
                  ? "Traffic is being redirected through MITM proxy"
                  : "Enable to redirect this tool's traffic"}
              </p>
            </div>
            <Toggle
              checked={!!dnsActive}
              onChange={(checked) => onDnsToggle(checked)}
              disabled={!serverRunning || loading}
            />
          </div>
        </div>
      )}
    </Card>
  );
}
