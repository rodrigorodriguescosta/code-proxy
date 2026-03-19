# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Is

Code Proxy is a self-contained OpenAI-compatible API proxy that routes requests to CLI agents (Claude Code, Codex, Gemini CLI) or API providers (Anthropic, OpenAI, Gemini, DeepSeek, Groq, generic). It's designed for use with Cursor and similar AI coding tools. Single Go binary with embedded Vue 3 frontend and SQLite database.

## Build & Development Commands

```bash
make build            # Full build: frontend → embedded in Go binary
make frontend         # Build Vue app only (output: backend/embed/dist/)
make backend          # Compile Go binary only
make dev              # Backend dev with air (hot reload)
make dev-frontend     # Frontend dev server (Vite, proxies /api → :3456)
make clean            # Remove build artifacts
```

The frontend builds into `backend/embed/dist/` and is embedded into the Go binary via `//go:embed` in `backend/embed/embed.go`. Always run `make frontend` before `make backend` (or just use `make build`).

No test suite exists in this project.

## Architecture

### Request Flow

```
Client (Cursor) → POST /v1/chat/completions
  → router.go: auth middleware (API key check if enabled)
  → chat.go: parse model prefix + effort suffix (e.g. "cc/claude-sonnet-4-6:max")
  → provider/registry.go: resolve prefix → provider
  → account/manager.go: select account (round-robin, respects cooldowns)
  → provider.Execute(): CLI exec or HTTP API call
  → SSE stream back to client
  → database: log request (tokens, cost)
```

### Model Prefix Routing

Models are addressed as `{prefix}/{model}:{effort}`. The prefix determines which provider handles the request:
- `cc/*` → Claude CLI, `codex/*` → Codex CLI, `gc/*` → Gemini CLI
- `anthropic/*` → Anthropic API, `openai/*` → OpenAI API, `gemini/*` → Gemini API
- `deepseek/*`, `groq/*`, `generic/*` → respective API providers

### Backend (Go)

Key modules under `backend/modules/`:
- **api/** — HTTP router, middleware, dashboard endpoints, chat completions handler
- **provider/** — Provider interface + implementations (CLI and API). `registry.go` maps prefixes to providers. `provider.go` defines the `Provider` interface (Name, Models, Execute, Category, IsAvailable).
- **account/** — Account selection strategy (round-robin/fill-first) and exponential backoff cooldown on rate limits
- **auth/** — OAuth2+PKCE flows for Claude/Codex/Gemini/GitHub Copilot. `configs.go` has provider OAuth configs. `callback.go` runs a local HTTP server to capture OAuth redirects.
- **database/** — SQLite with WAL mode. Auto-migrations on startup in `sqlite.go`. Tables: `accounts`, `api_keys`, `request_logs`, `settings`, `dashboard_sessions`, `model_cooldowns`.
- **config/** — Environment variable loading
- **tunnel/** — Cloudflare tunnel management

Entry point: `backend/main.go` — initializes DB, registers providers, starts HTTP server on port 3456.

### Frontend (Vue 3 + Vite + Tailwind CSS)

SPA with views in `frontend/src/views/`. API client in `frontend/src/api.js` uses `X-Dashboard-Token` header for dashboard auth. Routes defined inline in `frontend/src/main.js`.

Dashboard password is optional. When set, session tokens are stored in localStorage as `dashboard_token`.

### Key Environment Variables

| Variable | Default | Purpose |
|----------|---------|---------|
| `PORT` | `3456` | Server port |
| `DATA_DIR` | `~/.code-proxy` | SQLite DB location |
| `CLAUDE_MODEL` | `sonnet` | Default Claude model |
| `PROXY_REQUIRE_API_KEY` | `false` | Require API key for proxy requests |
| `USE_ACP` | `false` | Use ACP protocol instead of CLI exec |

### Auth Modes

Accounts can authenticate via:
1. **OAuth** (auth_mode: "oauth") — tokens auto-refresh every 5 minutes
2. **API Key** (auth_mode: "apikey") — stored directly, no refresh needed
3. **CLI detection** — checks if CLI binary exists locally
