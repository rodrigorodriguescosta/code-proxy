package database

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"log"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type DB struct {
	conn *sql.DB
}

func Open(path string) (*DB, error) {
	conn, err := sql.Open("sqlite", path+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	db := &DB{conn: conn}
	if err := db.migrate(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return db, nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}

func (db *DB) migrate() error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS api_keys (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			key_hash TEXT NOT NULL UNIQUE,
			is_active INTEGER DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS providers (
			id TEXT PRIMARY KEY,
			type TEXT NOT NULL,
			name TEXT NOT NULL,
			config TEXT DEFAULT '{}',
			is_active INTEGER DEFAULT 1,
			priority INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS request_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			api_key_id TEXT,
			provider TEXT,
			model TEXT,
			effort TEXT,
			input_tokens INTEGER DEFAULT 0,
			output_tokens INTEGER DEFAULT 0,
			duration_ms INTEGER DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_logs_created ON request_logs(created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_logs_provider ON request_logs(provider)`,
	}

	// Incremental migrations (ALTER TABLE - safe to re-run)
	alterMigrations := []string{
		`ALTER TABLE api_keys ADD COLUMN key_raw TEXT DEFAULT ''`,
		`ALTER TABLE request_logs ADD COLUMN account_id TEXT DEFAULT ''`,
		`ALTER TABLE request_logs ADD COLUMN estimated_cost REAL DEFAULT 0`,
	}
	for _, m := range alterMigrations {
		db.conn.Exec(m) // Ignore errors (column already exists)
	}

	for _, m := range migrations {
		if _, err := db.conn.Exec(m); err != nil {
			return fmt.Errorf("exec migration: %w\nSQL: %s", err, m)
		}
	}

	// Accounts table (multi-provider, multi-account)
	accountMigrations := []string{
		`CREATE TABLE IF NOT EXISTS accounts (
			id TEXT PRIMARY KEY,
			provider_type TEXT NOT NULL,
			label TEXT NOT NULL DEFAULT '',
			auth_mode TEXT NOT NULL DEFAULT 'none',
			access_token TEXT DEFAULT '',
			refresh_token TEXT DEFAULT '',
			api_key TEXT DEFAULT '',
			expires_at DATETIME,
			metadata TEXT DEFAULT '{}',
			is_active INTEGER DEFAULT 1,
			priority INTEGER DEFAULT 0,
			cooldown_until DATETIME,
			backoff_level INTEGER DEFAULT 0,
			last_error TEXT DEFAULT '',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_accounts_provider ON accounts(provider_type)`,
		`CREATE INDEX IF NOT EXISTS idx_accounts_active ON accounts(is_active)`,
		`CREATE TABLE IF NOT EXISTS model_cooldowns (
			account_id TEXT NOT NULL,
			model TEXT NOT NULL,
			cooldown_until DATETIME NOT NULL,
			backoff_level INTEGER DEFAULT 0,
			PRIMARY KEY (account_id, model)
		)`,
	}
	for _, m := range accountMigrations {
		if _, err := db.conn.Exec(m); err != nil {
			return fmt.Errorf("exec account migration: %w\nSQL: %s", err, m)
		}
	}

	// Dashboard sessions table
	db.conn.Exec(`CREATE TABLE IF NOT EXISTS dashboard_sessions (
		token TEXT PRIMARY KEY,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)

	// Combos table (model combos with fallback)
	db.conn.Exec(`CREATE TABLE IF NOT EXISTS combos (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL UNIQUE,
		models TEXT DEFAULT '[]',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)

	// Seed default provider if empty
	var count int
	db.conn.QueryRow("SELECT COUNT(*) FROM providers").Scan(&count)
	if count == 0 {
		db.conn.Exec(`INSERT INTO providers (id, type, name, is_active, priority) VALUES (?, ?, ?, 1, 0)`,
			generateID(), "claude", "Claude Code CLI")
		log.Println("[DB] Seeded default Claude provider")
	}

	return nil
}

// --- API Keys ---

func (db *DB) CreateApiKey(name string) (*ApiKey, error) {
	id := generateID()
	rawKey := "sk-" + generateRandom(32)
	keyHash := hashKey(rawKey)

	_, err := db.conn.Exec(
		`INSERT INTO api_keys (id, name, key_hash, key_raw, is_active) VALUES (?, ?, ?, ?, 1)`,
		id, name, keyHash, rawKey,
	)
	if err != nil {
		return nil, err
	}

	return &ApiKey{
		ID:        id,
		Name:      name,
		Key:       rawKey,
		IsActive:  true,
		CreatedAt: time.Now(),
	}, nil
}

func (db *DB) ListApiKeys() ([]ApiKey, error) {
	rows, err := db.conn.Query(`SELECT id, name, COALESCE(key_raw, ''), is_active, created_at FROM api_keys ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []ApiKey
	for rows.Next() {
		var k ApiKey
		if err := rows.Scan(&k.ID, &k.Name, &k.Key, &k.IsActive, &k.CreatedAt); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, nil
}

func (db *DB) ValidateApiKey(rawKey string) (string, bool) {
	h := hashKey(rawKey)
	var id string
	var active bool
	err := db.conn.QueryRow(`SELECT id, is_active FROM api_keys WHERE key_hash = ?`, h).Scan(&id, &active)
	if err != nil {
		return "", false
	}
	return id, active
}

func (db *DB) ToggleApiKey(id string, active bool) error {
	_, err := db.conn.Exec(`UPDATE api_keys SET is_active = ? WHERE id = ?`, active, id)
	return err
}

func (db *DB) DeleteApiKey(id string) error {
	_, err := db.conn.Exec(`DELETE FROM api_keys WHERE id = ?`, id)
	return err
}

func (db *DB) GetAllKeyHashes() []string {
	rows, err := db.conn.Query(`SELECT key_hash FROM api_keys WHERE is_active = 1`)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var hashes []string
	for rows.Next() {
		var h string
		rows.Scan(&h)
		hashes = append(hashes, h)
	}
	return hashes
}

// GetApiKeyInfo returns the key ID and name for a validated raw key
func (db *DB) GetApiKeyInfo(rawKey string) (id, name string, ok bool) {
	h := hashKey(rawKey)
	err := db.conn.QueryRow(`SELECT id, name FROM api_keys WHERE key_hash = ? AND is_active = 1`, h).Scan(&id, &name)
	if err != nil {
		return "", "", false
	}
	return id, name, true
}

// --- Providers ---

func (db *DB) ListProviders() ([]Provider, error) {
	rows, err := db.conn.Query(`SELECT id, type, name, config, is_active, priority, created_at FROM providers ORDER BY priority`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var providers []Provider
	for rows.Next() {
		var p Provider
		if err := rows.Scan(&p.ID, &p.Type, &p.Name, &p.Config, &p.IsActive, &p.Priority, &p.CreatedAt); err != nil {
			return nil, err
		}
		providers = append(providers, p)
	}
	return providers, nil
}

func (db *DB) CreateProvider(typ, name, config string) (*Provider, error) {
	id := generateID()
	var maxPriority int
	db.conn.QueryRow(`SELECT COALESCE(MAX(priority), -1) FROM providers`).Scan(&maxPriority)

	_, err := db.conn.Exec(
		`INSERT INTO providers (id, type, name, config, is_active, priority) VALUES (?, ?, ?, ?, 1, ?)`,
		id, typ, name, config, maxPriority+1,
	)
	if err != nil {
		return nil, err
	}

	return &Provider{
		ID:        id,
		Type:      typ,
		Name:      name,
		Config:    config,
		IsActive:  true,
		Priority:  maxPriority + 1,
		CreatedAt: time.Now(),
	}, nil
}

func (db *DB) UpdateProvider(id string, name string, config string, active bool) error {
	_, err := db.conn.Exec(
		`UPDATE providers SET name = ?, config = ?, is_active = ? WHERE id = ?`,
		name, config, active, id,
	)
	return err
}

func (db *DB) DeleteProvider(id string) error {
	_, err := db.conn.Exec(`DELETE FROM providers WHERE id = ?`, id)
	return err
}

// --- Settings ---

func (db *DB) GetSettings() Settings {
	s := Settings{
		DefaultModel:  "sonnet",
		LogRetention:  30,
		RequireApiKey: true,
	}

	rows, err := db.conn.Query(`SELECT key, value FROM settings`)
	if err != nil {
		return s
	}
	defer rows.Close()

	for rows.Next() {
		var k, v string
		rows.Scan(&k, &v)
		switch k {
		case "tunnel_enabled":
			s.TunnelEnabled = v == "true"
		case "tunnel_url":
			s.TunnelURL = v
		case "tunnel_token":
			s.TunnelToken = v
		case "default_model":
			s.DefaultModel = v
		case "log_retention_days":
			fmt.Sscanf(v, "%d", &s.LogRetention)
		case "require_api_key":
			s.RequireApiKey = v != "false"
		case "dashboard_password":
			if v != "" {
				s.DashboardPassword = "***"
			}
		}
	}
	return s
}

func (db *DB) SetSetting(key, value string) error {
	_, err := db.conn.Exec(
		`INSERT INTO settings (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = ?`,
		key, value, value,
	)
	return err
}

func (db *DB) GetSetting(key string) string {
	var v string
	db.conn.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&v)
	return v
}

// --- Dashboard Auth ---

func (db *DB) SetDashboardPassword(password string) error {
	if password == "" {
		return db.SetSetting("dashboard_password", "")
	}
	h := sha256.Sum256([]byte(password))
	hash := hex.EncodeToString(h[:])
	return db.SetSetting("dashboard_password", hash)
}

func (db *DB) ValidateDashboardPassword(password string) bool {
	stored := db.GetSetting("dashboard_password")
	if stored == "" {
		return true // No password set
	}
	h := sha256.Sum256([]byte(password))
	hash := hex.EncodeToString(h[:])
	return hash == stored
}

func (db *DB) HasDashboardPassword() bool {
	return db.GetSetting("dashboard_password") != ""
}

func (db *DB) CreateDashboardSession() string {
	token := generateRandom(32)
	db.conn.Exec(`INSERT INTO dashboard_sessions (token) VALUES (?)`, token)
	// Clean old sessions (older than 7 days)
	db.conn.Exec(`DELETE FROM dashboard_sessions WHERE created_at < datetime('now', '-7 days')`)
	return token
}

func (db *DB) ValidateDashboardSession(token string) bool {
	if token == "" {
		return false
	}
	var count int
	db.conn.QueryRow(`SELECT COUNT(*) FROM dashboard_sessions WHERE token = ? AND created_at > datetime('now', '-7 days')`, token).Scan(&count)
	return count > 0
}

// --- Request Logs ---

func (db *DB) LogRequest(apiKeyID, providerType, model, effort, accountID string, inputTokens, outputTokens int, estimatedCost float64, durationMs int64) {
	db.conn.Exec(
		`INSERT INTO request_logs (api_key_id, provider, model, effort, account_id, input_tokens, output_tokens, estimated_cost, duration_ms) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		apiKeyID, providerType, model, effort, accountID, inputTokens, outputTokens, estimatedCost, durationMs,
	)
}

func (db *DB) ListLogs(limit, offset int) ([]RequestLog, int, error) {
	var total int
	db.conn.QueryRow(`SELECT COUNT(*) FROM request_logs`).Scan(&total)

	rows, err := db.conn.Query(`
		SELECT l.id, l.api_key_id, COALESCE(k.name, 'unknown'), l.provider, l.model, l.effort,
			   COALESCE(l.account_id, ''), l.input_tokens, l.output_tokens,
			   COALESCE(l.estimated_cost, 0), l.duration_ms, l.created_at
		FROM request_logs l
		LEFT JOIN api_keys k ON l.api_key_id = k.id
		ORDER BY l.created_at DESC
		LIMIT ? OFFSET ?
	`, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var logs []RequestLog
	for rows.Next() {
		var l RequestLog
		if err := rows.Scan(&l.ID, &l.ApiKeyID, &l.ApiKeyName, &l.Provider, &l.Model, &l.Effort,
			&l.AccountID, &l.InputTokens, &l.OutputTokens, &l.EstimatedCost, &l.DurationMs, &l.CreatedAt); err != nil {
			return nil, 0, err
		}
		logs = append(logs, l)
	}
	return logs, total, nil
}

func (db *DB) GetStats() map[string]any {
	return db.GetStatsForPeriod("")
}

// GetStatsForPeriod returns stats filtered by period (24h, 7d, 30d, 60d, or empty for all time)
func (db *DB) GetStatsForPeriod(period string) map[string]any {
	stats := map[string]any{}

	whereClause := ""
	whereClauseAliased := "" // for queries with table alias "l"
	switch period {
	case "24h":
		whereClause = "WHERE created_at >= datetime('now', '-1 day')"
		whereClauseAliased = "WHERE l.created_at >= datetime('now', '-1 day')"
	case "7d":
		whereClause = "WHERE created_at >= datetime('now', '-7 days')"
		whereClauseAliased = "WHERE l.created_at >= datetime('now', '-7 days')"
	case "30d":
		whereClause = "WHERE created_at >= datetime('now', '-30 days')"
		whereClauseAliased = "WHERE l.created_at >= datetime('now', '-30 days')"
	case "60d":
		whereClause = "WHERE created_at >= datetime('now', '-60 days')"
		whereClauseAliased = "WHERE l.created_at >= datetime('now', '-60 days')"
	}

	// Total requests
	var totalRequests int
	db.conn.QueryRow(fmt.Sprintf(`SELECT COUNT(*) FROM request_logs %s`, whereClause)).Scan(&totalRequests)
	stats["total_requests"] = totalRequests

	// Today requests
	var todayRequests int
	db.conn.QueryRow(`SELECT COUNT(*) FROM request_logs WHERE created_at >= date('now')`).Scan(&todayRequests)
	stats["today_requests"] = todayRequests

	// Active keys/providers
	var totalKeys int
	db.conn.QueryRow(`SELECT COUNT(*) FROM api_keys WHERE is_active = 1`).Scan(&totalKeys)
	stats["active_keys"] = totalKeys

	var totalProviders int
	db.conn.QueryRow(`SELECT COUNT(*) FROM providers WHERE is_active = 1`).Scan(&totalProviders)
	stats["active_providers"] = totalProviders

	// Token totals (period-filtered)
	var totalInputTokens, totalOutputTokens int64
	db.conn.QueryRow(fmt.Sprintf(`SELECT COALESCE(SUM(input_tokens),0), COALESCE(SUM(output_tokens),0) FROM request_logs %s`, whereClause)).
		Scan(&totalInputTokens, &totalOutputTokens)
	stats["total_input_tokens"] = totalInputTokens
	stats["total_output_tokens"] = totalOutputTokens

	// Estimated cost (period-filtered)
	var estCost float64
	costRows, _ := db.conn.Query(fmt.Sprintf(`
		SELECT model, COALESCE(SUM(input_tokens),0), COALESCE(SUM(output_tokens),0), COALESCE(SUM(estimated_cost),0)
		FROM request_logs %s GROUP BY model
	`, whereClause))
	if costRows != nil {
		defer costRows.Close()
		for costRows.Next() {
			var m string
			var inTok, outTok int64
			var loggedCost float64
			costRows.Scan(&m, &inTok, &outTok, &loggedCost)
			if loggedCost > 0 {
				estCost += loggedCost
			} else {
				// Fallback calculation from tokens
				inRate, outRate := ModelCostRates(m)
				estCost += float64(inTok)/1_000_000*inRate + float64(outTok)/1_000_000*outRate
			}
		}
	}
	stats["estimated_cost"] = estCost

	// Top models (period-filtered)
	modelQuery := fmt.Sprintf(`
		SELECT model, COUNT(*) as cnt, COALESCE(SUM(input_tokens),0), COALESCE(SUM(output_tokens),0)
		FROM request_logs %s
		GROUP BY model ORDER BY cnt DESC LIMIT 10
	`, whereClause)
	rows, _ := db.conn.Query(modelQuery)
	if rows != nil {
		defer rows.Close()
		topModels := []map[string]any{}
		for rows.Next() {
			var model string
			var count int
			var inTok, outTok int64
			rows.Scan(&model, &count, &inTok, &outTok)
			topModels = append(topModels, map[string]any{
				"model":         model,
				"count":         count,
				"input_tokens":  inTok,
				"output_tokens": outTok,
			})
		}
		stats["top_models"] = topModels
	}

	// Per API key breakdown (period-filtered)
	keyQuery := fmt.Sprintf(`
		SELECT l.api_key_id, COALESCE(k.name, 'unknown'), COALESCE(k.key_raw, ''),
			   COUNT(*), COALESCE(SUM(l.input_tokens),0), COALESCE(SUM(l.output_tokens),0),
			   COALESCE(SUM(l.estimated_cost),0)
		FROM request_logs l
		LEFT JOIN api_keys k ON l.api_key_id = k.id
		%s
		GROUP BY l.api_key_id
		ORDER BY COUNT(*) DESC
	`, whereClause)
	keyRows, _ := db.conn.Query(keyQuery)
	if keyRows != nil {
		defer keyRows.Close()
		keyStats := []map[string]any{}
		for keyRows.Next() {
			var keyID, keyName, keyRaw string
			var cnt int
			var inTok, outTok int64
			var cost float64
			keyRows.Scan(&keyID, &keyName, &keyRaw, &cnt, &inTok, &outTok, &cost)
			masked := maskKey(keyRaw)
			keyStats = append(keyStats, map[string]any{
				"key_id":        keyID,
				"key_name":      keyName,
				"key_masked":    masked,
				"requests":      cnt,
				"input_tokens":  inTok,
				"output_tokens": outTok,
				"cost":          cost,
			})
		}
		stats["per_key"] = keyStats
	}

	// Per API key per model breakdown (period-filtered)
	keyModelQuery := fmt.Sprintf(`
		SELECT l.api_key_id, COALESCE(k.name, 'unknown'), l.model,
			   COUNT(*), COALESCE(SUM(l.input_tokens),0), COALESCE(SUM(l.output_tokens),0),
			   COALESCE(SUM(l.estimated_cost),0)
		FROM request_logs l
		LEFT JOIN api_keys k ON l.api_key_id = k.id
		%s
		GROUP BY l.api_key_id, l.model
		ORDER BY l.api_key_id, COUNT(*) DESC
	`, whereClause)
	kmRows, _ := db.conn.Query(keyModelQuery)
	if kmRows != nil {
		defer kmRows.Close()
		keyModels := []map[string]any{}
		for kmRows.Next() {
			var keyID, keyName, model string
			var cnt int
			var inTok, outTok int64
			var cost float64
			kmRows.Scan(&keyID, &keyName, &model, &cnt, &inTok, &outTok, &cost)
			keyModels = append(keyModels, map[string]any{
				"key_id":        keyID,
				"key_name":      keyName,
				"model":         model,
				"requests":      cnt,
				"input_tokens":  inTok,
				"output_tokens": outTok,
				"cost":          cost,
			})
		}
		stats["per_key_models"] = keyModels
	}

	// Recent requests (always last 20)
	recentRows, _ := db.conn.Query(`
		SELECT l.model, l.input_tokens, l.output_tokens, COALESCE(l.estimated_cost, 0),
			   l.duration_ms, l.created_at, COALESCE(k.name, ''), COALESCE(k.key_raw, ''),
			   l.provider, COALESCE(l.effort, '')
		FROM request_logs l
		LEFT JOIN api_keys k ON l.api_key_id = k.id
		ORDER BY l.created_at DESC LIMIT 20
	`)
	if recentRows != nil {
		defer recentRows.Close()
		recent := []map[string]any{}
		for recentRows.Next() {
			var m, keyName, keyRaw, prov, effort string
			var inTok, outTok int
			var cost float64
			var dur int64
			var t time.Time
			recentRows.Scan(&m, &inTok, &outTok, &cost, &dur, &t, &keyName, &keyRaw, &prov, &effort)
			recent = append(recent, map[string]any{
				"model":          m,
				"input_tokens":   inTok,
				"output_tokens":  outTok,
				"estimated_cost": cost,
				"duration_ms":    dur,
				"created_at":     t,
				"key_name":       keyName,
				"key_masked":     maskKey(keyRaw),
				"provider":       prov,
				"effort":         effort,
			})
		}
		stats["recent_requests"] = recent
	}

	return stats
}

// GetAccountUsageForPeriod returns aggregated usage per connected account.
// period accepts: 24h, 7d, 30d, 60d (or empty for all time).
func (db *DB) GetAccountUsageForPeriod(period string) []map[string]any {
	// Build a safe period condition for request_logs filtering.
	cond := "1=1"
	switch period {
	case "24h":
		cond = "l.created_at >= datetime('now', '-1 day')"
	case "7d":
		cond = "l.created_at >= datetime('now', '-7 days')"
	case "30d":
		cond = "l.created_at >= datetime('now', '-30 days')"
	case "60d":
		cond = "l.created_at >= datetime('now', '-60 days')"
	}

	rows, err := db.conn.Query(fmt.Sprintf(`
		SELECT
			a.id,
			a.provider_type,
			a.label,
			a.auth_mode,
			COALESCE(COUNT(l.id), 0) as requests,
			COALESCE(SUM(l.input_tokens), 0) as input_tokens,
			COALESCE(SUM(l.output_tokens), 0) as output_tokens,
			COALESCE(SUM(l.estimated_cost), 0) as estimated_cost,
			MAX(l.created_at) as last_used_at
		FROM accounts a
		LEFT JOIN request_logs l
		  ON l.account_id = a.id AND %s
		WHERE a.is_active = 1
		GROUP BY a.id, a.provider_type, a.label, a.auth_mode
		ORDER BY requests DESC, a.created_at DESC
	`, cond))
	if err != nil {
		return []map[string]any{}
	}
	defer rows.Close()

	var res []map[string]any
	for rows.Next() {
		var accountID, providerType, label, authMode string
		var requests int
		var inTok, outTok int64
		var cost float64
		var lastUsed *time.Time
		if err := rows.Scan(&accountID, &providerType, &label, &authMode, &requests, &inTok, &outTok, &cost, &lastUsed); err != nil {
			continue
		}
		var lastUsedISO string
		if lastUsed != nil {
			lastUsedISO = lastUsed.UTC().Format(time.RFC3339)
		}
		res = append(res, map[string]any{
			"account_id":     accountID,
			"provider_type":  providerType,
			"label":          label,
			"auth_mode":      authMode,
			"requests":       requests,
			"input_tokens":  inTok,
			"output_tokens": outTok,
			"estimated_cost": cost,
			"last_used_at":  lastUsedISO,
		})
	}
	return res
}

// GetAccountRecentRequests returns most recent request logs for an account.
func (db *DB) GetAccountRecentRequests(accountID string, limit int, period string) []map[string]any {
	if limit <= 0 {
		limit = 20
	}

	cond := "1=1"
	switch period {
	case "24h":
		cond = "l.created_at >= datetime('now', '-1 day')"
	case "7d":
		cond = "l.created_at >= datetime('now', '-7 days')"
	case "30d":
		cond = "l.created_at >= datetime('now', '-30 days')"
	case "60d":
		cond = "l.created_at >= datetime('now', '-60 days')"
	}

	rows, err := db.conn.Query(fmt.Sprintf(`
		SELECT
			l.model,
			l.input_tokens,
			l.output_tokens,
			COALESCE(l.estimated_cost, 0) as estimated_cost,
			l.duration_ms,
			l.created_at,
			COALESCE(k.name, '') as key_name,
			COALESCE(k.key_raw, '') as key_raw,
			l.provider,
			COALESCE(l.effort, '') as effort
		FROM request_logs l
		LEFT JOIN api_keys k ON l.api_key_id = k.id
		WHERE l.account_id = ? AND %s
		ORDER BY l.created_at DESC
		LIMIT ?
	`, cond), accountID, limit)
	if err != nil {
		return []map[string]any{}
	}
	defer rows.Close()

	var res []map[string]any
	for rows.Next() {
		var model, keyName, keyRaw, provider, effort string
		var inTok, outTok int
		var cost float64
		var dur int64
		var t time.Time
		if err := rows.Scan(&model, &inTok, &outTok, &cost, &dur, &t, &keyName, &keyRaw, &provider, &effort); err != nil {
			continue
		}
		res = append(res, map[string]any{
			"model":           model,
			"input_tokens":   inTok,
			"output_tokens":  outTok,
			"estimated_cost": cost,
			"duration_ms":    dur,
			"created_at":      t.Format(time.RFC3339),
			"key_name":       keyName,
			"key_masked":     maskKey(keyRaw),
			"provider":       provider,
			"effort":         effort,
		})
	}
	return res
}

func (db *DB) CleanOldLogs(retentionDays int) {
	db.conn.Exec(`DELETE FROM request_logs WHERE created_at < datetime('now', ?)`,
		fmt.Sprintf("-%d days", retentionDays))
}

// --- Helpers ---

func generateID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func generateRandom(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func hashKey(key string) string {
	h := sha256.Sum256([]byte(key))
	return hex.EncodeToString(h[:])
}

// maskKey returns first 6 + "..." + last 4 chars of a key
func maskKey(key string) string {
	if key == "" {
		return ""
	}
	if len(key) <= 12 {
		return key[:3] + "..."
	}
	return key[:6] + "..." + key[len(key)-4:]
}

// ModelCostRates returns input and output cost per million tokens for a model
func ModelCostRates(model string) (inputRate, outputRate float64) {
	lower := strings.ToLower(model)
	switch {
	case strings.Contains(lower, "opus"):
		return 15.0, 75.0
	case strings.Contains(lower, "haiku"):
		return 0.25, 1.25
	case strings.Contains(lower, "gpt-5.3"), strings.Contains(lower, "gpt-5.2"):
		return 10.0, 30.0
	case strings.Contains(lower, "gpt-5.1"), strings.Contains(lower, "gpt-5"):
		return 5.0, 15.0
	case strings.Contains(lower, "gpt-4o-mini"), strings.Contains(lower, "4.1-mini"), strings.Contains(lower, "4.1-nano"):
		return 0.15, 0.60
	case strings.Contains(lower, "gpt-4o"), strings.Contains(lower, "gpt-4.1"):
		return 2.5, 10.0
	case strings.Contains(lower, "o3-mini"), strings.Contains(lower, "o4-mini"):
		return 1.1, 4.4
	case strings.Contains(lower, "o3"), strings.Contains(lower, "o1"):
		return 10.0, 40.0
	case strings.Contains(lower, "gemini-2.5-pro"):
		return 1.25, 10.0
	case strings.Contains(lower, "gemini"):
		return 0.15, 0.60
	case strings.Contains(lower, "deepseek"):
		return 0.14, 0.28
	default:
		return 3.0, 15.0 // default sonnet rates
	}
}
