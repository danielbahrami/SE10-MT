package regex

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/danielbahrami/se10-mt/internal/analyzer"
	"github.com/danielbahrami/se10-mt/internal/graphdb"
	"github.com/danielbahrami/se10-mt/internal/postgres"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// Encapsulates the context and Neo4j driver
type RegexAnalyzer struct {
	*analyzer.BaseAnalyzer
}

var _ analyzer.Analyzer = (*RegexAnalyzer)(nil)

// Holds the outcome of a query analysis
type AnalysisResult = analyzer.AnalysisResult

// Creates a new Analyzer instance
func New(ctx context.Context, driver neo4j.DriverWithContext) *RegexAnalyzer {
	return &RegexAnalyzer{BaseAnalyzer: analyzer.NewBaseAnalyzer(ctx, driver)}
}

func (analyzer *RegexAnalyzer) analyzeQuery(cypher string, perm *postgres.Permissions) (*AnalysisResult, error) {
	log.Println("Analyzing the following query:", cypher)
	analysis := &AnalysisResult{Allowed: true, Violations: []string{}}
	initialViolations := len(analysis.Violations)

	// Node Label Check
	// Use regex that matches node definitions in parentheses
	nodeLabelRegex, err := regexp.Compile(`\(\s*[A-Za-z0-9]*\s*:\s*([A-Za-z0-9_]+)`)
	if err != nil {
		return nil, fmt.Errorf("%s", err.Error())
	}

	labelMatches := nodeLabelRegex.FindAllStringSubmatch(cypher, -1)
	allowedLabels := make(map[string]bool)
	for _, l := range perm.AllowedLabels {
		allowedLabels[strings.ToLower(l)] = true
	}

	labelsFound := make(map[string]bool)
	for _, match := range labelMatches {
		if len(match) < 2 {
			continue
		}
		label := strings.ToLower(match[1])
		labelsFound[label] = true
		if !allowedLabels[label] {
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
	relRegex, err := regexp.Compile(`-\[\s*[^\]]*:\s*([A-Za-z0-9_]+)`)
	if err != nil {
		return nil, fmt.Errorf("%s", err.Error())
	}

	relMatches := relRegex.FindAllStringSubmatch(cypher, -1)
	allowedRels := make(map[string]bool)
	for _, rel := range perm.AllowedRelationships {
		allowedRels[strings.ToLower(rel)] = true
	}

	for _, match := range relMatches {
		if len(match) < 2 {
			continue
		}
		relType := strings.ToLower(match[1])
		if !allowedRels[relType] {
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
	propRegex, err := regexp.Compile(`\.\s*([A-Za-z0-9_]+)`)
	if err != nil {
		return nil, fmt.Errorf("%s", err.Error())
	}

	propMatches := propRegex.FindAllStringSubmatch(cypher, -1)
	allowedProps := make(map[string]bool)
	for _, props := range perm.AllowedProperties {
		for _, prop := range props {
			allowedProps[strings.ToLower(prop)] = true
		}
	}

	for _, match := range propMatches {
		if len(match) < 2 {
			continue
		}
		prop := strings.ToLower(match[1])
		if !allowedProps[prop] {
			log.Printf("Property check failed: property '%s' is not allowed", match[1])
			analysis.Violations = append(analysis.Violations, fmt.Sprintf("disallowed property '%s'", match[1]))
			analysis.Allowed = false
		}
	}

	if len(analysis.Violations) == initialViolations {
		log.Printf("Property check passed. Allowed properties: %+v\n", allowedProps)
	} else {
		log.Printf("Property check completed with violations. Allowed properties: %+v\n", allowedProps)
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
	opPerms := make(map[string]postgres.OperationPermissions)
	for k, v := range perm.OperationPermissions {
		opPerms[strings.ToLower(k)] = v
	}

	for label := range labelsFound {
		if perms, ok := opPerms[label]; ok {
			log.Printf("For label '%s': operation '%s', permission read=%v\n", label, operation, perms.Read)
			var allowed bool
			switch operation {
			case "read":
				allowed = perms.Read
			case "create":
				allowed = perms.Create
			case "update":
				allowed = perms.Update
			case "delete":
				allowed = perms.Delete
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

func (r *RegexAnalyzer) rewriteQuery(cypher string, analysis *AnalysisResult) (string, bool, error) {
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
		return "", false, fmt.Errorf("Cannot safely rewrite the query")
	}

	// Extract the RETURN clause from the query
	// This assumes the RETURN clause is at the end of the query
	retRegex, err := regexp.Compile(`(?i)return\s+(.+)$`)
	if err != nil {
		return "", false, fmt.Errorf("%s", err.Error())
	}

	matches := retRegex.FindStringSubmatch(cypher)
	if len(matches) < 2 {
		log.Println("Rewriting fails due to no RETURN clause being found")
		return "", false, fmt.Errorf("Cannot safely rewrite the query")
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
		log.Println("Rewriting fails due to resulting in an empty RETURN clause")
		return "", false, fmt.Errorf("Cannot safely rewrite the query")
	}

	// Rebuild the new RETURN clause
	newReturnClause := "RETURN " + strings.Join(allowedFields, ", ")

	// Replace the original RETURN clause with the new one
	rewrittenQuery := retRegex.ReplaceAllString(cypher, newReturnClause)
	log.Println("Rewriting succeeded. New query:", rewrittenQuery)
	return rewrittenQuery, true, nil // 'true' indicates a rewritten query was returned
}

func (r *RegexAnalyzer) AnalyzeAndExecute(cypher string, perm *postgres.Permissions) ([]map[string]any, bool, string, []string, error) {
	log.Println("Analyzing with Regex Analyzer...")
	analysis, err := r.analyzeQuery(cypher, perm)
	if err != nil {
		return nil, false, "", nil, err
	}

	// Execute the query if it passed analysis
	if analysis.Allowed {
		log.Println("Query deemed safe. Executing original query...")
		results, err := graphdb.QueryHandler(r.Ctx, r.Driver, cypher)
		if err != nil {
			return nil, false, "", analysis.Violations, fmt.Errorf("%s", err.Error())
		}
		return results, false, "", analysis.Violations, nil
	}

	// Otherwise attempt to rewrite the query
	log.Println("Query is unsafe. Attempting to rewrite...")
	rewrittenQuery, wasRewritten, err := r.rewriteQuery(cypher, analysis)
	if err != nil {
		return nil, wasRewritten, rewrittenQuery, analysis.Violations, analyzer.ForbiddenQueryErr
	}

	log.Println("Rewritten query accepted. Executing rewritten query...")
	results, err := graphdb.QueryHandler(r.Ctx, r.Driver, rewrittenQuery)
	if err != nil {
		return nil, wasRewritten, rewrittenQuery, analysis.Violations, fmt.Errorf("%s", err.Error())
	}

	return results, wasRewritten, rewrittenQuery, analysis.Violations, nil
}
