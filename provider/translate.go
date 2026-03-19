package provider

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// --- OpenAI → Anthropic Messages API ---

// TranslateOpenAIToAnthropic translates an OpenAI chat/completions request to the Claude Messages API
func TranslateOpenAIToAnthropic(body []byte, model string) ([]byte, string, error) {
	var openAI struct {
		Model       string            `json:"model"`
		Messages    []json.RawMessage `json:"messages"`
		Stream      bool              `json:"stream"`
		MaxTokens   int               `json:"max_tokens,omitempty"`
		Temperature *float64          `json:"temperature,omitempty"`
		TopP        *float64          `json:"top_p,omitempty"`
		Tools       json.RawMessage   `json:"tools,omitempty"`
		ToolChoice  json.RawMessage   `json:"tool_choice,omitempty"`
		Stop        json.RawMessage   `json:"stop,omitempty"`
	}
	if err := json.Unmarshal(body, &openAI); err != nil {
		return nil, "", fmt.Errorf("parse OpenAI request: %w", err)
	}

	// Build the Anthropic request
	claude := map[string]any{
		"model":  mapModelToAnthropic(model),
		"stream": openAI.Stream,
	}

	// Max tokens (required by Claude)
	maxTokens := openAI.MaxTokens
	if maxTokens == 0 {
		maxTokens = 8192
	}
	claude["max_tokens"] = maxTokens

	if openAI.Temperature != nil {
		claude["temperature"] = *openAI.Temperature
	}
	if openAI.TopP != nil {
		claude["top_p"] = *openAI.TopP
	}
	if openAI.Stop != nil {
		claude["stop_sequences"] = openAI.Stop
	}

	// Split system and convert messages
	var systemParts []string
	var claudeMessages []map[string]any

	for _, rawMsg := range openAI.Messages {
		var msg struct {
			Role       string          `json:"role"`
			Content    json.RawMessage `json:"content"`
			ToolCalls  json.RawMessage `json:"tool_calls,omitempty"`
			ToolCallID string          `json:"tool_call_id,omitempty"`
		}
		if json.Unmarshal(rawMsg, &msg) != nil {
			continue
		}

		switch msg.Role {
		case "system":
			// System goes into the top-level field
			text := extractTextFromContent(msg.Content)
			if text != "" {
				systemParts = append(systemParts, text)
			}

		case "user":
			claudeMsg := map[string]any{
				"role":    "user",
				"content": convertContent(msg.Content),
			}
			claudeMessages = append(claudeMessages, claudeMsg)

		case "assistant":
			claudeMsg := map[string]any{
				"role": "assistant",
			}
			// If there are tool_calls, convert them into content blocks
			if msg.ToolCalls != nil {
				content := convertAssistantWithToolCalls(msg.Content, msg.ToolCalls)
				claudeMsg["content"] = content
			} else {
				claudeMsg["content"] = convertContent(msg.Content)
			}
			claudeMessages = append(claudeMessages, claudeMsg)

		case "tool":
			// Tool result → Claude tool_result block
			claudeMsg := map[string]any{
				"role": "user",
				"content": []map[string]any{{
					"type":        "tool_result",
					"tool_use_id": msg.ToolCallID,
					"content":     extractTextFromContent(msg.Content),
				}},
			}
			claudeMessages = append(claudeMessages, claudeMsg)
		}
	}

	if len(systemParts) > 0 {
		claude["system"] = strings.Join(systemParts, "\n\n")
	}
	claude["messages"] = claudeMessages

	// Convert tools
	if openAI.Tools != nil {
		claudeTools := convertToolsToAnthropic(openAI.Tools)
		if claudeTools != nil {
			claude["tools"] = claudeTools
		}
	}

	// Convert tool_choice
	if openAI.ToolChoice != nil {
		claude["tool_choice"] = convertToolChoiceToAnthropic(openAI.ToolChoice)
	}

	result, err := json.Marshal(claude)
	return result, "application/json", err
}

// --- Anthropic SSE → OpenAI SSE ---

