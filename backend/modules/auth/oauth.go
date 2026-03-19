package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// OAuthTokens is the result of an OAuth authentication
type OAuthTokens struct {
	AccessToken  string         `json:"access_token"`
	RefreshToken string         `json:"refresh_token"`
	IDToken      string         `json:"id_token,omitempty"`
	ExpiresIn    int            `json:"expires_in,omitempty"`
	ExpiresAt    time.Time      `json:"expires_at"`
	Scope        string         `json:"scope,omitempty"`
	TokenType    string         `json:"token_type,omitempty"`
	RawResponse  map[string]any `json:"raw_response,omitempty"`
}

// OAuthFlow manages an OAuth authorization flow
type OAuthFlow struct {
	config   OAuthConfig
	verifier string
	state    string
	callback *CallbackServer
}

// FlowManager manages active OAuth flows (by flow_id)
type FlowManager struct {
	mu    sync.RWMutex
	flows map[string]*OAuthFlow
}

// NewFlowManager creates an OAuth flow manager
func NewFlowManager() *FlowManager {
	return &FlowManager{
		flows: make(map[string]*OAuthFlow),
	}
}

// StartFlow starts a new OAuth flow for a provider
func (fm *FlowManager) StartFlow(providerName string) (flowID, authURL string, err error) {
	cfg, ok := GetConfig(providerName)
	if !ok {
		return "", "", fmt.Errorf("OAuth config not found for provider %q", providerName)
	}

	flow := &OAuthFlow{config: cfg}

	// Generate PKCE if needed
	var challenge string
	if cfg.UsePKCE {
		flow.verifier, challenge, err = GeneratePKCE()
		if err != nil {
			return "", "", fmt.Errorf("PKCE generation: %w", err)
		}
	}

	// Generate state
	flow.state, err = GenerateState()
	if err != nil {
		return "", "", fmt.Errorf("state generation: %w", err)
	}

	// Start callback server (optional — if it fails, manual paste mode still works)
	if cfg.CallbackPort > 0 {
		flow.callback = NewCallbackServer(cfg.CallbackPort)
		if err := flow.callback.Start(); err != nil {
			log.Printf("[AUTH] WARNING: callback server failed to start on port %d: %v (manual paste mode only)", cfg.CallbackPort, err)
			flow.callback = nil // Disable automatic callback but keep manual paste mode
		}
	}

	// Build authorization URL
	params := url.Values{
		"client_id":     {cfg.ClientID},
		"redirect_uri":  {cfg.RedirectURI()},
		"response_type": {"code"},
		"state":         {flow.state},
		"scope":         {strings.Join(cfg.Scopes, " ")},
	}

	if cfg.UsePKCE {
		params.Set("code_challenge", challenge)
		params.Set("code_challenge_method", "S256")
	}

	// Extra params from the provider
	for k, v := range cfg.ExtraParams {
		params.Set(k, v)
	}

	authURL = cfg.AuthURL + "?" + params.Encode()

	// Generate flow ID
	flowID, err = GenerateState()
	if err != nil {
		flow.callback.Stop()
		return "", "", err
	}
	flowID = flowID[:16] // Shorten

	// Register flow
	fm.mu.Lock()
	fm.flows[flowID] = flow
	fm.mu.Unlock()

	// Auto-cleanup after 10 minutes
	go func() {
		time.Sleep(10 * time.Minute)
		fm.mu.Lock()
		if f, ok := fm.flows[flowID]; ok {
			if f.callback != nil {
				f.callback.Stop()
			}
			delete(fm.flows, flowID)
		}
		fm.mu.Unlock()
	}()

	log.Printf("[AUTH] OAuth flow started: provider=%s, flowId=%s", providerName, flowID)
	return flowID, authURL, nil
}

// WaitForCallback waits for the callback of an OAuth flow
func (fm *FlowManager) WaitForCallback(flowID string, timeout time.Duration) (*OAuthTokens, error) {
	fm.mu.RLock()
	flow, ok := fm.flows[flowID]
	fm.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("flow %q not found or expired", flowID)
	}

	if flow.callback == nil {
		return nil, fmt.Errorf("callback server not available — use manual mode (paste URL)")
	}

	// Wait for callback
	result, err := flow.callback.WaitForResult(timeout)
	if err != nil {
		return nil, err
	}

	// Validate state
	if result.State != "" && result.State != flow.state {
		return nil, fmt.Errorf("state mismatch: expected %s, got %s", flow.state, result.State)
	}

	// Exchange code for tokens
	tokens, err := exchangeCode(flow.config, result.Code, flow.verifier, flow.state)
	if err != nil {
		return nil, fmt.Errorf("token exchange: %w", err)
	}

	// Cleanup
	flow.callback.Stop()
	fm.mu.Lock()
	delete(fm.flows, flowID)
	fm.mu.Unlock()

	return tokens, nil
}

