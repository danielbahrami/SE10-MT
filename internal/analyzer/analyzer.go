package analyzer

import (
	"context"
	"fmt"

	"github.com/danielbahrami/se10-mt/internal/graphdb"
	"github.com/danielbahrami/se10-mt/internal/postgres"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// Analyzer encapsulates the context and Neo4j driver
type Analyzer struct {
	ctx    context.Context
	driver neo4j.DriverWithContext
}

// Creates a new Analyzer instance
func New(ctx context.Context, driver neo4j.DriverWithContext) *Analyzer {
	return &Analyzer{ctx: ctx, driver: driver}
}

// Performs analysis on the given Cypher query using the provided permissions and then executes the (possibly modified) query
func (analyzer *Analyzer) AnalyzeAndExecute(cypher string, perm *postgres.Permissions) ([]map[string]any, error) {

	// Analysis here

	results, err := graphdb.QueryHandler(analyzer.ctx, analyzer.driver, cypher)
	if err != nil {
		return nil, fmt.Errorf("Failed to execute query: %w", err)
	}

	return results, nil
}
