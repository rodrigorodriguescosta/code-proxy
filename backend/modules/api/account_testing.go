package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"code-proxy/modules/auth"
	"code-proxy/modules/database"
)

// TestResult represents the result of testing an account connection
type TestResult struct {
	Valid      bool   `json:"valid"`
	Error      string `json:"error,omitempty"`
	LatencyMs  int64  `json:"latency_ms"`
	StatusCode int    `json:"status_code,omitempty"`
	Refreshed  bool   `json:"refreshed,omitempty"`
}

// testAccount tests a single account connection
func testAccount(w http.ResponseWriter, db *database.DB, id string) {
	acct, err := db.GetAccount(id)
	if err != nil {
		writeError(w, "Account not found", http.StatusNotFound)
		return
	}

	var result TestResult
	start := time.Now()

	if acct.AuthMode == "oauth" {
		result = testOAuthAccount(acct, db)
	} else if acct.AuthMode == "apikey" {
		result = testApiKeyAccount(acct)
	} else {
		result = TestResult{Valid: false, Error: "Unknown auth mode: " + acct.AuthMode}
	}

	result.LatencyMs = time.Since(start).Milliseconds()

	// Update account status in DB based on test result
	if result.Valid {
		db.ClearAccountCooldown(id)
	} else if result.Error != "" {
		db.SetAccountCooldown(id, time.Now().Add(1*time.Minute), 0, result.Error)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// testOAuthAccount tests an OAuth account by checking token validity
func testOAuthAccount(acct *database.Account, db *database.DB) TestResult {
	if acct.AccessToken == "" {
		return TestResult{Valid: false, Error: "No access token"}
	}

	// Check if token is expired
	if acct.ExpiresAt != nil && acct.ExpiresAt.Before(time.Now().Add(-5*time.Minute)) {
		// Try to refresh
		if acct.RefreshToken != "" {
			providerName := mapProviderTypeToOAuth(acct.ProviderType)
			cfg, ok := auth.GetConfig(providerName)
			if ok {
				tokens, err := auth.RefreshTokens(cfg, acct.RefreshToken)
				if err != nil {
					return TestResult{Valid: false, Error: "Token expired, refresh failed: " + err.Error()}
				}
				expiresAt := tokens.ExpiresAt
				db.UpdateAccountTokens(acct.ID, tokens.AccessToken, tokens.RefreshToken, &expiresAt)
				return TestResult{Valid: true, Refreshed: true}
			}
		}
		return TestResult{Valid: false, Error: "Token expired, no refresh token"}
	}

	// For Claude/Anthropic OAuth, try a lightweight API call
	if acct.ProviderType == "claude-cli" || acct.ProviderType == "anthropic-api" {
		return testHTTPEndpoint("https://api.anthropic.com/v1/models", map[string]string{
			"x-api-key":         acct.AccessToken,
			"anthropic-version": "2023-06-01",
		})
	}

	// For Codex OAuth — token is only usable via CLI, can't test against OpenAI API directly
	if acct.ProviderType == "codex-cli" {
		return TestResult{Valid: true}
	}

	// For OpenAI API OAuth
	if acct.ProviderType == "openai-api" {
		return testChatEndpoint("https://api.openai.com/v1/chat/completions", map[string]string{
			"Authorization": "Bearer " + acct.AccessToken,
		}, "gpt-4o-mini")
	}

	// For Gemini OAuth
	if acct.ProviderType == "gemini-cli" || acct.ProviderType == "gemini-api" {
		return testHTTPEndpoint("https://www.googleapis.com/oauth2/v1/userinfo?alt=json", map[string]string{
			"Authorization": "Bearer " + acct.AccessToken,
		})
	}

	// For Antigravity
	if acct.ProviderType == "antigravity" {
		return testHTTPEndpoint("https://www.googleapis.com/oauth2/v1/userinfo?alt=json", map[string]string{
			"Authorization": "Bearer " + acct.AccessToken,
		})
	}

	// Default: token exists and not expired = valid
	return TestResult{Valid: true}
}

// testApiKeyAccount tests an API key account by hitting the provider's endpoint
func testApiKeyAccount(acct *database.Account) TestResult {
	if acct.APIKey == "" {
		return TestResult{Valid: false, Error: "No API key configured"}
	}

	baseURL := acct.Metadata["base_url"]
	subtype := acct.Metadata["provider_subtype"]

	// Determine the test endpoint based on provider type/subtype
	type testConfig struct {
		url     string
		headers map[string]string
	}

	var cfg testConfig

	switch {
	case acct.ProviderType == "anthropic-api":
		cfg = testConfig{
			url: "https://api.anthropic.com/v1/models",
			headers: map[string]string{
				"x-api-key":         acct.APIKey,
				"anthropic-version": "2023-06-01",
			},
		}
	case acct.ProviderType == "openai-api":
		cfg = testConfig{
			url:     "https://api.openai.com/v1/models",
			headers: map[string]string{"Authorization": "Bearer " + acct.APIKey},
		}
	case acct.ProviderType == "gemini-api":
		cfg = testConfig{
			url:     fmt.Sprintf("https://generativelanguage.googleapis.com/v1/models?key=%s", acct.APIKey),
			headers: map[string]string{},
		}
	case subtype == "groq":
		cfg = testConfig{
			url:     "https://api.groq.com/openai/v1/models",
			headers: map[string]string{"Authorization": "Bearer " + acct.APIKey},
		}
	case subtype == "xai":
		cfg = testConfig{
			url:     "https://api.x.ai/v1/models",
			headers: map[string]string{"Authorization": "Bearer " + acct.APIKey},
		}
	case subtype == "mistral":
		cfg = testConfig{
			url:     "https://api.mistral.ai/v1/models",
			headers: map[string]string{"Authorization": "Bearer " + acct.APIKey},
		}
	case subtype == "together":
		cfg = testConfig{
			url:     "https://api.together.xyz/v1/models",
			headers: map[string]string{"Authorization": "Bearer " + acct.APIKey},
		}
	case subtype == "fireworks":
		cfg = testConfig{
			url:     "https://api.fireworks.ai/inference/v1/models",
			headers: map[string]string{"Authorization": "Bearer " + acct.APIKey},
		}
	case subtype == "cerebras":
		cfg = testConfig{
			url:     "https://api.cerebras.ai/v1/models",
			headers: map[string]string{"Authorization": "Bearer " + acct.APIKey},
		}
	case subtype == "openrouter":
		cfg = testConfig{
			url:     "https://openrouter.ai/api/v1/auth/key",
			headers: map[string]string{"Authorization": "Bearer " + acct.APIKey},
		}
	case subtype == "perplexity":
		cfg = testConfig{
			url:     "https://api.perplexity.ai/models",
			headers: map[string]string{"Authorization": "Bearer " + acct.APIKey},
		}
	case subtype == "nvidia":
		cfg = testConfig{
			url:     "https://integrate.api.nvidia.com/v1/models",
			headers: map[string]string{"Authorization": "Bearer " + acct.APIKey},
		}
	case subtype == "cohere":
		cfg = testConfig{
			url:     "https://api.cohere.ai/v1/models",
			headers: map[string]string{"Authorization": "Bearer " + acct.APIKey},
		}
	case baseURL != "":
		// Generic OpenAI-compatible: hit {baseUrl}/v1/models or {baseUrl}/models
		url := strings.TrimRight(baseURL, "/")
		if !strings.HasSuffix(url, "/v1") {
			url += "/v1"
		}
		url += "/models"
		cfg = testConfig{
			url:     url,
			headers: map[string]string{"Authorization": "Bearer " + acct.APIKey},
		}
	default:
		return TestResult{Valid: false, Error: "No test endpoint configured for this provider"}
	}

	return testHTTPEndpoint(cfg.url, cfg.headers)
}

// testChatEndpoint makes a minimal chat completion request to validate credentials
func testChatEndpoint(url string, headers map[string]string, model string) TestResult {
	client := &http.Client{Timeout: 15 * time.Second}

	body := fmt.Sprintf(`{"model":"%s","messages":[{"role":"user","content":"hi"}],"max_tokens":1}`, model)
	req, err := http.NewRequest("POST", url, strings.NewReader(body))
	if err != nil {
		return TestResult{Valid: false, Error: "Failed to create request: " + err.Error()}
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return TestResult{Valid: false, Error: "Connection failed: " + err.Error()}
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return TestResult{Valid: true, StatusCode: resp.StatusCode}
	}

	errMsg := fmt.Sprintf("HTTP %d", resp.StatusCode)
	if len(respBody) > 0 {
		var errResp map[string]any
		if json.Unmarshal(respBody, &errResp) == nil {
			if e, ok := errResp["error"]; ok {
				switch v := e.(type) {
				case string:
					errMsg = v
				case map[string]any:
					if msg, ok := v["message"].(string); ok {
						errMsg = msg
					}
				}
			}
		}
	}

	log.Printf("[TEST] Chat test failed: url=%s status=%d error=%s", url, resp.StatusCode, errMsg)
	return TestResult{Valid: false, Error: errMsg, StatusCode: resp.StatusCode}
}

// testHTTPEndpoint makes a GET request to validate credentials
func testHTTPEndpoint(url string, headers map[string]string) TestResult {
	client := &http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return TestResult{Valid: false, Error: "Failed to create request: " + err.Error()}
	}

	req.Header.Set("Accept", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return TestResult{Valid: false, Error: "Connection failed: " + err.Error()}
	}
	defer resp.Body.Close()

	// Read body for error details
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return TestResult{Valid: true, StatusCode: resp.StatusCode}
	}

	// Parse error message
	errMsg := fmt.Sprintf("HTTP %d", resp.StatusCode)
	if len(body) > 0 {
		var errResp map[string]any
		if json.Unmarshal(body, &errResp) == nil {
			if e, ok := errResp["error"]; ok {
				switch v := e.(type) {
				case string:
					errMsg = v
				case map[string]any:
					if msg, ok := v["message"].(string); ok {
						errMsg = msg
					}
				}
			}
		}
	}

	log.Printf("[TEST] Account test failed: url=%s status=%d error=%s", url, resp.StatusCode, errMsg)
	return TestResult{Valid: false, Error: errMsg, StatusCode: resp.StatusCode}
}
