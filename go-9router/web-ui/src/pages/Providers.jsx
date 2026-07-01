import { useState, useEffect, useMemo } from "react";
import { Link } from "react-router-dom";
import { cn } from "../lib/cn";
import Card from "../components/Card";
import Button from "../components/Button";
import Badge from "../components/Badge";
import Input from "../components/Input";
import Modal from "../components/Modal";
import Toggle from "../components/Toggle";
import ProviderIcon from "../components/ProviderIcon";
import { useHeaderSearchStore } from "../store/headerSearchStore";
import { useNotificationStore } from "../store/notificationStore";
import {
  OAUTH_PROVIDERS, APIKEY_PROVIDERS, FREE_PROVIDERS, FREE_TIER_PROVIDERS, WEB_COOKIE_PROVIDERS,
} from "../constants/providers";

const ALL_FREE_PROVIDERS = { ...FREE_PROVIDERS, ...FREE_TIER_PROVIDERS };

function getProviderStats(connections, providerId) {
  const provConns = connections.filter((c) => c.provider === providerId);
  if (provConns.length === 0) return null;
  const connected = provConns.filter((c) => c.isActive && (!c.testStatus || c.testStatus === "active" || c.testStatus === "success" || c.testStatus === "unknown")).length;
  const error = provConns.filter((c) => c.testStatus === "error" || c.testStatus === "expired" || c.testStatus === "unavailable").length;
  const allDisabled = provConns.every((c) => !c.isActive);
  return { total: provConns.length, connected, error, allDisabled };
}

function ProviderCard({ provider, stats }) {
  return (
    <Link to={`/providers/${provider.id}`}
      className="block bg-surface border border-border-subtle rounded-[12px] shadow-[var(--shadow-soft)] p-4 hover:shadow-[0_0_20px_-5px_rgba(229,106,74,0.15)] hover:border-brand-500/25 transition-all duration-300 dark:bg-gradient-to-b dark:from-surface dark:to-surface-2/20">
      <div className="flex items-start justify-between mb-3">
        <ProviderIcon src={`/providers/${provider.id}.png`} alt={provider.name} size={40} className="rounded-lg object-contain"
          fallbackText={provider.textIcon || provider.name?.slice(0, 2)?.toUpperCase()} fallbackColor={provider.color} />
        {stats && (
          <div className="flex items-center gap-1.5">
            {stats.connected > 0 && <Badge variant="success" size="sm" dot>{stats.connected}</Badge>}
            {stats.error > 0 && <Badge variant="error" size="sm" dot>{stats.error}</Badge>}
            {stats.allDisabled && <Badge variant="warning" size="sm">Disabled</Badge>}
          </div>
        )}
      </div>
      <h3 className="text-sm font-semibold text-text-main truncate">{provider.name}</h3>
      <p className="text-xs text-text-muted mt-0.5">{stats ? `${stats.total} connection${stats.total !== 1 ? "s" : ""}` : "Not connected"}</p>
      {provider.deprecated && <p className="text-[10px] text-amber-500 mt-1 truncate">⚠️ Risk</p>}
    </Link>
  );
}

