package parser

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	"github.com/danielbahrami/se10-mt/internal/analyzer"
	"github.com/danielbahrami/se10-mt/internal/graphdb"
	"github.com/danielbahrami/se10-mt/internal/parser"
	"github.com/danielbahrami/se10-mt/internal/postgres"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

// Encapsulates the context and Neo4j driver
type ParserAnalyzer struct {
	*analyzer.BaseAnalyzer
}

var _ analyzer.Analyzer = (*ParserAnalyzer)(nil)

// Holds the outcome of a query analysis
type AnalysisResult = analyzer.AnalysisResult

type TreeListener struct {
	*parser.BaseCypherListener
	labelsFound map[string]bool
	relFound    map[string]bool
	propsFound  map[string]bool
	hasCreate   bool
	hasUpdate   bool
	hasDelete   bool
}

// Creates a new Analyzer instance
func New(ctx context.Context, driver neo4j.DriverWithContext) *ParserAnalyzer {
	return &ParserAnalyzer{BaseAnalyzer: analyzer.NewBaseAnalyzer(ctx, driver)}
}

func newTreeListener() *TreeListener {
	return &TreeListener{
		BaseCypherListener: &parser.BaseCypherListener{},
		labelsFound:        make(map[string]bool),
		relFound:           make(map[string]bool),
		propsFound:         make(map[string]bool),
	}
}

// Returns the ANTLR parse-tree for an input Cypher string
func parse(input string) antlr.ParseTree {
	is := antlr.NewInputStream(input)
	lex := parser.NewCypherLexer(is)
	tokens := antlr.NewCommonTokenStream(lex, 0)
	p := parser.NewCypherParser(tokens)
	p.BuildParseTrees = true
	return p.OC_Cypher()
}

func (l *TreeListener) EnterOC_NodePattern(ctx *parser.OC_NodePatternContext) {
	labelsCtx := ctx.OC_NodeLabels()
	if labelsCtx == nil {
		return
	}

	for _, nodeLabelCtx := range labelsCtx.AllOC_NodeLabel() {
		labelNameCtx := nodeLabelCtx.OC_LabelName()
		if labelNameCtx == nil {
			continue
		}
		name := labelNameCtx.GetText()
		l.labelsFound[strings.ToLower(name)] = true
	}
}

func (l *TreeListener) EnterOC_RelationshipPattern(ctx *parser.OC_RelationshipPatternContext) {
	rd := ctx.OC_RelationshipDetail()
	if rd == nil {
		return
	}

	rtCtxs := rd.OC_RelationshipTypes()
	if rtCtxs == nil {
		return
	}

	for _, relTypeCtx := range rtCtxs.AllOC_RelTypeName() {
		text := relTypeCtx.GetText()
		rel := strings.TrimPrefix(text, ":")
		l.relFound[strings.ToLower(rel)] = true
	}
}

func (l *TreeListener) EnterOC_PropertyLookup(ctx *parser.OC_PropertyLookupContext) {
	pkCtx := ctx.OC_PropertyKeyName()
	if pkCtx == nil {
		return
	}

	name := pkCtx.GetText()
	l.propsFound[strings.ToLower(name)] = true
}

func (l *TreeListener) EnterOC_Create(ctx *parser.OC_CreateContext) {
	l.hasCreate = true
}

func (l *TreeListener) EnterOC_Set(ctx *parser.OC_SetContext) {
	l.hasUpdate = true
}

func (l *TreeListener) EnterOC_Delete(ctx *parser.OC_DeleteContext) {
	l.hasDelete = true
}

