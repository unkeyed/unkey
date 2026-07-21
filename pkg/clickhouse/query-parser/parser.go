package queryparser

import (
	"context"
	"fmt"

	clickhouse "github.com/AfterShip/clickhouse-sql-parser/parser"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
)

const (
	maxQueryBytes = 16 * 1024
	maxASTNodes   = 1_000
)

// NewParser creates a new parser
func NewParser(config Config) *Parser {
	return &Parser{
		stmt:             nil,
		config:           config,
		cteNames:         make(map[string]bool),
		extractedColumns: make(map[string]map[string]struct{}),
		hasFromSubquery:  false,
	}
}

func parseBoundedStatements(query string) ([]clickhouse.Expr, error) {
	if len(query) > maxQueryBytes {
		return nil, fault.New("analytics query is too long",
			fault.Code(codes.User.BadRequest.InvalidAnalyticsQuery.URN()),
			fault.Public(fmt.Sprintf("Analytics query is too long; maximum length is %d bytes", maxQueryBytes)),
		)
	}

	parser := clickhouse.NewParser(query)
	stmts, err := parser.ParseStmts()
	if err != nil {
		return nil, invalidSyntaxError(err)
	}

	nodes := 0
	for _, stmt := range stmts {
		withinLimit := clickhouse.Walk(stmt, func(clickhouse.Expr) bool {
			nodes++
			return nodes <= maxASTNodes
		})
		if !withinLimit {
			return nil, fault.New("analytics query AST is too complex",
				fault.Code(codes.User.BadRequest.InvalidAnalyticsQuery.URN()),
				fault.Public(fmt.Sprintf("Analytics query is too complex; maximum AST node count is %d", maxASTNodes)),
			)
		}
	}

	return stmts, nil
}

// Parse parses and rewrites a query
func (p *Parser) Parse(ctx context.Context, query string) (string, error) {
	stmts, err := parseBoundedStatements(query)
	if err != nil {
		return "", err
	}

	if len(stmts) == 0 {
		return "", fault.New("no statements found",
			fault.Code(codes.User.BadRequest.InvalidAnalyticsQuery.URN()),
			fault.Public("No SQL statements found"),
		)
	}

	// Only allow SELECT
	stmt, ok := stmts[0].(*clickhouse.SelectQuery)
	if !ok {
		return "", fault.New("only SELECT queries allowed",
			fault.Code(codes.User.BadRequest.InvalidAnalyticsQueryType.URN()),
			fault.Public("Only SELECT queries are allowed"),
		)
	}

	p.stmt = stmt
	if err := p.validateSettings(); err != nil {
		return "", err
	}
	p.collectExtractedColumns()
	p.collectFromSubqueries()

	// Build CTE registry FIRST so we know which table references are CTEs
	p.buildCTERegistry()

	// Inject security filters
	p.injectSecurityFilters()
	if err := p.rewriteTables(); err != nil {
		return "", err
	}

	p.injectWorkspaceFilter()

	p.enforceLimit()

	if err := p.validateFunctions(); err != nil {
		return "", err
	}

	if err := p.validateTimeRange(); err != nil {
		return "", err
	}

	return p.stmt.String(), nil
}

// HasFromSubquery reports whether any FROM clause contains a nested SELECT.
func (p *Parser) HasFromSubquery() bool {
	return p.hasFromSubquery
}

func (p *Parser) collectFromSubqueries() {
	p.hasFromSubquery = false
	clickhouse.Walk(p.stmt, func(node clickhouse.Expr) bool {
		selectQuery, ok := node.(*clickhouse.SelectQuery)
		if !ok || selectQuery.From == nil {
			return true
		}
		clickhouse.Walk(selectQuery.From, func(fromNode clickhouse.Expr) bool {
			if _, nested := fromNode.(*clickhouse.SelectQuery); nested {
				p.hasFromSubquery = true
				return false
			}
			return true
		})
		return !p.hasFromSubquery
	})
}

func invalidSyntaxError(err error) error {
	return fault.Wrap(err,
		fault.Code(codes.User.BadRequest.InvalidAnalyticsQuery.URN()),
		fault.Public(fmt.Sprintf("Invalid SQL syntax: %v", err)),
	)
}
