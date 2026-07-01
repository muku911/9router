// Provider definitions — ported from Next.js src/shared/constants/providers.js
// Only the fields needed for the dashboard UI are included (id, name, icon, color, textIcon, etc.)

const RISK_NOTICE = "⚠️ Risk Notice: This provider uses a subscription/OAuth session not officially licensed for proxy/router use. Account may be restricted or banned. Use at your own risk.";

export const FREE_PROVIDERS = {
  kiro: { id: "kiro", alias: "kr", name: "Kiro AI", icon: "psychology_alt", color: "#FF6B35", deprecated: true },
  "gemini-cli": { id: "gemini-cli", alias: "gc", name: "Gemini CLI", icon: "terminal", color: "#4285F4", deprecated: true },
  qoder: { id: "qoder", alias: "qd", name: "Qoder", icon: "water_drop", color: "#EC4899", deprecated: true },
  opencode: { id: "opencode", alias: "oc", name: "OpenCode Free", icon: "terminal", color: "#E87040", textIcon: "OC", noAuth: true },
};

export const FREE_TIER_PROVIDERS = {
  openrouter: { id: "openrouter", alias: "openrouter", name: "OpenRouter", icon: "router", color: "#F97316", textIcon: "OR" },
  nvidia: { id: "nvidia", alias: "nvidia", name: "NVIDIA NIM", icon: "developer_board", color: "#76B900", textIcon: "NV" },
  ollama: { id: "ollama", alias: "ollama", name: "Ollama Cloud", icon: "cloud", color: "#ffffff", textIcon: "OL" },
  vertex: { id: "vertex", alias: "vx", name: "Vertex AI", icon: "cloud", color: "#4285F4", textIcon: "VX" },
  gemini: { id: "gemini", alias: "gemini", name: "Gemini", icon: "diamond", color: "#4285F4", textIcon: "GE" },
  "cloudflare-ai": { id: "cloudflare-ai", alias: "cf", name: "Cloudflare", icon: "cloud", color: "#F38020", textIcon: "CF" },
  byteplus: { id: "byteplus", alias: "bpm", name: "BytePlus ModelArk", icon: "cloud", color: "#2563EB", textIcon: "BP" },
};

export const OAUTH_PROVIDERS = {
  claude: { id: "claude", alias: "cc", name: "Claude Code", icon: "smart_toy", color: "#D97757", deprecated: true },
  antigravity: { id: "antigravity", alias: "ag", name: "Antigravity", icon: "rocket_launch", color: "#F59E0B", deprecated: true },
  codex: { id: "codex", alias: "cx", name: "OpenAI Codex", icon: "code", color: "#3B82F6", deprecated: true },
  github: { id: "github", alias: "gh", name: "GitHub Copilot", icon: "code", color: "#333333", deprecated: true },
  cursor: { id: "cursor", alias: "cu", name: "Cursor IDE", icon: "edit_note", color: "#00D4AA" },
  xai: { id: "xai", alias: "xai", name: "xAI (Grok)", icon: "auto_awesome", color: "#1DA1F2", textIcon: "XA" },
  kilocode: { id: "kilocode", alias: "kc", name: "Kilo Code", icon: "code", color: "#FF6B35", textIcon: "KC" },
  cline: { id: "cline", alias: "cl", name: "Cline", icon: "smart_toy", color: "#5B9BD5", textIcon: "CL" },
};

