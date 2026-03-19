package provider

import (
	"context"
	"fmt"
)

// OpenAIAPI is the provider for the OpenAI API (direct pass-through)
type OpenAIAPI struct{}

// NewOpenAIAPI creates an OpenAI API provider
func NewOpenAIAPI() *OpenAIAPI {
	return &OpenAIAPI{}
}

func (p *OpenAIAPI) Name() string      { return "openai-api" }
func (p *OpenAIAPI) Category() string  { return "api" }
func (p *OpenAIAPI) IsAvailable() bool { return true }

func (p *OpenAIAPI) Models() []Model {
	return []Model{
		// Codex (GPT 5.x) — same cx/ prefix as CLI
		{ID: "cx/gpt-5.3-codex", Name: "GPT 5.3 Codex", OwnedBy: "openai"},
		{ID: "cx/gpt-5.3-codex-xhigh", Name: "GPT 5.3 Codex (xHigh)", OwnedBy: "openai"},
		{ID: "cx/gpt-5.3-codex-high", Name: "GPT 5.3 Codex (High)", OwnedBy: "openai"},
		{ID: "cx/gpt-5.3-codex-low", Name: "GPT 5.3 Codex (Low)", OwnedBy: "openai"},
		{ID: "cx/gpt-5.3-codex-none", Name: "GPT 5.3 Codex (None)", OwnedBy: "openai"},
		{ID: "cx/gpt-5.3-codex-spark", Name: "GPT 5.3 Codex Spark", OwnedBy: "openai"},
		{ID: "codex/5.4", Name: "GPT 5.4 Codex", OwnedBy: "openai"},
		{ID: "codex/5.4-xhigh", Name: "GPT 5.4 Codex (xHigh)", OwnedBy: "openai"},
		{ID: "codex/5.4-high", Name: "GPT 5.4 Codex (High)", OwnedBy: "openai"},
		{ID: "codex/5.4-low", Name: "GPT 5.4 Codex (Low)", OwnedBy: "openai"},
		{ID: "codex/5.4-none", Name: "GPT 5.4 Codex (None)", OwnedBy: "openai"},
		{ID: "codex/5.4-spark", Name: "GPT 5.4 Codex Spark", OwnedBy: "openai"},
		{ID: "cx/gpt-5.2-codex", Name: "GPT 5.2 Codex", OwnedBy: "openai"},
		{ID: "cx/gpt-5.2", Name: "GPT 5.2", OwnedBy: "openai"},
		{ID: "cx/gpt-5.1-codex", Name: "GPT 5.1 Codex", OwnedBy: "openai"},
		{ID: "cx/gpt-5.1-codex-max", Name: "GPT 5.1 Codex Max", OwnedBy: "openai"},
		{ID: "cx/gpt-5.1-codex-mini", Name: "GPT 5.1 Codex Mini", OwnedBy: "openai"},
		{ID: "cx/gpt-5.1-codex-mini-high", Name: "GPT 5.1 Codex Mini (High)", OwnedBy: "openai"},
		{ID: "cx/gpt-5.1", Name: "GPT 5.1", OwnedBy: "openai"},
		{ID: "cx/gpt-5-codex", Name: "GPT 5 Codex", OwnedBy: "openai"},
		{ID: "cx/gpt-5-codex-mini", Name: "GPT 5 Codex Mini", OwnedBy: "openai"},
		// GPT 4.x
		{ID: "openai/gpt-4o", Name: "GPT-4o", OwnedBy: "openai"},
		{ID: "openai/gpt-4o-mini", Name: "GPT-4o Mini", OwnedBy: "openai"},
		{ID: "openai/gpt-4.1", Name: "GPT-4.1", OwnedBy: "openai"},
		{ID: "openai/gpt-4.1-mini", Name: "GPT-4.1 Mini", OwnedBy: "openai"},
		{ID: "openai/gpt-4.1-nano", Name: "GPT-4.1 Nano", OwnedBy: "openai"},
		// Reasoning
		{ID: "openai/o1", Name: "o1", OwnedBy: "openai"},
		{ID: "openai/o1-mini", Name: "o1 Mini", OwnedBy: "openai"},
		{ID: "openai/o3", Name: "o3", OwnedBy: "openai"},
		{ID: "openai/o3-mini", Name: "o3 Mini", OwnedBy: "openai"},
		{ID: "openai/o4-mini", Name: "o4 Mini", OwnedBy: "openai"},
	}
}

func (p *OpenAIAPI) Execute(ctx context.Context, req *Request) (<-chan Event, error) {
	if req.Account == nil {
		return nil, fmt.Errorf("OpenAI API requires a configured account (API key or OAuth)")
	}

	baseURL := "https://api.openai.com"
	authHeader := "Bearer " + req.Account.AuthToken()

	// Direct pass-through — format is already OpenAI
	return proxyExecute(ctx, req, baseURL, authHeader, nil, nil, nil)
}
