package auth

import "fmt"

// OAuthConfig defines OAuth parameters specific to each provider
type OAuthConfig struct {
	Provider     string
	ClientID     string
	AuthURL      string
	TokenURL     string
	Scopes       []string
	CallbackPort int
	CallbackPath string            // Callback path (default: "/auth/callback")
	ContentType  string            // "application/json" or "application/x-www-form-urlencoded"
	UsePKCE      bool
	ExtraParams  map[string]string // Additional authorization URL parameters
}

// RedirectURI returns the full redirect_uri for this provider
func (c OAuthConfig) RedirectURI() string {
	path := c.CallbackPath
	if path == "" {
		path = "/auth/callback"
	}
	return fmt.Sprintf("http://localhost:%d%s", c.CallbackPort, path)
}

// Predefined OAuth configs by provider
var Configs = map[string]OAuthConfig{
	"claude": {
		Provider:     "claude",
		ClientID:     "9d1c250a-e61b-44d9-88ed-5944d1962f5e",
		AuthURL:      "https://console.anthropic.com/v1/oauth/authorize",
		TokenURL:     "https://console.anthropic.com/v1/oauth/token",
		Scopes:       []string{"org:create_api_key", "user:profile", "user:inference"},
		CallbackPort: 54545,
		ContentType:  "application/json",
		UsePKCE:      true,
	},
	"codex": {
		Provider:     "codex",
		ClientID:     "app_EMoamEEZ73f0CkXaXp7hrann",
		AuthURL:      "https://auth.openai.com/oauth/authorize",
		TokenURL:     "https://auth.openai.com/oauth/token",
		Scopes:       []string{"openid", "email", "profile", "offline_access"},
		CallbackPort: 1455,
		ContentType:  "application/x-www-form-urlencoded",
		UsePKCE:      true,
		ExtraParams: map[string]string{
			"codex_cli_simplified_flow":  "true",
			"id_token_add_organizations": "true",
			"originator":                 "codex_cli_rs",
			"prompt":                     "login",
		},
	},
	"gemini": {
		Provider:     "gemini",
		ClientID:     "681255809395-oo8ft2oprdrnp9e3aqf6av3hmdib135j.apps.googleusercontent.com",
		AuthURL:      "https://accounts.google.com/o/oauth2/v2/auth",
		TokenURL:     "https://oauth2.googleapis.com/token",
		Scopes:       []string{"https://www.googleapis.com/auth/cloud-platform", "https://www.googleapis.com/auth/userinfo.email"},
		CallbackPort: 8085,
		ContentType:  "application/x-www-form-urlencoded",
		UsePKCE:      true,
		ExtraParams: map[string]string{
			"access_type": "offline",
			"prompt":      "consent",
		},
	},
	"antigravity": {
		Provider:     "antigravity",
		ClientID:     "681255809395-oo8ft2oprdrnp9e3aqf6av3hmdib135j.apps.googleusercontent.com",
		AuthURL:      "https://accounts.google.com/o/oauth2/v2/auth",
		TokenURL:     "https://oauth2.googleapis.com/token",
		Scopes:       []string{"https://www.googleapis.com/auth/cloud-platform", "https://www.googleapis.com/auth/userinfo.email"},
		CallbackPort: 8086,
		ContentType:  "application/x-www-form-urlencoded",
		UsePKCE:      true,
		ExtraParams: map[string]string{
			"access_type": "offline",
			"prompt":      "consent",
		},
	},
	"github": {
		Provider:     "github",
		ClientID:     "Iv1.b507a08c87ecfe98", // GitHub Copilot CLI
		AuthURL:      "https://github.com/login/device/code",
		TokenURL:     "https://github.com/login/oauth/access_token",
		Scopes:       []string{"copilot"},
		CallbackPort: 0, // Device code flow — no callback
		ContentType:  "application/json",
		UsePKCE:      false,
	},
}

// GetConfig returns the OAuth config for a provider
func GetConfig(providerName string) (OAuthConfig, bool) {
	cfg, ok := Configs[providerName]
	return cfg, ok
}

// ListOAuthProviders returns a list of providers that support OAuth
func ListOAuthProviders() []string {
	providers := make([]string, 0, len(Configs))
	for k := range Configs {
		providers = append(providers, k)
	}
	return providers
}
