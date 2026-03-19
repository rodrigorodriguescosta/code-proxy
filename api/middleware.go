package api

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"code-proxy/database"
)

type contextKey string

const ctxKeyApiKeyID contextKey = "api_key_id"
const ctxKeyApiKeyName contextKey = "api_key_name"

// CORS middleware
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-API-Key, X-Dashboard-Token")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Logging middleware
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		log.Printf("[HTTP] >>> %s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
		log.Printf("[HTTP] <<< %s %s (%v)", r.Method, r.URL.Path, time.Since(start))
	})
}

// Auth middleware - validates API key for /v1/ endpoints (unused legacy, kept for reference)
func authMiddleware(validKeys []string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/v1/") {
			next.ServeHTTP(w, r)
			return
		}

		if len(validKeys) == 0 {
			next.ServeHTTP(w, r)
			return
		}

		token := ""
		auth := r.Header.Get("Authorization")
		if strings.HasPrefix(auth, "Bearer ") {
			token = strings.TrimPrefix(auth, "Bearer ")
		}
		if token == "" {
			token = r.Header.Get("X-API-Key")
		}

		if token == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":{"message":"Missing API key","type":"authentication_error"}}`))
			return
		}

		valid := false
		for _, k := range validKeys {
			if k == token {
				valid = true
				break
			}
		}

		if !valid {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":{"message":"Invalid API key","type":"authentication_error"}}`))
			return
		}

		next.ServeHTTP(w, r)
	})
}

// extractBearerToken extracts the Bearer token from Authorization header or X-API-Key
func extractBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return r.Header.Get("X-API-Key")
}

// setApiKeyContext adds API key ID and name to request context
func setApiKeyContext(r *http.Request, keyID, keyName string) *http.Request {
	ctx := context.WithValue(r.Context(), ctxKeyApiKeyID, keyID)
	ctx = context.WithValue(ctx, ctxKeyApiKeyName, keyName)
	return r.WithContext(ctx)
}

// GetApiKeyID returns the API key ID from request context
func GetApiKeyID(r *http.Request) string {
	if v, ok := r.Context().Value(ctxKeyApiKeyID).(string); ok {
		return v
	}
	return ""
}

// GetApiKeyName returns the API key name from request context
func GetApiKeyName(r *http.Request) string {
	if v, ok := r.Context().Value(ctxKeyApiKeyName).(string); ok {
		return v
	}
	return ""
}

// dashboardAuthMiddleware enforces dashboard password protection for /api/* routes.
// If a dashboard password is not configured, it allows all requests.
// Excludes /api/auth/* so login/status and password management work without a token.
func dashboardAuthMiddleware(db *database.DB, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Always allow CORS preflight.
		if r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}

		if db == nil {
			next.ServeHTTP(w, r)
			return
		}

		// Only protect dashboard APIs.
		if !strings.HasPrefix(r.URL.Path, "/api/") {
			next.ServeHTTP(w, r)
			return
		}

		// Allow auth endpoints without token.
		if strings.HasPrefix(r.URL.Path, "/api/auth/") || r.URL.Path == "/api/auth/status" {
			next.ServeHTTP(w, r)
			return
		}

		if !db.HasDashboardPassword() {
			next.ServeHTTP(w, r)
			return
		}

		token := r.Header.Get("X-Dashboard-Token")
		if token == "" || !db.ValidateDashboardSession(token) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":{"message":"Dashboard auth required","type":"authentication_error"}}`))
			return
		}

		next.ServeHTTP(w, r)
	})
}
