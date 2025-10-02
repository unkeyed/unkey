package queryparser

import (
	"context"

	clickhouse "github.com/AfterShip/clickhouse-sql-parser/parser"
	"github.com/unkeyed/unkey/go/pkg/fault"
)

// NewParser creates a new parser
func NewParser(config Config) *Parser {
	return &Parser{config: config}
}

// Parse parses and rewrites a query
func (p *Parser) Parse(ctx context.Context, query string) (string, error) {
	// Parse SQL
	parser := clickhouse.NewParser(query)
	stmts, err := parser.ParseStmts()
	if err != nil {
		return "", fault.Wrap(err, fault.Public("Invalid SQL syntax"))
	}

	if len(stmts) == 0 {
		return "", fault.New("no statements found", fault.Public("No SQL statements found"))
	}

	// Only allow SELECT
	stmt, ok := stmts[0].(*clickhouse.SelectQuery)
	if !ok {
		return "", fault.New("only SELECT queries allowed", fault.Public("Only SELECT queries are allowed"))
	}

	p.stmt = stmt

	// Build alias lookup map
	p.buildAliasMap()

	// Do all the rewriting
	if err := p.rewriteVirtualColumns(ctx); err != nil {
		return "", err
	}

	if err := p.rewriteTables(); err != nil {
		return "", err
	}

	if err := p.injectWorkspaceFilter(); err != nil {
		return "", err
	}

	if err := p.enforceLimit(); err != nil {
		return "", err
	}

	if err := p.validateFunctions(); err != nil {
		return "", err
	}

	return p.stmt.String(), nil
}
