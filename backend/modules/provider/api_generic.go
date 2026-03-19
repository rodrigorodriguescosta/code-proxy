package provider

import (
	"context"
	"fmt"
)

// GenericOpenAI is the provider for any OpenAI-compatible API
// (DeepSeek, Groq, Together, local Ollama, etc.)
// The base URL is configured via account metadata
type GenericOpenAI struct{}

// NewGenericOpenAI creates a generic OpenAI-compatible provider
func NewGenericOpenAI() *GenericOpenAI {
	return &GenericOpenAI{}
}

func (p *GenericOpenAI) Name() string      { return "generic-openai" }
func (p *GenericOpenAI) Category() string  { return "api" }
func (p *GenericOpenAI) IsAvailable() bool { return true }

func (p *GenericOpenAI) Models() []Model {
	return []Model{
		// DeepSeek
		{ID: "deepseek/deepseek-chat", Name: "DeepSeek Chat", OwnedBy: "deepseek"},
		{ID: "deepseek/deepseek-coder", Name: "DeepSeek Coder", OwnedBy: "deepseek"},
		{ID: "deepseek/deepseek-reasoner", Name: "DeepSeek Reasoner", OwnedBy: "deepseek"},
		// Groq
		{ID: "groq/llama-3.3-70b-versatile", Name: "Llama 3.3 70B (Groq)", OwnedBy: "groq"},
		{ID: "groq/llama-4-maverick-17b-128e-instruct", Name: "Llama 4 Maverick (Groq)", OwnedBy: "groq"},
		// Together
		{ID: "together/meta-llama/Llama-3.3-70B-Instruct-Turbo", Name: "Llama 3.3 70B (Together)", OwnedBy: "together"},
		{ID: "together/deepseek-ai/DeepSeek-R1", Name: "DeepSeek R1 (Together)", OwnedBy: "together"},
		// Ollama (local)
		{ID: "ollama/llama3.3", Name: "Llama 3.3 (Ollama)", OwnedBy: "ollama"},
	}
}

func (p *GenericOpenAI) Execute(ctx context.Context, req *Request) (<-chan Event, error) {
	if req.Account == nil {
		return nil, fmt.Errorf("Generic OpenAI provider requires a configured account")
	}

	// Base URL from account metadata or known defaults
	baseURL := req.Account.BaseURL()
	if baseURL == "" {
		baseURL = inferBaseURL(req.Account.ProviderType, req.Model)
	}
	if baseURL == "" {
		return nil, fmt.Errorf("base_url not configured for account %s", req.Account.Label)
	}

	authHeader := "Bearer " + req.Account.AuthToken()

	// Direct pass-through — OpenAI format
	return proxyExecute(ctx, req, baseURL, authHeader, nil, nil, nil)
}

// inferBaseURL tries to infer the base URL from the provider type or model
func inferBaseURL(providerType, model string) string {
	defaults := map[string]string{
		"deepseek":   "https://api.deepseek.com",
		"groq":       "https://api.groq.com/openai",
		"together":   "https://api.together.xyz",
		"fireworks":  "https://api.fireworks.ai/inference",
		"mistral":    "https://api.mistral.ai",
		"ollama":     "http://localhost:11434",
		"perplexity": "https://api.perplexity.ai",
	}

	// Try by provider type
	for key, url := range defaults {
		if providerType == key || providerType == key+"-api" {
			return url
		}
	}

	// Try by model prefix
	for key, url := range defaults {
		if len(model) > len(key) && model[:len(key)] == key {
			return url
		}
	}

	return ""
}
