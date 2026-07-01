import { useState, useEffect } from "react";
import { Link } from "react-router-dom";
import Card from "../components/Card";
import Badge from "../components/Badge";
import { cn } from "../lib/cn";

const CLI_TOOLS = [
  { id: "claude", name: "Claude Code", icon: "smart_toy", color: "#D97757", description: "Anthropic's CLI coding agent" },
  { id: "openclaw", name: "Open Claw", icon: "terminal", color: "#D97757", description: "Open-source Claude Code fork" },
  { id: "codex", name: "OpenAI Codex CLI", icon: "code", color: "#3B82F6", description: "OpenAI's CLI coding agent" },
  { id: "opencode", name: "OpenCode", icon: "terminal", color: "#E87040", description: "Go-based coding agent" },
  { id: "hermes", name: "Hermes Agent", icon: "smart_toy", color: "#8B5CF6", description: "Multi-provider coding agent" },
  { id: "droid", name: "Factory Droid", icon: "precision_manufacturing", color: "#10B981", description: "Factory AI coding agent" },
  { id: "cursor", name: "Cursor IDE", icon: "edit_note", color: "#00D4AA", description: "AI-first code editor" },
  { id: "cline", name: "Cline", icon: "smart_toy", color: "#5B9BD5", description: "VS Code AI assistant" },
  { id: "kilo", name: "Kilo Code", icon: "code", color: "#FF6B35", description: "VS Code AI coding extension" },
  { id: "roo", name: "Roo", icon: "smart_toy", color: "#7C3AED", description: "Roo Code assistant" },
  { id: "continue", name: "Continue", icon: "smart_toy", color: "#0F6FFF", description: "Open-source AI assistant" },
  { id: "amp", name: "Amp CLI", icon: "terminal", color: "#FF4081", description: "Sourcegraph coding agent" },
  { id: "qwen", name: "Qwen Code", icon: "code", color: "#7C3AED", description: "Alibaba's coding agent" },
  { id: "deepseek-tui", name: "DeepSeek TUI", icon: "terminal", color: "#4D6BFE", description: "DeepSeek terminal client" },
  { id: "jcode", name: "jcode", icon: "code", color: "#F59E0B", description: "Rust-based coding agent" },
];

const MITM_TOOLS = [
  { id: "antigravity", name: "Antigravity", icon: "rocket_launch", color: "#F59E0B" },
  { id: "copilot", name: "GitHub Copilot", icon: "code", color: "#333333" },
  { id: "kiro", name: "Kiro", icon: "psychology_alt", color: "#FF6B35" },
];

export default function CliTools() {
  const [statuses, setStatuses] = useState({});
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetch("/api/cli-tools/all-statuses")
      .then((r) => r.json())
      .then((data) => setStatuses(data))
      .catch(() => setStatuses({}))
      .finally(() => setLoading(false));
  }, []);

  if (loading) {
    return (
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
        {[1, 2, 3, 4, 5, 6].map((i) => (
          <div key={i} className="bg-surface border border-border-subtle rounded-[12px] shadow-[var(--shadow-soft)] p-4 animate-pulse h-24" />
        ))}
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-8">
      {/* Regular CLI Tools */}
      <section>
        <h2 className="text-lg font-semibold mb-4 flex items-center gap-2">
          <span className="material-symbols-outlined text-primary">terminal</span>
          CLI Tools
        </h2>
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
          {CLI_TOOLS.map((tool) => {
            const toolStatus = statuses[tool.id];
            return (
              <ToolCard
                key={tool.id}
                tool={tool}
                status={toolStatus}
              />
            );
          })}
        </div>
      </section>

      {/* MITM Tools */}
      <section>
        <h2 className="text-lg font-semibold mb-4 flex items-center gap-2">
          <span className="material-symbols-outlined text-primary">security</span>
          MITM Tools
        </h2>
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
          {MITM_TOOLS.map((tool) => (
            <Link
              key={tool.id}
              to="/mitm"
              className={cn(
                "flex items-center gap-3 p-4 rounded-[12px] border border-border-subtle bg-surface",
                "shadow-[var(--shadow-soft)] hover:shadow-[0_0_20px_-5px_rgba(229,106,74,0.15)]",
                "hover:border-brand-500/25 transition-all duration-300"
              )}
            >
              <div
                className="flex items-center justify-center size-10 rounded-lg shrink-0"
                style={{ backgroundColor: `${tool.color}20`, color: tool.color }}
              >
                <span className="material-symbols-outlined text-[20px]">{tool.icon}</span>
              </div>
              <div className="flex-1 min-w-0">
                <h3 className="text-sm font-semibold text-text-main">{tool.name}</h3>
                <Badge variant="primary" size="sm" className="mt-1">MITM</Badge>
              </div>
              <span className="material-symbols-outlined text-text-muted text-[18px]">chevron_right</span>
            </Link>
          ))}
        </div>
      </section>
    </div>
  );
}

function ToolCard({ tool, status }) {
  const getStatusInfo = () => {
    if (!status) return { label: "Unknown", variant: "default" };
    if (status.has9Router) return { label: "Connected", variant: "success" };
    if (status.installed) return { label: "Not configured", variant: "warning" };
    return { label: "Not installed", variant: "default" };
  };

  const statusInfo = getStatusInfo();

  return (
    <div
      className={cn(
        "flex items-center gap-3 p-4 rounded-[12px] border border-border-subtle bg-surface",
        "shadow-[var(--shadow-soft)] hover:shadow-[0_0_20px_-5px_rgba(229,106,74,0.15)]",
        "hover:border-brand-500/25 transition-all duration-300"
      )}
    >
      <div
        className="flex items-center justify-center size-10 rounded-lg shrink-0"
        style={{ backgroundColor: `${tool.color}20`, color: tool.color }}
      >
        <span className="material-symbols-outlined text-[20px]">{tool.icon}</span>
      </div>
      <div className="flex-1 min-w-0">
        <h3 className="text-sm font-semibold text-text-main truncate">{tool.name}</h3>
        <p className="text-xs text-text-muted truncate">{tool.description}</p>
      </div>
      <Badge variant={statusInfo.variant} size="sm" dot>
        {statusInfo.label}
      </Badge>
    </div>
  );
}
