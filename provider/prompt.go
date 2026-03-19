package provider

import (
	"encoding/json"
	"fmt"
	"strings"
)

// chatRequest is the OpenAI structure from the RawBody
type chatRequest struct {
	Model    string    `json:"model"`
	Messages []message `json:"messages"`
	Stream   bool      `json:"stream"`
}

type message struct {
	Role    string `json:"role"`
	Content any    `json:"content"` // string or []ContentPart
}

// ExtractPrompt extracts system prompt and user prompt from an OpenAI RawBody
func ExtractPrompt(rawBody json.RawMessage) (systemPrompt, userPrompt string) {
	var req chatRequest
	if err := json.Unmarshal(rawBody, &req); err != nil {
		return "", string(rawBody)
	}
	if len(req.Messages) > 0 {
		return buildPrompt(req.Messages)
	}

	// Fallback: some clients may not send OpenAI "messages" and instead use
	// keys like "prompt" or "input".
	var root map[string]any
	if err := json.Unmarshal(rawBody, &root); err != nil {
		return "", string(rawBody)
	}

	// Common simple shapes.
	if v, ok := root["prompt"].(string); ok && strings.TrimSpace(v) != "" {
		return "", v
	}
	if v, ok := root["input"].(string); ok && strings.TrimSpace(v) != "" {
		return "", v
	}
	if v, ok := root["text"].(string); ok && strings.TrimSpace(v) != "" {
		return "", v
	}

	// Attempt to interpret input as an array of chat-like items.
	if arr, ok := root["input"].([]any); ok && len(arr) > 0 {
		var pseudo []message
		for _, el := range arr {
			switch t := el.(type) {
			case string:
				pseudo = append(pseudo, message{Role: "user", Content: t})
			case map[string]any:
				role, _ := t["role"].(string)
				if role == "" {
					role = "user"
				}
				content := t["content"]
				if content == nil {
					if x, ok := t["text"].(string); ok {
						content = x
					}
				}
				pseudo = append(pseudo, message{Role: role, Content: content})
			}
		}
		if len(pseudo) > 0 {
			return buildPrompt(pseudo)
		}
	}

	return "", ""
}

// buildPrompt converts OpenAI messages into prompt + system prompt
func buildPrompt(messages []message) (systemPrompt string, userPrompt string) {
	var system strings.Builder
	var conversation strings.Builder

	for _, msg := range messages {
		content := extractContent(msg.Content)
		switch msg.Role {
		case "system":
			if system.Len() > 0 {
				system.WriteString("\n\n")
			}
			system.WriteString(content)
		case "user":
			conversation.WriteString(content)
			conversation.WriteString("\n")
		case "assistant":
			conversation.WriteString("[Previous response: ")
			conversation.WriteString(content)
			conversation.WriteString("]\n")
		default:
			// Some clients use additional roles (e.g. "developer").
			// Treat unknown roles as part of the user instruction stream.
			conversation.WriteString(content)
			conversation.WriteString("\n")
		}
	}

	return system.String(), strings.TrimSpace(conversation.String())
}

// extractContent handles OpenAI chat message content.
// It supports either:
// - a plain string
// - an array of parts (e.g. [{type:"text", text:"..."}])
func extractContent(content any) string {
	switch v := content.(type) {
	case string:
		return v
	case []any:
		var parts []string
		for _, item := range v {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}

			// Most common: { "type": "text", "text": "..." }
			if text, ok := m["text"].(string); ok && text != "" {
				parts = append(parts, text)
				continue
			}

			// Some clients may use other keys.
			if text, ok := m["value"].(string); ok && text != "" {
				parts = append(parts, text)
				continue
			}
		}

		// Avoid returning an empty string when the content is an array we don't fully recognize.
		if len(parts) > 0 {
			return strings.Join(parts, "\n")
		}
		return fmt.Sprintf("%v", content)
	}
	return fmt.Sprintf("%v", content)
}
