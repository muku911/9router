package executor

import (
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

// TranslateResponseChunk translates a single SSE chunk from the target provider format
// back to OpenAI chat.completion.chunk format.
// Returns the translated JSON string, or empty string to skip.
func TranslateResponseChunk(targetFormat string, data string, model string) string {
	switch targetFormat {
	case FormatClaude:
		return claudeToOpenAI(data, model)
	case FormatGemini:
		return geminiToOpenAI(data, model)
	case FormatOpenAI:
		return data // Already OpenAI format
	default:
		return data
	}
}

// --- Claude → OpenAI ---

func claudeToOpenAI(data string, model string) string {
	var event map[string]interface{}
	if err := json.Unmarshal([]byte(data), &event); err != nil {
		return ""
	}

	eventType, _ := event["type"].(string)

	switch eventType {
	case "content_block_delta":
		delta, _ := event["delta"].(map[string]interface{})
		if delta == nil {
			return ""
		}
		deltaType, _ := delta["type"].(string)

		switch deltaType {
		case "text_delta":
			text, _ := delta["text"].(string)
			return makeOpenAIChunk(model, map[string]interface{}{
				"content": text,
			}, nil)

		case "thinking_delta":
			text, _ := delta["thinking"].(string)
			return makeOpenAIChunk(model, map[string]interface{}{
				"reasoning_content": text,
			}, nil)

		case "input_json_delta":
			partial, _ := delta["partial_json"].(string)
			idx, _ := event["index"].(float64)
			return makeOpenAIChunk(model, map[string]interface{}{
				"tool_calls": []interface{}{
					map[string]interface{}{
						"index": int(idx),
						"function": map[string]interface{}{
							"arguments": partial,
						},
					},
				},
			}, nil)
		}

	case "content_block_start":
		cb, _ := event["content_block"].(map[string]interface{})
		if cb == nil {
			return ""
		}
		cbType, _ := cb["type"].(string)
		if cbType == "tool_use" {
			idx, _ := event["index"].(float64)
			return makeOpenAIChunk(model, map[string]interface{}{
				"tool_calls": []interface{}{
					map[string]interface{}{
						"index": int(idx),
						"id":    cb["id"],
						"type":  "function",
						"function": map[string]interface{}{
							"name":      cb["name"],
							"arguments": "",
						},
					},
				},
			}, nil)
		}

	case "message_delta":
		delta, _ := event["delta"].(map[string]interface{})
		if delta == nil {
			return ""
		}
		stopReason, _ := delta["stop_reason"].(string)
		finishReason := "stop"
		if stopReason == "tool_use" {
			finishReason = "tool_calls"
		} else if stopReason == "max_tokens" {
			finishReason = "length"
		}
		return makeOpenAIChunk(model, map[string]interface{}{}, &finishReason)

	case "message_stop":
		return "[DONE]"

	case "ping", "message_start", "content_block_stop":
		return "" // Skip these events
	}

	return ""
}

// --- Gemini → OpenAI ---

func geminiToOpenAI(data string, model string) string {
	var resp map[string]interface{}
	if err := json.Unmarshal([]byte(data), &resp); err != nil {
		return ""
	}

	candidates, _ := resp["candidates"].([]interface{})
	if len(candidates) == 0 {
		return ""
	}

	candidate, _ := candidates[0].(map[string]interface{})
	if candidate == nil {
		return ""
	}

	content, _ := candidate["content"].(map[string]interface{})
	if content == nil {
		return ""
	}

	parts, _ := content["parts"].([]interface{})
	if len(parts) == 0 {
		return ""
	}

	// Check finish reason
	var finishReason *string
	if fr, ok := candidate["finishReason"].(string); ok && fr != "" {
		mapped := "stop"
		switch fr {
		case "MAX_TOKENS":
			mapped = "length"
		case "SAFETY":
			mapped = "content_filter"
		}
		finishReason = &mapped
	}

	// Process parts
	for _, part := range parts {
		pMap, _ := part.(map[string]interface{})
		if pMap == nil {
			continue
		}

		// Text content
		if text, ok := pMap["text"].(string); ok {
			return makeOpenAIChunk(model, map[string]interface{}{
				"content": text,
			}, finishReason)
		}

		// Function call
		if fc, ok := pMap["functionCall"].(map[string]interface{}); ok {
			argsJSON, _ := json.Marshal(fc["args"])
			return makeOpenAIChunk(model, map[string]interface{}{
				"tool_calls": []interface{}{
					map[string]interface{}{
						"index": 0,
						"id":    "call_" + uuid.New().String()[:8],
						"type":  "function",
						"function": map[string]interface{}{
							"name":      fc["name"],
							"arguments": string(argsJSON),
						},
					},
				},
			}, &[]string{"tool_calls"}[0])
		}

		// Thought content
		if thought, ok := pMap["thought"].(bool); ok && thought {
			if text, ok := pMap["text"].(string); ok {
				return makeOpenAIChunk(model, map[string]interface{}{
					"reasoning_content": text,
				}, nil)
			}
		}
	}

	return ""
}

// makeOpenAIChunk builds a chat.completion.chunk JSON string.
func makeOpenAIChunk(model string, delta map[string]interface{}, finishReason *string) string {
	choice := map[string]interface{}{
		"index": 0,
		"delta": delta,
	}
	if finishReason != nil {
		choice["finish_reason"] = *finishReason
	}

	chunk := map[string]interface{}{
		"id":      "chatcmpl-" + uuid.New().String()[:12],
		"object":  "chat.completion.chunk",
		"created": 0,
		"model":   model,
		"choices": []interface{}{choice},
	}

	b, err := json.Marshal(chunk)
	if err != nil {
		return ""
	}
	return string(b)
}

// TranslateNonStreamingResponse translates a full (non-streaming) response
// from target format back to OpenAI chat.completion format.
func TranslateNonStreamingResponse(targetFormat string, responseBody []byte, model string) ([]byte, error) {
	switch targetFormat {
	case FormatClaude:
		return claudeFullToOpenAI(responseBody, model)
	case FormatGemini:
		return geminiFullToOpenAI(responseBody, model)
	default:
		return responseBody, nil
	}
}

func claudeFullToOpenAI(body []byte, model string) ([]byte, error) {
	var resp map[string]interface{}
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, nil
	}

	content, _ := resp["content"].([]interface{})
	var text string
	var toolCalls []interface{}

	for _, c := range content {
		cMap, _ := c.(map[string]interface{})
		if cMap == nil {
			continue
		}
		if cMap["type"] == "text" {
			text, _ = cMap["text"].(string)
		}
		if cMap["type"] == "tool_use" {
			argsJSON, _ := json.Marshal(cMap["input"])
			toolCalls = append(toolCalls, map[string]interface{}{
				"id":   cMap["id"],
				"type": "function",
				"function": map[string]interface{}{
					"name":      cMap["name"],
					"arguments": string(argsJSON),
				},
			})
		}
	}

	message := map[string]interface{}{
		"role":    "assistant",
		"content": text,
	}
	if len(toolCalls) > 0 {
		message["tool_calls"] = toolCalls
	}

	finishReason := "stop"
	stopReason, _ := resp["stop_reason"].(string)
	if stopReason == "tool_use" {
		finishReason = "tool_calls"
	} else if stopReason == "max_tokens" {
		finishReason = "length"
	}

	result := map[string]interface{}{
		"id":      "chatcmpl-" + uuid.New().String()[:12],
		"object":  "chat.completion",
		"created": 0,
		"model":   model,
		"choices": []interface{}{
			map[string]interface{}{
				"index":         0,
				"message":       message,
				"finish_reason": finishReason,
			},
		},
		"usage": resp["usage"],
	}

	return json.Marshal(result)
}

