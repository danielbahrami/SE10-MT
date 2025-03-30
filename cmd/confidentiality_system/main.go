package main

import (
	"log"

	"github.com/danielbahrami/se10-mt/internal/api"
	"github.com/danielbahrami/se10-mt/internal/postgres"
	"github.com/gin-gonic/gin"
)

func main() {
	// Connect to Postgres
	dbpool, err := postgres.ConnectPostgres()
	if err != nil {
		log.Fatalf("Failed to connect to Postgres: %v", err)
	}
	defer dbpool.Close()

	// Create Gin router
	router := gin.Default()

	// Setup API routes
	api.SetupRoutes(router, dbpool)

	// Start the server on port 9090
	if err := router.Run(":9090"); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
