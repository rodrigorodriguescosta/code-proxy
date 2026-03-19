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

// proxyExecute is the shared HTTP proxy logic for API providers
// It receives an OpenAI request, proxies it to the upstream, and returns events
func proxyExecute(ctx context.Context, req *Request, baseURL, authHeader string, extraHeaders map[string]string, translateReq RequestTranslator, translateResp ResponseTranslator) (<-chan Event, error) {
	// Translate request if needed
	body := []byte(req.RawBody)
	contentType := "application/json"
	var err error

	if translateReq != nil {
		body, contentType, err = translateReq(body, req.Model)
		if err != nil {
			return nil, fmt.Errorf("translate request: %w", err)
		}
	} else {
		// Ensure stream is set correctly in the body
		body = ensureStreamFlag(body, req.Stream)
	}

	url := baseURL + "/v1/chat/completions"

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", contentType)
	if authHeader != "" {
		httpReq.Header.Set("Authorization", authHeader)
	}
	for k, v := range extraHeaders {
		httpReq.Header.Set(k, v)
	}

	log.Printf("[PROXY] %s → %s (stream=%v, %d bytes)", req.Model, url, req.Stream, len(body))

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("upstream request: %w", err)
	}

	// Check HTTP error
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
		go streamProxy(resp, events, translateResp)
	} else {
		go nonStreamProxy(resp, events, translateResp)
	}

	return events, nil
}

// streamProxy reads SSE from the upstream and emits events
func streamProxy(resp *http.Response, events chan<- Event, translateResp ResponseTranslator) {
	defer close(events)
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)

	for scanner.Scan() {
		line := scanner.Text()

		if line == "" {
			continue
		}

		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		if data == "[DONE]" {
			events <- Event{Type: "done"}
			return
		}

		if translateResp != nil {
			// Translate upstream SSE into OpenAI format
			translated, err := translateResp([]byte(data))
			if err != nil {
				log.Printf("[PROXY] Translate error: %v", err)
				continue
			}
			if translated != nil {
				events <- Event{Type: "sse_chunk", JSON: string(translated)}
			}
		} else {
			// Pass-through: already in OpenAI format
			events <- Event{Type: "sse_chunk", JSON: data}
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("[PROXY] Stream scan error: %v", err)
		events <- Event{Type: "error", Text: err.Error()}
	}

	events <- Event{Type: "done"}
}

// nonStreamProxy reads the complete upstream response and emits it as an event
func nonStreamProxy(resp *http.Response, events chan<- Event, translateResp ResponseTranslator) {
	defer close(events)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		events <- Event{Type: "error", Text: err.Error()}
		return
	}

	if translateResp != nil {
		translated, err := translateResp(body)
		if err != nil {
			events <- Event{Type: "error", Text: err.Error()}
			return
		}
		body = translated
	}

	// Extract response content text to emit as a "text" event
	var chatResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if json.Unmarshal(body, &chatResp) == nil && len(chatResp.Choices) > 0 {
		events <- Event{Type: "text", Text: chatResp.Choices[0].Message.Content}
	} else {
		// If it couldn't parse, emit the full JSON as an sse_chunk
		events <- Event{Type: "sse_chunk", JSON: string(body)}
	}

	events <- Event{Type: "done"}
}

// ensureStreamFlag ensures the "stream" field is present in the JSON body
func ensureStreamFlag(body []byte, stream bool) []byte {
	var obj map[string]json.RawMessage
	if json.Unmarshal(body, &obj) != nil {
		return body
	}
	b, _ := json.Marshal(stream)
	obj["stream"] = b
	result, _ := json.Marshal(obj)
	return result
}

// --- Types ---

// RequestTranslator translates an OpenAI request into the upstream format
type RequestTranslator func(body []byte, model string) ([]byte, string, error) // (body, contentType, error)

// ResponseTranslator translates an upstream response/SSE chunk into OpenAI format
type ResponseTranslator func(data []byte) ([]byte, error)

// UpstreamError represents an upstream HTTP error
type UpstreamError struct {
	StatusCode int
	Body       string
}

func (e *UpstreamError) Error() string {
	return fmt.Sprintf("upstream HTTP %d: %s", e.StatusCode, truncateStr(e.Body, 200))
}

func truncateStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