export default function Providers() {
  const [connections, setConnections] = useState([]);
  const [compatibleNodes, setCompatibleNodes] = useState([]);
  const [loading, setLoading] = useState(true);
  const [showAllApiKey, setShowAllApiKey] = useState(false);
  const [showAddCompatModal, setShowAddCompatModal] = useState(null); // "openai" | "anthropic" | null
  const [compatForm, setCompatForm] = useState({ name: "", prefix: "", baseUrl: "", apiType: "chat" });
  const [addCompatLoading, setAddCompatLoading] = useState(false);
  const [testingBatch, setTestingBatch] = useState(null);
  const [testResults, setTestResults] = useState(null);
  const notify = useNotificationStore.getState;

  const query = useHeaderSearchStore((s) => s.query);
  const register = useHeaderSearchStore((s) => s.register);
  const unregister = useHeaderSearchStore((s) => s.unregister);

  useEffect(() => { register("Search providers..."); return () => unregister(); }, [register, unregister]);
  useEffect(() => { fetchAll(); }, []);

  const fetchAll = async () => {
    try {
      const [connRes, nodesRes] = await Promise.all([
        fetch("/api/providers"),
        fetch("/api/provider-nodes"),
      ]);
      if (connRes.ok) { const d = await connRes.json(); setConnections(d.connections || []); }
      if (nodesRes.ok) { const d = await nodesRes.json(); setCompatibleNodes(d.nodes || []); }
    } catch { /* ignore */ }
    setLoading(false);
  };

  const handleBatchTest = async (mode) => {
    setTestingBatch(mode);
    try {
      const res = await fetch("/api/providers/test-batch", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ mode }),
      });
      if (res.ok) {
        const data = await res.json();
        setTestResults(data);
      }
    } catch { /* ignore */ }
    setTestingBatch(null);
  };

  const handleAddCompatible = async () => {
    if (!compatForm.name.trim() || !compatForm.baseUrl.trim()) return;
    setAddCompatLoading(true);
    try {
      const res = await fetch("/api/provider-nodes", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          ...compatForm,
          type: showAddCompatModal === "anthropic" ? "anthropic-compatible" : "openai-compatible",
        }),
      });
      if (res.ok) {
        setShowAddCompatModal(null);
        setCompatForm({ name: "", prefix: "", baseUrl: "", apiType: "chat" });
        await fetchAll();
        notify().success("Compatible provider added");
      }
    } catch { /* ignore */ }
    setAddCompatLoading(false);
  };

  const filterByQuery = (providers) => {
    if (!query) return Object.values(providers);
    const q = query.toLowerCase();
    return Object.values(providers).filter((p) => p.name.toLowerCase().includes(q) || p.id.toLowerCase().includes(q));
  };

  const oauthProviders = useMemo(() => filterByQuery(OAUTH_PROVIDERS), [query]);
  const freeProviders = useMemo(() => filterByQuery(ALL_FREE_PROVIDERS), [query]);
  const apikeyProviders = useMemo(() => filterByQuery(APIKEY_PROVIDERS), [query]);
  const cookieProviders = useMemo(() => filterByQuery(WEB_COOKIE_PROVIDERS), [query]);
  const INITIAL_APIKEY_COUNT = 20;
  const visibleApiKey = showAllApiKey ? apikeyProviders : apikeyProviders.slice(0, INITIAL_APIKEY_COUNT);

  if (loading) {
    return (<div className="flex flex-col gap-8">{[1, 2, 3].map((i) => (<div key={i} className="bg-surface border border-border-subtle rounded-[12px] shadow-[var(--shadow-soft)] p-6 animate-pulse h-48" />))}</div>);
  }

  return (
    <div className="flex flex-col gap-8">
      {/* Custom Compatible Providers */}
      <section>
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-semibold flex items-center gap-2">
            <span className="material-symbols-outlined text-primary">hub</span>
            Custom Providers
          </h2>
          <div className="flex items-center gap-2">
            <Button variant="secondary" size="sm" icon="add" onClick={() => setShowAddCompatModal("openai")}>OpenAI Compatible</Button>
            <Button variant="secondary" size="sm" icon="add" onClick={() => setShowAddCompatModal("anthropic")}>Anthropic Compatible</Button>
          </div>
        </div>
        {compatibleNodes.length > 0 ? (
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-3">
            {compatibleNodes.map((node) => (
              <div key={node.id} className="bg-surface border border-border-subtle rounded-[12px] shadow-[var(--shadow-soft)] p-4 hover:border-brand-500/25 transition-all">
                <div className="flex items-center gap-2 mb-2">
                  <span className="material-symbols-outlined text-primary text-[20px]">{node.type === "anthropic-compatible" ? "smart_toy" : "hub"}</span>
                  <h3 className="text-sm font-semibold text-text-main truncate">{node.name}</h3>
                </div>
                <code className="text-[10px] text-text-muted font-mono truncate block">{node.baseUrl}</code>
                <Badge variant="info" size="sm" className="mt-2">{node.type === "anthropic-compatible" ? "Anthropic" : "OpenAI"}</Badge>
              </div>
            ))}
          </div>
        ) : (
          <p className="text-sm text-text-muted">No custom compatible providers. Add one to connect to your own API endpoints.</p>
        )}
      </section>

      {/* OAuth Providers */}
      {oauthProviders.length > 0 && (
        <section>
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-lg font-semibold flex items-center gap-2">
              <span className="material-symbols-outlined text-primary">lock</span> OAuth Providers
            </h2>
            <Button variant="ghost" size="sm" icon={testingBatch === "oauth" ? "progress_activity" : "play_arrow"}
              onClick={() => handleBatchTest("oauth")} loading={testingBatch === "oauth"}>Test All</Button>
          </div>
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-3">
            {oauthProviders.map((p) => <ProviderCard key={p.id} provider={p} stats={getProviderStats(connections, p.id)} />)}
          </div>
        </section>
      )}

      {/* Free Providers */}
      {freeProviders.length > 0 && (
        <section>
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-lg font-semibold flex items-center gap-2">
              <span className="material-symbols-outlined text-primary">stars</span> Free &amp; Free Tier Providers
            </h2>
            <Button variant="ghost" size="sm" icon={testingBatch === "free" ? "progress_activity" : "play_arrow"}
              onClick={() => handleBatchTest("free")} loading={testingBatch === "free"}>Test All</Button>
          </div>
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-3">
            {freeProviders.map((p) => <ProviderCard key={p.id} provider={p} stats={getProviderStats(connections, p.id)} />)}
          </div>
        </section>
      )}

      {/* API Key Providers */}
      {visibleApiKey.length > 0 && (
        <section>
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-lg font-semibold flex items-center gap-2">
              <span className="material-symbols-outlined text-primary">key</span> API Key Providers
            </h2>
            <Button variant="ghost" size="sm" icon={testingBatch === "apikey" ? "progress_activity" : "play_arrow"}
              onClick={() => handleBatchTest("apikey")} loading={testingBatch === "apikey"}>Test All</Button>
          </div>
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-3">
            {visibleApiKey.map((p) => <ProviderCard key={p.id} provider={p} stats={getProviderStats(connections, p.id)} />)}
          </div>
          {!showAllApiKey && apikeyProviders.length > INITIAL_APIKEY_COUNT && (
            <button onClick={() => setShowAllApiKey(true)} className="mt-3 text-sm text-primary hover:underline">
              Show all {apikeyProviders.length} providers
            </button>
          )}
        </section>
      )}

      {/* Cookie Providers */}
      {cookieProviders.length > 0 && (
        <section>
          <h2 className="text-lg font-semibold mb-4 flex items-center gap-2">
            <span className="material-symbols-outlined text-primary">cookie</span> Web Cookie Providers
          </h2>
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-3">
            {cookieProviders.map((p) => <ProviderCard key={p.id} provider={p} stats={getProviderStats(connections, p.id)} />)}
          </div>
        </section>
      )}

      {/* Empty search */}
      {oauthProviders.length === 0 && freeProviders.length === 0 && apikeyProviders.length === 0 && cookieProviders.length === 0 && (
        <div className="text-center py-20">
          <span className="material-symbols-outlined text-[48px] text-text-muted mb-4">search_off</span>
          <p className="text-text-muted">No providers match your search</p>
        </div>
      )}

      {/* Add Compatible Provider Modal */}
      <Modal isOpen={!!showAddCompatModal}
        title={`Add ${showAddCompatModal === "anthropic" ? "Anthropic" : "OpenAI"} Compatible Provider`}
        onClose={() => { if (!addCompatLoading) { setShowAddCompatModal(null); setCompatForm({ name: "", prefix: "", baseUrl: "", apiType: "chat" }); } }}>
        <div className="flex flex-col gap-4">
          <Input label="Provider Name" value={compatForm.name} onChange={(e) => setCompatForm({ ...compatForm, name: e.target.value })} placeholder="My Custom LLM" />
          <Input label="Routing Prefix" value={compatForm.prefix} onChange={(e) => setCompatForm({ ...compatForm, prefix: e.target.value })} placeholder="my-llm" hint="Used as provider/model prefix (e.g. my-llm/gpt-4)" />
          <Input label="Base URL" value={compatForm.baseUrl} onChange={(e) => setCompatForm({ ...compatForm, baseUrl: e.target.value })}
            placeholder={showAddCompatModal === "anthropic" ? "https://api.example.com/v1/messages" : "https://api.example.com/v1/chat/completions"} />
          {showAddCompatModal === "openai" && (
            <div className="flex items-center gap-3">
              <label className="text-sm font-medium text-text-main">API Type:</label>
              {["chat", "responses"].map((t) => (
                <button key={t} onClick={() => setCompatForm({ ...compatForm, apiType: t })}
                  className={cn("px-3 py-1.5 rounded text-sm border transition-colors",
                    compatForm.apiType === t ? "bg-primary text-white border-primary" : "border-border text-text-muted hover:bg-surface-2")}>
                  {t === "chat" ? "Chat" : "Responses"}
                </button>
              ))}
            </div>
          )}
          <div className="flex gap-2">
            <Button onClick={handleAddCompatible} fullWidth disabled={!compatForm.name.trim() || !compatForm.baseUrl.trim() || addCompatLoading} loading={addCompatLoading}>
              Add Provider
            </Button>
            <Button variant="ghost" fullWidth onClick={() => { setShowAddCompatModal(null); setCompatForm({ name: "", prefix: "", baseUrl: "", apiType: "chat" }); }} disabled={addCompatLoading}>
              Cancel
            </Button>
          </div>
        </div>
      </Modal>

      {/* Test Results Modal */}
      <Modal isOpen={!!testResults} title="Test Results" onClose={() => setTestResults(null)} size="lg">
        {testResults && (
          <div className="space-y-3">
            <div className="flex items-center gap-4 text-sm">
              <Badge variant="success" size="md">{testResults.summary?.passed || 0} Passed</Badge>
              <Badge variant="error" size="md">{testResults.summary?.failed || 0} Failed</Badge>
              <span className="text-text-muted">Total: {testResults.summary?.total || 0}</span>
            </div>
            {(testResults.results || []).length === 0 ? (
              <p className="text-sm text-text-muted py-4 text-center">No connections to test</p>
            ) : (
              <div className="space-y-1 max-h-[400px] overflow-y-auto custom-scrollbar">
                {(testResults.results || []).map((r, i) => (
                  <div key={i} className="flex items-center justify-between py-2 border-b border-border-subtle/50 last:border-b-0">
                    <div className="flex items-center gap-2 min-w-0">
                      <span className={cn("size-2 rounded-full shrink-0", r.valid ? "bg-green-500" : "bg-red-500")} />
                      <span className="text-sm truncate">{r.connectionName || r.provider}</span>
                    </div>
                    <div className="flex items-center gap-2 shrink-0">
                      {r.latencyMs && <span className="text-xs text-text-muted">{r.latencyMs}ms</span>}
                      <Badge variant={r.valid ? "success" : "error"} size="sm">{r.valid ? "Pass" : "Fail"}</Badge>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        )}
      </Modal>
    </div>
  );
}
