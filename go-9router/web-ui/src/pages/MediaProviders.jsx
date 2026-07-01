import { useState, useEffect } from "react";
import { useParams, Link } from "react-router-dom";
import Card from "../components/Card";
import Badge from "../components/Badge";
import ProviderIcon from "../components/ProviderIcon";
import { AI_PROVIDERS, MEDIA_PROVIDER_KINDS } from "../constants/providers";

// Map of which providers support which media kinds
// This is a simplified version — full mapping would come from getProvidersByKind()
const KIND_PROVIDERS = {
  embedding: ["openai", "anthropic", "gemini", "mistral", "cohere", "nvidia", "huggingface", "deepseek", "vertex"],
  image: ["openai", "anthropic", "deepseek", "nvidia", "fireworks", "together"],
  tts: ["openai", "elevenlabs", "deepgram", "minimax"],
  stt: ["openai", "deepgram"],
  webSearch: ["openai", "perplexity"],
  webFetch: [],
  video: [],
  music: [],
};

export default function MediaProviders() {
  const { kind } = useParams();
  const [connections, setConnections] = useState([]);
  const [loading, setLoading] = useState(true);

  const kindConfig = MEDIA_PROVIDER_KINDS.find((k) => k.id === kind);

  useEffect(() => {
    fetch("/api/providers")
      .then((r) => r.json())
      .then((data) => setConnections(data.connections || []))
      .catch(() => {})
      .finally(() => setLoading(false));
  }, [kind]);

  const providerIds = KIND_PROVIDERS[kind] || [];

  if (loading) {
    return (
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-3">
        {[1, 2, 3, 4].map((i) => (
          <div key={i} className="bg-surface border border-border-subtle rounded-[12px] shadow-[var(--shadow-soft)] p-4 animate-pulse h-24" />
        ))}
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-6">
      {providerIds.length === 0 ? (
        <Card>
          <div className="text-center py-12">
            <span className="material-symbols-outlined text-[40px] text-text-muted mb-3">
              {kindConfig?.icon || "perm_media"}
            </span>
            <h3 className="text-lg font-semibold mb-2">{kindConfig?.label || kind}</h3>
            <p className="text-text-muted text-sm">
              No providers configured for this media type yet.
            </p>
          </div>
        </Card>
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-3">
          {providerIds.map((providerId) => {
            const info = AI_PROVIDERS[providerId];
            if (!info) return null;
            const conns = connections.filter((c) => c.provider === providerId);
            const activeCount = conns.filter((c) => c.isActive).length;

            return (
              <Link
                key={providerId}
                to={`/providers/${providerId}`}
                className="block bg-surface border border-border-subtle rounded-[12px] shadow-[var(--shadow-soft)] p-4 hover:shadow-[0_0_20px_-5px_rgba(229,106,74,0.15)] hover:border-brand-500/25 transition-all duration-300"
              >
                <div className="flex items-start justify-between mb-3">
                  <ProviderIcon
                    src={`/providers/${providerId}.png`}
                    alt={info.name}
                    size={36}
                    className="rounded-lg object-contain"
                    fallbackText={info.textIcon || info.name?.slice(0, 2)?.toUpperCase()}
                    fallbackColor={info.color}
                  />
                  {activeCount > 0 && (
                    <Badge variant="success" size="sm" dot>{activeCount}</Badge>
                  )}
                </div>
                <h3 className="text-sm font-semibold text-text-main truncate">{info.name}</h3>
                <p className="text-xs text-text-muted mt-0.5">
                  {conns.length > 0 ? `${conns.length} connection${conns.length !== 1 ? "s" : ""}` : "Not connected"}
                </p>
              </Link>
            );
          })}
        </div>
      )}
    </div>
  );
}
