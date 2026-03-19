package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// GeminiAPI is the provider for the Google Gemini API with format translation
type GeminiAPI struct{}

// NewGeminiAPI creates a Gemini API provider
func NewGeminiAPI() *GeminiAPI {
	return &GeminiAPI{}
}

func (p *GeminiAPI) Name() string      { return "gemini-api" }
func (p *GeminiAPI) Category() string   { return "api" }
func (p *GeminiAPI) IsAvailable() bool  { return true }

func (p *GeminiAPI) Models() []Model {
	return []Model{
		{ID: "gemini/gemini-2.5-pro", Name: "Gemini 2.5 Pro", OwnedBy: "google"},
		{ID: "gemini/gemini-2.5-flash", Name: "Gemini 2.5 Flash", OwnedBy: "google"},
		{ID: "gemini/gemini-2.5-flash-lite", Name: "Gemini 2.5 Flash Lite", OwnedBy: "google"},
		{ID: "gemini/gemini-2.0-flash", Name: "Gemini 2.0 Flash", OwnedBy: "google"},
	}
}

func (p *GeminiAPI) Execute(ctx context.Context, req *Request) (<-chan Event, error) {
	if req.Account == nil {
		return nil, fmt.Errorf("Gemini API requires a configured account")
	}

	// Translate request OpenAI -> Gemini generateContent
	geminiBody, model, err := translateOpenAIToGemini(req.RawBody, req.Model)
	if err != nil {
		return nil, fmt.Errorf("translate to gemini: %w", err)
	}

	// Build URL with model name
	action := "generateContent"
	if req.Stream {
		action = "streamGenerateContent?alt=sse"
	}

	baseURL := "https://generativelanguage.googleapis.com/v1beta"
	projectID := ""
	if req.Account.Metadata != nil {
		projectID = req.Account.Metadata["project_id"]
	}

	var url string
	if projectID != "" {
		// Cloud AI Companion (Antigravity/Gemini CLI mode)
		baseURL = fmt.Sprintf("https://us-central1-aiplatform.googleapis.com/v1/projects/%s/locations/us-central1/publishers/google/models", projectID)
		url = fmt.Sprintf("%s/%s:%s", baseURL, model, action)
	} else {
		url = fmt.Sprintf("%s/models/%s:%s", baseURL, model, action)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(geminiBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if req.Account.AccessToken != "" {
		httpReq.Header.Set("Authorization", "Bearer "+req.Account.AccessToken)
	} else if req.Account.APIKey != "" {
		// API key via query param
		q := httpReq.URL.Query()
		q.Set("key", req.Account.APIKey)
		httpReq.URL.RawQuery = q.Encode()
	}

	log.Printf("[GEMINI] %s → %s (stream=%v)", req.Model, url, req.Stream)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("upstream request: %w", err)
	}

	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		errBody, _ := io.ReadAll(resp.Body)
		return nil, &UpstreamError{StatusCode: resp.StatusCode, Body: string(errBody)}
	}

	events := make(chan Event, 128)

	if req.Stream {
		go p.streamResponse(resp, events)
	} else {
		go p.nonStreamResponse(resp, events)
	}

	return events, nil
}

func (p *GeminiAPI) streamResponse(resp *http.Response, events chan<- Event) {
	defer close(events)
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)

	sentRole := false

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")

		chunk, err := translateGeminiStreamToOpenAI([]byte(data), !sentRole)
		if err != nil || chunk == nil {
			continue
		}
		sentRole = true
		events <- Event{Type: "sse_chunk", JSON: string(chunk)}
	}

	events <- Event{Type: "done"}
}

func (p *GeminiAPI) nonStreamResponse(resp *http.Response, events chan<- Event) {
	defer close(events)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		events <- Event{Type: "error", Text: err.Error()}
		return
	}

	translated, err := translateGeminiResponseToOpenAI(body)
	if err != nil {
		events <- Event{Type: "text", Text: string(body)}
		events <- Event{Type: "done"}
		return
	}

	var chatResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if json.Unmarshal(translated, &chatResp) == nil && len(chatResp.Choices) > 0 {
		events <- Event{Type: "text", Text: chatResp.Choices[0].Message.Content}
	}
	events <- Event{Type: "done"}
}

// --- Translation OpenAI -> Gemini ---

func translateOpenAIToGemini(body []byte, model string) ([]byte, string, error) {
	var openAI struct {
		Model       string            `json:"model"`
		Messages    []json.RawMessage `json:"messages"`
		Stream      bool              `json:"stream"`
		MaxTokens   int               `json:"max_tokens,omitempty"`
		Temperature *float64          `json:"temperature,omitempty"`
		Tools       json.RawMessage   `json:"tools,omitempty"`
	}
	if err := json.Unmarshal(body, &openAI); err != nil {
		return nil, "", err
	}

	geminiModel := mapModelToGemini(model)

	gemini := map[string]any{}

	// Convert messages to contents
	var systemInstruction string
	var contents []map[string]any

	for _, rawMsg := range openAI.Messages {
		var msg struct {
			Role    string          `json:"role"`
			Content json.RawMessage `json:"content"`
		}
		if json.Unmarshal(rawMsg, &msg) != nil {
			continue
		}

		text := extractTextFromContent(msg.Content)

		switch msg.Role {
		case "system":
			systemInstruction = text
		case "user":
			contents = append(contents, map[string]any{
				"role":  "user",
				"parts": []map[string]any{{"text": text}},
			})
		case "assistant":
			contents = append(contents, map[string]any{
				"role":  "model",
				"parts": []map[string]any{{"text": text}},
			})
		}
	}

	gemini["contents"] = contents

	if systemInstruction != "" {
		gemini["systemInstruction"] = map[string]any{
			"parts": []map[string]any{{"text": systemInstruction}},
		}
	}

	// Generation config
	genConfig := map[string]any{}
	if openAI.MaxTokens > 0 {
		genConfig["maxOutputTokens"] = openAI.MaxTokens
	}
	if openAI.Temperature != nil {
		genConfig["temperature"] = *openAI.Temperature
	}
	if len(genConfig) > 0 {
		gemini["generationConfig"] = genConfig
	}

	// Convert tools
	if openAI.Tools != nil {
		geminiTools := convertToolsToGemini(openAI.Tools)
		if geminiTools != nil {
			gemini["tools"] = geminiTools
		}
	}

	result, err := json.Marshal(gemini)
	return result, geminiModel, err
}

