package main

import (
	"context"
	"log"
	"net/http"

	"github.com/danielbahrami/se10-mt/internal/analyzer/parser"
	"github.com/danielbahrami/se10-mt/internal/analyzer/regex"
	"github.com/danielbahrami/se10-mt/internal/api"
	"github.com/danielbahrami/se10-mt/internal/graphdb"
	"github.com/danielbahrami/se10-mt/internal/postgres"
)

func main() {
	ctx := context.Background()

	// Connect to Postgres
	dbpool, err := postgres.ConnectPostgres()
	if err != nil {
		log.Fatalf("Failed to connect to Postgres: %v", err)
	}
	defer dbpool.Close()

	// Connect to Neo4j
	driver, err := graphdb.ConnectNeo4j(ctx)
	if err != nil {
		log.Fatalf("Failed to connect to Neo4j: %v", err)
	}
	defer driver.Close(ctx)

	// Create regex and parser analyzers
	regexAnalyzer := regex.New(ctx, driver)
	parserAnalyzer := parser.New(ctx, driver)

	// Create ServeMux
	mux := http.NewServeMux()

	// Setup API routes with both analyzers
	api.SetupRoutes(mux, dbpool, regexAnalyzer, parserAnalyzer)

	// Start the server on port 9090
	if err := http.ListenAndServe(":9090", mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
