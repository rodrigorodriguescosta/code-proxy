package api

import (
	"encoding/json"
	"net/http"
	"regexp"

	"code-proxy/modules/database"
)

var validComboName = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// registerComboRoutes registers combo management endpoints
func registerComboRoutes(mux *http.ServeMux, db *database.DB) {
	// GET/POST /api/combos
	mux.HandleFunc("/api/combos", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			listCombos(w, db)
		case "POST":
			createCombo(w, r, db)
		default:
			writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})

	// GET/PUT/DELETE /api/combos/{id}
	mux.HandleFunc("/api/combos/", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Path[len("/api/combos/"):]
		if id == "" {
			writeError(w, "Combo ID required", http.StatusBadRequest)
			return
		}
		switch r.Method {
		case "GET":
			getCombo(w, db, id)
		case "PUT":
			updateCombo(w, r, db, id)
		case "DELETE":
			deleteCombo(w, db, id)
		default:
			writeError(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
}

func listCombos(w http.ResponseWriter, db *database.DB) {
	combos, err := db.ListCombos()
	if err != nil {
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if combos == nil {
		combos = []database.Combo{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"combos": combos})
}

func createCombo(w http.ResponseWriter, r *http.Request, db *database.DB) {
	var req struct {
		Name   string   `json:"name"`
		Models []string `json:"models"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		writeError(w, "Name is required", http.StatusBadRequest)
		return
	}
	if !validComboName.MatchString(req.Name) {
		writeError(w, "Name must contain only letters, numbers, - and _", http.StatusBadRequest)
		return
	}
	if db.ComboNameExists(req.Name, "") {
		writeError(w, "A combo with this name already exists", http.StatusConflict)
		return
	}

	if req.Models == nil {
		req.Models = []string{}
	}

	combo, err := db.CreateCombo(req.Name, req.Models)
	if err != nil {
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(combo)
}

func getCombo(w http.ResponseWriter, db *database.DB, id string) {
	combo, err := db.GetCombo(id)
	if err != nil {
		writeError(w, "Combo not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(combo)
}

func updateCombo(w http.ResponseWriter, r *http.Request, db *database.DB, id string) {
	var req struct {
		Name   string   `json:"name"`
		Models []string `json:"models"`
	}
	if err := readJSON(r, &req); err != nil {
		writeError(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Check combo exists
	existing, err := db.GetCombo(id)
	if err != nil {
		writeError(w, "Combo not found", http.StatusNotFound)
		return
	}

	name := req.Name
	if name == "" {
		name = existing.Name
	}
	if !validComboName.MatchString(name) {
		writeError(w, "Name must contain only letters, numbers, - and _", http.StatusBadRequest)
		return
	}
	if db.ComboNameExists(name, id) {
		writeError(w, "A combo with this name already exists", http.StatusConflict)
		return
	}

	models := req.Models
	if models == nil {
		models = existing.Models
	}

	if err := db.UpdateCombo(id, name, models); err != nil {
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func deleteCombo(w http.ResponseWriter, db *database.DB, id string) {
	if err := db.DeleteCombo(id); err != nil {
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
