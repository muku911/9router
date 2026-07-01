import { useState, useEffect } from "react";
import Card from "../components/Card";
import Button from "../components/Button";
import Badge from "../components/Badge";
import Input from "../components/Input";
import Modal, { ConfirmModal } from "../components/Modal";
import Toggle from "../components/Toggle";
import { useNotificationStore } from "../store/notificationStore";

export default function ProxyPools() {
  const [pools, setPools] = useState([]);
  const [loading, setLoading] = useState(true);
  const [showModal, setShowModal] = useState(false);
  const [editPool, setEditPool] = useState(null);
  const [confirmState, setConfirmState] = useState(null);
  const [testingId, setTestingId] = useState(null);
  const [form, setForm] = useState({ name: "", proxyUrl: "", noProxy: "", strictProxy: false });

  useEffect(() => { fetchPools(); }, []);

  const fetchPools = async () => {
    try {
      const res = await fetch("/api/proxy-pools");
      if (res.ok) {
        const data = await res.json();
        setPools(data.pools || []);
      }
    } catch { /* ignore */ }
    setLoading(false);
  };

  const openCreate = () => {
    setEditPool(null);
    setForm({ name: "", proxyUrl: "", noProxy: "", strictProxy: false });
    setShowModal(true);
  };

  const openEdit = (pool) => {
    setEditPool(pool);
    setForm({
      name: pool.name || "",
      proxyUrl: pool.proxyUrl || "",
      noProxy: pool.noProxy || "",
      strictProxy: pool.strictProxy || false,
    });
    setShowModal(true);
  };

  const handleSave = async () => {
    if (!form.name.trim() || !form.proxyUrl.trim()) return;
    try {
      const method = editPool ? "PUT" : "POST";
      const url = editPool ? `/api/proxy-pools/${editPool.id}` : "/api/proxy-pools";
      const res = await fetch(url, {
        method,
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(form),
      });
      if (res.ok) {
        setShowModal(false);
        await fetchPools();
      }
    } catch { /* ignore */ }
  };

  const handleDelete = (pool) => {
    setConfirmState({
      title: "Delete Proxy Pool",
      message: `Delete proxy pool "${pool.name}"?${pool.boundConnectionCount > 0 ? " Warning: this pool is bound to connections." : ""}`,
      onConfirm: async () => {
        setConfirmState(null);
        try {
          await fetch(`/api/proxy-pools/${pool.id}`, { method: "DELETE" });
          await fetchPools();
        } catch { /* ignore */ }
      },
    });
  };

  const handleToggle = async (pool, isActive) => {
    try {
      await fetch(`/api/proxy-pools/${pool.id}`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ ...pool, isActive }),
      });
      setPools((prev) => prev.map((p) => (p.id === pool.id ? { ...p, isActive } : p)));
    } catch { /* ignore */ }
  };

  const handleTest = async (pool) => {
    setTestingId(pool.id);
    try {
      const res = await fetch(`/api/proxy-pools/${pool.id}/test`, { method: "POST" });
      const data = await res.json();
      setPools((prev) =>
        prev.map((p) =>
          p.id === pool.id
            ? { ...p, testStatus: data.valid ? "active" : "error", lastError: data.error }
            : p
        )
      );
    } catch { /* ignore */ }
    setTestingId(null);
  };

  if (loading) {
    return (
      <div className="flex flex-col gap-4">
        {[1, 2, 3].map((i) => (
          <div key={i} className="bg-surface border border-border-subtle rounded-[12px] shadow-[var(--shadow-soft)] p-6 animate-pulse h-20" />
        ))}
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-6">
      <div className="flex items-center justify-between">
        <p className="text-sm text-text-muted">{pools.length} proxy pool{pools.length !== 1 ? "s" : ""}</p>
        <Button icon="add" onClick={openCreate} size="sm">Add Pool</Button>
      </div>

      {pools.length === 0 ? (
        <Card>
          <div className="text-center py-12">
            <div className="inline-flex items-center justify-center w-16 h-16 rounded-full bg-primary/10 text-primary mb-4">
              <span className="material-symbols-outlined text-[32px]">lan</span>
            </div>
            <p className="text-text-main font-medium mb-1">No proxy pools</p>
            <p className="text-sm text-text-muted mb-4">Add proxy pools to route traffic through</p>
            <Button icon="add" onClick={openCreate}>Add Pool</Button>
          </div>
        </Card>
      ) : (
        <div className="flex flex-col gap-3">
          {pools.map((pool) => (
            <Card key={pool.id} padding="sm">
              <div className="flex items-center justify-between">
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    <h3 className="text-sm font-semibold text-text-main">{pool.name}</h3>
                    {pool.testStatus === "active" && <Badge variant="success" size="sm" dot>OK</Badge>}
                    {pool.testStatus === "error" && <Badge variant="error" size="sm" dot>Error</Badge>}
                    {!pool.isActive && <Badge variant="warning" size="sm">Disabled</Badge>}
                    {pool.strictProxy && <Badge variant="info" size="sm">Strict</Badge>}
                  </div>
                  <code className="text-xs text-text-muted font-mono mt-1 block truncate">{pool.proxyUrl}</code>
                  {pool.lastError && <p className="text-xs text-red-500 mt-1 truncate">{pool.lastError}</p>}
                </div>
                <div className="flex items-center gap-2 shrink-0 ml-3">
                  <button
                    onClick={() => handleTest(pool)}
                    disabled={testingId === pool.id}
                    className="p-1.5 hover:bg-surface-2 rounded text-text-muted hover:text-primary transition-colors"
                    title="Test"
                  >
                    <span className={`material-symbols-outlined text-[18px] ${testingId === pool.id ? "animate-spin" : ""}`}>
                      {testingId === pool.id ? "progress_activity" : "speed"}
                    </span>
                  </button>
                  <button
                    onClick={() => openEdit(pool)}
                    className="p-1.5 hover:bg-surface-2 rounded text-text-muted hover:text-primary transition-colors"
                    title="Edit"
                  >
                    <span className="material-symbols-outlined text-[18px]">edit</span>
                  </button>
                  <Toggle size="sm" checked={pool.isActive ?? true} onChange={(v) => handleToggle(pool, v)} />
                  <button
                    onClick={() => handleDelete(pool)}
                    className="p-1.5 hover:bg-red-500/10 rounded text-red-500 transition-colors"
                    title="Delete"
                  >
                    <span className="material-symbols-outlined text-[18px]">delete</span>
                  </button>
                </div>
              </div>
            </Card>
          ))}
        </div>
      )}

      {/* Create/Edit Modal */}
      <Modal isOpen={showModal} title={editPool ? "Edit Proxy Pool" : "Add Proxy Pool"} onClose={() => setShowModal(false)}>
        <div className="flex flex-col gap-4">
          <Input label="Name" value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} placeholder="US Proxy 1" />
          <Input label="Proxy URL" value={form.proxyUrl} onChange={(e) => setForm({ ...form, proxyUrl: e.target.value })} placeholder="http://user:pass@proxy.example.com:8080" hint="HTTP, HTTPS, or SOCKS5 proxy URL" />
          <Input label="No Proxy (comma-separated)" value={form.noProxy} onChange={(e) => setForm({ ...form, noProxy: e.target.value })} placeholder="localhost,127.0.0.1" />
          <Toggle checked={form.strictProxy} onChange={(v) => setForm({ ...form, strictProxy: v })} label="Strict Proxy" description="Fail requests if proxy is unreachable (instead of direct)" />
          <div className="flex gap-2">
            <Button onClick={handleSave} fullWidth disabled={!form.name.trim() || !form.proxyUrl.trim()}>
              {editPool ? "Update" : "Create"}
            </Button>
            <Button variant="ghost" fullWidth onClick={() => setShowModal(false)}>Cancel</Button>
          </div>
        </div>
      </Modal>

      <ConfirmModal
        isOpen={!!confirmState}
        onClose={() => setConfirmState(null)}
        onConfirm={confirmState?.onConfirm}
        title={confirmState?.title || "Confirm"}
        message={confirmState?.message}
        variant="danger"
      />
    </div>
  );
}
