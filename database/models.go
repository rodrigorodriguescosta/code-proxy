package database

import "time"

type ApiKey struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Key       string    `json:"key,omitempty"`
	KeyHash   string    `json:"-"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

type Provider struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Name      string    `json:"name"`
	Config    string    `json:"config,omitempty"`
	IsActive  bool      `json:"is_active"`
	Priority  int       `json:"priority"`
	CreatedAt time.Time `json:"created_at"`
}

type Settings struct {
	TunnelEnabled  bool   `json:"tunnel_enabled"`
	TunnelURL      string `json:"tunnel_url"`
	TunnelToken    string `json:"tunnel_token,omitempty"`
	DefaultModel   string `json:"default_model"`
	LogRetention   int    `json:"log_retention_days"`
	RequireApiKey  bool   `json:"require_api_key"`
	DashboardPassword string `json:"dashboard_password,omitempty"`
}

type RequestLog struct {
	ID            int64     `json:"id"`
	ApiKeyID      string    `json:"api_key_id,omitempty"`
	ApiKeyName    string    `json:"api_key_name,omitempty"`
	Provider      string    `json:"provider"`
	Model         string    `json:"model"`
	Effort        string    `json:"effort,omitempty"`
	AccountID     string    `json:"account_id,omitempty"`
	InputTokens   int       `json:"input_tokens"`
	OutputTokens  int       `json:"output_tokens"`
	EstimatedCost float64   `json:"estimated_cost"`
	DurationMs    int64     `json:"duration_ms"`
	CreatedAt     time.Time `json:"created_at"`
}
