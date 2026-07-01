import { useState, useEffect, useRef } from "react";
import { useParams, Link } from "react-router-dom";
import { cn } from "../lib/cn";
import Card from "../components/Card";
import Button from "../components/Button";
import Badge from "../components/Badge";
import Input from "../components/Input";
import Modal, { ConfirmModal } from "../components/Modal";
import Toggle from "../components/Toggle";
import ProviderIcon from "../components/ProviderIcon";
import { useCopyToClipboard } from "../hooks/useCopyToClipboard";
import { useNotificationStore } from "../store/notificationStore";
import { AI_PROVIDERS } from "../constants/providers";

export default function ProviderDetail() {
  const { id: providerId } = useParams();
  const providerInfo = AI_PROVIDERS[providerId];
  const [connections, setConnections] = useState([]);
  const [models, setModels] = useState([]);
  const [modelAliases, setModelAliases] = useState({});
  const [disabledModels, setDisabledModels] = useState([]);
  const [proxyPools, setProxyPools] = useState([]);
  const [loading, setLoading] = useState(true);
  const [showAddModal, setShowAddModal] = useState(false);
  const [showAddModelModal, setShowAddModelModal] = useState(false);
  const [editingConn, setEditingConn] = useState(null);
  const [newKeyName, setNewKeyName] = useState("");
  const [newApiKey, setNewApiKey] = useState("");
  const [newModelId, setNewModelId] = useState("");
  const [addLoading, setAddLoading] = useState(false);
  const [confirmState, setConfirmState] = useState(null);
  const [testingId, setTestingId] = useState(null);
  const [testingModel, setTestingModel] = useState(null);
  const { copied, copy } = useCopyToClipboard();
  const notify = useNotificationStore.getState;

  useEffect(() => { fetchAll(); }, [providerId]);

  const fetchAll = async () => {
    setLoading(true);
    try {
      const [connRes, poolRes] = await Promise.all([
        fetch("/api/providers"),
        fetch("/api/proxy-pools"),
      ]);
      if (connRes.ok) {
        const data = await connRes.json();
        const filtered = (data.connections || [])
          .filter((c) => c.provider === providerId)
          .sort((a, b) => (a.priority || 0) - (b.priority || 0));
        setConnections(filtered);
      }
      if (poolRes.ok) {
        const data = await poolRes.json();
        setProxyPools((data.pools || []).filter((p) => p.isActive));
      }
    } catch { /* ignore */ }
    // Fetch model aliases
    try {
      const res = await fetch("/api/models/alias");
      if (res.ok) {
        const data = await res.json();
        setModelAliases(data.aliases || data || {});
      }
    } catch { /* ignore */ }
    // Fetch disabled models
    try {
      const res = await fetch(`/api/models/disabled?providerAlias=${providerId}`);
      if (res.ok) {
        const data = await res.json();
        setDisabledModels(data.disabled || []);
      }
    } catch { /* ignore */ }
    setLoading(false);
  };

  // --- Connection handlers ---
  const handleAddConnection = async () => {
    if (!newApiKey.trim()) return;
    setAddLoading(true);
    try {
      const res = await fetch("/api/providers", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          provider: providerId, authType: "apikey",
          name: newKeyName.trim() || `${providerInfo?.name || providerId} Key`,
          apiKey: newApiKey.trim(),
        }),
      });
      if (res.ok) {
        setShowAddModal(false);
        setNewKeyName(""); setNewApiKey("");
        await fetchAll();
      }
    } catch { /* ignore */ }
    setAddLoading(false);
  };

  const handleToggleActive = async (connId, isActive) => {
    try {
      const res = await fetch(`/api/providers/${connId}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ isActive }),
      });
      if (res.ok) setConnections((prev) => prev.map((c) => c.id === connId ? { ...c, isActive } : c));
    } catch { /* ignore */ }
  };

  const handleDeleteConn = (connId, connName) => {
    setConfirmState({
      title: "Delete Connection",
      message: `Delete connection "${connName}"? This cannot be undone.`,
      onConfirm: async () => {
        setConfirmState(null);
        try {
          const res = await fetch(`/api/providers/${connId}`, { method: "DELETE" });
          if (res.ok) setConnections((prev) => prev.filter((c) => c.id !== connId));
        } catch { /* ignore */ }
      },
    });
  };

  const handleTestConn = async (connId) => {
    setTestingId(connId);
    try {
      const res = await fetch(`/api/providers/${connId}/test`, { method: "POST" });
      const data = await res.json();
      setConnections((prev) => prev.map((c) =>
        c.id === connId ? { ...c, testStatus: data.valid ? "active" : "error", lastError: data.error || null } : c
      ));
    } catch { /* ignore */ }
    setTestingId(null);
  };

  const handleSwapPriority = async (index, direction) => {
    const targetIndex = index + direction;
    if (targetIndex < 0 || targetIndex >= connections.length) return;
    const a = connections[index];
    const b = connections[targetIndex];
    try {
      await Promise.all([
        fetch(`/api/providers/${a.id}`, { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ priority: b.priority }) }),
        fetch(`/api/providers/${b.id}`, { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ priority: a.priority }) }),
      ]);
      await fetchAll();
    } catch { /* ignore */ }
  };

  const handleUpdateProxy = async (connId, proxyPoolId) => {
    try {
      await fetch(`/api/providers/${connId}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ proxyPoolId }),
      });
      await fetchAll();
    } catch { /* ignore */ }
  };

  const handleEditSave = async (connId, patch) => {
    try {
      const res = await fetch(`/api/providers/${connId}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(patch),
      });
      if (res.ok) {
        setEditingConn(null);
        await fetchAll();
      }
    } catch { /* ignore */ }
  };

  // --- Model handlers ---
  const handleTestModel = async (modelId) => {
    setTestingModel(modelId);
    try {
      const res = await fetch("/api/models/test", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ model: `${providerId}/${modelId}` }),
      });
      // Result shown via test status
    } catch { /* ignore */ }
    setTimeout(() => setTestingModel(null), 1500);
  };

  const handleSetAlias = async (alias, model) => {
    try {
      await fetch("/api/models/alias", {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ alias, model: `${providerId}/${model}` }),
      });
      setModelAliases((prev) => ({ ...prev, [alias]: `${providerId}/${model}` }));
      notify().success(`Alias "${alias}" set`);
    } catch { /* ignore */ }
  };

  const handleDeleteAlias = async (alias) => {
    try {
      await fetch(`/api/models/alias?alias=${encodeURIComponent(alias)}`, { method: "DELETE" });
      setModelAliases((prev) => { const next = { ...prev }; delete next[alias]; return next; });
    } catch { /* ignore */ }
  };

  const handleToggleModelDisabled = async (modelId, disable) => {
    try {
      if (disable) {
        await fetch("/api/models/disabled", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ providerAlias: providerId, id: modelId }),
        });
        setDisabledModels((prev) => [...prev, modelId]);
      } else {
        await fetch(`/api/models/disabled?providerAlias=${providerId}&id=${encodeURIComponent(modelId)}`, { method: "DELETE" });
        setDisabledModels((prev) => prev.filter((m) => m !== modelId));
      }
    } catch { /* ignore */ }
  };

  const handleAddCustomModel = async () => {
    if (!newModelId.trim()) return;
    const model = newModelId.trim();
    await handleSetAlias(model, model);
    setNewModelId("");
    setShowAddModelModal(false);
  };

  // --- Computed ---
  const getStatusBadge = (conn) => {
    if (!conn.isActive) return <Badge variant="warning" size="sm">Disabled</Badge>;
    const st = conn.testStatus;
    if (st === "error" || st === "expired") return <Badge variant="error" size="sm" dot>Error</Badge>;
    if (st === "active" || st === "success") return <Badge variant="success" size="sm" dot>Active</Badge>;
    return <Badge variant="default" size="sm">Unknown</Badge>;
  };

  // Get reverse alias map: model -> alias
  const reverseAliases = {};
  for (const [alias, target] of Object.entries(modelAliases)) {
    if (typeof target === "string" && target.startsWith(`${providerId}/`)) {
      const modelName = target.replace(`${providerId}/`, "");
      reverseAliases[modelName] = alias;
    }
  }

  if (loading) {
    return (
      <div className="flex flex-col gap-6">
        {[1, 2, 3].map((i) => <div key={i} className="bg-surface border border-border-subtle rounded-[12px] shadow-[var(--shadow-soft)] p-6 animate-pulse h-32" />)}
      </div>
    );
  }

  if (!providerInfo) {
    return (
      <div className="text-center py-20">
        <span className="material-symbols-outlined text-[48px] text-text-muted mb-4">error</span>
        <h2 className="text-xl font-semibold mb-2">Provider Not Found</h2>
        <p className="text-text-muted mb-4">Provider "{providerId}" is not recognized.</p>
        <Link to="/providers"><Button variant="secondary" icon="arrow_back">Back to Providers</Button></Link>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-6">
      {/* Header */}
      <div className="flex items-center gap-4">
        <Link to="/providers" className="p-2 hover:bg-surface-2 rounded-lg transition-colors text-text-muted hover:text-text-main">
          <span className="material-symbols-outlined text-[20px]">arrow_back</span>
        </Link>
        <ProviderIcon src={`/providers/${providerInfo.id}.png`} alt={providerInfo.name} size={48} className="rounded-xl object-contain"
          fallbackText={providerInfo.textIcon || providerInfo.name?.slice(0, 2)?.toUpperCase()} fallbackColor={providerInfo.color} />
        <div className="flex-1 min-w-0">
          <h1 className="text-xl font-bold text-text-main">{providerInfo.name}</h1>
          <p className="text-sm text-text-muted">{connections.length} connection{connections.length !== 1 ? "s" : ""}</p>
        </div>
      </div>

      {/* Deprecation banner */}
      {providerInfo.deprecated && (
        <div className="flex items-center gap-2 px-4 py-3 rounded-lg bg-amber-500/10 border border-amber-500/20 text-amber-700 dark:text-amber-400 text-sm">
          <span className="material-symbols-outlined text-[18px]">warning</span>
          <p>⚠️ Risk Notice: This provider may restrict or ban accounts used with proxies.</p>
        </div>
      )}

      {/* Connections Card */}
      <Card>
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-semibold flex items-center gap-2">
            <span className="material-symbols-outlined text-primary">cable</span>
            Connections
          </h2>
          <Button icon="add" onClick={() => setShowAddModal(true)} size="sm">Add Connection</Button>
        </div>

        {connections.length === 0 ? (
          <div className="text-center py-12">
            <div className="inline-flex items-center justify-center w-16 h-16 rounded-full bg-primary/10 text-primary mb-4">
              <span className="material-symbols-outlined text-[32px]">cable</span>
            </div>
            <p className="text-text-main font-medium mb-1">No connections</p>
            <p className="text-sm text-text-muted mb-4">Add your first connection to get started</p>
            <Button icon="add" onClick={() => setShowAddModal(true)}>Add Connection</Button>
          </div>
        ) : (
          <div className="flex flex-col divide-y divide-border-subtle/50">
            {connections.map((conn, index) => (
              <ConnectionRow
                key={conn.id}
                connection={conn}
                proxyPools={proxyPools}
                isFirst={index === 0}
                isLast={index === connections.length - 1}
                onMoveUp={() => handleSwapPriority(index, -1)}
                onMoveDown={() => handleSwapPriority(index, 1)}
                onToggleActive={(checked) => handleToggleActive(conn.id, checked)}
                onUpdateProxy={(poolId) => handleUpdateProxy(conn.id, poolId)}
                onEdit={() => setEditingConn(conn)}
                onDelete={() => handleDeleteConn(conn.id, conn.name || conn.email || "Connection")}
                onTest={() => handleTestConn(conn.id)}
                testingId={testingId}
              />
            ))}
          </div>
        )}
      </Card>

      {/* Models Card */}
      <Card>
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-semibold flex items-center gap-2">
            <span className="material-symbols-outlined text-primary">model_training</span>
            Available Models
          </h2>
          <div className="flex items-center gap-2">
            <Button variant="ghost" size="sm" icon="add" onClick={() => setShowAddModelModal(true)}>Add Model</Button>
          </div>
        </div>

        {/* Model aliases (custom models added by user) */}
        {Object.entries(reverseAliases).length > 0 || Object.keys(modelAliases).length > 0 ? (
          <div className="space-y-1">
            {Object.entries(modelAliases)
              .filter(([, target]) => typeof target === "string" && target.startsWith(`${providerId}/`))
              .map(([alias, target]) => {
                const modelId = target.replace(`${providerId}/`, "");
                const isDisabled = disabledModels.includes(modelId);
                return (
                  <ModelRow
                    key={alias}
                    modelId={modelId}
                    alias={alias !== modelId ? alias : null}
                    fullModel={target}
                    isDisabled={isDisabled}
                    copied={copied}
                    onCopy={copy}
                    onTest={() => handleTestModel(modelId)}
                    isTesting={testingModel === modelId}
                    onDisable={() => handleToggleModelDisabled(modelId, !isDisabled)}
                    onDeleteAlias={() => handleDeleteAlias(alias)}
                  />
                );
              })}
          </div>
        ) : (
          <div className="text-center py-8">
            <span className="material-symbols-outlined text-[32px] text-text-muted mb-2">model_training</span>
            <p className="text-sm text-text-muted">No models configured. Click "Add Model" to add one.</p>
          </div>
        )}
      </Card>

      {/* Add Connection Modal */}
      <Modal isOpen={showAddModal} title="Add Connection" onClose={() => { if (!addLoading) { setShowAddModal(false); setNewKeyName(""); setNewApiKey(""); } }}>
        <div className="flex flex-col gap-4">
          <Input label="Connection Name" value={newKeyName} onChange={(e) => setNewKeyName(e.target.value)} placeholder={`${providerInfo.name} Key`} />
          <Input label="API Key" type="password" value={newApiKey} onChange={(e) => setNewApiKey(e.target.value)} placeholder="Enter your API key" />
          <div className="flex gap-2">
            <Button onClick={handleAddConnection} fullWidth disabled={!newApiKey.trim() || addLoading} loading={addLoading}>Add</Button>
            <Button onClick={() => { setShowAddModal(false); setNewKeyName(""); setNewApiKey(""); }} variant="ghost" fullWidth disabled={addLoading}>Cancel</Button>
          </div>
        </div>
      </Modal>

      {/* Add Custom Model Modal */}
      <Modal isOpen={showAddModelModal} title="Add Custom Model" onClose={() => { setShowAddModelModal(false); setNewModelId(""); }} size="sm">
        <div className="flex flex-col gap-4">
          <Input label="Model ID" value={newModelId} onChange={(e) => setNewModelId(e.target.value)} placeholder="e.g. gpt-4o-mini"
            hint={`Will be accessible as ${providerId}/${newModelId || "model-id"}`} />
          <div className="flex gap-2">
            <Button onClick={handleAddCustomModel} fullWidth disabled={!newModelId.trim()}>Add Model</Button>
            <Button variant="ghost" fullWidth onClick={() => { setShowAddModelModal(false); setNewModelId(""); }}>Cancel</Button>
          </div>
        </div>
      </Modal>

      {/* Edit Connection Modal */}
      {editingConn && (
        <EditConnectionModal
          connection={editingConn}
          onClose={() => setEditingConn(null)}
          onSave={(patch) => handleEditSave(editingConn.id, patch)}
        />
      )}

      {/* Confirm Modal */}
      <ConfirmModal isOpen={!!confirmState} onClose={() => setConfirmState(null)} onConfirm={confirmState?.onConfirm}
        title={confirmState?.title || "Confirm"} message={confirmState?.message} variant="danger" />
    </div>
  );
}

// --- ConnectionRow ---
function ConnectionRow({ connection, proxyPools, isFirst, isLast, onMoveUp, onMoveDown, onToggleActive, onUpdateProxy, onEdit, onDelete, onTest, testingId }) {
  const [showProxyDropdown, setShowProxyDropdown] = useState(false);
  const proxyRef = useRef(null);

  const boundProxyPoolId = connection.providerSpecificData?.proxyPoolId || connection.proxyPoolId || null;
  const boundPool = boundProxyPoolId ? (proxyPools || []).find((p) => p.id === boundProxyPoolId) : null;
  const hasProxy = !!boundProxyPoolId;

  useEffect(() => {
    if (!showProxyDropdown) return;
    const handler = (e) => { if (proxyRef.current && !proxyRef.current.contains(e.target)) setShowProxyDropdown(false); };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, [showProxyDropdown]);

  const authType = connection.authType || "apikey";
  const authIcon = authType === "oauth" ? "lock" : authType === "cookie" ? "cookie" : "key";
  const authLabel = authType === "oauth" ? "OAuth" : authType === "cookie" ? "Cookie" : "API Key";
  const displayName = connection.name || connection.email || connection.displayName || "Connection";
  const isTesting = testingId === connection.id;

  const getStatusVariant = () => {
    if (!connection.isActive) return "default";
    const st = connection.testStatus;
    if (st === "active" || st === "success") return "success";
    if (st === "error" || st === "expired" || st === "unavailable") return "error";
    return "default";
  };

  return (
    <div className={cn("group flex min-w-0 flex-col gap-3 rounded-lg p-2 transition-colors hover:bg-black/[0.02] dark:hover:bg-white/[0.02] sm:flex-row sm:items-center sm:justify-between", !connection.isActive && "opacity-60")}>
      <div className="flex min-w-0 flex-1 items-start gap-2 sm:items-center sm:gap-3">
        {/* Priority arrows */}
        <div className="flex shrink-0 flex-col">
          <button onClick={onMoveUp} disabled={isFirst} className={cn("p-0.5 rounded", isFirst ? "text-text-muted/30 cursor-not-allowed" : "hover:bg-surface-2 text-text-muted hover:text-primary")}>
            <span className="material-symbols-outlined text-sm">keyboard_arrow_up</span>
          </button>
          <button onClick={onMoveDown} disabled={isLast} className={cn("p-0.5 rounded", isLast ? "text-text-muted/30 cursor-not-allowed" : "hover:bg-surface-2 text-text-muted hover:text-primary")}>
            <span className="material-symbols-outlined text-sm">keyboard_arrow_down</span>
          </button>
        </div>
        <span className="material-symbols-outlined shrink-0 text-base text-text-muted">{authIcon}</span>
        <div className="flex-1 min-w-0">
          <p className="text-sm font-medium truncate">{displayName}</p>
          <div className="mt-1 flex min-w-0 flex-wrap items-center gap-1.5">
            <Badge variant={getStatusVariant()} size="sm" dot>
              {!connection.isActive ? "disabled" : (connection.testStatus || "Unknown")}
            </Badge>
            <Badge variant="default" size="sm">{authLabel}</Badge>
            {hasProxy && <Badge variant={boundPool?.isActive ? "success" : "error"} size="sm">Proxy</Badge>}
            {connection.lastError && connection.isActive && (
              <span className="max-w-full truncate text-xs text-red-500 sm:max-w-[300px]" title={connection.lastError}>{connection.lastError}</span>
            )}
            <span className="text-xs text-text-muted">#{connection.priority}</span>
          </div>
          {hasProxy && boundPool && (
            <div className="mt-1">
              <span className="text-[11px] text-text-muted truncate">Pool: {boundPool.name}</span>
            </div>
          )}
        </div>
      </div>
      <div className="flex w-full items-center justify-between gap-2 sm:w-auto sm:justify-end">
        <div className="grid flex-1 grid-cols-4 gap-1 sm:flex sm:flex-none">
          {/* Test */}
          <button onClick={onTest} disabled={isTesting} className="flex flex-col items-center rounded px-2 py-1 text-text-muted hover:bg-black/5 hover:text-primary dark:hover:bg-white/5">
            <span className={cn("material-symbols-outlined text-[18px]", isTesting && "animate-spin")}>{isTesting ? "progress_activity" : "science"}</span>
            <span className="text-[10px] leading-tight">Test</span>
          </button>
          {/* Proxy */}
          {(proxyPools || []).length > 0 && (
            <div className="relative" ref={proxyRef}>
              <button onClick={() => setShowProxyDropdown((v) => !v)} className={cn("flex w-full flex-col items-center rounded px-2 py-1 transition-colors hover:bg-black/5 dark:hover:bg-white/5", hasProxy ? "text-primary" : "text-text-muted hover:text-primary")}>
                <span className="material-symbols-outlined text-[18px]">lan</span>
                <span className="text-[10px] leading-tight">Proxy</span>
              </button>
              {showProxyDropdown && (
                <div className="absolute right-0 top-full z-50 mt-1 min-w-[160px] rounded-lg border border-border bg-surface py-1 shadow-lg">
                  <button onClick={() => { onUpdateProxy(null); setShowProxyDropdown(false); }} className={cn("w-full text-left px-3 py-1.5 text-sm hover:bg-black/5 dark:hover:bg-white/5", !boundProxyPoolId ? "text-primary font-medium" : "text-text-main")}>None</button>
                  {proxyPools.map((pool) => (
                    <button key={pool.id} onClick={() => { onUpdateProxy(pool.id); setShowProxyDropdown(false); }}
                      className={cn("w-full text-left px-3 py-1.5 text-sm hover:bg-black/5 dark:hover:bg-white/5", boundProxyPoolId === pool.id ? "text-primary font-medium" : "text-text-main")}>{pool.name}</button>
                  ))}
                </div>
              )}
            </div>
          )}
          {/* Edit */}
          <button onClick={onEdit} className="flex flex-col items-center rounded px-2 py-1 text-text-muted hover:bg-black/5 hover:text-primary dark:hover:bg-white/5">
            <span className="material-symbols-outlined text-[18px]">edit</span>
            <span className="text-[10px] leading-tight">Edit</span>
          </button>
          {/* Delete */}
          <button onClick={onDelete} className="flex flex-col items-center rounded px-2 py-1 text-red-500 hover:bg-red-500/10">
            <span className="material-symbols-outlined text-[18px]">delete</span>
            <span className="text-[10px] leading-tight">Delete</span>
          </button>
        </div>
        <Toggle size="sm" checked={connection.isActive ?? true} onChange={onToggleActive} />
      </div>
    </div>
  );
}

// --- ModelRow ---
function ModelRow({ modelId, alias, fullModel, isDisabled, copied, onCopy, onTest, isTesting, onDisable, onDeleteAlias }) {
  return (
    <div className={cn("group flex items-center gap-2 rounded-lg border px-3 py-2 hover:bg-surface-2/50 transition-colors", isDisabled ? "border-red-500/20 opacity-60" : "border-border-subtle")}>
      <span className="material-symbols-outlined shrink-0 text-base text-text-muted">smart_toy</span>
      <div className="flex-1 min-w-0">
        <code className="truncate rounded bg-surface-2 px-1.5 py-0.5 font-mono text-xs text-text-muted">{fullModel}</code>
        {alias && <span className="ml-2 text-[10px] text-primary">alias: {alias}</span>}
      </div>
      <div className="flex items-center gap-1 opacity-100 sm:opacity-0 sm:group-hover:opacity-100 transition-opacity">
        {onTest && (
          <button onClick={onTest} disabled={isTesting} className="p-1 rounded text-text-muted hover:bg-surface-2 hover:text-primary">
            <span className={cn("material-symbols-outlined text-[16px]", isTesting && "animate-spin")}>{isTesting ? "progress_activity" : "science"}</span>
          </button>
        )}
        <button onClick={() => onCopy(fullModel, modelId)} className="p-1 rounded text-text-muted hover:bg-surface-2 hover:text-primary">
          <span className="material-symbols-outlined text-[16px]">{copied === modelId ? "check" : "content_copy"}</span>
        </button>
        <button onClick={onDisable} className={cn("p-1 rounded", isDisabled ? "text-green-500 hover:bg-green-500/10" : "text-amber-500 hover:bg-amber-500/10")} title={isDisabled ? "Enable" : "Disable"}>
          <span className="material-symbols-outlined text-[16px]">{isDisabled ? "toggle_on" : "toggle_off"}</span>
        </button>
        {onDeleteAlias && (
          <button onClick={onDeleteAlias} className="p-1 rounded text-red-500 hover:bg-red-500/10" title="Remove model">
            <span className="material-symbols-outlined text-[16px]">close</span>
          </button>
        )}
      </div>
    </div>
  );
}

// --- EditConnectionModal ---
function EditConnectionModal({ connection, onClose, onSave }) {
  const [name, setName] = useState(connection.name || "");
  const [defaultModel, setDefaultModel] = useState(connection.defaultModel || "");
  const [saving, setSaving] = useState(false);

  const handleSave = async () => {
    setSaving(true);
    await onSave({ name, defaultModel });
    setSaving(false);
  };

  return (
    <Modal isOpen={true} title="Edit Connection" onClose={onClose}>
      <div className="flex flex-col gap-4">
        <Input label="Display Name" value={name} onChange={(e) => setName(e.target.value)} placeholder="Connection name" />
        <Input label="Default Model" value={defaultModel} onChange={(e) => setDefaultModel(e.target.value)} placeholder="e.g. gpt-4o (optional)" hint="Override the model for requests routed through this connection" />
        <div className="flex gap-2">
          <Button onClick={handleSave} fullWidth loading={saving}>Save</Button>
          <Button variant="ghost" fullWidth onClick={onClose} disabled={saving}>Cancel</Button>
        </div>
      </div>
    </Modal>
  );
}
