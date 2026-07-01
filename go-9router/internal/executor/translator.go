package executor

import "encoding/json"

// TranslateRequest translates the request body from OpenAI format to the target provider's format.
// Follows the open-sse hub-and-spoke pattern: source → OpenAI → target.
func TranslateRequest(targetFormat string, model string, body map[string]interface{}, stream bool) map[string]interface{} {
	switch targetFormat {
	case FormatClaude:
		return openaiToClaude(model, body, stream)
	case FormatGemini:
		return openaiToGemini(model, body, stream)
	case FormatOpenAI:
		return body
	default:
		return body
	}
}

// --- OpenAI → Claude ---

func openaiToClaude(model string, body map[string]interface{}, stream bool) map[string]interface{} {
	messages, _ := body["messages"].([]interface{})
	if messages == nil {
		return body
	}

	result := map[string]interface{}{
		"model":  model,
		"stream": stream,
	}

	var systemParts []string
	var claudeMessages []interface{}

	for _, msg := range messages {
		m, ok := msg.(map[string]interface{})
		if !ok {
			continue
		}
		role, _ := m["role"].(string)

		if role == "system" {
			if content, ok := m["content"].(string); ok {
				systemParts = append(systemParts, content)
			}
			continue
		}

		if role == "assistant" {
			if toolCalls, ok := m["tool_calls"].([]interface{}); ok && len(toolCalls) > 0 {
				var contentParts []interface{}
				if text, ok := m["content"].(string); ok && text != "" {
					contentParts = append(contentParts, map[string]interface{}{"type": "text", "text": text})
				}
				for _, tc := range toolCalls {
					tcMap, _ := tc.(map[string]interface{})
					if tcMap == nil {
						continue
					}
					fn, _ := tcMap["function"].(map[string]interface{})
					if fn == nil {
						continue
					}
					contentParts = append(contentParts, map[string]interface{}{
						"type": "tool_use", "id": tcMap["id"], "name": fn["name"], "input": parseJSONValue(fn["arguments"]),
					})
				}
				claudeMessages = append(claudeMessages, map[string]interface{}{"role": "assistant", "content": contentParts})
				continue
			}
		}

		if role == "tool" {
			toolCallID, _ := m["tool_call_id"].(string)
			content, _ := m["content"].(string)
			claudeMessages = append(claudeMessages, map[string]interface{}{
				"role": "user",
				"content": []interface{}{
					map[string]interface{}{"type": "tool_result", "tool_use_id": toolCallID, "content": content},
				},
			})
			continue
		}

		claudeRole := role
		if role != "assistant" {
			claudeRole = "user"
		}
		claudeMsg := map[string]interface{}{"role": claudeRole}
		if content, ok := m["content"].(string); ok {
			claudeMsg["content"] = content
		} else {
			claudeMsg["content"] = m["content"]
		}
		claudeMessages = append(claudeMessages, claudeMsg)
	}

	result["messages"] = claudeMessages
	if len(systemParts) > 0 {
		combined := ""
		for i, s := range systemParts {
			if i > 0 {
				combined += "\n\n"
			}
			combined += s
		}
		result["system"] = combined
	}

	if maxTokens, ok := body["max_tokens"]; ok {
		result["max_tokens"] = maxTokens
	} else {
		result["max_tokens"] = 8192
	}
	if temp, ok := body["temperature"]; ok {
		result["temperature"] = temp
	}
	if topP, ok := body["top_p"]; ok {
		result["top_p"] = topP
	}

	if tools, ok := body["tools"].([]interface{}); ok && len(tools) > 0 {
		var claudeTools []interface{}
		for _, t := range tools {
			tMap, _ := t.(map[string]interface{})
			if tMap == nil {
				continue
			}
			fn, _ := tMap["function"].(map[string]interface{})
			if fn == nil {
				continue
			}
			claudeTools = append(claudeTools, map[string]interface{}{
				"name": fn["name"], "description": fn["description"], "input_schema": fn["parameters"],
			})
		}
		result["tools"] = claudeTools
	}

	return result
}

// --- OpenAI → Gemini ---

func openaiToGemini(model string, body map[string]interface{}, stream bool) map[string]interface{} {
	messages, _ := body["messages"].([]interface{})
	if messages == nil {
		return body
	}

	var contents []interface{}
	var systemInstruction interface{}

	for _, msg := range messages {
		m, ok := msg.(map[string]interface{})
		if !ok {
			continue
		}
		role, _ := m["role"].(string)

		if role == "system" {
			if content, ok := m["content"].(string); ok {
				systemInstruction = map[string]interface{}{
					"parts": []interface{}{map[string]interface{}{"text": content}},
				}
			}
			continue
		}

		geminiRole := "user"
		if role == "assistant" {
			geminiRole = "model"
		}

		var parts []interface{}
		if content, ok := m["content"].(string); ok {
			parts = append(parts, map[string]interface{}{"text": content})
		} else if contentArr, ok := m["content"].([]interface{}); ok {
			for _, part := range contentArr {
				pMap, _ := part.(map[string]interface{})
				if pMap == nil {
					continue
				}
				if pMap["type"] == "text" {
					parts = append(parts, map[string]interface{}{"text": pMap["text"]})
				}
			}
		}

		if role == "assistant" {
			if toolCalls, ok := m["tool_calls"].([]interface{}); ok {
				for _, tc := range toolCalls {
					tcMap, _ := tc.(map[string]interface{})
					if tcMap == nil {
						continue
					}
					fn, _ := tcMap["function"].(map[string]interface{})
					if fn == nil {
						continue
					}
					parts = append(parts, map[string]interface{}{
						"functionCall": map[string]interface{}{"name": fn["name"], "args": parseJSONValue(fn["arguments"])},
					})
				}
			}
		}

		if role == "tool" {
			content, _ := m["content"].(string)
			parts = append(parts, map[string]interface{}{
				"functionResponse": map[string]interface{}{
					"name":     m["name"],
					"response": map[string]interface{}{"content": content},
				},
			})
		}

		if len(parts) > 0 {
			contents = append(contents, map[string]interface{}{"role": geminiRole, "parts": parts})
		}
	}

	result := map[string]interface{}{"contents": contents}
	if systemInstruction != nil {
		result["systemInstruction"] = systemInstruction
	}

	genConfig := map[string]interface{}{}
	if temp, ok := body["temperature"]; ok {
		genConfig["temperature"] = temp
	}
	if maxTokens, ok := body["max_tokens"]; ok {
		genConfig["maxOutputTokens"] = maxTokens
	}
	if topP, ok := body["top_p"]; ok {
		genConfig["topP"] = topP
	}
	if len(genConfig) > 0 {
		result["generationConfig"] = genConfig
	}

	if tools, ok := body["tools"].([]interface{}); ok && len(tools) > 0 {
		var funcDecls []interface{}
		for _, t := range tools {
			tMap, _ := t.(map[string]interface{})
			if tMap == nil {
				continue
			}
			fn, _ := tMap["function"].(map[string]interface{})
			if fn == nil {
				continue
			}
			funcDecls = append(funcDecls, map[string]interface{}{
				"name": fn["name"], "description": fn["description"], "parameters": fn["parameters"],
			})
		}
		result["tools"] = []interface{}{map[string]interface{}{"functionDeclarations": funcDecls}}
	}

	return result
}

func parseJSONValue(v interface{}) interface{} {
	s, ok := v.(string)
	if !ok {
		return v
	}
	var result interface{}
	if err := json.Unmarshal([]byte(s), &result); err != nil {
		return v
	}
	return result
}
