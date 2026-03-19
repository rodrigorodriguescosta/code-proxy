# Code Proxy

A unified proxy that lets you use **Claude Code**, **OpenAI Codex**, **Gemini** and other AI coding agents through any OpenAI-compatible client — like **Cursor**, VS Code, or any tool that speaks the OpenAI API.

## Why Code Proxy?

AI coding tools offer two very different modes:

| Mode | What happens | Agent | Best for |
|------|-------------|-------|----------|
| **CLI (subscription)** | Routes through `claude` / `codex` binary | Full code agent (reads files, runs commands, edits code) | Complex tasks — the AI *does* the work |
| **API (pay-per-token)** | Direct HTTP to provider API | Chat-only — answers questions | Quick questions, code review |

**Code Proxy exposes both modes as a single OpenAI-compatible API**, so Cursor (or any client) can use either one — just by picking a model name.

### Key benefits

- **Centralize subscriptions** — connect multiple Claude/Codex/Gemini accounts and share them across your team
- **Use Cursor's chat with your subscription** — Claude Code and Codex subscriptions include the code agent; Code Proxy lets Cursor leverage that agent instead of consuming API tokens
- **Track usage** — see token counts, costs, and request history per model, per account, per API key
- **Control access** — create API keys, set quotas, revoke access instantly
- **One endpoint, all providers** — Claude, GPT, Gemini, DeepSeek, Groq, Together, Ollama — all through `http://localhost:3456/v1`

## How it works

```
Cursor / VS Code / any client
        │
        ▼
   Code Proxy (:3456)
   ┌─────────────────────────┐
   │  OpenAI-compatible API  │
   │  /v1/chat/completions   │
   │  /v1/models             │
   └────────┬────────────────┘
            │ routes by model prefix
            ├── cc/*        → Claude CLI agent (subscription)
            ├── codex/*     → Codex CLI agent (subscription)
            ├── gc/*        → Gemini CLI
            ├── anthropic/* → Anthropic Messages API (pay-per-token)
            ├── openai/*    → OpenAI API (pay-per-token)
            ├── gemini/*    → Gemini API (pay-per-token)
            ├── deepseek/*  → DeepSeek API
            ├── groq/*      → Groq API
            └── generic/*   → Any OpenAI-compatible endpoint
```

### CLI vs API — what's the difference?

When you select a **CLI model** (e.g. `cc/claude-sonnet-4-6`), Code Proxy spawns the actual `claude` or `codex` binary on your machine. This means:

- The request uses your **subscription** (Claude Max, Codex Pro, etc.) — not API tokens
- The AI runs as a **full code agent** — it can read your files, run commands, search your codebase
- Cursor acts as the interface, but the agent doing the work is Claude Code or Codex

When you select an **API model** (e.g. `anthropic/claude-sonnet-4-6`), Code Proxy sends the request directly to the provider's HTTP API:

- Uses **API keys** or **OAuth tokens** — pay-per-token pricing
- Chat-only — no file access, no command execution
- Lower latency for quick questions

## Installation

### From source

```bash
git clone https://github.com/rodrigorodriguescosta/code-proxy.git
cd code-proxy
go build -o code-proxy .
./code-proxy
```

### Using `go install`

```bash
go install github.com/rodrigorodriguescosta/code-proxy@latest
code-proxy
```

### Download binary

