package translator

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type TranslateState struct {
	ToolNameMap   map[string]string
	ToolCallAccum map[int]*AccumulatedToolCall
	ResponseID    string
	ModelVersion  string
	Usage         *OpenAIUsage
}

type AccumulatedToolCall struct {
	ID        string
	Name      string
	Arguments string
}

func NewTranslateState(toolNameMap map[string]string) *TranslateState {
	return &TranslateState{
		ToolNameMap:   toolNameMap,
		ToolCallAccum: make(map[int]*AccumulatedToolCall),
		ResponseID:    fmt.Sprintf("chatcmpl-%d", time.Now().Unix()),
	}
}

// Convert Antigravity single SSE response chunk to OpenAI Chat Completion chunk format
func AntigravityToOpenAIResponse(agRes *AntigravityResponse, state *TranslateState) (*ChatCompletionChunk, error) {
	if agRes == nil || agRes.Response == nil {
		return nil, nil
	}

	detail := agRes.Response
	if len(detail.Candidates) == 0 {
		// Just handle usage metadata updates if candidates is empty
		if detail.UsageMetadata != nil {
			state.Usage = mapUsage(detail.UsageMetadata)
		}
		return nil, nil
	}

	candidate := detail.Candidates[0]
	finishReason := candidate.FinishReason

	// Map finish reason
	var openAIFinishReason interface{} = nil
	if finishReason != "" {
		switch strings.ToUpper(finishReason) {
		case "STOP":
			openAIFinishReason = "stop"
		case "MAX_TOKENS":
			openAIFinishReason = "length"
		case "SAFETY":
			openAIFinishReason = "content_filter"
		default:
			openAIFinishReason = "stop"
		}
	}

	// Update global info
	if detail.ModelVersion != "" {
		state.ModelVersion = detail.ModelVersion
	}
	if detail.ResponseID != "" {
		state.ResponseID = detail.ResponseID
	}
	if detail.UsageMetadata != nil {
		state.Usage = mapUsage(detail.UsageMetadata)
	}

	delta := Delta{}
	var toolCalls []ToolCall

	// Process parts
	for _, part := range candidate.Content.Parts {
		// Thought/Reasoning
		if part.Thought && part.Text != "" {
			delta.ReasoningContent = part.Text
		}

		// Plain text content
		if !part.Thought && part.Text != "" {
			delta.Content = part.Text
		}

		// Function/Tool Call
		if part.FunctionCall != nil {
			fc := part.FunctionCall
			name := fc.Name
			if uncloaked, ok := state.ToolNameMap[name]; ok {
				name = uncloaked
			}

			// Clean the suffix _ide if it's there
			if strings.HasSuffix(name, "_ide") {
				name = strings.TrimSuffix(name, "_ide")
			}

			argsBytes, _ := json.Marshal(fc.Args)
			argsStr := string(argsBytes)
			if argsStr == "" || argsStr == "null" {
				argsStr = "{}"
			}

			tcIndex := len(toolCalls)
			toolCalls = append(toolCalls, ToolCall{
				Index: &tcIndex,
				ID:    fc.ID,
				Type:  "function",
				Function: ToolFunction{
					Name:      name,
					Arguments: argsStr,
				},
			})
		}
	}

	if len(toolCalls) > 0 {
		delta.ToolCalls = toolCalls
		if openAIFinishReason == nil {
			openAIFinishReason = "tool_calls"
		}
	}

	// If everything is empty and it's not a stop chunk, skip
	if delta.Content == "" && delta.ReasoningContent == "" && len(delta.ToolCalls) == 0 && openAIFinishReason == nil {
		return nil, nil
	}

	chunk := &ChatCompletionChunk{
		ID:      state.ResponseID,
		Object:  "chat.completion.chunk",
		Created: time.Now().Unix(),
		Model:   state.ModelVersion,
		Choices: []ChoiceChunk{
			{
				Index:        0,
				Delta:        delta,
				FinishReason: openAIFinishReason,
			},
		},
	}

	if state.Usage != nil {
		chunk.Usage = state.Usage
	}

	return chunk, nil
}

func mapUsage(metadata *AGUsageMetadata) *OpenAIUsage {
	usage := &OpenAIUsage{
		PromptTokens:     metadata.PromptTokenCount,
		CompletionTokens: metadata.CandidatesTokenCount,
		TotalTokens:      metadata.TotalTokenCount,
	}

	if metadata.ThoughtsTokenCount > 0 {
		usage.CompletionTokensDetails = &CompletionTokensDetails{
			ReasoningTokens: metadata.ThoughtsTokenCount,
		}
	}

	if metadata.CachedContentTokenCount > 0 {
		usage.PromptTokensDetails = &PromptTokensDetails{
			CachedTokens: metadata.CachedContentTokenCount,
		}
	}

	return usage
}
