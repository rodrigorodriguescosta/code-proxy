package provider

import (
	"fmt"
	"strings"
	"sync"
)

// Registry manages registered providers and resolves model → provider
type Registry struct {
	mu        sync.RWMutex
	providers map[string]Provider // providerType -> Provider instance
	defaults  string             // default provider type
}

// NewRegistry creates an empty registry
func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[string]Provider),
	}
}

// Register adds a provider to the registry
func (r *Registry) Register(providerType string, p Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[providerType] = p
	// First registered becomes default
	if r.defaults == "" {
		r.defaults = providerType
	}
}

// SetDefault sets the default provider
func (r *Registry) SetDefault(providerType string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.defaults = providerType
}

// Get returns a provider by type
func (r *Registry) Get(providerType string) (Provider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.providers[providerType]
	return p, ok
}

// ResolveProvider resolves a model ID to the correct provider and a cleaned model ID
func (r *Registry) ResolveProvider(model string) (Provider, string, string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	providerType, cleanModel := parseModelPrefix(model)

	if providerType != "" {
		if p, ok := r.providers[providerType]; ok {
			return p, providerType, cleanModel, nil
		}
		return nil, "", "", fmt.Errorf("provider %q not found for model %q", providerType, model)
	}

	// No prefix: uses default provider
	if r.defaults != "" {
		if p, ok := r.providers[r.defaults]; ok {
			return p, r.defaults, cleanModel, nil
		}
	}

	// Fallback: first registered provider
	for pt, p := range r.providers {
		return p, pt, cleanModel, nil
	}

	return nil, "", "", fmt.Errorf("no providers registered")
}

// AllModels returns all models from all providers
func (r *Registry) AllModels() []Model {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var models []Model
	for _, p := range r.providers {
		models = append(models, p.Models()...)
	}
	return models
}

// ModelsForProvider returns models for a specific provider
func (r *Registry) ModelsForProvider(providerType string) []Model {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if p, ok := r.providers[providerType]; ok {
		return p.Models()
	}
	return nil
}

// ProviderStatus returns status info for all registered providers
type ProviderStatusInfo struct {
	Type      string `json:"type"`
	Name      string `json:"name"`
	Category  string `json:"category"`
	Available bool   `json:"available"`
}

func (r *Registry) ProviderStatuses() []ProviderStatusInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var statuses []ProviderStatusInfo
	for pt, p := range r.providers {
		statuses = append(statuses, ProviderStatusInfo{
			Type:      pt,
			Name:      p.Name(),
			Category:  p.Category(),
			Available: p.IsAvailable(),
		})
	}
	return statuses
}

// ListProviders returns the registered provider types
func (r *Registry) ListProviders() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	types := make([]string, 0, len(r.providers))
	for pt := range r.providers {
		types = append(types, pt)
	}
	return types
}

// parseModelPrefix extracts the provider type from the model prefix
// Ex: "cc/claude-opus-4-6" -> ("claude-cli", "claude-opus-4-6")
// Ex: "openai/gpt-4o" -> ("openai-api", "gpt-4o")
// Ex: "sonnet" -> ("", "sonnet")
func parseModelPrefix(model string) (providerType, cleanModel string) {
	lower := strings.ToLower(model)

	// Explicit prefixes
	prefixes := map[string]string{
		// CLI prefixes (local binary execution)
		"cli-cc/":    "claude-cli",
		"cli-codex/": "codex-cli",
		"cli-gc/":    "gemini-cli",
		// OAuth/API prefixes
		"cc/":         "claude-cli",
		"cx/":         "codex-cli",
		"codex/":      "codex-cli",
		"gc/":         "gemini-cli",
		"gemini-cli/": "gemini-cli",
		"anthropic/":  "anthropic-api",
		"openai/":     "openai-api",
		"gemini/":     "gemini-api",
		"deepseek/":   "generic-openai",
		"groq/":       "generic-openai",
		"together/":   "generic-openai",
		"ollama/":     "generic-openai",
	}

	for prefix, pt := range prefixes {
		if strings.HasPrefix(lower, prefix) {
			return pt, model[len(prefix):]
		}
	}

	// Detection by model name (no prefix)
	if strings.HasPrefix(lower, "gpt-") || strings.HasPrefix(lower, "o1") || strings.HasPrefix(lower, "o3") || strings.HasPrefix(lower, "o4") {
		return "openai-api", model
	}

	return "", model
}
