package api

import (
	"encoding/json"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Request struct {
	Cypher string `json:"cypher"`
}

func SetupRoutes(mux *http.ServeMux, dbpool *pgxpool.Pool) {
	// Health endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("Ok"))
	})

	// Query endpoint
	mux.HandleFunc("/query", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Authenticate user
		user, err := AuthenticateUser(r, dbpool)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		// Decode JSON request body
		var req Request
		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if req.Cypher == "" {
			http.Error(w, "The 'cypher' field is required", http.StatusBadRequest)
			return
		}

		response := map[string]any{
			"message": "User authenticated and query processed",
			"user":    user.Email,
			"cypher":  req.Cypher,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})
}