// TranslateAnthropicStreamToOpenAI translates an SSE event from Claude into OpenAI format
func TranslateAnthropicStreamToOpenAI(data []byte) ([]byte, error) {
	var event struct {
		Type  string          `json:"type"`
		Index int             `json:"index"`
		Delta json.RawMessage `json:"delta,omitempty"`

		// message_start
		Message *struct {
			ID    string `json:"id"`
			Model string `json:"model"`
			Usage *struct {
				InputTokens  int `json:"input_tokens"`
				OutputTokens int `json:"output_tokens"`
			} `json:"usage"`
		} `json:"message,omitempty"`

		// content_block_start
		ContentBlock *struct {
			Type  string `json:"type"`
			ID    string `json:"id,omitempty"`
			Name  string `json:"name,omitempty"`
			Text  string `json:"text,omitempty"`
			Input any    `json:"input,omitempty"`
		} `json:"content_block,omitempty"`
	}

	if err := json.Unmarshal(data, &event); err != nil {
		return nil, nil // Ignore lines that don't parse
	}

	chatID := fmt.Sprintf("chatcmpl-%d", time.Now().UnixNano())
	created := time.Now().Unix()

	switch event.Type {
	case "message_start":
		// Send role
		if event.Message != nil {
			chatID = event.Message.ID
		}
		chunk := map[string]any{
			"id":      chatID,
			"object":  "chat.completion.chunk",
			"created": created,
			"choices": []map[string]any{{
				"index": 0,
				"delta": map[string]any{"role": "assistant"},
			}},
		}
		return json.Marshal(chunk)

	case "content_block_start":
		if event.ContentBlock != nil && event.ContentBlock.Type == "tool_use" {
			// Start of tool_use
			chunk := map[string]any{
				"id":      chatID,
				"object":  "chat.completion.chunk",
				"created": created,
				"choices": []map[string]any{{
					"index": 0,
					"delta": map[string]any{
						"tool_calls": []map[string]any{{
							"index": event.Index,
							"id":    event.ContentBlock.ID,
							"type":  "function",
							"function": map[string]any{
								"name":      event.ContentBlock.Name,
								"arguments": "",
							},
						}},
					},
				}},
			}
			return json.Marshal(chunk)
		}
		return nil, nil

	case "content_block_delta":
		var delta struct {
			Type        string `json:"type"`
			Text        string `json:"text,omitempty"`
			PartialJSON string `json:"partial_json,omitempty"`
		}
		if json.Unmarshal(event.Delta, &delta) != nil {
			return nil, nil
		}

		switch delta.Type {
		case "text_delta":
			chunk := map[string]any{
				"id":      chatID,
				"object":  "chat.completion.chunk",
				"created": created,
				"choices": []map[string]any{{
					"index": 0,
					"delta": map[string]any{"content": delta.Text},
				}},
			}
			return json.Marshal(chunk)

		case "input_json_delta":
			chunk := map[string]any{
				"id":      chatID,
				"object":  "chat.completion.chunk",
				"created": created,
				"choices": []map[string]any{{
					"index": 0,
					"delta": map[string]any{
						"tool_calls": []map[string]any{{
							"index": event.Index,
							"function": map[string]any{
								"arguments": delta.PartialJSON,
							},
						}},
					},
				}},
			}
			return json.Marshal(chunk)
		}
		return nil, nil

	case "message_delta":
		var delta struct {
			StopReason string `json:"stop_reason"`
		}
		if json.Unmarshal(event.Delta, &delta) != nil {
			return nil, nil
		}

		finishReason := "stop"
		if delta.StopReason == "tool_use" {
			finishReason = "tool_calls"
		}

		chunk := map[string]any{
			"id":      chatID,
			"object":  "chat.completion.chunk",
			"created": created,
			"choices": []map[string]any{{
				"index":         0,
				"delta":         map[string]any{},
				"finish_reason": finishReason,
			}},
		}
		return json.Marshal(chunk)

	case "message_stop":
		return nil, nil // Handled in proxyStream as [DONE]
	}

	return nil, nil
}

// --- Anthropic non-stream → OpenAI ---

