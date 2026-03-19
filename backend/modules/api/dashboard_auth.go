package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"code-proxy/modules/account"
	"code-proxy/modules/auth"
	"code-proxy/modules/database"
	"code-proxy/modules/provider"
)

// oauthFlows is the global OAuth flow manager
var oauthFlows = auth.NewFlowManager()

// registerAccountRoutes registers account management endpoints
func registerAccountRoutes(mux *http.ServeMux, db *database.DB, acctMgr *account.Manager, registry *provider.Registry) {
	// GET /api/accounts/usage?period=24h|7d|30d|60d
	// Returns aggregated usage (tokens, estimated cost, last used) per connected account.
	mux.HandleFunc("/api/accounts/usage", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		period := r.URL.Query().Get("period")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(db.GetAccountUsageForPeriod(period))
	})

	// GET/POST /api/accounts
	mux.HandleFunc("/api/accounts", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			listAccounts(w, r, db)
		case "POST":
			createAccount(w, r, db)
		default:
			writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// OAuth routes and account routes with ID
	mux.HandleFunc("/api/accounts/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path[len("/api/accounts/"):]

		// POST /api/accounts/oauth/start
		if path == "oauth/start" && r.Method == "POST" {
			startOAuth(w, r, db)
			return
		}

		// POST /api/accounts/oauth/callback
		if path == "oauth/callback" && r.Method == "POST" {
			handleOAuthCallback(w, r, db)
			return
		}

		// GET /api/accounts/oauth/providers — list available OAuth providers
		if path == "oauth/providers" && r.Method == "GET" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(auth.ListOAuthProviders())
			return
		}

		// ID routes: /api/accounts/{id} or /api/accounts/{id}/...
		id := path
		suffix := ""
		if idx := strings.Index(path, "/"); idx > 0 {
			id = path[:idx]
			suffix = path[idx+1:]
		}

		if id == "" {
			writeError(w, "Account ID required", http.StatusBadRequest)
			return
		}

		switch {
		case suffix == "refresh" && r.Method == "POST":
			refreshAccount(w, db, id)
		case suffix == "test" && r.Method == "POST":
			testAccount(w, db, id)
		case suffix == "quota" && r.Method == "GET":
			getAccountQuota(w, db, id)
		case suffix == "status" && r.Method == "GET":
			accountStatus(w, db, id)
		case suffix == "" && r.Method == "PUT":
			updateAccount(w, r, db, id)
		case suffix == "" && r.Method == "DELETE":
			deleteAccount(w, db, id)
		default:
			writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	_ = acctMgr
	_ = registry
}

// --- Accounts CRUD ---

func listAccounts(w http.ResponseWriter, r *http.Request, db *database.DB) {
	providerType := r.URL.Query().Get("provider_type")
	accounts, err := db.ListAccounts(providerType)
	if err != nil {
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if accounts == nil {
		accounts = []database.Account{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(accounts)
}

func createAccount(w http.ResponseWriter, r *http.Request, db *database.DB) {
	var req struct {
		ProviderType string            `json:"provider_type"`
		Label        string            `json:"label"`
		AuthMode     string            `json:"auth_mode"`
		APIKey       string            `json:"api_key"`
		Metadata     map[string]string `json:"metadata"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.ProviderType == "" {
		writeError(w, "provider_type is required", http.StatusBadRequest)
		return
	}
	if req.AuthMode == "" {
		if req.APIKey != "" {
			req.AuthMode = "apikey"
		} else {
			req.AuthMode = "none"
		}
	}
	if req.Label == "" {
		req.Label = req.ProviderType
	}

	acct, err := db.CreateAccountFull(
		req.ProviderType, req.Label, req.AuthMode,
		"", "", req.APIKey, nil, req.Metadata,
	)
	if err != nil {
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(acct)
}

func updateAccount(w http.ResponseWriter, r *http.Request, db *database.DB, id string) {
	// Partial updates: the frontend may send only one field (e.g. label or is_active).
	// Using pointers prevents missing fields from being deserialized into zero-values and
	// overwriting other properties unintentionally.
	var req struct {
		Label    *string `json:"label"`
		IsActive *bool   `json:"is_active"`
		Priority *int    `json:"priority"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	acct, err := db.GetAccount(id)
	if err != nil {
		writeError(w, "Account not found", http.StatusNotFound)
		return
	}

	label := acct.Label
	isActive := acct.IsActive
	priority := acct.Priority

	if req.Label != nil {
		label = *req.Label
	}
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	if req.Priority != nil {
		priority = *req.Priority
	}

	if err := db.UpdateAccount(id, label, isActive, priority); err != nil {
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func deleteAccount(w http.ResponseWriter, db *database.DB, id string) {
	if err := db.DeleteAccount(id); err != nil {
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func accountStatus(w http.ResponseWriter, db *database.DB, id string) {
	acct, err := db.GetAccount(id)
	if err != nil {
		writeError(w, "Account not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(acct)
}

func refreshAccount(w http.ResponseWriter, db *database.DB, id string) {
	acct, err := db.GetAccount(id)
	if err != nil {
		writeError(w, "Account not found", http.StatusNotFound)
		return
	}

	if acct.RefreshToken == "" {
		writeError(w, "No refresh token available", http.StatusBadRequest)
		return
	}

	// Determine provider to fetch OAuth config
	providerName := mapProviderTypeToOAuth(acct.ProviderType)
	cfg, ok := auth.GetConfig(providerName)
	if !ok {
		writeError(w, "OAuth config not found for provider", http.StatusBadRequest)
		return
	}

	tokens, err := auth.RefreshTokens(cfg, acct.RefreshToken)
	if err != nil {
		writeError(w, "Refresh failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Update in database
	expiresAt := tokens.ExpiresAt
	if err := db.UpdateAccountTokens(id, tokens.AccessToken, tokens.RefreshToken, &expiresAt); err != nil {
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status":     "ok",
		"expires_at": expiresAt,
	})
}

// --- OAuth Flow ---

func startOAuth(w http.ResponseWriter, r *http.Request, db *database.DB) {
	var req struct {
		ProviderType string `json:"provider_type"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Map provider_type to OAuth config name
	providerName := mapProviderTypeToOAuth(req.ProviderType)
	if providerName == "" {
		providerName = req.ProviderType
	}

	flowID, authURL, err := oauthFlows.StartFlow(providerName)
	if err != nil {
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"flow_id":  flowID,
		"auth_url": authURL,
	})
}

func handleOAuthCallback(w http.ResponseWriter, r *http.Request, db *database.DB) {
	var req struct {
		FlowID       string `json:"flow_id"`
		CallbackURL  string `json:"callback_url"`
		ProviderType string `json:"provider_type"` // Used to create the account
		Label        string `json:"label"`          // Custom label (optional)
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.FlowID == "" {
		writeError(w, "flow_id is required", http.StatusBadRequest)
		return
	}

	var tokens *auth.OAuthTokens
	var err error

	if req.CallbackURL != "" {
		// Manual mode: paste the URL
		tokens, err = oauthFlows.SubmitCallback(req.FlowID, req.CallbackURL)
	} else {
		// Automatic mode: wait for callback on the local server
		tokens, err = oauthFlows.WaitForCallback(req.FlowID, 5*time.Minute)
	}

	if err != nil {
		writeError(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Create account with the tokens
	providerType := req.ProviderType
	if providerType == "" {
		providerType = "anthropic-api" // default
	}

	label := req.Label
	if label == "" {
		label = providerType
		// Try to extract email from the token
		if email, ok := tokens.RawResponse["email"].(string); ok && email != "" {
			label = email
		}
	}

	metadata := map[string]string{}
	if tokens.IDToken != "" {
		metadata["id_token"] = tokens.IDToken
	}

	expiresAt := tokens.ExpiresAt
	acct, err := db.CreateAccountFull(
		providerType, label, "oauth",
		tokens.AccessToken, tokens.RefreshToken, "",
		&expiresAt, metadata,
	)
	if err != nil {
		writeError(w, "Failed to save account: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(acct)
}

// mapProviderTypeToOAuth maps provider type to OAuth config name
func mapProviderTypeToOAuth(providerType string) string {
	mapping := map[string]string{
		"claude-cli":     "claude",
		"anthropic-api":  "claude",
		"codex-cli":      "codex",
		"openai-api":     "codex",
		"gemini-cli":     "gemini",
		"gemini-api":     "gemini",
		"antigravity":    "antigravity",
		"github-copilot": "github",
	}
	if name, ok := mapping[providerType]; ok {
		return name
	}
	return providerType
}
