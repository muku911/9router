package translator

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

var (
	sanitizeRegex = regexp.MustCompile(`[^a-zA-Z0-9_.:\-]`)
	letterRegex   = regexp.MustCompile(`^[a-zA-Z_]`)
)

func sanitizeFunctionName(name string) string {
	if name == "" {
		return "_unknown"
	}
	s := sanitizeRegex.ReplaceAllString(name, "_")
	if !letterRegex.MatchString(s) {
		s = "_" + s
	}
	if len(s) > 64 {
		s = s[:64]
	}
	return s
}

// Convert OpenAI API request to Antigravity internal request
func OpenAIRequestToAntigravity(openaiReq *OpenAIRequest, project string, sessionID string) (*AntigravityRequest, map[string]string, error) {
	agReq := &AntigravityRequest{
		Project:     project,
		Model:       openaiReq.Model,
		UserAgent:   "antigravity",
		RequestType: "agent",
		RequestID:   fmt.Sprintf("agent-%s", uuid.New().String()),
		Request: AGRequestDetail{
			SessionID: sessionID,
		},
	}

	// 1. Generation Config
	genConfig := &AGGenerationConfig{}
	if openaiReq.MaxTokens > 0 {
		// cap at max tokens
		if openaiReq.MaxTokens > 16384 {
			genConfig.MaxOutputTokens = 16384
		} else {
			genConfig.MaxOutputTokens = openaiReq.MaxTokens
		}
	}
	if openaiReq.Temperature != nil {
		genConfig.Temperature = openaiReq.Temperature
	}
	if openaiReq.TopP != nil {
		genConfig.TopP = openaiReq.TopP
	}

	// Handle thinking budget mapping (simplification based on reasoning_effort or reasoning_tokens)
	if openaiReq.ReasoningEffort != "" {
		budget := 1024
		if openaiReq.ReasoningEffort == "medium" {
			budget = 4096
		} else if openaiReq.ReasoningEffort == "high" {
			budget = 8192
		}
		genConfig.ThinkingConfig = &AGThinkingConfig{
			ThinkingBudget: budget,
		}
	}

	agReq.Request.GenerationConfig = genConfig

	// 2. System Instruction & Messages
	var systemParts []AGPart
	var contents []AGContent

	for _, msg := range openaiReq.Messages {
		if msg.Role == "system" {
			// Extract system prompt text
			if contentStr, ok := msg.Content.(string); ok {
				systemParts = append(systemParts, AGPart{Text: contentStr})
			} else if partsBytes, err := json.Marshal(msg.Content); err == nil {
				var parts []ContentPart
				if err := json.Unmarshal(partsBytes, &parts); err == nil {
					for _, p := range parts {
						if p.Type == "text" {
							systemParts = append(systemParts, AGPart{Text: p.Text})
						}
					}
				}
			}
			continue
		}

		// Regular contents (user / assistant / tool)
		role := msg.Role
		if role == "assistant" {
			role = "model"
		}

		parts := []AGPart{}

		// Handle reasoning_content (thought)
		if msg.ReasoningContent != "" {
			parts = append(parts, AGPart{Thought: true, Text: msg.ReasoningContent})
		}

		// Handle Content
		if msg.Content != nil {
			if contentStr, ok := msg.Content.(string); ok && contentStr != "" {
				parts = append(parts, AGPart{Text: contentStr})
			} else {
				// Try deserializing complex content parts (image, etc.)
				contentBytes, err := json.Marshal(msg.Content)
				if err == nil {
					var rawParts []ContentPart
					if err := json.Unmarshal(contentBytes, &rawParts); err == nil {
						for _, p := range rawParts {
							if p.Type == "text" && p.Text != "" {
								parts = append(parts, AGPart{Text: p.Text})
							} else if p.Type == "image_url" && p.ImageURL != nil {
								// Parse data URL format: data:<mime>;base64,<data>
								url := p.ImageURL.URL
								if strings.HasPrefix(url, "data:") {
									commaIdx := strings.Index(url, ",")
									if commaIdx != -1 {
										meta := url[5:commaIdx]
										data := url[commaIdx+1:]
										mime := strings.Split(meta, ";")[0]
										parts = append(parts, AGPart{
											InlineData: &AGInlineData{
												MimeType: mime,
												Data:     data,
											},
										})
									}
								}
							}
						}
					}
				}
			}
		}

		// Handle ToolCalls (assistant role calling tools)
		for _, tc := range msg.ToolCalls {
			var args map[string]interface{}
			_ = json.Unmarshal([]byte(tc.Function.Arguments), &args)

			parts = append(parts, AGPart{
				FunctionCall: &AGFunctionCall{
					ID:   tc.ID,
					Name: sanitizeFunctionName(tc.Function.Name),
					Args: args,
				},
			})
		}

		// Handle Tool response (tool role returning result)
		if msg.Role == "tool" {
			role = "user" // functionResponse is treated as user role by Gemini/AG
			var responseMap map[string]interface{}
			contentStr, isStr := msg.Content.(string)
			if isStr {
				if err := json.Unmarshal([]byte(contentStr), &responseMap); err != nil {
					responseMap = map[string]interface{}{"result": contentStr}
				}
			} else {
				resBytes, _ := json.Marshal(msg.Content)
				_ = json.Unmarshal(resBytes, &responseMap)
			}

			parts = append(parts, AGPart{
				FunctionResponse: &AGFunctionResponse{
					ID:       msg.ToolCallID,
					Name:     msg.ToolCallID, // using id or function name
					Response: responseMap,
				},
			})
		}

		if len(parts) > 0 {
			contents = append(contents, AGContent{
				Role:  role,
				Parts: parts,
			})
		}
	}

	if len(systemParts) > 0 {
		agReq.Request.SystemInstruction = &AGSystemInstruction{
			Parts: systemParts,
		}
	}
	agReq.Request.Contents = contents

	// 3. Tools Cloaking and translation
	var tools []AGTool
	toolNameMap := make(map[string]string) // cloaked -> original

	if len(openaiReq.Tools) > 0 {
		agDeclarations := []AGFunctionDeclaration{}
		for _, t := range openaiReq.Tools {
			if t.Type == "function" {
				origName := t.Function.Name
				// Simple tool cloaking suffixing
				cloakedName := origName + "_ide"
				toolNameMap[cloakedName] = origName

				// Sanitize tool parameters types
				normalizedParams := normalizeSchemaTypes(t.Function.Parameters)

				agDeclarations = append(agDeclarations, AGFunctionDeclaration{
					Name:        sanitizeFunctionName(cloakedName),
					Description: t.Function.Description,
					Parameters:  normalizedParams,
				})
			}
		}

		if len(agDeclarations) > 0 {
			tools = append(tools, AGTool{
				FunctionDeclarations: agDeclarations,
			})
			agReq.Request.Tools = tools
			agReq.Request.ToolConfig = &AGToolConfig{
				FunctionCallingConfig: AGFunctionCallingConfig{
					Mode: "VALIDATED",
				},
			}
		}
	}

	return agReq, toolNameMap, nil
}

// Convert schema types to lowercase and remove enumDescriptions to match Google format
func normalizeSchemaTypes(schema map[string]interface{}) map[string]interface{} {
	if schema == nil {
		return map[string]interface{}{"type": "object", "properties": map[string]interface{}{}}
	}

	result := make(map[string]interface{})
	for k, v := range schema {
		if k == "type" {
			if typeStr, ok := v.(string); ok {
				result[k] = strings.ToLower(typeStr)
				continue
			}
		}
		if k == "enumDescriptions" {
			continue // skip
		}

		if subMap, ok := v.(map[string]interface{}); ok {
			result[k] = normalizeSchemaTypes(subMap)
		} else if itemsList, ok := v.([]interface{}); ok {
			// recursively normalize items list if needed, or map elements
			normalizedItems := make([]interface{}, len(itemsList))
			for i, item := range itemsList {
				if itemMap, ok := item.(map[string]interface{}); ok {
					normalizedItems[i] = normalizeSchemaTypes(itemMap)
				} else {
					normalizedItems[i] = item
				}
			}
			result[k] = normalizedItems
		} else {
			result[k] = v
		}
	}
	return result
}
