package account

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"code-proxy/modules/database"
	"code-proxy/modules/provider"
)

// Manager manages account selection, cooldown and refresh
type Manager struct {
	db       *database.DB
	mu       sync.RWMutex
	strategy string         // "fill-first" or "round-robin"
	cursors  map[string]int // providerType -> current index (round-robin)
}

// NewManager creates an AccountManager
func NewManager(db *database.DB) *Manager {
	return &Manager{
		db:       db,
		strategy: "fill-first",
		cursors:  make(map[string]int),
	}
}

// SetStrategy sets the selection strategy
func (m *Manager) SetStrategy(strategy string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.strategy = strategy
}

// Select picks the next available account for the provider+model
func (m *Manager) Select(providerType, model string) (*provider.Account, error) {
	if m.db == nil {
		return nil, nil // No database, no accounts
	}

	accounts, err := m.db.GetAvailableAccounts(providerType)
	if err != nil {
		return nil, fmt.Errorf("fetch accounts: %w", err)
	}

	if len(accounts) == 0 {
		// No accounts configured = provider works without auth
		return nil, nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	var selected *database.Account

	switch m.strategy {
	case "round-robin":
		key := providerType
		idx := m.cursors[key] % len(accounts)
		selected = &accounts[idx]
		m.cursors[key] = idx + 1

	default: // fill-first
		selected = &accounts[0]
	}

	return dbAccountToProvider(selected), nil
}

// ReportSuccess clears cooldown for an account
func (m *Manager) ReportSuccess(accountID, model string) {
	if m.db == nil {
		return
	}
	m.db.ClearAccountCooldown(accountID)
}

// ReportError applies cooldown with exponential backoff
func (m *Manager) ReportError(accountID, model string, httpStatus int, errText string) {
	if m.db == nil {
		return
	}

	acct, err := m.db.GetAccount(accountID)
	if err != nil {
		return
	}

	duration := CooldownForStatus(httpStatus, acct.BackoffLevel)
	until := time.Now().Add(duration)
	newLevel := acct.BackoffLevel + 1

	log.Printf("[ACCOUNT] Cooldown %s: status=%d, duration=%s, level=%d",
		accountID[:8], httpStatus, duration, newLevel)

	m.db.SetAccountCooldown(accountID, until, newLevel, errText)
}

// RefreshLoop checks for expiring tokens and refreshes them in background
func (m *Manager) RefreshLoop(ctx context.Context, interval time.Duration, refreshFn func(acct *provider.Account) error) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.refreshExpiring(refreshFn)
		}
	}
}

func (m *Manager) refreshExpiring(refreshFn func(acct *provider.Account) error) {
	if m.db == nil || refreshFn == nil {
		return
	}

	// Fetch accounts expiring in the next 10 minutes
	accounts, err := m.db.GetExpiringAccounts(10 * time.Minute)
	if err != nil {
		log.Printf("[ACCOUNT] Error fetching expiring accounts: %v", err)
		return
	}

	for _, a := range accounts {
		acct := dbAccountToProvider(&a)
		if err := refreshFn(acct); err != nil {
			log.Printf("[ACCOUNT] Refresh error %s (%s): %v", a.ID[:8], a.Label, err)
		} else {
			log.Printf("[ACCOUNT] Refresh OK: %s (%s)", a.ID[:8], a.Label)
		}
	}
}

// dbAccountToProvider converts database.Account to provider.Account
func dbAccountToProvider(a *database.Account) *provider.Account {
	var expiresAt time.Time
	if a.ExpiresAt != nil {
		expiresAt = *a.ExpiresAt
	}

	return &provider.Account{
		ID:           a.ID,
		ProviderType: a.ProviderType,
		Label:        a.Label,
		AuthMode:     a.AuthMode,
		AccessToken:  a.AccessToken,
		RefreshToken: a.RefreshToken,
		APIKey:       a.APIKey,
		ExpiresAt:    expiresAt,
		Metadata:     a.Metadata,
		IsActive:     a.IsActive,
		Priority:     a.Priority,
	}
}
