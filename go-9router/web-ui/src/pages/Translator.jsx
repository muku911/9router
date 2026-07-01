import { useState } from "react";
import Card from "../components/Card";
import Button from "../components/Button";

const STEPS = [
  { id: 1, label: "Client Request", file: "1_req_client.json", icon: "input" },
  { id: 2, label: "Source Body", file: "2_req_source.json", icon: "data_object" },
  { id: 3, label: "OpenAI Intermediate", file: "3_req_openai.json", icon: "swap_horiz" },
  { id: 4, label: "Target Request", file: "4_req_target.json", icon: "send" },
  { id: 5, label: "Provider Response", file: "5_res_provider.json", icon: "cloud_download" },
  { id: 6, label: "OpenAI Response", file: "6_res_openai.json", icon: "swap_horiz" },
  { id: 7, label: "Client Response", file: "7_res_client.txt", icon: "output" },
];

export default function Translator() {
  const [stepData, setStepData] = useState({});
  const [expandedStep, setExpandedStep] = useState(1);
  const [loading, setLoading] = useState({});

  const loadStep = async (step) => {
    setLoading((prev) => ({ ...prev, [step.id]: true }));
    try {
      const res = await fetch(`/api/translator/load?file=${step.file}`);
      if (res.ok) {
        const data = await res.json();
        setStepData((prev) => ({ ...prev, [step.id]: data.content || JSON.stringify(data, null, 2) }));
      }
    } catch { /* ignore */ }
    setLoading((prev) => ({ ...prev, [step.id]: false }));
  };

  const formatStep = (stepId) => {
    const content = stepData[stepId];
    if (!content) return;
    try {
      const parsed = JSON.parse(content);
      setStepData((prev) => ({ ...prev, [stepId]: JSON.stringify(parsed, null, 2) }));
    } catch { /* not JSON */ }
  };

  const copyStep = (stepId) => {
    const content = stepData[stepId];
    if (content) navigator.clipboard.writeText(content);
  };

  return (
    <div className="flex flex-col gap-3">
      <p className="text-sm text-text-muted mb-2">
        Debug the 7-step translation pipeline between client and provider formats.
        Load log files from <code className="px-1 bg-surface-2 rounded text-xs">logs/translator/</code> to inspect each step.
      </p>

      {STEPS.map((step) => (
        <Card key={step.id} padding="none">
          {/* Step header */}
          <button
            onClick={() => setExpandedStep(expandedStep === step.id ? null : step.id)}
            className="w-full flex items-center justify-between p-3 hover:bg-surface-2/30 transition-colors"
          >
            <div className="flex items-center gap-2">
              <span className="flex items-center justify-center size-7 rounded-lg bg-brand-500/10 text-brand-500 text-xs font-bold">
                {step.id}
              </span>
              <span className="material-symbols-outlined text-text-muted text-[18px]">{step.icon}</span>
              <span className="text-sm font-medium text-text-main">{step.label}</span>
            </div>
            <span
              className="material-symbols-outlined text-text-muted text-[20px] transition-transform"
              style={{ transform: expandedStep === step.id ? "rotate(180deg)" : "rotate(0deg)" }}
            >
              expand_more
            </span>
          </button>

          {/* Step content */}
          {expandedStep === step.id && (
            <div className="border-t border-border-subtle p-3">
              <div className="flex items-center gap-2 mb-2">
                <Button variant="ghost" size="sm" icon="folder_open" onClick={() => loadStep(step)} loading={loading[step.id]}>
                  Load
                </Button>
                <Button variant="ghost" size="sm" icon="code" onClick={() => formatStep(step.id)} disabled={!stepData[step.id]}>
                  Format
                </Button>
                <Button variant="ghost" size="sm" icon="content_copy" onClick={() => copyStep(step.id)} disabled={!stepData[step.id]}>
                  Copy
                </Button>
              </div>
              <textarea
                value={stepData[step.id] || ""}
                onChange={(e) => setStepData((prev) => ({ ...prev, [step.id]: e.target.value }))}
                placeholder={`Load ${step.file} or paste content here...`}
                className="w-full h-[200px] p-3 text-xs font-mono bg-black text-green-400 rounded-lg border border-border-subtle resize-y focus:outline-none focus:ring-1 focus:ring-brand-500/30 custom-scrollbar"
                spellCheck={false}
              />
            </div>
          )}
        </Card>
      ))}
    </div>
  );
}
