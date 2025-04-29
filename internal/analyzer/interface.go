package analyzer

import (
	"errors"

	"github.com/danielbahrami/se10-mt/internal/postgres"
)

type Analyzer interface {
	AnalyzeAndExecute(
		cypher string,
		perm *postgres.Permissions,
	) (
		[]map[string]any, bool, string, []string, error,
	)
}

var ForbiddenQueryErr = errors.New("")
