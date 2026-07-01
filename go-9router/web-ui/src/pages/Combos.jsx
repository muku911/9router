import { useState, useEffect } from "react";
import Card from "../components/Card";
import Button from "../components/Button";
import Badge from "../components/Badge";
import Input from "../components/Input";
import Modal, { ConfirmModal } from "../components/Modal";
import { useCopyToClipboard } from "../hooks/useCopyToClipboard";

export default function Combos() {
  const [combos, setCombos] = useState([]);
  const [loading, setLoading] = useState(true);
  const [showModal, setShowModal] = useState(false);
  const [editCombo, setEditCombo] = useState(null);
  const [confirmState, setConfirmState] = useState(null);
  const [form, setForm] = useState({ name: "", models: [""] });
  const { copied, copy } = useCopyToClipboard();

  useEffect(() => { fetchCombos(); }, []);

  const fetchCombos = async () => {
    try {
      const res = await fetch("/api/combos");
      if (res.ok) {
        const data = await res.json();
        setCombos((data.combos || []).filter((c) => !c.kind));
      }
    } catch { /* ignore */ }
    setLoading(false);
  };

  const openCreate = () => {
    setEditCombo(null);
    setForm({ name: "", models: [""] });
    setShowModal(true);
  };

  const openEdit = (combo) => {
    setEditCombo(combo);
    setForm({ name: combo.name, models: [...combo.models, ""] });
    setShowModal(true);
  };

  const handleSave = async () => {
    const name = form.name.trim();
    const models = form.models.filter((m) => m.trim());
    if (!name || models.length === 0) return;
    if (!/^[a-zA-Z0-9_.\-]+$/.test(name)) return;

    try {
      const method = editCombo ? "PUT" : "POST";
      const url = editCombo ? `/api/combos/${editCombo.id}` : "/api/combos";
      const res = await fetch(url, {
        method,
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ name, models }),
      });
      if (res.ok) {
        setShowModal(false);
        await fetchCombos();
      }
    } catch { /* ignore */ }
  };

  const handleDelete = (combo) => {
    setConfirmState({
      title: "Delete Combo",
      message: `Delete combo "${combo.name}"? This cannot be undone.`,
      onConfirm: async () => {
        setConfirmState(null);
        try {
          await fetch(`/api/combos/${combo.id}`, { method: "DELETE" });
          await fetchCombos();
        } catch { /* ignore */ }
      },
    });
  };

  const updateModel = (index, value) => {
    setForm((prev) => {
      const models = [...prev.models];
      models[index] = value;
      // Auto-add empty slot at end
      if (index === models.length - 1 && value.trim()) {
        models.push("");
      }
      return { ...prev, models };
    });
  };

  const removeModel = (index) => {
    setForm((prev) => ({
      ...prev,
      models: prev.models.filter((_, i) => i !== index),
    }));
  };

  const moveModel = (index, direction) => {
    setForm((prev) => {
      const models = [...prev.models];
      const targetIndex = index + direction;
      if (targetIndex < 0 || targetIndex >= models.length) return prev;
      [models[index], models[targetIndex]] = [models[targetIndex], models[index]];
      return { ...prev, models };
    });
  };

  if (loading) {
    return (
      <div className="flex flex-col gap-4">
        {[1, 2].map((i) => (
          <div key={i} className="bg-surface border border-border-subtle rounded-[12px] shadow-[var(--shadow-soft)] p-6 animate-pulse h-32" />
        ))}
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-6">
      <div className="flex items-center justify-between">
        <p className="text-sm text-text-muted">{combos.length} combo{combos.length !== 1 ? "s" : ""}</p>
        <Button icon="add" onClick={openCreate} size="sm">Create Combo</Button>
      </div>

      {combos.length === 0 ? (
        <Card>
          <div className="text-center py-12">
            <div className="inline-flex items-center justify-center w-16 h-16 rounded-full bg-primary/10 text-primary mb-4">
              <span className="material-symbols-outlined text-[32px]">layers</span>
            </div>
            <p className="text-text-main font-medium mb-1">No combos yet</p>
            <p className="text-sm text-text-muted mb-4">Create a combo to chain multiple models with automatic fallback</p>
            <Button icon="add" onClick={openCreate}>Create Combo</Button>
          </div>
        </Card>
      ) : (
        <div className="flex flex-col gap-3">
          {combos.map((combo) => (
            <Card key={combo.id} padding="sm">
              <div className="flex items-center justify-between">
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2 mb-2">
                    <h3 className="text-sm font-semibold text-text-main">{combo.name}</h3>
                    <Badge variant="primary" size="sm">{combo.models?.length || 0} models</Badge>
                  </div>
                  <div className="flex flex-wrap gap-1.5">
                    {(combo.models || []).map((model, i) => (
                      <span
                        key={`${model}-${i}`}
                        className="inline-flex items-center gap-1 px-2 py-0.5 rounded bg-surface-2 text-xs text-text-muted font-mono"
                      >
                        <span className="text-[10px] text-text-subtle">{i + 1}.</span>
                        {model}
                      </span>
                    ))}
                  </div>
                </div>
                <div className="flex items-center gap-1.5 shrink-0 ml-3">
                  <button
                    onClick={() => copy(combo.name, combo.id)}
                    className="p-1.5 hover:bg-surface-2 rounded text-text-muted hover:text-primary transition-colors"
                    title="Copy combo name"
                  >
                    <span className="material-symbols-outlined text-[18px]">
                      {copied === combo.id ? "check" : "content_copy"}
                    </span>
                  </button>
                  <button
                    onClick={() => openEdit(combo)}
                    className="p-1.5 hover:bg-surface-2 rounded text-text-muted hover:text-primary transition-colors"
                    title="Edit"
                  >
                    <span className="material-symbols-outlined text-[18px]">edit</span>
                  </button>
                  <button
                    onClick={() => handleDelete(combo)}
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
      <Modal isOpen={showModal} title={editCombo ? "Edit Combo" : "Create Combo"} onClose={() => setShowModal(false)} size="md">
        <div className="flex flex-col gap-4">
          <Input
            label="Combo Name"
            value={form.name}
            onChange={(e) => setForm({ ...form, name: e.target.value })}
            placeholder="my-combo"
            hint="Use as model name: letters, numbers, hyphens, dots, underscores"
            error={form.name && !/^[a-zA-Z0-9_.\-]*$/.test(form.name) ? "Invalid characters" : null}
          />
          <div>
            <label className="text-sm font-medium text-text-main mb-2 block">
              Models (in fallback order)
            </label>
            <div className="space-y-2">
              {form.models.map((model, i) => (
                <div key={i} className="flex items-center gap-1.5">
                  <span className="text-xs text-text-muted w-5 text-right shrink-0">{i + 1}.</span>
                  <input
                    type="text"
                    value={model}
                    onChange={(e) => updateModel(i, e.target.value)}
                    placeholder="e.g. openai/gpt-4o"
                    className="flex-1 px-3 py-2 text-sm rounded-[10px] bg-surface-2 border border-transparent focus:border-brand-500/40 focus:ring-2 focus:ring-brand-500/30 focus:outline-none text-text-main placeholder:text-text-muted/50"
                  />
                  {model.trim() && (
                    <>
                      <button onClick={() => moveModel(i, -1)} disabled={i === 0} className="p-1 hover:bg-surface-2 rounded text-text-muted disabled:opacity-30">
                        <span className="material-symbols-outlined text-[16px]">arrow_upward</span>
                      </button>
                      <button onClick={() => moveModel(i, 1)} disabled={i >= form.models.filter((m) => m.trim()).length - 1} className="p-1 hover:bg-surface-2 rounded text-text-muted disabled:opacity-30">
                        <span className="material-symbols-outlined text-[16px]">arrow_downward</span>
                      </button>
                      <button onClick={() => removeModel(i)} className="p-1 hover:bg-red-500/10 rounded text-red-500">
                        <span className="material-symbols-outlined text-[16px]">close</span>
                      </button>
                    </>
                  )}
                </div>
              ))}
            </div>
          </div>
          <div className="flex gap-2">
            <Button onClick={handleSave} fullWidth disabled={!form.name.trim() || !form.models.some((m) => m.trim())}>
              {editCombo ? "Update" : "Create"}
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
