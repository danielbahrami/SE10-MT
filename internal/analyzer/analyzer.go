package analyzer

import (
	"context"
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/danielbahrami/se10-mt/internal/graphdb"
	"github.com/danielbahrami/se10-mt/internal/postgres"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// Encapsulates the context and Neo4j driver
type Analyzer struct {
	ctx    context.Context
	driver neo4j.DriverWithContext
}

// Holds the outcome of a query analysis
type AnalysisResult struct {
	Allowed    bool
	Violations []string
}

// Returned when a query is unsafe and rewriting failed
var ForbiddenQueryErr = errors.New("")

// Creates a new Analyzer instance
func New(ctx context.Context, driver neo4j.DriverWithContext) *Analyzer {
	return &Analyzer{ctx: ctx, driver: driver}
}

func (analyzer *Analyzer) AnalyzeQuery(cypher string, perm *postgres.Permissions) (*AnalysisResult, error) {
	log.Println("Analyzing the following query:", cypher)

	analysis := &AnalysisResult{
		Allowed:    true,
		Violations: []string{},
	}

	// Node Label Check
	initialViolations := len(analysis.Violations)

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
			analysis.Violations = append(analysis.Violations, fmt.Sprintf("disallowed label '%s'", match[1]))
			analysis.Allowed = false
		}
	}

	if len(analysis.Violations) == initialViolations {
		log.Printf("Label check passed. Labels found: %+v\n", labelsFound)
	} else {
		log.Printf("Label check completed with violations. Labels found: %+v\n", labelsFound)
	}

	// Relationship Check
	initialViolations = len(analysis.Violations)
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
			analysis.Violations = append(analysis.Violations, fmt.Sprintf("disallowed relationship type '%s'", match[1]))
			analysis.Allowed = false
		}
	}

	if len(analysis.Violations) == initialViolations {
		log.Println("Relationship check passed")
	} else {
		log.Println("Relationship check completed with violations")
	}

	// Property Check
	initialViolations = len(analysis.Violations)
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
			analysis.Violations = append(analysis.Violations, fmt.Sprintf("disallowed property '%s'", match[1]))
			analysis.Allowed = false
		}
	}

	if len(analysis.Violations) == initialViolations {
		log.Printf("Property check passed. Allowed properties: %+v\n", allowedPropSet)
	} else {
		log.Printf("Property check completed with violations. Allowed properties: %+v\n", allowedPropSet)
	}

	// Operation Check
	initialViolations = len(analysis.Violations)
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
				analysis.Violations = append(analysis.Violations, fmt.Sprintf("operation '%s' is not allowed on label '%s'", operation, label))
				analysis.Allowed = false
			}
		}
	}

	if len(analysis.Violations) == initialViolations {
		log.Printf("Operation check passed. Operation: %s\n", operation)
	} else {
		log.Printf("Operation check completed with violations. Operation: %s\n", operation)
	}

	log.Println("Analysis complete with violations:", analysis.Violations)

	return analysis, nil
}

func (analyzer *Analyzer) RewriteQuery(cypher string, analysis *AnalysisResult, perm *postgres.Permissions) (string, bool, error) {
	log.Println("Attempting to rewrite the query. Violations:", analysis.Violations)

	// Determine if there are any violations other than disallowed properties
	nonPropertyViolations := false
	var disallowedProps []string
	for _, v := range analysis.Violations {
		if !strings.HasPrefix(v, "disallowed property") {
			nonPropertyViolations = true
			break
		} else {
			parts := strings.Split(v, "'")
			if len(parts) >= 2 {
				disallowedProps = append(disallowedProps, parts[1])
			}
		}
	}

	if nonPropertyViolations {
		log.Println("Rewriting not possible due to violations other than disallowed properties")
		return "", false, fmt.Errorf("Cannot safely modify the query")
	}

	// Extract the RETURN clause from the query
	// This assumes the RETURN clause is at the end of the query
	retRegex := regexp.MustCompile(`(?i)return\s+(.+)$`)
	matches := retRegex.FindStringSubmatch(cypher)
	if len(matches) < 2 {
		log.Println("No RETURN clause found; rewriting fails.")
		return "", false, fmt.Errorf("rewriting failed; cannot safely modify the query")
	}

	originalReturnClause := matches[1]

	// Split the RETURN clause into individual fields
	fields := strings.Split(originalReturnClause, ",")
	var allowedFields []string

fieldLoop:
	for _, field := range fields {
		trimmed := strings.TrimSpace(field)
		// Skip fields that reference any disallowed property
		for _, prop := range disallowedProps {
			if strings.Contains(strings.ToLower(trimmed), "."+strings.ToLower(prop)) {
				log.Printf("Removing field '%s' due to disallowed property '%s'\n", trimmed, prop)
				continue fieldLoop
			}
		}
		allowedFields = append(allowedFields, trimmed)
	}

	if len(allowedFields) == 0 {
		log.Println("Rewriting resulted in an empty RETURN clause; rewriting fails.")
		return "", false, fmt.Errorf("rewriting failed; cannot safely modify the query")
	}

	// Rebuild the new RETURN clause
	newReturnClause := "RETURN " + strings.Join(allowedFields, ", ")

	// Replace the original RETURN clause with the new one
	rewrittenQuery := retRegex.ReplaceAllString(cypher, newReturnClause)

	log.Println("Rewriting succeeded. New query:", rewrittenQuery)
	return rewrittenQuery, true, nil // 'true' indicates a rewritten query was returned
}

// Uses the analysis result and then either executes the original query if allowed or calls the rewriter
func (analyzer *Analyzer) AnalyzeAndExecute(cypher string, perm *postgres.Permissions) ([]map[string]any, bool, error) {
	analysis, err := analyzer.AnalyzeQuery(cypher, perm)
	if err != nil {
		return nil, false, fmt.Errorf("%s", err.Error())
	}

	// Execute the query if it passed analysis
	if analysis.Allowed {
		log.Println("Query deemed safe. Executing original query...")
		results, err := graphdb.QueryHandler(analyzer.ctx, analyzer.driver, cypher)
		if err != nil {
			return nil, false, fmt.Errorf("%s", err.Error())
		}
		return results, false, nil
	}

	// Otherwise attempt to rewrite the query
	log.Println("Query is unsafe. Attempting to rewrite...")
	rewrittenQuery, wasRewritten, err := analyzer.RewriteQuery(cypher, analysis, perm)
	if err != nil {
		return nil, wasRewritten, ForbiddenQueryErr
	}

	log.Println("Rewritten query accepted. Executing rewritten query...")
	results, err := graphdb.QueryHandler(analyzer.ctx, analyzer.driver, rewrittenQuery)
	if err != nil {
		return nil, wasRewritten, fmt.Errorf("%s", err.Error())
	}

	return results, wasRewritten, nil
}
