package graphdb

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type QueryResult = map[string]any

func ConnectNeo4j(ctx context.Context) (neo4j.DriverWithContext, error) {
	dbHost := os.Getenv("NEO4J_HOST")
	dbPort := os.Getenv("NEO4J_PORT")
	dbUser := os.Getenv("NEO4J_USER")
	dbPassword := os.Getenv("NEO4J_PASSWORD")
	dbUri := fmt.Sprintf("bolt://%s:%s", dbHost, dbPort)

	driver, err := neo4j.NewDriverWithContext(dbUri, neo4j.BasicAuth(dbUser, dbPassword, ""))
	if err != nil {
		return nil, fmt.Errorf("Failed to create Neo4j driver: %w", err)
	}

	if err = driver.VerifyConnectivity(ctx); err != nil {
		return nil, fmt.Errorf("Unable to connect to Neo4j: %w", err)
	}

	log.Println("Connected to Neo4j")
	return driver, nil
}

func QueryHandler(
	ctx context.Context,
	driver neo4j.DriverWithContext,
	cypher string) ([]QueryResult, error) {
	parameters := map[string]any{}
	result, err := neo4j.ExecuteQuery(ctx, driver,
		cypher,
		parameters,
		neo4j.EagerResultTransformer,
		neo4j.ExecuteQueryWithDatabase("neo4j"))
	if err != nil {
		return nil, fmt.Errorf("QueryHandler failed: %w", err)
	}

	var records []QueryResult
	for _, record := range result.Records {
		records = append(records, record.AsMap())
	}

	log.Printf("The query `%v` returned %v records in %+v.\n",
		result.Summary.Query().Text(),
		len(result.Records),
		result.Summary.ResultAvailableAfter())

	return records, nil
}