export const APIKEY_PROVIDERS = {
  glm: { id: "glm", alias: "glm", name: "GLM Coding", icon: "code", color: "#2563EB", textIcon: "GL" },
  "glm-cn": { id: "glm-cn", alias: "glm-cn", name: "GLM (China)", icon: "code", color: "#DC2626", textIcon: "GC" },
  kimi: { id: "kimi", alias: "kimi", name: "Kimi", icon: "psychology", color: "#1E3A8A", textIcon: "KM" },
  minimax: { id: "minimax", alias: "minimax", name: "Minimax Coding", icon: "memory", color: "#7C3AED", textIcon: "MM" },
  "minimax-cn": { id: "minimax-cn", alias: "minimax-cn", name: "Minimax (China)", icon: "memory", color: "#DC2626", textIcon: "MC" },
  alicode: { id: "alicode", alias: "alicode", name: "Alibaba", icon: "cloud", color: "#FF6A00", textIcon: "ALi" },
  "alicode-intl": { id: "alicode-intl", alias: "alicode-intl", name: "Alibaba Intl", icon: "cloud", color: "#FF6A00", textIcon: "ALi" },
  "xiaomi-mimo": { id: "xiaomi-mimo", alias: "mimo", name: "Xiaomi MiMo", icon: "smart_toy", color: "#FF6900", textIcon: "XM" },
  "volcengine-ark": { id: "volcengine-ark", alias: "ark", name: "Volcengine Ark", icon: "cloud", color: "#1677FF", textIcon: "ARK" },
  openai: { id: "openai", alias: "openai", name: "OpenAI", icon: "auto_awesome", color: "#10A37F", textIcon: "OA" },
  "vercel-ai-gateway": { id: "vercel-ai-gateway", alias: "vercel", name: "Vercel AI Gateway", icon: "deployed_code", color: "#111827", textIcon: "VG" },
  anthropic: { id: "anthropic", alias: "anthropic", name: "Anthropic", icon: "smart_toy", color: "#D97757", textIcon: "AN" },
  "opencode-go": { id: "opencode-go", alias: "ocg", name: "OpenCode Go", icon: "terminal", color: "#E87040", textIcon: "OC" },
  azure: { id: "azure", alias: "azure", name: "Azure OpenAI", icon: "cloud", color: "#0078D4", textIcon: "AZ" },
  deepseek: { id: "deepseek", alias: "ds", name: "DeepSeek", icon: "bolt", color: "#4D6BFE", textIcon: "DS" },
  groq: { id: "groq", alias: "groq", name: "Groq", icon: "speed", color: "#F55036", textIcon: "GQ" },
  mistral: { id: "mistral", alias: "mistral", name: "Mistral", icon: "air", color: "#FF7000", textIcon: "MI" },
  perplexity: { id: "perplexity", alias: "pplx", name: "Perplexity", icon: "search", color: "#20808D", textIcon: "PP" },
  together: { id: "together", alias: "together", name: "Together AI", icon: "group_work", color: "#0F6FFF", textIcon: "TG" },
  fireworks: { id: "fireworks", alias: "fireworks", name: "Fireworks AI", icon: "local_fire_department", color: "#7B2EF2", textIcon: "FW" },
  cerebras: { id: "cerebras", alias: "cerebras", name: "Cerebras", icon: "memory", color: "#FF4F00", textIcon: "CB" },
  cohere: { id: "cohere", alias: "cohere", name: "Cohere", icon: "hub", color: "#39594D", textIcon: "CO" },
  nebius: { id: "nebius", alias: "nebius", name: "Nebius AI", icon: "cloud", color: "#6C5CE7", textIcon: "NB" },
  hyperbolic: { id: "hyperbolic", alias: "hyp", name: "Hyperbolic", icon: "bolt", color: "#00D4FF", textIcon: "HY" },
  elevenlabs: { id: "elevenlabs", alias: "el", name: "ElevenLabs", icon: "record_voice_over", color: "#6C47FF", textIcon: "EL" },
  deepgram: { id: "deepgram", alias: "dg", name: "Deepgram", icon: "mic", color: "#13EF93", textIcon: "DG" },
  huggingface: { id: "huggingface", alias: "hf", name: "HuggingFace", icon: "face", color: "#FFD21E", textIcon: "HF" },
  blackbox: { id: "blackbox", alias: "bb", name: "Blackbox AI", icon: "smart_toy", color: "#5B5FEF", textIcon: "BB" },
  chutes: { id: "chutes", alias: "ch", name: "Chutes AI", icon: "water_drop", color: "#ffffff", textIcon: "CH" },
};

export const WEB_COOKIE_PROVIDERS = {
  "grok-web": { id: "grok-web", alias: "gw", name: "Grok Web", icon: "auto_awesome", color: "#1DA1F2", textIcon: "GW" },
  "perplexity-web": { id: "perplexity-web", alias: "pw", name: "Perplexity Web", icon: "search", color: "#20808D", textIcon: "PW" },
};

// Media provider kinds — each kind maps to a route and endpoint config
export const MEDIA_PROVIDER_KINDS = [
  { id: "embedding",   label: "Embedding",      icon: "data_array",        endpoint: { method: "POST", path: "/v1/embeddings" } },
  { id: "image",       label: "Text to Image",  icon: "brush",             endpoint: { method: "POST", path: "/v1/images/generations" } },
  { id: "imageToText", label: "Image to Text",  icon: "image_search",      endpoint: { method: "POST", path: "/v1/images/understanding" } },
  { id: "tts",         label: "Text To Speech", icon: "record_voice_over", endpoint: { method: "POST", path: "/v1/audio/speech" } },
  { id: "stt",         label: "Speech To Text", icon: "mic",               endpoint: { method: "POST", path: "/v1/audio/transcriptions" } },
  { id: "webSearch",   label: "Web Search",     icon: "travel_explore",    endpoint: { method: "POST", path: "/v1/search" } },
  { id: "webFetch",    label: "Web Fetch",      icon: "language",          endpoint: { method: "POST", path: "/v1/web/fetch" } },
  { id: "video",       label: "Video",          icon: "movie",             endpoint: { method: "POST", path: "/v1/video/generations" } },
  { id: "music",       label: "Music",          icon: "music_note",        endpoint: { method: "POST", path: "/v1/audio/music" } },
];

// All providers combined
export const AI_PROVIDERS = { ...FREE_PROVIDERS, ...FREE_TIER_PROVIDERS, ...OAUTH_PROVIDERS, ...APIKEY_PROVIDERS, ...WEB_COOKIE_PROVIDERS };

export const AUTH_METHODS = {
  oauth: { id: "oauth", name: "OAuth", icon: "lock" },
  apikey: { id: "apikey", name: "API Key", icon: "key" },
  cookie: { id: "cookie", name: "Browser Cookie", icon: "cookie" },
};

// Helper: Get provider by alias
export function getProviderByAlias(alias) {
  for (const provider of Object.values(AI_PROVIDERS)) {
    if (provider.alias === alias || provider.id === alias) {
      return provider;
    }
  }
  return null;
}
