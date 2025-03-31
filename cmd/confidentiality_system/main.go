package main

import (
	"log"
	"net/http"

	"github.com/danielbahrami/se10-mt/internal/api"
	"github.com/danielbahrami/se10-mt/internal/postgres"
)

func main() {
	// Connect to Postgres
	dbpool, err := postgres.ConnectPostgres()
	if err != nil {
		log.Fatalf("Failed to connect to Postgres: %v", err)
	}
	defer dbpool.Close()

	// Create ServeMux
	mux := http.NewServeMux()

	// Setup API routes
	api.SetupRoutes(mux, dbpool)

	// Start the server on port 9090
	if err := http.ListenAndServe(":9090", mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
