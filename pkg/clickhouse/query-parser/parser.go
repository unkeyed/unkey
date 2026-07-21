package queryparser

import (
	"context"
	"fmt"

	clickhouse "github.com/AfterShip/clickhouse-sql-parser/parser"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
)

const (
	queryBytesMax = 16 * 1024
	astNodesMax   = 1_000
)

// NewParser creates a new parser
func NewParser(config Config) *Parser {
	return &Parser{
		stmt:                nil,
		config:              config,
		cteNames:            make(map[string]bool),
		columnValues:        make(map[string]map[string]struct{}),
		fromSubqueryPresent: false,
	}
}

func parseStatementsBounded(query string) ([]clickhouse.Expr, error) {
	if len(query) > queryBytesMax {
		return nil, fault.New("analytics query is too long",
			fault.Code(codes.User.BadRequest.InvalidAnalyticsQuery.URN()),
			fault.Public(fmt.Sprintf("Analytics query is too long; maximum length is %d bytes", queryBytesMax)),
		)
	}

	parser := clickhouse.NewParser(query)
	statements, err := parser.ParseStmts()
	if err != nil {
		return nil, invalidSyntaxError(err)
	}

	astNodesCount := 0
	for _, stmt := range statements {
		astNodesLimitValid := clickhouse.Walk(stmt, func(clickhouse.Expr) bool {
			astNodesCount++
			return astNodesCount <= astNodesMax
		})
		if !astNodesLimitValid {
			return nil, fault.New("analytics query AST is too complex",
				fault.Code(codes.User.BadRequest.InvalidAnalyticsQuery.URN()),
				fault.Public(fmt.Sprintf("Analytics query is too complex; maximum AST node count is %d", astNodesMax)),
			)
		}
	}

	return statements, nil
}

// Parse parses and rewrites a query
func (p *Parser) Parse(ctx context.Context, query string) (string, error) {
	statements, err := parseStatementsBounded(query)
	if err != nil {
		return "", err
	}

	if len(statements) == 0 {
		return "", fault.New("no statements found",
			fault.Code(codes.User.BadRequest.InvalidAnalyticsQuery.URN()),
			fault.Public("No SQL statements found"),
		)
	}

	// Only allow SELECT
	stmt, ok := statements[0].(*clickhouse.SelectQuery)
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
	p.collectColumnValues()
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

// FromSubqueryPresent reports whether any FROM clause contains a nested SELECT.
func (p *Parser) FromSubqueryPresent() bool {
	return p.fromSubqueryPresent
}

func (p *Parser) collectFromSubqueries() {
	p.fromSubqueryPresent = false
	clickhouse.Walk(p.stmt, func(node clickhouse.Expr) bool {
		selectQuery, ok := node.(*clickhouse.SelectQuery)
		if !ok || selectQuery.From == nil {
			return true
		}
		clickhouse.Walk(selectQuery.From, func(fromNode clickhouse.Expr) bool {
			if _, nested := fromNode.(*clickhouse.SelectQuery); nested {
				p.fromSubqueryPresent = true
				return false
			}
			return true
		})
		return !p.fromSubqueryPresent
	})
}

func invalidSyntaxError(err error) error {
	return fault.Wrap(err,
		fault.Code(codes.User.BadRequest.InvalidAnalyticsQuery.URN()),
		fault.Public(fmt.Sprintf("Invalid SQL syntax: %v", err)),
	)
}