// SubmitCallback manually submits a callback URL
func (fm *FlowManager) SubmitCallback(flowID, callbackURL string) (*OAuthTokens, error) {
	fm.mu.RLock()
	flow, ok := fm.flows[flowID]
	fm.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("flow %q not found or expired", flowID)
	}

	// Parse the URL to extract code and state
	u, err := url.Parse(callbackURL)
	if err != nil {
		return nil, fmt.Errorf("invalid callback URL: %w", err)
	}

	code := u.Query().Get("code")
	state := u.Query().Get("state")

	// Try fragment if the code wasn't found in the query
	if code == "" && u.Fragment != "" {
		vals, _ := url.ParseQuery(u.Fragment)
		code = vals.Get("code")
		if state == "" {
			state = vals.Get("state")
		}
	}

	if code == "" {
		return nil, fmt.Errorf("no authorization code found in URL")
	}

	// Validate state if present
	if state != "" && state != flow.state {
		return nil, fmt.Errorf("state mismatch")
	}

	// Exchange code for tokens
	tokens, err := exchangeCode(flow.config, code, flow.verifier, flow.state)
	if err != nil {
		return nil, fmt.Errorf("token exchange: %w", err)
	}

	// Cleanup
	if flow.callback != nil {
		flow.callback.Stop()
	}
	fm.mu.Lock()
	delete(fm.flows, flowID)
	fm.mu.Unlock()

	return tokens, nil
}

// exchangeCode exchanges an authorization code for tokens
func exchangeCode(cfg OAuthConfig, code, verifier, state string) (*OAuthTokens, error) {
	redirectURI := cfg.RedirectURI()

	var reqBody io.Reader
	var contentType string

	if cfg.ContentType == "application/json" {
		// JSON body (Claude)
		payload := map[string]string{
			"grant_type":   "authorization_code",
			"code":         code,
			"client_id":    cfg.ClientID,
			"redirect_uri": redirectURI,
		}
		if verifier != "" {
			payload["code_verifier"] = verifier
		}
		if state != "" {
			payload["state"] = state
		}
		b, _ := json.Marshal(payload)
		reqBody = bytes.NewReader(b)
		contentType = "application/json"
	} else {
		// Form-urlencoded (OpenAI, Google)
		params := url.Values{
			"grant_type":   {"authorization_code"},
			"code":         {code},
			"client_id":    {cfg.ClientID},
			"redirect_uri": {redirectURI},
		}
		if verifier != "" {
			params.Set("code_verifier", verifier)
		}
		reqBody = strings.NewReader(params.Encode())
		contentType = "application/x-www-form-urlencoded"
	}

	req, err := http.NewRequest("POST", cfg.TokenURL, reqBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("token request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("parse token response: %w", err)
	}

	tokens := &OAuthTokens{
		RawResponse: raw,
	}

	// Extract common fields
	if v, ok := raw["access_token"].(string); ok {
		tokens.AccessToken = v
	}
	if v, ok := raw["refresh_token"].(string); ok {
		tokens.RefreshToken = v
	}
	if v, ok := raw["id_token"].(string); ok {
		tokens.IDToken = v
	}
	if v, ok := raw["scope"].(string); ok {
		tokens.Scope = v
	}
	if v, ok := raw["token_type"].(string); ok {
		tokens.TokenType = v
	}

	// Calculate expiration
	if v, ok := raw["expires_in"].(float64); ok {
		tokens.ExpiresIn = int(v)
		tokens.ExpiresAt = time.Now().Add(time.Duration(v) * time.Second)
	} else {
		// Default: 1 hour
		tokens.ExpiresAt = time.Now().Add(1 * time.Hour)
	}

	log.Printf("[AUTH] Token exchange OK: provider=%s, expires=%s, hasRefresh=%v",
		cfg.Provider, tokens.ExpiresAt.Format(time.RFC3339), tokens.RefreshToken != "")

	return tokens, nil
}

// RefreshTokens refreshes an OAuth token
func RefreshTokens(cfg OAuthConfig, refreshToken string) (*OAuthTokens, error) {
	var reqBody io.Reader
	var contentType string

	if cfg.ContentType == "application/json" {
		payload := map[string]string{
			"grant_type":    "refresh_token",
			"refresh_token": refreshToken,
			"client_id":     cfg.ClientID,
		}
		b, _ := json.Marshal(payload)
		reqBody = bytes.NewReader(b)
		contentType = "application/json"
	} else {
		params := url.Values{
			"grant_type":    {"refresh_token"},
			"refresh_token": {refreshToken},
			"client_id":     {cfg.ClientID},
		}
		reqBody = strings.NewReader(params.Encode())
		contentType = "application/x-www-form-urlencoded"
	}

	req, err := http.NewRequest("POST", cfg.TokenURL, reqBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("refresh request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("refresh failed (HTTP %d): %s", resp.StatusCode, string(body))
	}

	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}

	tokens := &OAuthTokens{RawResponse: raw}

	if v, ok := raw["access_token"].(string); ok {
		tokens.AccessToken = v
	}
	if v, ok := raw["refresh_token"].(string); ok {
		tokens.RefreshToken = v
	} else {
		// Keep the original refresh token if no new one was returned
		tokens.RefreshToken = refreshToken
	}
	if v, ok := raw["expires_in"].(float64); ok {
		tokens.ExpiresIn = int(v)
		tokens.ExpiresAt = time.Now().Add(time.Duration(v) * time.Second)
	} else {
		tokens.ExpiresAt = time.Now().Add(1 * time.Hour)
	}

	log.Printf("[AUTH] Token refresh OK: provider=%s, expires=%s",
		cfg.Provider, tokens.ExpiresAt.Format(time.RFC3339))

	return tokens, nil
}
