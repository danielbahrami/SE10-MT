package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/danielbahrami/se10-mt/internal/analyzer"
	"github.com/danielbahrami/se10-mt/internal/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
)

func SetupRoutes(mux *http.ServeMux, dbpool *pgxpool.Pool, analyzerInstance *analyzer.Analyzer) {
	// Health endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte("Ok"))
	})

	// Query endpoint
	mux.HandleFunc("/query", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "", http.StatusMethodNotAllowed)
			return
		}

		// Authenticate user
		user, err := AuthenticateUser(r, dbpool)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		// Decode JSON payload body
		var payload struct {
			Cypher string `json:"cypher"`
		}

		decoder := json.NewDecoder(r.Body)
		if err := decoder.Decode(&payload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		if payload.Cypher == "" {
			http.Error(w, "The 'cypher' field is required", http.StatusBadRequest)
			return
		}

		// Retrieve user permissions
		perm, err := postgres.GetUserPermissions(r.Context(), dbpool, user)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Provide the Cypher query and the user's permissions to the analyzer
		results, wasRewritten, err := analyzerInstance.AnalyzeAndExecute(payload.Cypher, perm)
		if err != nil {
			if errors.Is(err, analyzer.ForbiddenQueryErr) {
				http.Error(w, err.Error(), http.StatusForbidden)
				return
			}
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// Return the results to the client
		if wasRewritten {
			w.Header().Set("Query-Rewritten", "true")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(results)
	})
}