func geminiFullToOpenAI(body []byte, model string) ([]byte, error) {
	var resp map[string]interface{}
	if err := json.Unmarshal(body, &resp); err != nil {
		return body, nil
	}

	candidates, _ := resp["candidates"].([]interface{})
	if len(candidates) == 0 {
		return body, nil
	}

	candidate, _ := candidates[0].(map[string]interface{})
	content, _ := candidate["content"].(map[string]interface{})
	parts, _ := content["parts"].([]interface{})

	var text string
	var toolCalls []interface{}

	for _, p := range parts {
		pMap, _ := p.(map[string]interface{})
		if pMap == nil {
			continue
		}
		if t, ok := pMap["text"].(string); ok {
			text += t
		}
		if fc, ok := pMap["functionCall"].(map[string]interface{}); ok {
			argsJSON, _ := json.Marshal(fc["args"])
			toolCalls = append(toolCalls, map[string]interface{}{
				"id":   "call_" + uuid.New().String()[:8],
				"type": "function",
				"function": map[string]interface{}{
					"name":      fc["name"],
					"arguments": string(argsJSON),
				},
			})
		}
	}

	message := map[string]interface{}{
		"role":    "assistant",
		"content": text,
	}
	if len(toolCalls) > 0 {
		message["tool_calls"] = toolCalls
	}

	finishReason := "stop"
	if fr, ok := candidate["finishReason"].(string); ok {
		if fr == "MAX_TOKENS" {
			finishReason = "length"
		}
	}

	result := map[string]interface{}{
		"id":      "chatcmpl-" + uuid.New().String()[:12],
		"object":  "chat.completion",
		"created": 0,
		"model":   model,
		"choices": []interface{}{
			map[string]interface{}{
				"index":         0,
				"message":       message,
				"finish_reason": finishReason,
			},
		},
	}

	// Map usage
	if usageMeta, ok := resp["usageMetadata"].(map[string]interface{}); ok {
		result["usage"] = map[string]interface{}{
			"prompt_tokens":     usageMeta["promptTokenCount"],
			"completion_tokens": usageMeta["candidatesTokenCount"],
			"total_tokens":      usageMeta["totalTokenCount"],
		}
	}

	return json.Marshal(result)
}

// FormatSSEData wraps a data string into SSE wire format.
func FormatSSEData(data string) string {
	return fmt.Sprintf("data: %s\n\n", data)
}