// TranslateAnthropicResponseToOpenAI translates a complete Claude response into OpenAI format
func TranslateAnthropicResponseToOpenAI(data []byte) ([]byte, error) {
	var claude struct {
		ID           string `json:"id"`
		Type         string `json:"type"`
		Role         string `json:"role"`
		Model        string `json:"model"`
		StopReason   string `json:"stop_reason"`
		Content      []struct {
			Type  string          `json:"type"`
			Text  string          `json:"text,omitempty"`
			ID    string          `json:"id,omitempty"`
			Name  string          `json:"name,omitempty"`
			Input json.RawMessage `json:"input,omitempty"`
		} `json:"content"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(data, &claude); err != nil {
		return nil, err
	}

	// Extract text and tool_calls
	var textParts []string
	var toolCalls []map[string]any

	for _, block := range claude.Content {
		switch block.Type {
		case "text":
			textParts = append(textParts, block.Text)
		case "tool_use":
			args, _ := json.Marshal(block.Input)
			toolCalls = append(toolCalls, map[string]any{
				"id":   block.ID,
				"type": "function",
				"function": map[string]any{
					"name":      block.Name,
					"arguments": string(args),
				},
			})
		}
	}

	content := strings.Join(textParts, "")
	finishReason := "stop"
	if claude.StopReason == "tool_use" {
		finishReason = "tool_calls"
	}

	message := map[string]any{
		"role":    "assistant",
		"content": content,
	}
	if len(toolCalls) > 0 {
		message["tool_calls"] = toolCalls
	}

	openAI := map[string]any{
		"id":      claude.ID,
		"object":  "chat.completion",
		"created": time.Now().Unix(),
		"model":   claude.Model,
		"choices": []map[string]any{{
			"index":         0,
			"message":       message,
			"finish_reason": finishReason,
		}},
		"usage": map[string]any{
			"prompt_tokens":     claude.Usage.InputTokens,
			"completion_tokens": claude.Usage.OutputTokens,
			"total_tokens":      claude.Usage.InputTokens + claude.Usage.OutputTokens,
		},
	}

	return json.Marshal(openAI)
}

// --- Helpers ---

func extractTextFromContent(raw json.RawMessage) string {
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}

	var parts []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if json.Unmarshal(raw, &parts) == nil {
		var texts []string
		for _, p := range parts {
			if p.Type == "text" {
				texts = append(texts, p.Text)
			}
		}
		return strings.Join(texts, "\n")
	}

	return string(raw)
}

func convertContent(raw json.RawMessage) any {
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	// If it's an array, it may contain images etc. — pass through
	var arr []any
	if json.Unmarshal(raw, &arr) == nil {
		return arr
	}
	return string(raw)
}

func convertAssistantWithToolCalls(content, toolCalls json.RawMessage) []map[string]any {
	var blocks []map[string]any

	// Add text block if present
	text := extractTextFromContent(content)
	if text != "" {
		blocks = append(blocks, map[string]any{
			"type": "text",
			"text": text,
		})
	}

	// Convert tool_calls into tool_use blocks
	var calls []struct {
		ID       string `json:"id"`
		Function struct {
			Name      string `json:"name"`
			Arguments string `json:"arguments"`
		} `json:"function"`
	}
	if json.Unmarshal(toolCalls, &calls) == nil {
		for _, tc := range calls {
			var input any
			json.Unmarshal([]byte(tc.Function.Arguments), &input)
			if input == nil {
				input = map[string]any{}
			}
			blocks = append(blocks, map[string]any{
				"type":  "tool_use",
				"id":    tc.ID,
				"name":  tc.Function.Name,
				"input": input,
			})
		}
	}

	return blocks
}

func convertToolsToAnthropic(raw json.RawMessage) []map[string]any {
	var openAITools []struct {
		Type     string `json:"type"`
		Function struct {
			Name        string          `json:"name"`
			Description string          `json:"description"`
			Parameters  json.RawMessage `json:"parameters"`
		} `json:"function"`
	}
	if json.Unmarshal(raw, &openAITools) != nil {
		return nil
	}

	var claudeTools []map[string]any
	for _, t := range openAITools {
		tool := map[string]any{
			"name":         t.Function.Name,
			"description":  t.Function.Description,
			"input_schema": json.RawMessage(t.Function.Parameters),
		}
		claudeTools = append(claudeTools, tool)
	}
	return claudeTools
}

func convertToolChoiceToAnthropic(raw json.RawMessage) any {
	var s string
	if json.Unmarshal(raw, &s) == nil {
		switch s {
		case "auto":
			return map[string]string{"type": "auto"}
		case "none":
			return map[string]string{"type": "auto"} // Claude doesn't have "none", use "auto"
		case "required":
			return map[string]string{"type": "any"}
		}
	}

	// OpenAI object format: {"type":"function","function":{"name":"..."}}
	var obj struct {
		Type     string `json:"type"`
		Function struct {
			Name string `json:"name"`
		} `json:"function"`
	}
	if json.Unmarshal(raw, &obj) == nil && obj.Function.Name != "" {
		return map[string]string{
			"type": "tool",
			"name": obj.Function.Name,
		}
	}

	return map[string]string{"type": "auto"}
}

func mapModelToAnthropic(model string) string {
	lower := strings.ToLower(model)

	// If it's already full anthropic format
	if strings.HasPrefix(lower, "claude-") {
		return model
	}

	// Map short names
	switch {
	case strings.Contains(lower, "opus"):
		return "claude-opus-4-6-20250610"
	case strings.Contains(lower, "haiku"):
		return "claude-haiku-4-5-20251001"
	default:
		return "claude-sonnet-4-6-20250514"
	}
}
