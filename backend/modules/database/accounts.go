package database

import (
	"encoding/json"
	"time"
)

// Account represents an authenticated credential stored in the database
type Account struct {
	ID           string            `json:"id"`
	ProviderType string            `json:"provider_type"`
	Label        string            `json:"label"`
	AuthMode     string            `json:"auth_mode"`
	AccessToken  string            `json:"access_token,omitempty"`
	RefreshToken string            `json:"refresh_token,omitempty"`
	APIKey       string            `json:"api_key,omitempty"`
	ExpiresAt    *time.Time        `json:"expires_at,omitempty"`
	Metadata     map[string]string `json:"metadata,omitempty"`
	IsActive     bool              `json:"is_active"`
	Priority     int               `json:"priority"`
	CooldownUntil *time.Time       `json:"cooldown_until,omitempty"`
	BackoffLevel int               `json:"backoff_level"`
	LastError    string            `json:"last_error,omitempty"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

// CreateAccount creates a new account in the database
func (db *DB) CreateAccount(providerType, label, authMode string) (*Account, error) {
	id := generateID()
	now := time.Now()

	_, err := db.conn.Exec(
		`INSERT INTO accounts (id, provider_type, label, auth_mode, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		id, providerType, label, authMode, now, now,
	)
	if err != nil {
		return nil, err
	}

	return &Account{
		ID:           id,
		ProviderType: providerType,
		Label:        label,
		AuthMode:     authMode,
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

// CreateAccountFull creates an account with all fields set
func (db *DB) CreateAccountFull(providerType, label, authMode, accessToken, refreshToken, apiKey string, expiresAt *time.Time, metadata map[string]string) (*Account, error) {
	id := generateID()
	now := time.Now()

	metadataJSON := "{}"
	if metadata != nil {
		b, _ := json.Marshal(metadata)
		metadataJSON = string(b)
	}

	var expAt interface{}
	if expiresAt != nil {
		expAt = *expiresAt
	}

	_, err := db.conn.Exec(
		`INSERT INTO accounts (id, provider_type, label, auth_mode, access_token, refresh_token, api_key, expires_at, metadata, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, providerType, label, authMode, accessToken, refreshToken, apiKey, expAt, metadataJSON, now, now,
	)
	if err != nil {
		return nil, err
	}

	return &Account{
		ID:           id,
		ProviderType: providerType,
		Label:        label,
		AuthMode:     authMode,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		APIKey:       apiKey,
		ExpiresAt:    expiresAt,
		Metadata:     metadata,
		IsActive:     true,
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

// UpdateAccountTokens updates the tokens of an account
func (db *DB) UpdateAccountTokens(id, accessToken, refreshToken string, expiresAt *time.Time) error {
	var expAt interface{}
	if expiresAt != nil {
		expAt = *expiresAt
	}

	_, err := db.conn.Exec(
		`UPDATE accounts SET access_token = ?, refresh_token = ?, expires_at = ?, updated_at = ? WHERE id = ?`,
		accessToken, refreshToken, expAt, time.Now(), id,
	)
	return err
}

// UpdateAccountAPIKey updates an account's API key
func (db *DB) UpdateAccountAPIKey(id, apiKey string) error {
	_, err := db.conn.Exec(
		`UPDATE accounts SET api_key = ?, updated_at = ? WHERE id = ?`,
		apiKey, time.Now(), id,
	)
	return err
}

// ListAccounts lists accounts by provider type (empty = all)
func (db *DB) ListAccounts(providerType string) ([]Account, error) {
	query := `SELECT id, provider_type, label, auth_mode, access_token, refresh_token, api_key,
			  expires_at, metadata, is_active, priority, cooldown_until, backoff_level,
			  last_error, created_at, updated_at
			  FROM accounts`
	var args []interface{}

	if providerType != "" {
		query += ` WHERE provider_type = ?`
		args = append(args, providerType)
	}
	query += ` ORDER BY priority, created_at`

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanAccounts(rows)
}

// GetAccount returns an account by ID
func (db *DB) GetAccount(id string) (*Account, error) {
	row := db.conn.QueryRow(
		`SELECT id, provider_type, label, auth_mode, access_token, refresh_token, api_key,
		 expires_at, metadata, is_active, priority, cooldown_until, backoff_level,
		 last_error, created_at, updated_at
		 FROM accounts WHERE id = ?`, id,
	)
	return scanAccount(row)
}

// GetAvailableAccounts returns active accounts without cooldown
func (db *DB) GetAvailableAccounts(providerType string) ([]Account, error) {
	rows, err := db.conn.Query(
		`SELECT id, provider_type, label, auth_mode, access_token, refresh_token, api_key,
		 expires_at, metadata, is_active, priority, cooldown_until, backoff_level,
		 last_error, created_at, updated_at
		 FROM accounts
		 WHERE provider_type = ? AND is_active = 1
		   AND (cooldown_until IS NULL OR cooldown_until < ?)
		 ORDER BY priority, created_at`,
		providerType, time.Now(),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanAccounts(rows)
}

// GetExpiringAccounts returns OAuth accounts that expire soon
func (db *DB) GetExpiringAccounts(withinDuration time.Duration) ([]Account, error) {
	threshold := time.Now().Add(withinDuration)
	rows, err := db.conn.Query(
		`SELECT id, provider_type, label, auth_mode, access_token, refresh_token, api_key,
		 expires_at, metadata, is_active, priority, cooldown_until, backoff_level,
		 last_error, created_at, updated_at
		 FROM accounts
		 WHERE auth_mode = 'oauth' AND is_active = 1
		   AND refresh_token != '' AND expires_at IS NOT NULL AND expires_at < ?`,
		threshold,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanAccounts(rows)
}

// UpdateAccount updates label, active, and priority
func (db *DB) UpdateAccount(id string, label string, isActive bool, priority int) error {
	_, err := db.conn.Exec(
		`UPDATE accounts SET label = ?, is_active = ?, priority = ?, updated_at = ? WHERE id = ?`,
		label, isActive, priority, time.Now(), id,
	)
	return err
}

// DeleteAccount removes an account
func (db *DB) DeleteAccount(id string) error {
	_, err := db.conn.Exec(`DELETE FROM accounts WHERE id = ?`, id)
	return err
}

// SetAccountCooldown sets cooldown for an account
func (db *DB) SetAccountCooldown(id string, until time.Time, backoffLevel int, lastError string) error {
	_, err := db.conn.Exec(
		`UPDATE accounts SET cooldown_until = ?, backoff_level = ?, last_error = ?, updated_at = ? WHERE id = ?`,
		until, backoffLevel, lastError, time.Now(), id,
	)
	return err
}

// ClearAccountCooldown clears an account's cooldown
func (db *DB) ClearAccountCooldown(id string) error {
	_, err := db.conn.Exec(
		`UPDATE accounts SET cooldown_until = NULL, backoff_level = 0, last_error = '', updated_at = ? WHERE id = ?`,
		time.Now(), id,
	)
	return err
}

// --- Scan helpers ---

func scanAccounts(rows interface{ Next() bool; Scan(...interface{}) error }) ([]Account, error) {
	var accounts []Account
	for rows.Next() {
		var a Account
		var expiresAt, cooldownUntil *time.Time
		var metadataJSON string

		if err := rows.Scan(
			&a.ID, &a.ProviderType, &a.Label, &a.AuthMode,
			&a.AccessToken, &a.RefreshToken, &a.APIKey,
			&expiresAt, &metadataJSON, &a.IsActive, &a.Priority,
			&cooldownUntil, &a.BackoffLevel,
			&a.LastError, &a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			return nil, err
		}

		a.ExpiresAt = expiresAt
		a.CooldownUntil = cooldownUntil
		if metadataJSON != "" && metadataJSON != "{}" {
			json.Unmarshal([]byte(metadataJSON), &a.Metadata)
		}

		accounts = append(accounts, a)
	}
	return accounts, nil
}

func scanAccount(row interface{ Scan(...interface{}) error }) (*Account, error) {
	var a Account
	var expiresAt, cooldownUntil *time.Time
	var metadataJSON string

	if err := row.Scan(
		&a.ID, &a.ProviderType, &a.Label, &a.AuthMode,
		&a.AccessToken, &a.RefreshToken, &a.APIKey,
		&expiresAt, &metadataJSON, &a.IsActive, &a.Priority,
		&cooldownUntil, &a.BackoffLevel,
		&a.LastError, &a.CreatedAt, &a.UpdatedAt,
	); err != nil {
		return nil, err
	}

	a.ExpiresAt = expiresAt
	a.CooldownUntil = cooldownUntil
	if metadataJSON != "" && metadataJSON != "{}" {
		json.Unmarshal([]byte(metadataJSON), &a.Metadata)
	}

	return &a, nil
}
