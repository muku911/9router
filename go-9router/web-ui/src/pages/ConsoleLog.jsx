import { useState, useEffect, useRef } from "react";
import Card from "../components/Card";
import Button from "../components/Button";
import Badge from "../components/Badge";
import { cn } from "../lib/cn";
import { CONSOLE_LOG_CONFIG } from "../constants/config";

const LOG_COLORS = {
  LOG: "text-green-400",
  INFO: "text-blue-400",
  WARN: "text-yellow-400",
  ERROR: "text-red-400",
  DEBUG: "text-purple-400",
};

function colorLine(line) {
  const matches = line.match(/\[(\w+)\]/g);
  if (matches && matches.length >= 2) {
    const level = matches[1].replace(/[[\]]/g, "").toUpperCase();
    return LOG_COLORS[level] || "text-green-400";
  }
  return "text-green-400";
}

export default function ConsoleLog() {
  const [logs, setLogs] = useState([]);
  const [connected, setConnected] = useState(false);
  const logRef = useRef(null);
  const esRef = useRef(null);

  useEffect(() => {
    const es = new EventSource("/api/translator/console-logs/stream");
    esRef.current = es;

    es.onopen = () => setConnected(true);
    es.onerror = () => setConnected(false);

    es.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data);

        if (msg.type === "init") {
          const initLogs = (msg.logs || []).slice(-CONSOLE_LOG_CONFIG.maxLines);
          setLogs(initLogs);
        } else if (msg.type === "line") {
          setLogs((prev) => {
            const next = [...prev, msg.line];
            if (next.length > CONSOLE_LOG_CONFIG.maxLines) {
              return next.slice(next.length - CONSOLE_LOG_CONFIG.maxLines);
            }
            return next;
          });
        } else if (msg.type === "clear") {
          setLogs([]);
        }
      } catch {
        // Ignore parse errors (e.g. keepalive pings)
      }
    };

    return () => {
      es.close();
      esRef.current = null;
    };
  }, []);

  // Auto-scroll to bottom on new logs
  useEffect(() => {
    if (logRef.current) {
      logRef.current.scrollTop = logRef.current.scrollHeight;
    }
  }, [logs]);

  const handleClear = async () => {
    try {
      await fetch("/api/translator/console-logs", { method: "DELETE" });
    } catch {
      // SSE clear event will handle the UI update
    }
  };

  return (
    <div className="flex flex-col gap-4">
      <Card padding="none">
        {/* Header */}
        <div className="flex items-center justify-between p-4 border-b border-border-subtle">
          <div className="flex items-center gap-3">
            <h2 className="text-sm font-semibold flex items-center gap-2">
              <span className="material-symbols-outlined text-primary text-[18px]">terminal</span>
              Console Output
            </h2>
            <Badge
              variant={connected ? "success" : "error"}
              size="sm"
              dot
            >
              {connected ? "Live" : "Disconnected"}
            </Badge>
          </div>
          <Button
            variant="ghost"
            size="sm"
            icon="delete_sweep"
            onClick={handleClear}
            disabled={logs.length === 0}
          >
            Clear
          </Button>
        </div>

        {/* Terminal */}
        <div
          ref={logRef}
          className="bg-black rounded-b-[12px] p-4 font-mono text-xs leading-relaxed overflow-y-auto custom-scrollbar"
          style={{ height: "calc(100vh - 220px)" }}
        >
          {logs.length === 0 ? (
            <div className="flex items-center justify-center h-full text-gray-500">
              <div className="text-center">
                <span className="material-symbols-outlined text-[32px] mb-2">monitor</span>
                <p>Waiting for log output...</p>
              </div>
            </div>
          ) : (
            logs.map((line, i) => (
              <div key={i} className={cn("whitespace-pre-wrap break-all", colorLine(line))}>
                {line}
              </div>
            ))
          )}
        </div>
      </Card>
    </div>
  );
}
