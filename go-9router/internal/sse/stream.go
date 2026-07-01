package sse

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"go-9router/internal/translator"
)

// Read SSE stream from upstream Antigravity server, translate, and write to client
func ProcessSSEStream(ctx context.Context, upstreamBody io.ReadCloser, w http.ResponseWriter, state *translator.TranslateState) error {
	defer upstreamBody.Close()

	flusher, ok := w.(http.Flusher)
	if !ok {
		return fmt.Errorf("response writer does not support flushing")
	}

	reader := bufio.NewReader(upstreamBody)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Read line by line until we find a full SSE event block
			line, err := reader.ReadBytes('\n')
			if err != nil {
				if err == io.EOF {
					return nil
				}
				return err
			}

			lineStr := string(line)
			if !strings.HasPrefix(lineStr, "data:") {
				// We only care about data: prefix in standard SSE
				continue
			}

			// Extract json payload
			dataPayload := strings.TrimSpace(strings.TrimPrefix(lineStr, "data:"))
			if dataPayload == "" {
				continue
			}

			// Handle stream termination markers if any
			if dataPayload == "[DONE]" {
				_, _ = w.Write([]byte("data: [DONE]\n\n"))
				flusher.Flush()
				return nil
			}

			// Unmarshal Antigravity chunk
			var agChunk translator.AntigravityResponse
			if err := json.Unmarshal([]byte(dataPayload), &agChunk); err != nil {
				// Log or skip malformed JSON from upstream
				continue
			}

			// Translate
			openaiChunk, err := translator.AntigravityToOpenAIResponse(&agChunk, state)
			if err != nil {
				return fmt.Errorf("failed to translate response chunk: %w", err)
			}

			if openaiChunk == nil {
				// Skip if chunk mapping returned nil (e.g. empty candidate but not finished yet)
				continue
			}

			// Marshal and write back to downstream client
			resBytes, err := json.Marshal(openaiChunk)
			if err != nil {
				continue
			}

			_, _ = w.Write([]byte("data: "))
			_, _ = w.Write(resBytes)
			_, _ = w.Write([]byte("\n\n"))
			flusher.Flush()

			// Check if finished
			if len(openaiChunk.Choices) > 0 && openaiChunk.Choices[0].FinishReason != nil {
				// Send final [DONE]
				_, _ = w.Write([]byte("data: [DONE]\n\n"))
				flusher.Flush()
				return nil
			}
		}
	}
}

// Processes non-streaming JSON responses
func ProcessJSONResponse(upstreamBody io.ReadCloser, state *translator.TranslateState) ([]byte, error) {
	defer upstreamBody.Close()

	bodyBytes, err := io.ReadAll(upstreamBody)
	if err != nil {
		return nil, err
	}

	var agRes translator.AntigravityResponse
	if err := json.Unmarshal(bodyBytes, &agRes); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON response: %w", err)
	}

	openaiChunk, err := translator.AntigravityToOpenAIResponse(&agRes, state)
	if err != nil {
		return nil, fmt.Errorf("failed to translate JSON response: %w", err)
	}

	if openaiChunk == nil {
		return nil, fmt.Errorf("translated response chunk is empty")
	}

	// OpenAI JSON response expects ChatCompletion structure instead of ChatCompletionChunk
	// We map the chunk contents to a full response object
	var finishReason string
	if openaiChunk.Choices[0].FinishReason != nil {
		finishReason = openaiChunk.Choices[0].FinishReason.(string)
	}

	openaiResponse := map[string]interface{}{
		"id":      openaiChunk.ID,
		"object":  "chat.completion",
		"created": openaiChunk.Created,
		"model":   openaiChunk.Model,
		"choices": []map[string]interface{}{
			{
				"index": 0,
				"message": map[string]interface{}{
					"role":             "assistant",
					"content":          openaiChunk.Choices[0].Delta.Content,
					"reasoning_content": openaiChunk.Choices[0].Delta.ReasoningContent,
					"tool_calls":       openaiChunk.Choices[0].Delta.ToolCalls,
				},
				"finish_reason": finishReason,
			},
		},
	}

	if openaiChunk.Usage != nil {
		openaiResponse["usage"] = openaiChunk.Usage
	}

	return json.Marshal(openaiResponse)
}
