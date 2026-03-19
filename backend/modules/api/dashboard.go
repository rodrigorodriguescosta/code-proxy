package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"

	"code-proxy/modules/account"
	"code-proxy/modules/database"
	"code-proxy/modules/provider"
)

func registerDashboardRoutes(mux *http.ServeMux, db *database.DB, acctMgr *account.Manager, registry *provider.Registry) {
	// API Keys
	mux.HandleFunc("/api/keys", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			listApiKeys(w, db)
		case "POST":
			createApiKey(w, r, db)
		default:
			writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/keys/", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Path[len("/api/keys/"):]
		if id == "" {
			writeError(w, "Key ID required", http.StatusBadRequest)
			return
		}
		switch r.Method {
		case "PUT":
			toggleApiKey(w, r, db, id)
		case "DELETE":
			deleteApiKey(w, db, id)
		default:
			writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Providers
	mux.HandleFunc("/api/providers", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			listProviders(w, db)
		case "POST":
			createProvider(w, r, db)
		default:
			writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/api/providers/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path[len("/api/providers/"):]
		if path == "" {
			writeError(w, "Provider ID required", http.StatusBadRequest)
			return
		}

		// GET /api/providers/{type}/models → listar modelos do provider
		if strings.HasSuffix(path, "/models") && r.Method == "GET" {
			providerType := strings.TrimSuffix(path, "/models")
			models := registry.ModelsForProvider(providerType)
			if models == nil {
				writeError(w, "Provider type not found: "+providerType, http.StatusNotFound)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{
				"provider_type": providerType,
				"models":        models,
			})
			return
		}

		id := path
		switch r.Method {
		case "PUT":
			updateProvider(w, r, db, id)
		case "DELETE":
			deleteProvider(w, db, id)
		default:
			writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Settings
	mux.HandleFunc("/api/settings", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			getSettings(w, db)
		case "PUT":
			updateSettings(w, r, db)
		default:
			writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Logs
	mux.HandleFunc("/api/logs", func(w http.ResponseWriter, r *http.Request) {
		listLogs(w, r, db)
	})

	// Stats (with optional period query param: 24h, 7d, 30d, 60d)
	mux.HandleFunc("/api/stats", func(w http.ResponseWriter, r *http.Request) {
		period := r.URL.Query().Get("period")
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(db.GetStatsForPeriod(period))
	})

	// Provider runtime status (CLI available, API categories)
	mux.HandleFunc("/api/providers/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(registry.ProviderStatuses())
	})

	// Dashboard authentication
	mux.HandleFunc("/api/auth/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		hasPassword := db.HasDashboardPassword()
		authenticated := !hasPassword
		if hasPassword {
			token := r.Header.Get("X-Dashboard-Token")
			authenticated = db.ValidateDashboardSession(token)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"has_password":  hasPassword,
			"authenticated": authenticated,
		})
	})

	mux.HandleFunc("/api/auth/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Password string `json:"password"`
		}
		if err := readJSON(r, &req); err != nil {
			writeError(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		if !db.ValidateDashboardPassword(req.Password) {
			writeError(w, "Invalid password", http.StatusUnauthorized)
			return
		}
		token := db.CreateDashboardSession()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"token": token})
	})

	mux.HandleFunc("/api/auth/password", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "PUT":
			var req struct {
				CurrentPassword string `json:"current_password"`
				NewPassword     string `json:"new_password"`
			}
			if err := readJSON(r, &req); err != nil {
				writeError(w, "Invalid JSON", http.StatusBadRequest)
				return
			}
			// If password exists, validate current
			if db.HasDashboardPassword() && !db.ValidateDashboardPassword(req.CurrentPassword) {
				writeError(w, "Current password is incorrect", http.StatusUnauthorized)
				return
			}
			if err := db.SetDashboardPassword(req.NewPassword); err != nil {
				writeError(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		case "DELETE":
			var req struct {
				CurrentPassword string `json:"current_password"`
			}
			if err := readJSON(r, &req); err != nil {
				writeError(w, "Invalid JSON", http.StatusBadRequest)
				return
			}
			if !db.ValidateDashboardPassword(req.CurrentPassword) {
				writeError(w, "Current password is incorrect", http.StatusUnauthorized)
				return
			}
			db.SetDashboardPassword("")
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		default:
			writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// Accounts (multi-provider, multi-account)
	registerAccountRoutes(mux, db, acctMgr, registry)
}

// --- API Keys ---

func listApiKeys(w http.ResponseWriter, db *database.DB) {
	keys, err := db.ListApiKeys()
	if err != nil {
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if keys == nil {
		keys = []database.ApiKey{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(keys)
}

func createApiKey(w http.ResponseWriter, r *http.Request, db *database.DB) {
	var req struct {
		Name string `json:"name"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		req.Name = "Unnamed Key"
	}

	key, err := db.CreateApiKey(req.Name)
	if err != nil {
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(key)
}

func toggleApiKey(w http.ResponseWriter, r *http.Request, db *database.DB, id string) {
	var req struct {
		IsActive bool `json:"is_active"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if err := db.ToggleApiKey(id, req.IsActive); err != nil {
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func deleteApiKey(w http.ResponseWriter, db *database.DB, id string) {
	if err := db.DeleteApiKey(id); err != nil {
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// --- Providers ---

func listProviders(w http.ResponseWriter, db *database.DB) {
	providers, err := db.ListProviders()
	if err != nil {
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if providers == nil {
		providers = []database.Provider{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(providers)
}

func createProvider(w http.ResponseWriter, r *http.Request, db *database.DB) {
	var req struct {
		Type   string `json:"type"`
		Name   string `json:"name"`
		Config string `json:"config"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	p, err := db.CreateProvider(req.Type, req.Name, req.Config)
	if err != nil {
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(p)
}

func updateProvider(w http.ResponseWriter, r *http.Request, db *database.DB, id string) {
	var req struct {
		Name     string `json:"name"`
		Config   string `json:"config"`
		IsActive bool   `json:"is_active"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if err := db.UpdateProvider(id, req.Name, req.Config, req.IsActive); err != nil {
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func deleteProvider(w http.ResponseWriter, db *database.DB, id string) {
	if err := db.DeleteProvider(id); err != nil {
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// --- Settings ---

func getSettings(w http.ResponseWriter, db *database.DB) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(db.GetSettings())
}

func updateSettings(w http.ResponseWriter, r *http.Request, db *database.DB) {
	var req map[string]string
	if err := readJSON(r, &req); err != nil {
		writeError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	for k, v := range req {
		db.SetSetting(k, v)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(db.GetSettings())
}

// --- Logs ---

func listLogs(w http.ResponseWriter, r *http.Request, db *database.DB) {
	limit := 50
	offset := 0
	if l := r.URL.Query().Get("limit"); l != "" {
		limit, _ = strconv.Atoi(l)
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		offset, _ = strconv.Atoi(o)
	}
	if limit > 200 {
		limit = 200
	}

	logs, total, err := db.ListLogs(limit, offset)
	if err != nil {
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if logs == nil {
		logs = []database.RequestLog{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"data":   logs,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// --- Helpers ---

func readJSON(r *http.Request, v any) error {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, v)
}
