package analyzer

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"

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

func (analyzer *Analyzer) AnalyzeAndExecute(cypher string, perm *postgres.Permissions) ([]map[string]any, error) {
	log.Println("AnalyzeAndExecute called with query:", cypher)

	// Node Label Check
	// Use regex that matches node definitions in parentheses
	nodeLabelRegex := regexp.MustCompile(`\(\s*[A-Za-z0-9]*\s*:\s*([A-Za-z0-9_]+)`)
	labelMatches := nodeLabelRegex.FindAllStringSubmatch(cypher, -1)

	allowedLabelSet := make(map[string]bool)
	for _, l := range perm.AllowedLabels {
		allowedLabelSet[strings.ToLower(l)] = true
	}

	labelsFound := make(map[string]bool)
	for _, match := range labelMatches {
		if len(match) < 2 {
			continue
		}
		label := strings.ToLower(match[1])
		labelsFound[label] = true
		if !allowedLabelSet[label] {
			log.Printf("Label check failed: label '%s' is not allowed", match[1])
			return nil, fmt.Errorf("Query contains disallowed label '%s'", match[1])
		}
	}

	log.Printf("Label check passed. Labels found: %+v\n", labelsFound)

	// Relationship Check
	relRegex := regexp.MustCompile(`-\[\s*[^\]]*:\s*([A-Za-z0-9_]+)`)
	relMatches := relRegex.FindAllStringSubmatch(cypher, -1)
	allowedRelSet := make(map[string]bool)
	for _, rp := range perm.AllowedRelationships {
		for _, t := range rp.Types {
			allowedRelSet[strings.ToLower(t)] = true
		}
	}

	for _, match := range relMatches {
		if len(match) < 2 {
			continue
		}
		relType := strings.ToLower(match[1])
		if !allowedRelSet[relType] {
			log.Printf("Relationship check failed: relationship type '%s' is not allowed", match[1])
			return nil, fmt.Errorf("Query contains disallowed relationship type '%s'", match[1])
		}
	}

	log.Printf("Relationship check passed\n")

	// Property Check
	propRegex := regexp.MustCompile(`\.\s*([A-Za-z0-9_]+)`)
	propMatches := propRegex.FindAllStringSubmatch(cypher, -1)
	allowedPropSet := make(map[string]bool)
	for _, props := range perm.AllowedProperties {
		for _, p := range props {
			allowedPropSet[strings.ToLower(p)] = true
		}
	}

	for _, match := range propMatches {
		if len(match) < 2 {
			continue
		}
		prop := strings.ToLower(match[1])
		if !allowedPropSet[prop] {
			log.Printf("Property check failed: property '%s' is not allowed", match[1])
			return nil, fmt.Errorf("Query contains disallowed property '%s'", match[1])
		}
	}

	log.Printf("Property check passed. Allowed properties: %+v\n", allowedPropSet)

	// Operation Check
	lowerQuery := strings.ToLower(cypher)
	operation := "read" // default for MATCH queries
	if strings.Contains(lowerQuery, "create") {
		operation = "create"
	} else if strings.Contains(lowerQuery, "set") {
		operation = "update"
	} else if strings.Contains(lowerQuery, "delete") {
		operation = "delete"
	}

	// Normalize the operation permissions keys
	opPermMap := make(map[string]postgres.OperationPermissions)
	for k, v := range perm.OperationPermissions {
		opPermMap[strings.ToLower(k)] = v
	}

	for label := range labelsFound {
		if opPerm, exists := opPermMap[label]; exists {
			log.Printf("For label '%s': operation '%s', permission read=%v\n", label, operation, opPerm.Read)
			var allowed bool
			switch operation {
			case "read":
				allowed = opPerm.Read
			case "create":
				allowed = opPerm.Create
			case "update":
				allowed = opPerm.Update
			case "delete":
				allowed = opPerm.Delete
			}
			if !allowed {
				log.Printf("Operation check failed for label '%s' for operation '%s'\n", label, operation)
				return nil, fmt.Errorf("Operation '%s' is not allowed on label '%s'", operation, label)
			}
		}
	}

	log.Printf("Operation check passed. Operation: %s\n", operation)
	log.Println("All analysis checks passed. Proceeding to execute query")

	results, err := graphdb.QueryHandler(analyzer.ctx, analyzer.driver, cypher)
	if err != nil {
		return nil, fmt.Errorf("Failed to execute query: %w", err)
	}

	return results, nil
}
