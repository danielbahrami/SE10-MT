package analyzer

import (
	"context"
	"errors"

	"github.com/danielbahrami/se10-mt/internal/postgres"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// Encapsulates the context and Neo4j driver
type BaseAnalyzer struct {
	Ctx    context.Context
	Driver neo4j.DriverWithContext
}

// Holds the outcome of a query analysis
type AnalysisResult struct {
	Allowed    bool
	Violations []string
}

type Analyzer interface {
	AnalyzeAndExecute(cypher string, perm *postgres.Permissions) ([]map[string]any, bool, string, []string, error)
}

func NewBaseAnalyzer(ctx context.Context, driver neo4j.DriverWithContext) *BaseAnalyzer {
	return &BaseAnalyzer{Ctx: ctx, Driver: driver}
}

// Returned when a query is unsafe and rewriting failed
var ForbiddenQueryErr = errors.New("")
