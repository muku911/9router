import Card from "../components/Card";
import Button from "../components/Button";
import { useCopyToClipboard } from "../hooks/useCopyToClipboard";

const SKILLS_RAW_BASE = "https://raw.githubusercontent.com/decolua/9router/master/skills/";

const SKILLS = [
  { id: "configure.md", name: "Auto-Configure", description: "Automatically detect and configure CLI tools for 9Router", icon: "settings_suggest", endpoint: "POST /api/cli-tools/*-settings", isEntry: true },
  { id: "smart-connect.md", name: "Smart Connect", description: "Add provider connections via natural language instructions", icon: "add_link", endpoint: "POST /api/providers" },
  { id: "health-check.md", name: "Health Check", description: "Run diagnostics on all provider connections", icon: "health_and_safety", endpoint: "POST /api/providers/test-batch" },
  { id: "usage-report.md", name: "Usage Report", description: "Generate a detailed usage analytics report", icon: "summarize", endpoint: "GET /api/usage/stats" },
  { id: "proxy-setup.md", name: "Proxy Setup", description: "Configure outbound proxies and relay pools", icon: "lan", endpoint: "POST /api/proxy-pools" },
  { id: "backup-restore.md", name: "Backup & Restore", description: "Export and import database backups", icon: "backup", endpoint: "GET/POST /api/settings/database" },
  { id: "model-alias.md", name: "Model Aliases", description: "Set up model name aliases for routing", icon: "swap_horiz", endpoint: "PUT /api/models/alias" },
  { id: "combo-create.md", name: "Combo Builder", description: "Create model combos with fallback chains", icon: "layers", endpoint: "POST /api/combos" },
];

export default function Skills() {
  const { copied, copy } = useCopyToClipboard();

  return (
    <div className="flex flex-col gap-6">
      {/* Instruction card */}
      <Card elev>
        <div className="flex items-start gap-3">
          <div className="flex items-center justify-center size-10 rounded-lg bg-brand-500/10 text-brand-500 shrink-0">
            <span className="material-symbols-outlined text-[22px]">magic_exchange</span>
          </div>
          <div>
            <h2 className="font-semibold mb-1">Agent Skills</h2>
            <p className="text-sm text-text-muted">
              Copy a skill link and paste it into your AI agent to give it 9Router superpowers.
              Each skill teaches the agent how to interact with a specific 9Router API.
            </p>
          </div>
        </div>
      </Card>

      {/* Skills list */}
      <Card padding="none">
        <div className="divide-y divide-border-subtle">
          {SKILLS.map((skill) => {
            const url = `${SKILLS_RAW_BASE}${skill.id}`;
            return (
              <div
                key={skill.id}
                className="flex items-center gap-3 px-4 py-3 hover:bg-surface-2/30 transition-colors"
              >
                <div className="flex items-center justify-center size-9 rounded-lg bg-brand-500/10 text-brand-500 shrink-0">
                  <span className="material-symbols-outlined text-[18px]">{skill.icon}</span>
                </div>
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2">
                    <h3 className="text-sm font-semibold text-text-main">{skill.name}</h3>
                    {skill.isEntry && (
                      <span className="px-1.5 py-0.5 text-[10px] font-bold rounded bg-brand-500/10 text-brand-500">START HERE</span>
                    )}
                  </div>
                  <p className="text-xs text-text-muted truncate">{skill.description}</p>
                </div>
                <code className="hidden sm:block text-[10px] text-text-muted font-mono bg-surface-2 px-2 py-1 rounded shrink-0">
                  {skill.endpoint}
                </code>
                <Button
                  variant="ghost"
                  size="sm"
                  icon={copied === skill.id ? "check" : "content_copy"}
                  onClick={() => copy(url, skill.id)}
                >
                  {copied === skill.id ? "Copied" : "Copy"}
                </Button>
              </div>
            );
          })}
        </div>
      </Card>

      {/* GitHub link */}
      <div className="text-center">
        <a
          href="https://github.com/decolua/9router/tree/master/skills"
          target="_blank"
          rel="noreferrer"
          className="text-sm text-primary hover:underline inline-flex items-center gap-1"
        >
          <span className="material-symbols-outlined text-[16px]">open_in_new</span>
          View all skills on GitHub
        </a>
      </div>
    </div>
  );
}
