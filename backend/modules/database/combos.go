package database

import (
	"encoding/json"
	"time"
)

// Combo represents a model combo with fallback support
type Combo struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Models    []string  `json:"models"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateCombo creates a new combo
func (db *DB) CreateCombo(name string, models []string) (*Combo, error) {
	id := generateID()
	now := time.Now()

	modelsJSON, _ := json.Marshal(models)

	_, err := db.conn.Exec(
		`INSERT INTO combos (id, name, models, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		id, name, string(modelsJSON), now, now,
	)
	if err != nil {
		return nil, err
	}

	return &Combo{
		ID:        id,
		Name:      name,
		Models:    models,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// ListCombos returns all combos
func (db *DB) ListCombos() ([]Combo, error) {
	rows, err := db.conn.Query(`SELECT id, name, models, created_at, updated_at FROM combos ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var combos []Combo
	for rows.Next() {
		var c Combo
		var modelsJSON string
		if err := rows.Scan(&c.ID, &c.Name, &modelsJSON, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(modelsJSON), &c.Models)
		if c.Models == nil {
			c.Models = []string{}
		}
		combos = append(combos, c)
	}
	return combos, nil
}

// GetCombo returns a combo by ID
func (db *DB) GetCombo(id string) (*Combo, error) {
	var c Combo
	var modelsJSON string
	err := db.conn.QueryRow(
		`SELECT id, name, models, created_at, updated_at FROM combos WHERE id = ?`, id,
	).Scan(&c.ID, &c.Name, &modelsJSON, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	json.Unmarshal([]byte(modelsJSON), &c.Models)
	if c.Models == nil {
		c.Models = []string{}
	}
	return &c, nil
}

// GetComboByName returns a combo by name (for runtime resolution)
func (db *DB) GetComboByName(name string) (*Combo, error) {
	var c Combo
	var modelsJSON string
	err := db.conn.QueryRow(
		`SELECT id, name, models, created_at, updated_at FROM combos WHERE name = ?`, name,
	).Scan(&c.ID, &c.Name, &modelsJSON, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	json.Unmarshal([]byte(modelsJSON), &c.Models)
	if c.Models == nil {
		c.Models = []string{}
	}
	return &c, nil
}

// UpdateCombo updates a combo's name and models
func (db *DB) UpdateCombo(id, name string, models []string) error {
	modelsJSON, _ := json.Marshal(models)
	_, err := db.conn.Exec(
		`UPDATE combos SET name = ?, models = ?, updated_at = ? WHERE id = ?`,
		name, string(modelsJSON), time.Now(), id,
	)
	return err
}

// DeleteCombo removes a combo
func (db *DB) DeleteCombo(id string) error {
	_, err := db.conn.Exec(`DELETE FROM combos WHERE id = ?`, id)
	return err
}

// ComboNameExists checks if a combo name already exists (excluding a given ID)
func (db *DB) ComboNameExists(name string, excludeID string) bool {
	var count int
	if excludeID != "" {
		db.conn.QueryRow(`SELECT COUNT(*) FROM combos WHERE name = ? AND id != ?`, name, excludeID).Scan(&count)
	} else {
		db.conn.QueryRow(`SELECT COUNT(*) FROM combos WHERE name = ?`, name).Scan(&count)
	}
	return count > 0
}
