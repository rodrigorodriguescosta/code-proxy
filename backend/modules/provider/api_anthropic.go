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
)

const anthropicBaseURL = "https://api.anthropic.com"
const anthropicVersion = "2023-06-01"

// AnthropicAPI is the provider for the Anthropic Messages API with format translation
type AnthropicAPI struct{}

// NewAnthropicAPI creates an Anthropic API provider
func NewAnthropicAPI() *AnthropicAPI {
	return &AnthropicAPI{}
}

func (p *AnthropicAPI) Name() string      { return "anthropic-api" }
func (p *AnthropicAPI) Category() string   { return "api" }
func (p *AnthropicAPI) IsAvailable() bool  { return true }

func (p *AnthropicAPI) Models() []Model {
	return []Model{
		{ID: "anthropic/claude-opus-4-6", Name: "Claude Opus 4.6", OwnedBy: "anthropic"},
		{ID: "anthropic/claude-sonnet-4-6", Name: "Claude Sonnet 4.6", OwnedBy: "anthropic"},
		{ID: "anthropic/claude-haiku-4-5", Name: "Claude Haiku 4.5", OwnedBy: "anthropic"},
	}
}

func (p *AnthropicAPI) Execute(ctx context.Context, req *Request) (<-chan Event, error) {
	if req.Account == nil {
		return nil, fmt.Errorf("Anthropic API requires a configured account")
	}

	// Translate request OpenAI -> Claude Messages API
	claudeBody, _, err := TranslateOpenAIToAnthropic(req.RawBody, req.Model)
	if err != nil {
		return nil, fmt.Errorf("translate to anthropic: %w", err)
	}

	// Build HTTP request
	url := anthropicBaseURL + "/v1/messages"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(claudeBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("anthropic-version", anthropicVersion)

	// Auth: OAuth token (Bearer) or API key (x-api-key)
	if req.Account.AuthMode == "oauth" && req.Account.AccessToken != "" {
		httpReq.Header.Set("Authorization", "Bearer "+req.Account.AccessToken)
	} else {
		httpReq.Header.Set("x-api-key", req.Account.AuthToken())
	}

	log.Printf("[ANTHROPIC] %s → %s (stream=%v, %d bytes)", req.Model, url, req.Stream, len(claudeBody))

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("upstream request: %w", err)
	}

	// Check for HTTP error
	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		errBody, _ := io.ReadAll(resp.Body)
		return nil, &UpstreamError{
			StatusCode: resp.StatusCode,
			Body:       string(errBody),
		}
	}

	events := make(chan Event, 128)

	if req.Stream {
		go p.streamResponse(resp, events)
	} else {
		go p.nonStreamResponse(resp, events)
	}

	return events, nil
}

// streamResponse reads Claude SSE and translates to OpenAI format
func (p *AnthropicAPI) streamResponse(resp *http.Response, events chan<- Event) {
	defer close(events)
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)

	var eventType string

	for scanner.Scan() {
		line := scanner.Text()

		if line == "" {
			continue
		}

		// Claude SSE uses "event: xxx" followed by "data: {json}"
		if strings.HasPrefix(line, "event: ") {
			eventType = strings.TrimPrefix(line, "event: ")
			continue
		}

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		// Inject type in JSON if event type is available
		if eventType != "" {
			var obj map[string]json.RawMessage
			if json.Unmarshal([]byte(data), &obj) == nil {
				if _, hasType := obj["type"]; !hasType {
					typeJSON, _ := json.Marshal(eventType)
					obj["type"] = typeJSON
					newData, _ := json.Marshal(obj)
					data = string(newData)
				}
			}
			eventType = ""
		}

		// Translate to OpenAI
		translated, err := TranslateAnthropicStreamToOpenAI([]byte(data))
		if err != nil {
			log.Printf("[ANTHROPIC] Translate error: %v", err)
			continue
		}
		if translated != nil {
			events <- Event{Type: "sse_chunk", JSON: string(translated)}
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("[ANTHROPIC] Stream scan error: %v", err)
		events <- Event{Type: "error", Text: err.Error()}
	}

	events <- Event{Type: "done"}
}

// nonStreamResponse reads the full response and translates it
func (p *AnthropicAPI) nonStreamResponse(resp *http.Response, events chan<- Event) {
	defer close(events)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		events <- Event{Type: "error", Text: err.Error()}
		return
	}

	translated, err := TranslateAnthropicResponseToOpenAI(body)
	if err != nil {
		log.Printf("[ANTHROPIC] Translate response error: %v", err)
		// Fallback: emit raw text
		events <- Event{Type: "text", Text: string(body)}
		events <- Event{Type: "done"}
		return
	}

	// Extract content to emit as text event
	var chatResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if json.Unmarshal(translated, &chatResp) == nil && len(chatResp.Choices) > 0 {
		events <- Event{Type: "text", Text: chatResp.Choices[0].Message.Content}
	} else {
		events <- Event{Type: "sse_chunk", JSON: string(translated)}
	}

	events <- Event{Type: "done"}
}