Grab the latest release for your platform from [Releases](https://github.com/rodrigorodriguescosta/code-proxy/releases).

```bash
# Linux/macOS
chmod +x code-proxy
./code-proxy

# Or move to your PATH
sudo mv code-proxy /usr/local/bin/
```

### Deploy on a VPS

Code Proxy is a single binary with an embedded SQLite database — no external dependencies.

```bash
# 1. Download
wget https://github.com/rodrigorodriguescosta/code-proxy/releases/latest/download/code-proxy-linux-amd64
chmod +x code-proxy-linux-amd64

# 2. Configure (optional)
export PORT=3456
export DATA_DIR=/var/lib/code-proxy
export PROXY_REQUIRE_API_KEY=true

# 3. Run
./code-proxy-linux-amd64
```

#### Systemd service (recommended for VPS)

```ini
# /etc/systemd/system/code-proxy.service
[Unit]
Description=Code Proxy
After=network.target

[Service]
Type=simple
User=code-proxy
ExecStart=/usr/local/bin/code-proxy
Environment=PORT=3456
Environment=DATA_DIR=/var/lib/code-proxy
Environment=PROXY_REQUIRE_API_KEY=true
Restart=always

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl enable --now code-proxy
```

#### Expose via Cloudflare Tunnel

Code Proxy has built-in Cloudflare tunnel support — no need for nginx or port forwarding.

**Quick Tunnel** (random URL — changes on every restart):

1. Open the dashboard → Tunnel → Enable Tunnel
2. A random `*.trycloudflare.com` URL is generated
3. Use this URL in Cursor — but note it **changes on every restart**

**Named Tunnel** (fixed URL — recommended):

The Quick Tunnel URL changes every time the tunnel restarts, which means you have to update Cursor settings constantly. For a **permanent URL**, use a Cloudflare Named Tunnel:

1. Go to [Cloudflare Zero Trust](https://one.dash.cloudflare.com) → Networks → Tunnels
2. Create a tunnel — name it "code-proxy" or similar
3. Add a public hostname (e.g. `proxy.yourdomain.com`) pointing to `http://localhost:3456`
4. Copy the tunnel token (starts with `eyJ...`)
5. In the dashboard → Tunnel → Configure Token → paste the token
6. Enable the tunnel — your fixed URL is now active

This way Cursor always points to `https://proxy.yourdomain.com/v1` and it never changes.

## Quick start with Cursor

> **Important:** Cursor does **not** support `localhost` as the OpenAI Base URL. You must expose Code Proxy via a **public URL** — either a VPS with a public IP, a Cloudflare tunnel, or any reverse proxy/tunnel solution. See [Tunnel setup](#expose-via-cloudflare-tunnel) below.

1. Start Code Proxy:
   ```bash
   code-proxy
   ```
2. Open the dashboard at `http://localhost:3456` and connect your accounts (OAuth or API key)
3. Enable a tunnel (Dashboard → Tunnel) or deploy on a VPS to get a public URL
4. In Cursor → Settings → Models → OpenAI API Key: enter any value (or a Code Proxy API key if you enabled `require_api_key`)
5. Set the Base URL to `https://your-public-url/v1`
6. Pick a model — use the prefix to choose the provider:

| Model ID | Provider | Mode |
|----------|----------|------|
| `cc/claude-sonnet-4-6` | Claude Code CLI | Subscription agent |
| `cc/claude-opus-4-6:max` | Claude Code CLI | Subscription (max effort) |
| `codex/5.3` | Codex CLI | Subscription agent |
| `anthropic/claude-sonnet-4-6` | Anthropic API | Pay-per-token |
| `openai/gpt-5.3` | OpenAI API | Pay-per-token |
| `gemini/gemini-2.5-pro` | Gemini API | Pay-per-token |
| `deepseek/deepseek-chat` | DeepSeek API | Pay-per-token |

### Effort levels

Append `:low`, `:medium`, `:high`, or `:max` to CLI models to control token usage:

```
cc/claude-sonnet-4-6:low    → fast, minimal tokens
cc/claude-sonnet-4-6:max    → thorough, more tokens
```

## Dashboard

The web dashboard at `http://localhost:3456` provides:

- **Providers** — connect accounts via OAuth or API key, see available models
- **Usage stats** — requests, tokens, and estimated costs over time
- **Account usage** — per-account breakdown of consumption
- **Request logs** — full history with model, tokens, cost, and duration
- **API keys** — create, toggle, and revoke access keys
- **Settings** — default model, tunnel, dashboard password, require API key

## Configuration

All settings are via environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `3456` | HTTP port |
| `DATA_DIR` | `~/.code-proxy` | SQLite database location |
| `CLAUDE_MODEL` | `sonnet` | Default model when none specified |
| `WORK_DIR` | current dir | Working directory for CLI agents |
| `PROXY_API_KEY` | — | Comma-separated fallback API keys |
| `PROXY_REQUIRE_API_KEY` | `false` | Require API key for all requests |
| `USE_ACP` | `false` | Use ACP protocol instead of CLI exec |
| `ACP_COMMAND` | — | Path to ACP subprocess binary |

## Multi-account & rotation

Connect multiple accounts of the same provider. Code Proxy will:

- **Round-robin** across active accounts
- **Auto-cooldown** accounts that hit rate limits (exponential backoff)
- **Refresh OAuth tokens** automatically before they expire (checks every 5 minutes)
- **Skip inactive** or expired accounts

This lets you pool subscriptions across a team without sharing credentials.

## API

Code Proxy implements the OpenAI Chat Completions API:

```bash
# Chat completion (streaming)
curl http://localhost:3456/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer your-api-key" \
  -d '{
    "model": "cc/claude-sonnet-4-6",
    "messages": [{"role": "user", "content": "Hello"}],
    "stream": true
  }'

# List models
curl http://localhost:3456/v1/models
```

## Security

- **Dashboard password** — optional password protection for the web UI
- **API key enforcement** — require Bearer token for all `/v1/*` requests
- **No credentials exposed** — API keys are stored hashed; OAuth tokens are encrypted at rest
- **CORS headers** — configurable for cross-origin access

## Prerequisites

For **CLI providers** (subscription mode), you need the respective CLI tool installed:

- **Claude Code**: `npm install -g @anthropic-ai/claude-code` and authenticate with `claude`
- **Codex**: `npm install -g @openai/codex` and authenticate with `codex`
- **Gemini CLI**: install and authenticate per Google's instructions

For **API providers**, you just need an API key or OAuth credentials — no CLI required.

## Author

Created by **Rodrigo Rodrigues** ([@rodrigorodriguescosta](https://github.com/rodrigorodriguescosta)).

## License

MIT
