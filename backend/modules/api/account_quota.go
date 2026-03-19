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

// QuotaInfo represents usage quota for a single metric/model
type QuotaInfo struct {
	Name       string  `json:"name"`
	Used       float64 `json:"used"`
	Total      float64 `json:"total"`
	Remaining  float64 `json:"remaining"`
	Percentage float64 `json:"percentage"` // remaining percentage (0-100)
	ResetAt    string  `json:"reset_at,omitempty"`
	Unlimited  bool    `json:"unlimited,omitempty"`
}

// QuotaResult represents the full quota response for an account
type QuotaResult struct {
	Plan    string      `json:"plan,omitempty"`
	Message string      `json:"message,omitempty"`
	Quotas  []QuotaInfo `json:"quotas"`
	Error   string      `json:"error,omitempty"`
}

// getAccountQuota fetches usage quota for a connected account
func getAccountQuota(w http.ResponseWriter, db *database.DB, id string) {
	acct, err := db.GetAccount(id)
	if err != nil {
		writeError(w, "Account not found", http.StatusNotFound)
		return
	}

	var result QuotaResult

	switch acct.ProviderType {
	case "claude-cli", "anthropic-api":
		result = getClaudeQuota(acct, db)
	case "codex-cli", "openai-api":
		result = getCodexQuota(acct, db)
	case "antigravity":
		result = getAntigravityQuota(acct, db)
	case "github-copilot":
		result = getGitHubCopilotQuota(acct, db)
	case "gemini-cli", "gemini-api":
		result = QuotaResult{Message: "Gemini uses Google Cloud quotas. Check Google Cloud Console for details."}
	default:
		result = QuotaResult{Message: fmt.Sprintf("Usage quota not available for %s", acct.ProviderType)}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// getClaudeQuota fetches Claude/Anthropic usage quota
func getClaudeQuota(acct *database.Account, db *database.DB) QuotaResult {
	token := acct.AccessToken
	if token == "" {
		token = acct.APIKey
	}
	if token == "" {
		return QuotaResult{Error: "No token available"}
	}

	// Try to get settings first (for org info)
	client := &http.Client{Timeout: 10 * time.Second}

	req, _ := http.NewRequest("GET", "https://api.anthropic.com/v1/settings", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := client.Do(req)
	if err != nil {
		return QuotaResult{Message: "Claude connected. Unable to fetch usage: " + err.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		// For API key accounts, no usage quota endpoint
		if acct.AuthMode == "apikey" {
			return QuotaResult{Message: "Claude API key connected. Usage tracked per request."}
		}
		return QuotaResult{Message: "Claude connected. Usage API requires admin permissions."}
	}

	var settings map[string]any
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	json.Unmarshal(body, &settings)

	plan, _ := settings["plan"].(string)
	orgName, _ := settings["organization_name"].(string)

	result := QuotaResult{
		Plan:    plan,
		Message: fmt.Sprintf("Connected to %s", orgName),
	}

	// Try org usage endpoint if we have org_id
	orgID, _ := settings["organization_id"].(string)
	if orgID != "" {
		usageReq, _ := http.NewRequest("GET",
			fmt.Sprintf("https://api.anthropic.com/v1/organizations/%s/usage", orgID), nil)
		usageReq.Header.Set("Authorization", "Bearer "+token)
		usageReq.Header.Set("Content-Type", "application/json")
		usageReq.Header.Set("anthropic-version", "2023-06-01")

		usageResp, err := client.Do(usageReq)
		if err == nil {
			defer usageResp.Body.Close()
			if usageResp.StatusCode == 200 {
				var usage map[string]any
				usageBody, _ := io.ReadAll(io.LimitReader(usageResp.Body, 8192))
				json.Unmarshal(usageBody, &usage)
				// Parse quota data if available
				result.Quotas = parseClaudeUsage(usage)
			}
		}
	}

	return result
}

func parseClaudeUsage(usage map[string]any) []QuotaInfo {
	// Claude usage API structure varies; return what's available
	var quotas []QuotaInfo
	// The API may return rate limits or usage data
	if limits, ok := usage["rate_limits"].(map[string]any); ok {
		for name, v := range limits {
			if lim, ok := v.(map[string]any); ok {
				total, _ := lim["limit"].(float64)
				used, _ := lim["used"].(float64)
				remaining := total - used
				pct := float64(0)
				if total > 0 {
					pct = (remaining / total) * 100
				}
				quotas = append(quotas, QuotaInfo{
					Name:       name,
					Used:       used,
					Total:      total,
					Remaining:  remaining,
					Percentage: pct,
				})
			}
		}
	}
	return quotas
}

// getCodexQuota fetches OpenAI/Codex usage quota
func getCodexQuota(acct *database.Account, db *database.DB) QuotaResult {
	token := acct.AccessToken
	if token == "" {
		token = acct.APIKey
	}
	if token == "" {
		return QuotaResult{Error: "No token available"}
	}

	// For API key accounts, no subscription quota
	if acct.AuthMode == "apikey" {
		return QuotaResult{Message: "OpenAI API key connected. Usage tracked per request."}
	}

	// For OAuth accounts (Codex subscription), fetch from ChatGPT backend API
	client := &http.Client{Timeout: 10 * time.Second}

	req, _ := http.NewRequest("GET", "https://chatgpt.com/backend-api/wham/usage", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return QuotaResult{Message: "Codex connected. Unable to fetch usage: " + err.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return QuotaResult{Message: "Codex connected. Usage API not accessible."}
	}

	var data map[string]any
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
	json.Unmarshal(body, &data)

	plan, _ := data["plan_type"].(string)
	rateLimit, _ := data["rate_limit"].(map[string]any)

	var quotas []QuotaInfo

	if rateLimit != nil {
		// Primary window (session)
		if pw, ok := rateLimit["primary_window"].(map[string]any); ok {
			usedPct, _ := pw["used_percent"].(float64)
			remaining := 100 - usedPct
			resetAt := ""
			if ra, ok := pw["reset_at"].(float64); ok && ra > 0 {
				resetAt = time.Unix(int64(ra), 0).UTC().Format(time.RFC3339)
			}
			quotas = append(quotas, QuotaInfo{
				Name:       "Session",
				Used:       usedPct,
				Total:      100,
				Remaining:  remaining,
				Percentage: remaining,
				ResetAt:    resetAt,
			})
		}

		// Secondary window (weekly)
		if sw, ok := rateLimit["secondary_window"].(map[string]any); ok {
			usedPct, _ := sw["used_percent"].(float64)
			remaining := 100 - usedPct
			resetAt := ""
			if ra, ok := sw["reset_at"].(float64); ok && ra > 0 {
				resetAt = time.Unix(int64(ra), 0).UTC().Format(time.RFC3339)
			}
			quotas = append(quotas, QuotaInfo{
				Name:       "Weekly",
				Used:       usedPct,
				Total:      100,
				Remaining:  remaining,
				Percentage: remaining,
				ResetAt:    resetAt,
			})
		}
	}

	return QuotaResult{
		Plan:   plan,
		Quotas: quotas,
	}
}

// getAntigravityQuota fetches Antigravity/Google Cloud Code usage quota
func getAntigravityQuota(acct *database.Account, db *database.DB) QuotaResult {
	if acct.AccessToken == "" {
		return QuotaResult{Error: "No access token available"}
	}

	// Refresh token if needed
	acct = refreshIfNeeded(acct, db)

	client := &http.Client{Timeout: 10 * time.Second}

	// Fetch available models with quota info
	req, _ := http.NewRequest("POST", "https://cloudcode-pa.googleapis.com/v1internal:fetchAvailableModels", stringReader("{}"))
	req.Header.Set("Authorization", "Bearer "+acct.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "code-proxy/1.0")
	req.Header.Set("x-request-source", "local")

	resp, err := client.Do(req)
	if err != nil {
		return QuotaResult{Message: "Antigravity connected. Unable to fetch quota: " + err.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return QuotaResult{Message: "Antigravity connected. Quota API not accessible."}
	}

	var data map[string]any
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 32768))
	json.Unmarshal(body, &data)

	var quotas []QuotaInfo

	// Important models to track
	importantModels := map[string]bool{
		"claude-opus-4-6-thinking": true,
		"claude-sonnet-4-6":       true,
		"gemini-3.1-pro-high":     true,
		"gemini-3.1-pro-low":      true,
		"gemini-3-flash":          true,
		"gpt-oss-120b-medium":     true,
	}

	if models, ok := data["models"].(map[string]any); ok {
		for modelKey, info := range models {
			modelInfo, ok := info.(map[string]any)
			if !ok {
				continue
			}

			quotaInfo, hasQuota := modelInfo["quotaInfo"].(map[string]any)
			if !hasQuota {
				continue
			}

			if !importantModels[modelKey] {
				continue
			}

			remainingFraction, _ := quotaInfo["remainingFraction"].(float64)
			remainingPct := remainingFraction * 100
			total := float64(1000) // Normalized base
			remaining := total * remainingFraction
			used := total - remaining

			displayName, _ := modelInfo["displayName"].(string)
			if displayName == "" {
				displayName = modelKey
			}

			resetTime := ""
			if rt, ok := quotaInfo["resetTime"].(string); ok {
				resetTime = rt
			}

			quotas = append(quotas, QuotaInfo{
				Name:       displayName,
				Used:       used,
				Total:      total,
				Remaining:  remaining,
				Percentage: remainingPct,
				ResetAt:    resetTime,
			})
		}
	}

	return QuotaResult{
		Plan:   "Antigravity",
		Quotas: quotas,
	}
}

// getGitHubCopilotQuota fetches GitHub Copilot usage quota
func getGitHubCopilotQuota(acct *database.Account, db *database.DB) QuotaResult {
	if acct.AccessToken == "" {
		return QuotaResult{Error: "No access token available"}
	}

	client := &http.Client{Timeout: 10 * time.Second}

	req, _ := http.NewRequest("GET", "https://api.github.com/copilot_internal/user", nil)
	req.Header.Set("Authorization", "token "+acct.AccessToken)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "GitHubCopilotChat/0.26.7")
	req.Header.Set("Editor-Version", "vscode/1.100.0")
	req.Header.Set("Editor-Plugin-Version", "copilot-chat/0.26.7")

	resp, err := client.Do(req)
	if err != nil {
		return QuotaResult{Message: "GitHub Copilot connected. Unable to fetch quota: " + err.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return QuotaResult{Message: "GitHub Copilot connected. Quota API not accessible."}
	}

	var data map[string]any
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8192))
	json.Unmarshal(body, &data)

	plan, _ := data["copilot_plan"].(string)
	var quotas []QuotaInfo

	resetDate := ""
	if rd, ok := data["quota_reset_date"].(string); ok {
		resetDate = rd
	}

	// Paid plan: quota_snapshots
	if snapshots, ok := data["quota_snapshots"].(map[string]any); ok {
		for name, v := range snapshots {
			snap, ok := v.(map[string]any)
			if !ok {
				continue
			}
			entitlement, _ := snap["entitlement"].(float64)
			remaining, _ := snap["remaining"].(float64)
			unlimited, _ := snap["unlimited"].(bool)
			used := entitlement - remaining
			pct := float64(0)
			if entitlement > 0 {
				pct = (remaining / entitlement) * 100
			}
			quotas = append(quotas, QuotaInfo{
				Name:       name,
				Used:       used,
				Total:      entitlement,
				Remaining:  remaining,
				Percentage: pct,
				ResetAt:    resetDate,
				Unlimited:  unlimited,
			})
		}
	}

	// Free plan: monthly_quotas
	if monthlyQuotas, ok := data["monthly_quotas"].(map[string]any); ok {
		usedQuotas, _ := data["limited_user_quotas"].(map[string]any)
		freeResetDate, _ := data["limited_user_reset_date"].(string)

		for name, v := range monthlyQuotas {
			total, _ := v.(float64)
			used := float64(0)
			if usedQuotas != nil {
				used, _ = usedQuotas[name].(float64)
			}
			remaining := total - used
			pct := float64(0)
			if total > 0 {
				pct = (remaining / total) * 100
			}
			quotas = append(quotas, QuotaInfo{
				Name:       name,
				Used:       used,
				Total:      total,
				Remaining:  remaining,
				Percentage: pct,
				ResetAt:    freeResetDate,
			})
		}
	}

	return QuotaResult{
		Plan:   plan,
		Quotas: quotas,
	}
}

// refreshIfNeeded refreshes OAuth tokens if expired
func refreshIfNeeded(acct *database.Account, db *database.DB) *database.Account {
	if acct.ExpiresAt == nil || acct.ExpiresAt.After(time.Now().Add(5*time.Minute)) {
		return acct // Not expired yet
	}

	if acct.RefreshToken == "" {
		return acct
	}

	providerName := mapProviderTypeToOAuth(acct.ProviderType)
	cfg, ok := auth.GetConfig(providerName)
	if !ok {
		return acct
	}

	tokens, err := auth.RefreshTokens(cfg, acct.RefreshToken)
	if err != nil {
		log.Printf("[QUOTA] Token refresh failed for %s: %v", acct.ID, err)
		return acct
	}

	expiresAt := tokens.ExpiresAt
	db.UpdateAccountTokens(acct.ID, tokens.AccessToken, tokens.RefreshToken, &expiresAt)
	acct.AccessToken = tokens.AccessToken
	acct.RefreshToken = tokens.RefreshToken
	acct.ExpiresAt = &expiresAt

	return acct
}

// stringReader creates an io.ReadCloser from a string
func stringReader(s string) io.ReadCloser {
	return io.NopCloser(strings.NewReader(s))
}
