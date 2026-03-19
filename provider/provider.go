package provider

import (
	"context"
	"encoding/json"
	"os/exec"
	"time"
)

// Event represents a streaming event from a provider
type Event struct {
	Type string  // "text", "done", "error", "sse_chunk"
	Text string  // Text content delta
	JSON string  // SSE chunk JSON (for sse_chunk — already in OpenAI format)
	Cost float64 // Cost in USD
}

// Request is the unified input for all providers
type Request struct {
	RawBody json.RawMessage // Complete OpenAI request (original JSON from client)
	Model   string          // Resolved model (e.g. "sonnet", "gpt-4o")
	Effort  string          // low/medium/high/max
	Stream  bool
	Account *Account // Selected credentials (nil for providers without auth)
}

// Provider is the unified interface for CLI and API providers
type Provider interface {
	Name() string
	Models() []Model
	Execute(ctx context.Context, req *Request) (<-chan Event, error)
	// Category returns "cli" for CLI-based providers or "api" for API-based providers
	Category() string
	// IsAvailable checks if the provider is usable at runtime
	// CLI providers check if the binary exists; API providers always return true
	IsAvailable() bool
}

// Model represents an available model
type Model struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	OwnedBy string `json:"owned_by"`
}

// Account represents an authenticated credential for a provider
type Account struct {
	ID           string            `json:"id"`
	ProviderType string            `json:"provider_type"`
	Label        string            `json:"label"`
	AuthMode     string            `json:"auth_mode"` // oauth, apikey, none
	AccessToken  string            `json:"access_token,omitempty"`
	RefreshToken string            `json:"refresh_token,omitempty"`
	APIKey       string            `json:"api_key,omitempty"`
	ExpiresAt    time.Time         `json:"expires_at,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	IsActive     bool              `json:"is_active"`
	Priority     int               `json:"priority"`
}

// AuthToken returns the token or API key for authentication
func (a *Account) AuthToken() string {
	if a.AccessToken != "" {
		return a.AccessToken
	}
	return a.APIKey
}

// BaseURL returns the custom base URL (for generic providers)
func (a *Account) BaseURL() string {
	if a.Metadata != nil {
		return a.Metadata["base_url"]
	}
	return ""
}

// CLIBinaryAvailable checks if a CLI binary exists in PATH
func CLIBinaryAvailable(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