func (p *ParserAnalyzer) analyzeQuery(cypher string, perm *postgres.Permissions) (*AnalysisResult, error) {
	log.Println("Analyzing the following query:", cypher)
	listener := newTreeListener()
	tree := parse(cypher)
	antlr.ParseTreeWalkerDefault.Walk(listener, tree)
	analysis := &AnalysisResult{Allowed: true, Violations: []string{}}
	initialViolations := len(analysis.Violations)

	// Node Label check
	allowedLabels := make(map[string]bool, len(perm.AllowedLabels))
	for _, l := range perm.AllowedLabels {
		allowedLabels[strings.ToLower(l)] = true
	}

	for label := range listener.labelsFound {
		if !allowedLabels[label] {
			log.Printf("Label check failed: label '%s' is not allowed", label)
			analysis.Violations = append(analysis.Violations, fmt.Sprintf("disallowed label '%s'", label))
			analysis.Allowed = false
		}
	}

	if len(analysis.Violations) == initialViolations {
		log.Printf("Label check passed. Labels found: %+v\n", listener.labelsFound)
	} else {
		log.Printf("Label check completed with violations. Labels found: %+v\n", listener.labelsFound)
	}

	// Relationship check
	initialViolations = len(analysis.Violations)
	allowedRels := make(map[string]bool, len(perm.AllowedRelationships))
	for _, rel := range perm.AllowedRelationships {
		allowedRels[strings.ToLower(rel)] = true
	}

	for rel := range listener.relFound {
		if !allowedRels[rel] {
			log.Printf("Relationship check failed: relationship type '%s' is not allowed", rel)
			analysis.Violations = append(analysis.Violations, fmt.Sprintf("disallowed relationship type '%s'", rel))
			analysis.Allowed = false
		}
	}

	if len(analysis.Violations) == initialViolations {
		log.Println("Relationship check passed")
	} else {
		log.Println("Relationship check completed with violations")
	}

	// Property check
	initialViolations = len(analysis.Violations)
	allowedProps := make(map[string]bool)
	for _, props := range perm.AllowedProperties {
		for _, prop := range props {
			allowedProps[strings.ToLower(prop)] = true
		}
	}

	for prop := range listener.propsFound {
		if !allowedProps[prop] {
			log.Printf("Property check failed: property '%s' is not allowed", prop)
			analysis.Violations = append(analysis.Violations, fmt.Sprintf("disallowed property '%s'", prop))
			analysis.Allowed = false
		}
	}

	if len(analysis.Violations) == initialViolations {
		log.Printf("Property check passed. Allowed properties: %+v\n", allowedProps)
	} else {
		log.Printf("Property check completed with violations. Allowed properties: %+v\n", allowedProps)
	}

	// Operation check
	initialViolations = len(analysis.Violations)
	operation := "read" // default for MATCH queries
	if listener.hasCreate {
		operation = "create"
	} else if listener.hasUpdate {
		operation = "update"
	} else if listener.hasDelete {
		operation = "delete"
	}

	// Normalize the operation permissions keys
	opPerms := make(map[string]postgres.OperationPermissions, len(perm.OperationPermissions))
	for k, v := range perm.OperationPermissions {
		opPerms[strings.ToLower(k)] = v
	}

	for label := range listener.labelsFound {
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

func (p *ParserAnalyzer) rewriteQuery(cypher string, analysis *AnalysisResult) (string, bool, error) {
	log.Println("Attempting to rewrite the query. Violations:", analysis.Violations)

	// Determine if there are any violations other than disallowed properties
	nonPropertyViolations := false
	var disallowedProps []string
	for _, v := range analysis.Violations {
		if !strings.HasPrefix(v, "disallowed property") {
			nonPropertyViolations = true
			break
		}
		parts := strings.Split(v, "'")
		if len(parts) >= 2 {
			disallowedProps = append(disallowedProps, parts[1])
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
	for _, field := range fields {
		trimmed := strings.TrimSpace(field)
		skip := false

		// Skip fields that reference any disallowed property
		for _, prop := range disallowedProps {
			if strings.Contains(strings.ToLower(trimmed), "."+strings.ToLower(prop)) {
				skip = true
				break
			}
		}
		if !skip {
			allowedFields = append(allowedFields, trimmed)
		}
	}

	if len(allowedFields) == 0 {
		return "", false, fmt.Errorf("Cannot safely rewrite the query")
	}

	// Rebuild the new RETURN clause
	newReturnClause := "RETURN " + strings.Join(allowedFields, ", ")

	// Replace the original RETURN clause with the new one
	rewrittenQuery := retRegex.ReplaceAllString(cypher, newReturnClause)
	log.Println("Rewriting succeeded. New query:", rewrittenQuery)
	return rewrittenQuery, true, nil // 'true' indicates a rewritten query was returned
}

func (p *ParserAnalyzer) AnalyzeAndExecute(cypher string, perm *postgres.Permissions) ([]map[string]any, bool, string, []string, error) {
	log.Println("Analyzing with Parser Analyzer...")
	analysis, err := p.analyzeQuery(cypher, perm)
	if err != nil {
		return nil, false, "", nil, err
	}

	// Execute the query if it passed analysis
	if analysis.Allowed {
		log.Println("Query deemed safe. Executing original query...")
		results, err := graphdb.QueryHandler(p.Ctx, p.Driver, cypher)
		if err != nil {
			return nil, false, "", analysis.Violations, err
		}
		return results, false, "", analysis.Violations, nil
	}

	// Otherwise attempt to rewrite the query
	log.Println("Query is unsafe. Attempting to rewrite...")
	rewritten, wasRewritten, err := p.rewriteQuery(cypher, analysis)
	if err != nil {
		return nil, wasRewritten, rewritten, analysis.Violations, analyzer.ForbiddenQueryErr
	}

	log.Println("Rewritten query accepted. Executing rewritten query...")
	results, err := graphdb.QueryHandler(p.Ctx, p.Driver, rewritten)
	if err != nil {
		return nil, wasRewritten, rewritten, analysis.Violations, err
	}

	return results, wasRewritten, rewritten, analysis.Violations, nil
}
