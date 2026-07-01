package translator

type AntigravityRequest struct {
	Project      string             `json:"project"`
	Model        string             `json:"model"`
	UserAgent    string             `json:"userAgent"`
	RequestType  string             `json:"requestType"`
	RequestID    string             `json:"requestId"`
	Request      AGRequestDetail    `json:"request"`
}

type AGRequestDetail struct {
	Contents          []AGContent        `json:"contents,omitempty"`
	SystemInstruction *AGSystemInstruction `json:"systemInstruction,omitempty"`
	Tools             []AGTool           `json:"tools,omitempty"`
	ToolConfig        *AGToolConfig      `json:"toolConfig,omitempty"`
	GenerationConfig  *AGGenerationConfig `json:"generationConfig,omitempty"`
	SessionID         string             `json:"sessionId,omitempty"`
}

type AGContent struct {
	Role  string   `json:"role"`
	Parts []AGPart `json:"parts"`
}

type AGPart struct {
	Text             string              `json:"text,omitempty"`
	Thought          bool                `json:"thought,omitempty"`
	ThoughtSignature string              `json:"thoughtSignature,omitempty"`
	InlineData       *AGInlineData       `json:"inlineData,omitempty"`
	FunctionCall     *AGFunctionCall     `json:"functionCall,omitempty"`
	FunctionResponse *AGFunctionResponse `json:"functionResponse,omitempty"`
}

type AGInlineData struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"` // base64 encoded
}

type AGFunctionCall struct {
	ID   string                 `json:"id,omitempty"`
	Name string                 `json:"name"`
	Args map[string]interface{} `json:"args"`
}

type AGFunctionResponse struct {
	ID       string                 `json:"id,omitempty"`
	Name     string                 `json:"name"`
	Response map[string]interface{} `json:"response"`
}

type AGSystemInstruction struct {
	Parts []AGPart `json:"parts"`
}

type AGTool struct {
	FunctionDeclarations []AGFunctionDeclaration `json:"functionDeclarations,omitempty"`
}

type AGFunctionDeclaration struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

type AGToolConfig struct {
	FunctionCallingConfig AGFunctionCallingConfig `json:"functionCallingConfig"`
}

type AGFunctionCallingConfig struct {
	Mode string `json:"mode"` // "VALIDATED", "AUTO", "NONE", etc.
}

type AGGenerationConfig struct {
	MaxOutputTokens  int               `json:"maxOutputTokens,omitempty"`
	Temperature      *float64          `json:"temperature,omitempty"`
	TopP             *float64          `json:"topP,omitempty"`
	ThinkingConfig   *AGThinkingConfig `json:"thinkingConfig,omitempty"`
}

type AGThinkingConfig struct {
	ThinkingBudget int `json:"thinkingBudget,omitempty"`
}

// Response structs

type AntigravityResponse struct {
	Response *AGResponseDetail `json:"response,omitempty"`
}

type AGResponseDetail struct {
	Candidates   []AGCandidate `json:"candidates,omitempty"`
	UsageMetadata *AGUsageMetadata `json:"usageMetadata,omitempty"`
	ModelVersion  string           `json:"modelVersion,omitempty"`
	ResponseID    string           `json:"responseId,omitempty"`
}

type AGCandidate struct {
	Content      AGContent `json:"content"`
	FinishReason string    `json:"finishReason,omitempty"`
}

type AGUsageMetadata struct {
	PromptTokenCount           int `json:"promptTokenCount"`
	CandidatesTokenCount       int `json:"candidatesTokenCount"`
	TotalTokenCount            int `json:"totalTokenCount"`
	ThoughtsTokenCount         int `json:"thoughtsTokenCount,omitempty"`
	CachedContentTokenCount    int `json:"cachedContentTokenCount,omitempty"`
}
