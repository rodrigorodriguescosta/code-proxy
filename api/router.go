package api

import (
	"encoding/json"
	"io/fs"
	"log"
	"net/http"
	"strings"

	"code-proxy/account"
	"code-proxy/database"
	"code-proxy/provider"
	"code-proxy/tunnel"
)

type Server struct {
	mux          *http.ServeMux
	registry     *provider.Registry
	acctMgr      *account.Manager
	db           *database.DB
	tunnel       *tunnel.Manager
	defaultModel string
	envApiKeys   []string // fallback API keys from env
	frontendFS   fs.FS    // embedded frontend
}

type ServerOptions struct {
	Registry     *provider.Registry
	AccountMgr   *account.Manager
	DefaultModel string
	EnvApiKeys   []string
	DB           *database.DB
	Tunnel       *tunnel.Manager
	FrontendFS   fs.FS
}

func NewServer(opts ServerOptions) *Server {
	s := &Server{
		mux:          http.NewServeMux(),
		registry:     opts.Registry,
		acctMgr:      opts.AccountMgr,
		db:           opts.DB,
		tunnel:       opts.Tunnel,
		defaultModel: opts.DefaultModel,
		envApiKeys:   opts.EnvApiKeys,
		frontendFS:   opts.FrontendFS,
	}
	s.registerRoutes()
	return s
}

// RegisterProvider registers a provider in the server registry
func (s *Server) RegisterProvider(providerType string, p provider.Provider) {
	s.registry.Register(providerType, p)
	log.Printf("[SERVER] Registered provider: %s (%s)", p.Name(), providerType)
}

func (s *Server) Handler() http.Handler {
	var handler http.Handler = s.mux
	handler = s.dbAuthMiddleware(handler)
	// Protect dashboard management APIs when dashboard password is configured.
	handler = dashboardAuthMiddleware(s.db, handler)
	handler = corsMiddleware(handler)
	handler = loggingMiddleware(handler)
	return handler
}

// dbAuthMiddleware validates API keys against DB + env keys for /v1/ endpoints.
// When require_api_key is false, requests pass through without auth (but still track key if provided).
func (s *Server) dbAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth for non-API paths
		if !strings.HasPrefix(r.URL.Path, "/v1/") {
			next.ServeHTTP(w, r)
			return
		}

		token := extractBearerToken(r)

		// Check if API key auth is required
		requireKey := true
		if s.db != nil {
			settings := s.db.GetSettings()
			requireKey = settings.RequireApiKey
		}

		// If token provided, try to set context regardless of require_key setting
		if token != "" && s.db != nil {
			if keyID, keyName, ok := s.db.GetApiKeyInfo(token); ok {
				r = setApiKeyContext(r, keyID, keyName)
				next.ServeHTTP(w, r)
				return
			}
			// Check env keys
			for _, k := range s.envApiKeys {
				if k == token {
					r = setApiKeyContext(r, "env", "env-key")
					next.ServeHTTP(w, r)
					return
				}
			}
			// Token provided but invalid
			if requireKey {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				w.Write([]byte(`{"error":{"message":"Invalid API key","type":"authentication_error"}}`))
				return
			}
		}

		// No token provided
		if !requireKey {
			next.ServeHTTP(w, r)
			return
		}

		// requireKey=true, no token, check if any keys exist
		hasEnvKeys := len(s.envApiKeys) > 0
		hasDbKeys := false
		if s.db != nil {
			keys, _ := s.db.ListApiKeys()
			hasDbKeys = len(keys) > 0
		}
		if !hasEnvKeys && !hasDbKeys {
			// No keys configured — allow through (first-use mode)
			next.ServeHTTP(w, r)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":{"message":"Missing API key","type":"authentication_error"}}`))
	})
}

func (s *Server) registerRoutes() {
	// OpenAI-compatible API
	s.mux.HandleFunc("/v1/chat/completions", handleChat(s.registry, s.acctMgr, s.defaultModel, s.db))
	s.mux.HandleFunc("/v1/models", s.handleModels)
	s.mux.HandleFunc("/v1/", s.handleV1Info)

	// Health
	s.mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	// Dashboard API
	if s.db != nil {
		registerDashboardRoutes(s.mux, s.db, s.acctMgr, s.registry)
	}

	// Tunnel API
	if s.tunnel != nil {
		s.tunnel.RegisterRoutes(s.mux, func(token string) {
			if s.db != nil {
				s.db.SetSetting("tunnel_token", token)
			}
		})
	}

	// Frontend (SPA fallback)
	if s.frontendFS != nil {
		fileServer := http.FileServer(http.FS(s.frontendFS))
		s.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			// Try serving the file directly
			path := r.URL.Path
			if path == "/" {
				path = "/index.html"
			}

			// Check if file exists in frontend FS
			f, err := s.frontendFS.Open(strings.TrimPrefix(path, "/"))
			if err == nil {
				f.Close()
				fileServer.ServeHTTP(w, r)
				return
			}

			// SPA fallback: serve index.html for non-API, non-file routes
			if !strings.HasPrefix(path, "/api/") && !strings.HasPrefix(path, "/v1/") {
				r.URL.Path = "/"
				fileServer.ServeHTTP(w, r)
				return
			}

			http.NotFound(w, r)
		})
	} else {
		s.mux.HandleFunc("/", s.handleRoot)
	}
}

func (s *Server) handleModels(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var models []ModelItem
	for _, m := range s.registry.AllModels() {
		models = append(models, ModelItem{
			ID:      m.ID,
			Object:  "model",
			OwnedBy: m.OwnedBy,
		})
	}

	json.NewEncoder(w).Encode(ModelsResponse{
		Object: "list",
		Data:   models,
	})
}

func (s *Server) handleV1Info(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":    "ok",
		"service":   "code-proxy",
		"endpoints": "/v1/chat/completions, /v1/models",
	})
}

func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status":    "ok",
		"service":   "code-proxy",
		"providers": s.registry.ListProviders(),
		"dashboard": "Frontend not built. Run: cd frontend-src && npm run build",
	})
}
