package database

import (
	"encoding/json"
	"fmt"
	"time"
)

// ExportData represents the full exportable state of the database
type ExportData struct {
	Version   int              `json:"version"`
	ExportedAt string          `json:"exported_at"`
	Accounts  []Account        `json:"accounts"`
	ApiKeys   []ApiKeyExport   `json:"api_keys"`
	Settings  map[string]string `json:"settings"`
	Logs      []RequestLog     `json:"logs,omitempty"`
}

// ApiKeyExport includes raw key for portability
type ApiKeyExport struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	KeyRaw    string    `json:"key_raw"`
	KeyHash   string    `json:"key_hash"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

// Export returns all data as a portable JSON-serializable struct.
// If includeLogs is true, request_logs are included (can be large).
func (db *DB) Export(includeLogs bool) (*ExportData, error) {
	data := &ExportData{
		Version:    1,
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
	}

	// Accounts
	accounts, err := db.ListAccounts("")
	if err != nil {
		return nil, fmt.Errorf("export accounts: %w", err)
	}
	data.Accounts = accounts

	// API keys (with raw key for re-import)
	rows, err := db.conn.Query(`SELECT id, name, COALESCE(key_raw, ''), key_hash, is_active, created_at FROM api_keys ORDER BY created_at`)
	if err != nil {
		return nil, fmt.Errorf("export api_keys: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var k ApiKeyExport
		if err := rows.Scan(&k.ID, &k.Name, &k.KeyRaw, &k.KeyHash, &k.IsActive, &k.CreatedAt); err != nil {
			return nil, err
		}
		data.ApiKeys = append(data.ApiKeys, k)
	}

	// Settings (all key-value pairs, including dashboard_password hash)
	data.Settings = make(map[string]string)
	srows, err := db.conn.Query(`SELECT key, value FROM settings`)
	if err != nil {
		return nil, fmt.Errorf("export settings: %w", err)
	}
	defer srows.Close()
	for srows.Next() {
		var k, v string
		srows.Scan(&k, &v)
		data.Settings[k] = v
	}

	// Logs (optional)
	if includeLogs {
		lrows, err := db.conn.Query(`
			SELECT id, COALESCE(api_key_id,''), '', provider, model, COALESCE(effort,''),
			       COALESCE(account_id,''), input_tokens, output_tokens,
			       COALESCE(estimated_cost,0), duration_ms, created_at
			FROM request_logs ORDER BY created_at
		`)
		if err != nil {
			return nil, fmt.Errorf("export logs: %w", err)
		}
		defer lrows.Close()
		for lrows.Next() {
			var l RequestLog
			if err := lrows.Scan(&l.ID, &l.ApiKeyID, &l.ApiKeyName, &l.Provider, &l.Model, &l.Effort,
				&l.AccountID, &l.InputTokens, &l.OutputTokens, &l.EstimatedCost, &l.DurationMs, &l.CreatedAt); err != nil {
				return nil, err
			}
			data.Logs = append(data.Logs, l)
		}
	}

	return data, nil
}

// Import restores data from an ExportData struct.
// mode: "merge" (skip existing IDs) or "replace" (wipe + insert).
func (db *DB) Import(data *ExportData, mode string) (*ImportResult, error) {
	if data.Version != 1 {
		return nil, fmt.Errorf("unsupported export version: %d", data.Version)
	}

	result := &ImportResult{}

	if mode == "replace" {
		db.conn.Exec(`DELETE FROM accounts`)
		db.conn.Exec(`DELETE FROM api_keys`)
		db.conn.Exec(`DELETE FROM settings`)
		db.conn.Exec(`DELETE FROM request_logs`)
	}

	// Import accounts
	for _, a := range data.Accounts {
		metadataJSON := "{}"
		if a.Metadata != nil {
			b, _ := json.Marshal(a.Metadata)
			metadataJSON = string(b)
		}
		var expAt interface{}
		if a.ExpiresAt != nil {
			expAt = *a.ExpiresAt
		}

		res, err := db.conn.Exec(
			`INSERT OR IGNORE INTO accounts (id, provider_type, label, auth_mode, access_token, refresh_token, api_key, expires_at, metadata, is_active, priority, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			a.ID, a.ProviderType, a.Label, a.AuthMode, a.AccessToken, a.RefreshToken, a.APIKey, expAt, metadataJSON, a.IsActive, a.Priority, a.CreatedAt, a.UpdatedAt,
		)
		if err == nil {
			if n, _ := res.RowsAffected(); n > 0 {
				result.AccountsImported++
			} else {
				result.AccountsSkipped++
			}
		}
	}

	// Import API keys
	for _, k := range data.ApiKeys {
		hash := k.KeyHash
		if hash == "" && k.KeyRaw != "" {
			hash = hashKey(k.KeyRaw)
		}
		res, err := db.conn.Exec(
			`INSERT OR IGNORE INTO api_keys (id, name, key_hash, key_raw, is_active, created_at)
			 VALUES (?, ?, ?, ?, ?, ?)`,
			k.ID, k.Name, hash, k.KeyRaw, k.IsActive, k.CreatedAt,
		)
		if err == nil {
			if n, _ := res.RowsAffected(); n > 0 {
				result.ApiKeysImported++
			} else {
				result.ApiKeysSkipped++
			}
		}
	}

	// Import settings
	for key, value := range data.Settings {
		if mode == "merge" {
			// Only set if not already present
			existing := db.GetSetting(key)
			if existing != "" {
				result.SettingsSkipped++
				continue
			}
		}
		db.SetSetting(key, value)
		result.SettingsImported++
	}

	// Import logs
	for _, l := range data.Logs {
		res, err := db.conn.Exec(
			`INSERT OR IGNORE INTO request_logs (id, api_key_id, provider, model, effort, account_id, input_tokens, output_tokens, estimated_cost, duration_ms, created_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			l.ID, l.ApiKeyID, l.Provider, l.Model, l.Effort, l.AccountID, l.InputTokens, l.OutputTokens, l.EstimatedCost, l.DurationMs, l.CreatedAt,
		)
		if err == nil {
			if n, _ := res.RowsAffected(); n > 0 {
				result.LogsImported++
			} else {
				result.LogsSkipped++
			}
		}
	}

	return result, nil
}

// ImportResult summarizes what was imported
type ImportResult struct {
	AccountsImported int `json:"accounts_imported"`
	AccountsSkipped  int `json:"accounts_skipped"`
	ApiKeysImported  int `json:"api_keys_imported"`
	ApiKeysSkipped   int `json:"api_keys_skipped"`
	SettingsImported int `json:"settings_imported"`
	SettingsSkipped  int `json:"settings_skipped"`
	LogsImported     int `json:"logs_imported"`
	LogsSkipped      int `json:"logs_skipped"`
}
