package neo4j

import (
	"context"
	"fmt"
	"os"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

func ConnectNeo4j(ctx context.Context) (neo4j.DriverWithContext, error) {
	dbHost := os.Getenv("NEO4J_HOST")
	dbPort := os.Getenv("NEO4J_PORT")
	dbUser := os.Getenv("NEO4J_USER")
	dbPassword := os.Getenv("NEO4J_PASSWORD")
	dbUri := fmt.Sprintf("bolt://%s:%s", dbHost, dbPort)

	driver, err := neo4j.NewDriverWithContext(dbUri, neo4j.BasicAuth(dbUser, dbPassword, ""))
	if err != nil {
		return nil, fmt.Errorf("failed to create neo4j driver: %w", err)
	}

	if err = driver.VerifyConnectivity(ctx); err != nil {
		return nil, fmt.Errorf("unable to connect to neo4j: %w", err)
	}

	fmt.Println("Connected to Neo4j")
	return driver, nil
}
