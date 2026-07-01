import { useState, useEffect } from "react";
import { useNavigate } from "react-router-dom";
import Card from "../components/Card";
import Button from "../components/Button";
import Input from "../components/Input";
import Toggle from "../components/Toggle";
import Select from "../components/Select";
import ThemeToggle from "../components/ThemeToggle";
import Modal, { ConfirmModal } from "../components/Modal";
import { useNotificationStore } from "../store/notificationStore";
import { LOCALE_NAMES, LOCALES, getCurrentLocale, reloadTranslations } from "../i18n";

export default function Profile() {
  const navigate = useNavigate();
  const [settings, setSettings] = useState(null);
  const [loading, setLoading] = useState(true);
  const [confirmState, setConfirmState] = useState(null);
  const [proxyTesting, setProxyTesting] = useState(false);
  const [oidcTesting, setOidcTesting] = useState(false);
  const notify = useNotificationStore.getState;

  // Password change form
  const [pwForm, setPwForm] = useState({ current: "", newPw: "", confirm: "" });
  const [pwLoading, setPwLoading] = useState(false);
  const [pwError, setPwError] = useState("");

  // Language
  const [currentLocale, setCurrentLocale] = useState("en");

  useEffect(() => {
    loadSettings();
    setCurrentLocale(getCurrentLocale());
  }, []);

  const loadSettings = async () => {
    try {
      const res = await fetch("/api/settings");
      if (res.ok) setSettings(await res.json());
    } catch { /* ignore */ }
    setLoading(false);
  };

  const patchSetting = async (patch) => {
    try {
      const res = await fetch("/api/settings", {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(patch),
      });
      if (res.ok) {
        const updated = await res.json();
        setSettings(updated);
        return updated;
      }
    } catch { /* ignore */ }
    return null;
  };

  const handlePasswordChange = async () => {
    setPwError("");
    if (pwForm.newPw !== pwForm.confirm) {
      setPwError("Passwords do not match");
      return;
    }
    if (pwForm.newPw.length < 4) {
      setPwError("Password must be at least 4 characters");
      return;
    }
    setPwLoading(true);
    try {
      const result = await patchSetting({
        currentPassword: pwForm.current,
        password: pwForm.newPw,
      });
      if (result) {
        setPwForm({ current: "", newPw: "", confirm: "" });
        notify().success("Password updated successfully");
      } else {
        setPwError("Failed to update password. Check current password.");
      }
    } catch {
      setPwError("Failed to update password");
    }
    setPwLoading(false);
  };

  const handleLocaleChange = async (locale) => {
    setCurrentLocale(locale);
    document.cookie = `locale=${locale};path=/;max-age=31536000`;
    try {
      await fetch("/api/locale", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ locale }),
      });
    } catch { /* ignore */ }
    await reloadTranslations();
    // Force re-render
    window.location.reload();
  };

  const handleExportDB = async () => {
    try {
      const res = await fetch("/api/settings/database");
      const data = await res.json();
      const blob = new Blob([JSON.stringify(data, null, 2)], { type: "application/json" });
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = `9router-backup-${new Date().toISOString().slice(0, 10)}.json`;
      a.click();
      URL.revokeObjectURL(url);
    } catch { /* ignore */ }
  };

  const handleImportDB = () => {
    const input = document.createElement("input");
    input.type = "file";
    input.accept = ".json";
    input.onchange = async (e) => {
      const file = e.target.files[0];
      if (!file) return;
      try {
        const text = await file.text();
        const data = JSON.parse(text);
        const res = await fetch("/api/settings/database", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(data),
        });
        if (res.ok) window.location.reload();
      } catch { /* ignore */ }
    };
    input.click();
  };

  const handleProxyTest = async () => {
    setProxyTesting(true);
    try {
      const res = await fetch("/api/settings/proxy-test", { method: "POST" });
      const data = await res.json();
      if (data.success) notify().success("Proxy connection successful");
      else notify().error(data.error || "Proxy test failed");
    } catch { notify().error("Proxy test failed"); }
    setProxyTesting(false);
  };

  const handleOidcTest = async () => {
    setOidcTesting(true);
    try {
      const res = await fetch("/api/auth/oidc/test", { method: "POST" });
      const data = await res.json();
      if (data.success) notify().success("OIDC configuration valid");
      else notify().error(data.error || "OIDC test failed");
    } catch { notify().error("OIDC test failed"); }
    setOidcTesting(false);
  };

  const handleLogout = async () => {
    try {
      await fetch("/api/auth/logout", { method: "POST" });
      navigate("/login");
    } catch { /* ignore */ }
  };

  const handleShutdown = () => {
    setConfirmState({
      title: "Shutdown Server",
      message: "Are you sure you want to shutdown the 9Router server? You will need to restart it manually.",
      onConfirm: async () => {
        setConfirmState(null);
        try { await fetch("/api/version/shutdown", { method: "POST" }); } catch { /* server stops */ }
      },
    });
  };

  if (loading) {
    return (
      <div className="flex flex-col gap-6 max-w-3xl">
        {[1, 2, 3, 4].map((i) => (
          <div key={i} className="bg-surface border border-border-subtle rounded-[12px] shadow-[var(--shadow-soft)] p-6 animate-pulse h-32" />
        ))}
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-6 max-w-3xl">
      {/* Appearance */}
      <Card title="Appearance" icon="palette">
        <div className="flex items-center justify-between">
          <div>
            <p className="font-medium">Theme</p>
            <p className="text-sm text-text-muted">Switch between light and dark mode</p>
          </div>
          <ThemeToggle variant="card" />
        </div>
      </Card>

      {/* Language */}
      <Card title="Language" icon="translate">
        <Select
          label="Display Language"
          value={currentLocale}
          onChange={(e) => handleLocaleChange(e.target.value)}
          options={LOCALES.map((l) => ({ value: l, label: LOCALE_NAMES[l] || l }))}
        />
      </Card>

      {/* Security */}
      <Card title="Security" icon="shield">
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <div>
              <p className="font-medium">Require Login</p>
              <p className="text-sm text-text-muted">Protect dashboard with password</p>
            </div>
            <Toggle
              checked={settings?.requireLogin || false}
              onChange={(v) => patchSetting({ requireLogin: v })}
            />
          </div>

          {/* Password change */}
          <div className="pt-3 border-t border-border-subtle">
            <p className="font-medium mb-3">{settings?.hasPassword ? "Change Password" : "Set Password"}</p>
            <div className="space-y-3">
              {settings?.hasPassword && (
                <Input
                  type="password"
                  label="Current Password"
                  value={pwForm.current}
                  onChange={(e) => setPwForm({ ...pwForm, current: e.target.value })}
                  placeholder="Enter current password"
                />
              )}
              <Input
                type="password"
                label="New Password"
                value={pwForm.newPw}
                onChange={(e) => setPwForm({ ...pwForm, newPw: e.target.value })}
                placeholder="Enter new password"
              />
              <Input
                type="password"
                label="Confirm Password"
                value={pwForm.confirm}
                onChange={(e) => setPwForm({ ...pwForm, confirm: e.target.value })}
                placeholder="Confirm new password"
                error={pwError || null}
              />
              <Button
                size="sm"
                onClick={handlePasswordChange}
                loading={pwLoading}
                disabled={!pwForm.newPw || !pwForm.confirm}
              >
                {settings?.hasPassword ? "Change Password" : "Set Password"}
              </Button>
            </div>
          </div>
        </div>
      </Card>

      {/* OIDC SSO */}
      <Card title="OIDC Single Sign-On" icon="key">
        <div className="space-y-3">
          <Select
            label="Auth Mode"
            value={settings?.authMode || "password"}
            onChange={(e) => patchSetting({ authMode: e.target.value })}
            options={[
              { value: "password", label: "Password only" },
              { value: "oidc", label: "OIDC only" },
              { value: "both", label: "Password + OIDC" },
            ]}
          />
          <Input
            label="Issuer URL"
            value={settings?.oidcIssuerUrl || ""}
            onChange={(e) => patchSetting({ oidcIssuerUrl: e.target.value })}
            placeholder="https://accounts.google.com"
          />
          <Input
            label="Client ID"
            value={settings?.oidcClientId || ""}
            onChange={(e) => patchSetting({ oidcClientId: e.target.value })}
            placeholder="your-client-id"
          />
          <Input
            label="Client Secret"
            type="password"
            value={settings?.oidcClientSecret || ""}
            onChange={(e) => patchSetting({ oidcClientSecret: e.target.value })}
            placeholder="your-client-secret"
          />
          <Input
            label="Scopes"
            value={settings?.oidcScopes || "openid profile email"}
            onChange={(e) => patchSetting({ oidcScopes: e.target.value })}
            placeholder="openid profile email"
          />
          <Input
            label="Login Button Label"
            value={settings?.oidcLoginLabel || ""}
            onChange={(e) => patchSetting({ oidcLoginLabel: e.target.value })}
            placeholder="Sign in with SSO"
          />
          <Button
            variant="secondary"
            size="sm"
            icon="science"
            onClick={handleOidcTest}
            loading={oidcTesting}
          >
            Test OIDC Configuration
          </Button>
        </div>
      </Card>

      {/* Routing Strategy */}
      <Card title="Routing Strategy" icon="route">
        <div className="space-y-4">
          {/* Provider strategy */}
          <div className="flex items-center justify-between">
            <div>
              <p className="font-medium">Provider Round-Robin</p>
              <p className="text-sm text-text-muted">Distribute requests across connections evenly</p>
            </div>
            <Toggle
              checked={settings?.fallbackStrategy === "round-robin"}
              onChange={(v) => patchSetting({ fallbackStrategy: v ? "round-robin" : "fill-first" })}
            />
          </div>
          {settings?.fallbackStrategy === "round-robin" && (
            <div className="flex items-center gap-3 pl-4 border-l-2 border-brand-500/20">
              <label className="text-sm text-text-muted shrink-0">Sticky limit:</label>
              <input
                type="number" min={1} max={100}
                value={settings?.stickyRoundRobinLimit || 1}
                onChange={(e) => patchSetting({ stickyRoundRobinLimit: parseInt(e.target.value) || 1 })}
                className="w-20 px-2 py-1.5 text-sm rounded-lg bg-surface-2 border border-transparent focus:border-brand-500/30 focus:outline-none text-text-main"
              />
            </div>
          )}

          {/* Combo strategy */}
          <div className="pt-3 border-t border-border-subtle">
            <div className="flex items-center justify-between">
              <div>
                <p className="font-medium">Combo Round-Robin</p>
                <p className="text-sm text-text-muted">Round-robin across combo model chains</p>
              </div>
              <Toggle
                checked={settings?.comboStrategy === "round-robin"}
                onChange={(v) => patchSetting({ comboStrategy: v ? "round-robin" : "fill-first" })}
              />
            </div>
            {settings?.comboStrategy === "round-robin" && (
              <div className="flex items-center gap-3 pl-4 border-l-2 border-brand-500/20 mt-3">
                <label className="text-sm text-text-muted shrink-0">Combo sticky limit:</label>
                <input
                  type="number" min={1} max={100}
                  value={settings?.comboStickyRoundRobinLimit || 1}
                  onChange={(e) => patchSetting({ comboStickyRoundRobinLimit: parseInt(e.target.value) || 1 })}
                  className="w-20 px-2 py-1.5 text-sm rounded-lg bg-surface-2 border border-transparent focus:border-brand-500/30 focus:outline-none text-text-main"
                />
              </div>
            )}
          </div>
        </div>
      </Card>

      {/* Network */}
      <Card title="Network" icon="lan">
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <div>
              <p className="font-medium">Outbound Proxy</p>
              <p className="text-sm text-text-muted">Route all upstream requests through a proxy</p>
            </div>
            <Toggle
              checked={settings?.outboundProxyEnabled || false}
              onChange={(v) => patchSetting({ outboundProxyEnabled: v })}
            />
          </div>
          {settings?.outboundProxyEnabled && (
            <div className="space-y-3 pl-4 border-l-2 border-brand-500/20">
              <Input
                label="Proxy URL"
                value={settings?.outboundProxyUrl || ""}
                onChange={(e) => patchSetting({ outboundProxyUrl: e.target.value })}
                placeholder="http://proxy.example.com:8080"
              />
              <Input
                label="No Proxy"
                value={settings?.outboundNoProxy || ""}
                onChange={(e) => patchSetting({ outboundNoProxy: e.target.value })}
                placeholder="localhost,127.0.0.1"
              />
              <Button variant="secondary" size="sm" icon="speed" onClick={handleProxyTest} loading={proxyTesting}>
                Test Connection
              </Button>
            </div>
          )}
        </div>
      </Card>

      {/* Observability */}
      <Card title="Observability" icon="monitoring">
        <div className="flex items-center justify-between">
          <div>
            <p className="font-medium">Enable Observability</p>
            <p className="text-sm text-text-muted">Record detailed request/response data for debugging</p>
          </div>
          <Toggle
            checked={settings?.enableObservability || false}
            onChange={(v) => patchSetting({ enableObservability: v })}
          />
        </div>
      </Card>

      {/* Backup & Restore */}
      <Card title="Backup & Restore" icon="backup">
        <div className="flex flex-col sm:flex-row gap-3">
          <Button variant="secondary" icon="download" onClick={handleExportDB} fullWidth>
            Export Database
          </Button>
          <Button variant="secondary" icon="upload" onClick={handleImportDB} fullWidth>
            Import Database
          </Button>
        </div>
        <p className="text-xs text-text-muted mt-2">
          Export downloads a JSON backup of all settings, connections, keys, and usage data.
        </p>
      </Card>

      {/* Danger Zone */}
      <Card>
        <h3 className="text-sm font-semibold text-red-500 mb-3 flex items-center gap-2">
          <span className="material-symbols-outlined text-[18px]">warning</span>
          Danger Zone
        </h3>
        <div className="flex flex-col sm:flex-row gap-3">
          <Button variant="danger" icon="logout" onClick={handleLogout}>
            Logout
          </Button>
          <Button variant="danger" icon="power_settings_new" onClick={handleShutdown}>
            Shutdown Server
          </Button>
        </div>
      </Card>

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