func convertToolsToGemini(raw json.RawMessage) []map[string]any {
	var openAITools []struct {
		Function struct {
			Name        string          `json:"name"`
			Description string          `json:"description"`
			Parameters  json.RawMessage `json:"parameters"`
		} `json:"function"`
	}
	if json.Unmarshal(raw, &openAITools) != nil {
		return nil
	}

	var declarations []map[string]any
	for _, t := range openAITools {
		decl := map[string]any{
			"name":        t.Function.Name,
			"description": t.Function.Description,
		}
		if t.Function.Parameters != nil {
			decl["parameters"] = json.RawMessage(t.Function.Parameters)
		}
		declarations = append(declarations, decl)
	}

	return []map[string]any{{
		"functionDeclarations": declarations,
	}}
}

// --- Translation Gemini -> OpenAI ---

func translateGeminiStreamToOpenAI(data []byte, includeRole bool) ([]byte, error) {
	var gemini struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text         string          `json:"text,omitempty"`
					FunctionCall json.RawMessage `json:"functionCall,omitempty"`
				} `json:"parts"`
				Role string `json:"role"`
			} `json:"content"`
			FinishReason string `json:"finishReason"`
		} `json:"candidates"`
	}

	if json.Unmarshal(data, &gemini) != nil || len(gemini.Candidates) == 0 {
		return nil, nil
	}

	cand := gemini.Candidates[0]
	chatID := fmt.Sprintf("chatcmpl-%d", time.Now().UnixNano())
	created := time.Now().Unix()

	// Extract text
	var text string
	for _, part := range cand.Content.Parts {
		if part.Text != "" {
			text += part.Text
		}
	}

	delta := map[string]any{}
	if includeRole {
		delta["role"] = "assistant"
	}
	if text != "" {
		delta["content"] = text
	}

	chunk := map[string]any{
		"id":      chatID,
		"object":  "chat.completion.chunk",
		"created": created,
		"choices": []map[string]any{{
			"index": 0,
			"delta": delta,
		}},
	}

	if cand.FinishReason != "" && cand.FinishReason != "STOP" {
		chunk["choices"] = []map[string]any{{
			"index":         0,
			"delta":         map[string]any{},
			"finish_reason": "stop",
		}}
	}

	return json.Marshal(chunk)
}

func translateGeminiResponseToOpenAI(data []byte) ([]byte, error) {
	var gemini struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text,omitempty"`
				} `json:"parts"`
			} `json:"content"`
			FinishReason string `json:"finishReason"`
		} `json:"candidates"`
		UsageMetadata struct {
			PromptTokenCount     int `json:"promptTokenCount"`
			CandidatesTokenCount int `json:"candidatesTokenCount"`
			TotalTokenCount      int `json:"totalTokenCount"`
		} `json:"usageMetadata"`
	}

	if err := json.Unmarshal(data, &gemini); err != nil {
		return nil, err
	}

	var text string
	if len(gemini.Candidates) > 0 {
		for _, part := range gemini.Candidates[0].Content.Parts {
			text += part.Text
		}
	}

	openAI := map[string]any{
		"id":      fmt.Sprintf("chatcmpl-%d", time.Now().UnixNano()),
		"object":  "chat.completion",
		"created": time.Now().Unix(),
		"choices": []map[string]any{{
			"index": 0,
			"message": map[string]any{
				"role":    "assistant",
				"content": text,
			},
			"finish_reason": "stop",
		}},
		"usage": map[string]any{
			"prompt_tokens":     gemini.UsageMetadata.PromptTokenCount,
			"completion_tokens": gemini.UsageMetadata.CandidatesTokenCount,
			"total_tokens":      gemini.UsageMetadata.TotalTokenCount,
		},
	}

	return json.Marshal(openAI)
}

func mapModelToGemini(model string) string {
	lower := strings.ToLower(model)
	switch {
	case strings.Contains(lower, "2.5-pro") || strings.Contains(lower, "pro"):
		return "gemini-2.5-pro-preview-06-05"
	case strings.Contains(lower, "2.5-flash") || strings.Contains(lower, "flash"):
		return "gemini-2.5-flash-preview-05-20"
	case strings.Contains(lower, "2.0"):
		return "gemini-2.0-flash"
	default:
		return "gemini-2.5-flash-preview-05-20"
	}
}
