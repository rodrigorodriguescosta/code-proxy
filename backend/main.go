package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"code-proxy/embed"
	"code-proxy/modules/account"
	"code-proxy/modules/api"
	"code-proxy/modules/config"
	"code-proxy/modules/database"
	"code-proxy/modules/provider"
	"code-proxy/modules/tunnel"
)

func main() {
	cfg := config.Load()

	// API keys from env (comma-separated)
	var envApiKeys []string
	if keys := os.Getenv("PROXY_API_KEY"); keys != "" {
		for _, k := range strings.Split(keys, ",") {
			k = strings.TrimSpace(k)
			if k != "" {
				envApiKeys = append(envApiKeys, k)
			}
		}
	}

	// Open SQLite database
	db, err := database.Open(cfg.DBPath)
	if err != nil {
		log.Printf("[MAIN] WARNING: Failed to open database: %v (running without persistence)", err)
	} else {
		defer db.Close()
		log.Printf("[MAIN] Database: %s", cfg.DBPath)
	}

	// Provider registry
	registry := provider.NewRegistry()

	// Register Claude CLI provider
	if cfg.UseACP {
		log.Printf("[MAIN] Mode: ACP (command: %s)", cfg.ACPCommand)
		registry.Register("claude-cli", provider.NewClaudeACP(cfg.WorkDir, cfg.ACPCommand, cfg.ACPArgs))
	} else {
		log.Println("[MAIN] Mode: CLI (exec claude)")
		registry.Register("claude-cli", provider.NewClaude(cfg.WorkDir))
	}

	// Register API providers (accounts added via dashboard)
	registry.Register("anthropic-api", provider.NewAnthropicAPI())
	registry.Register("openai-api", provider.NewOpenAIAPI())
	registry.Register("gemini-api", provider.NewGeminiAPI())
	registry.Register("generic-openai", provider.NewGenericOpenAI())

	// Register additional CLI providers
	registry.Register("codex-cli", provider.NewCodex(cfg.WorkDir))
	registry.Register("gemini-cli", provider.NewGeminiCLI(cfg.WorkDir))

	// Account manager
	acctMgr := account.NewManager(db)

	// Background token refresh (every 5 minutes)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go acctMgr.RefreshLoop(ctx, 5*time.Minute, nil) // refreshFn is configured once OAuth is ready

	// Tunnel manager
	tunnelMgr := tunnel.NewManager(cfg.Port, cfg.DataDir, func(url string) {
		if db != nil {
			db.SetSetting("tunnel_url", url)
		}
	}, func(enabled bool) {
		if db != nil {
			if enabled {
				db.SetSetting("tunnel_enabled", "true")
			} else {
				db.SetSetting("tunnel_enabled", "false")
			}
		}
	})

	// Create server
	frontendFS := embed.FS()
	server := api.NewServer(api.ServerOptions{
		Registry:     registry,
		AccountMgr:   acctMgr,
		DefaultModel: cfg.DefaultModel,
		EnvApiKeys:   envApiKeys,
		DB:           db,
		Tunnel:       tunnelMgr,
		FrontendFS:   frontendFS,
	})

	if len(envApiKeys) == 0 && db != nil {
		keys, _ := db.ListApiKeys()
		if len(keys) == 0 {
			log.Println("[MAIN] WARNING: No API keys configured. API is open to all requests.")
			log.Println("[MAIN] Create a key via dashboard or set PROXY_API_KEY env var.")
		}
	}

	// Auto-start tunnel if it was enabled before shutdown
	if db != nil {
		settings := db.GetSettings()
		tunnelMgr.AutoStart(settings.TunnelEnabled, settings.TunnelToken)
	}

	log.Printf("[MAIN] Code Proxy listening on :%s", cfg.Port)
	log.Printf("[MAIN] Default model: %s, Providers: %v", cfg.DefaultModel, registry.ListProviders())
	log.Println("[MAIN] Endpoints:")
	log.Println("[MAIN]   POST /v1/chat/completions  (OpenAI-compatible)")
	log.Println("[MAIN]   GET  /v1/models")
	log.Println("[MAIN]   GET  /health")
	if frontendFS != nil {
		log.Printf("[MAIN]   GET  /                      (Dashboard)")
	}
	log.Println("[MAIN]   GET  /api/keys, /api/providers, /api/accounts, /api/settings, /api/logs, /api/stats")
	log.Println("[MAIN]   POST /api/tunnel/enable, /api/tunnel/disable, /api/tunnel/status")

	if err := http.ListenAndServe(":"+cfg.Port, server.Handler()); err != nil {
		log.Fatal(err)
	}
}
