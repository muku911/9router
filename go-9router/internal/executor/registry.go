// Package executor provides a multi-provider execution framework.
// It mirrors the open-sse architecture: a registry of provider executors,
// each knowing how to build URLs, headers, and transform requests for their provider.
package executor

// Format identifiers — matches open-sse/translator/formats.js
const (
	FormatOpenAI         = "openai"
	FormatOpenAIResponse = "openai-responses"
	FormatClaude         = "claude"
	FormatGemini         = "gemini"
	FormatAntigravity    = "antigravity"
	FormatKiro           = "kiro"
	FormatCursor         = "cursor"
	FormatOllama         = "ollama"
	FormatVertex         = "vertex"
)

// ProviderConfig holds the configuration for a provider.
type ProviderConfig struct {
	BaseURL   string
	Format    string // target format for translation (e.g. "openai", "claude", "gemini")
	AuthStyle string // "bearer", "x-api-key", "x-goog-api-key"
	Headers   map[string]string
}

// Known provider configurations
var ProviderConfigs = map[string]ProviderConfig{
	// OpenAI-format providers (no translation needed)
	"openai":     {BaseURL: "https://api.openai.com/v1/chat/completions", Format: FormatOpenAI, AuthStyle: "bearer"},
	"deepseek":   {BaseURL: "https://api.deepseek.com/v1/chat/completions", Format: FormatOpenAI, AuthStyle: "bearer"},
	"groq":       {BaseURL: "https://api.groq.com/openai/v1/chat/completions", Format: FormatOpenAI, AuthStyle: "bearer"},
	"mistral":    {BaseURL: "https://api.mistral.ai/v1/chat/completions", Format: FormatOpenAI, AuthStyle: "bearer"},
	"together":   {BaseURL: "https://api.together.xyz/v1/chat/completions", Format: FormatOpenAI, AuthStyle: "bearer"},
	"fireworks":  {BaseURL: "https://api.fireworks.ai/inference/v1/chat/completions", Format: FormatOpenAI, AuthStyle: "bearer"},
	"perplexity": {BaseURL: "https://api.perplexity.ai/chat/completions", Format: FormatOpenAI, AuthStyle: "bearer"},
	"cerebras":   {BaseURL: "https://api.cerebras.ai/v1/chat/completions", Format: FormatOpenAI, AuthStyle: "bearer"},
	"nebius":     {BaseURL: "https://api.studio.nebius.ai/v1/chat/completions", Format: FormatOpenAI, AuthStyle: "bearer"},
	"hyperbolic": {BaseURL: "https://api.hyperbolic.xyz/v1/chat/completions", Format: FormatOpenAI, AuthStyle: "bearer"},
	"openrouter": {BaseURL: "https://openrouter.ai/api/v1/chat/completions", Format: FormatOpenAI, AuthStyle: "bearer"},
	"nvidia":     {BaseURL: "https://integrate.api.nvidia.com/v1/chat/completions", Format: FormatOpenAI, AuthStyle: "bearer"},
	"cohere":     {BaseURL: "https://api.cohere.com/v2/chat", Format: FormatOpenAI, AuthStyle: "bearer"},
	"huggingface": {BaseURL: "https://api-inference.huggingface.co/v1/chat/completions", Format: FormatOpenAI, AuthStyle: "bearer"},

	// Claude-format providers
	"anthropic": {BaseURL: "https://api.anthropic.com/v1/messages", Format: FormatClaude, AuthStyle: "x-api-key",
		Headers: map[string]string{"anthropic-version": "2023-06-01"}},
	"claude": {BaseURL: "https://api.anthropic.com/v1/messages", Format: FormatClaude, AuthStyle: "x-api-key",
		Headers: map[string]string{"anthropic-version": "2023-06-01"}},

	// Gemini-format providers
	"gemini": {BaseURL: "https://generativelanguage.googleapis.com/v1beta/models/", Format: FormatGemini, AuthStyle: "x-goog-api-key"},

	// Antigravity (already handled by existing executor, included for completeness)
	"antigravity": {BaseURL: "https://daily-cloudcode-pa.googleapis.com/v1internal/models/", Format: FormatAntigravity, AuthStyle: "bearer"},
}

// GetProviderFormat returns the target format for a provider, or "openai" as default.
func GetProviderFormat(provider string) string {
	if cfg, ok := ProviderConfigs[provider]; ok {
		return cfg.Format
	}
	// OpenAI-compatible and unknown providers default to OpenAI format
	return FormatOpenAI
}

// GetProviderConfig returns the config for a provider, falling back to openai defaults.
func GetProviderConfig(provider string) ProviderConfig {
	if cfg, ok := ProviderConfigs[provider]; ok {
		return cfg
	}
	return ProviderConfigs["openai"]
}
