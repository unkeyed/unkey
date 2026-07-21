package queryparser

import (
	"context"
	"fmt"

	clickhouse "github.com/AfterShip/clickhouse-sql-parser/parser"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
)

// NewParser creates a new parser
func NewParser(config Config) *Parser {
	return &Parser{
		stmt:     nil,
		config:   config,
		cteNames: make(map[string]bool),
	}
}

// Parse parses and rewrites a query
func (p *Parser) Parse(ctx context.Context, query string) (string, error) {
	if p.config.MaxQueryBytes > 0 && len(query) > p.config.MaxQueryBytes {
		return "", invalidQueryLimitError("query exceeds maximum length", "Analytics query exceeds the maximum length")
	}

	// Parse SQL
	parser := clickhouse.NewParser(query)
	stmts, err := parser.ParseStmts()
	if err != nil {
		return "", fault.Wrap(err,
			fault.Code(codes.User.BadRequest.InvalidAnalyticsQuery.URN()),
			fault.Public(fmt.Sprintf("Invalid SQL syntax: %v", err)),
		)
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

	if err := p.validateComplexity(); err != nil {
		return "", err
	}

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

func (p *Parser) validateComplexity() error {
	astNodes := 0
	projectedColumns := 0
	var countAST func(clickhouse.Expr)
	countAST = func(root clickhouse.Expr) {
		clickhouse.Walk(root, func(node clickhouse.Expr) bool {
			astNodes++
			if selectQuery, ok := node.(*clickhouse.SelectQuery); ok {
				projectedColumns += len(selectQuery.SelectItems)
				// AfterShip's walker omits EXCEPT, so count that branch explicitly.
				if selectQuery.Except != nil {
					countAST(selectQuery.Except)
				}
			}
			return true
		})
	}
	countAST(p.stmt)

	if p.config.MaxProjectedColumns > 0 && projectedColumns > p.config.MaxProjectedColumns {
		return invalidQueryLimitError("too many projected columns", "Analytics query projects too many columns")
	}
	if p.config.MaxASTNodes > 0 && astNodes > p.config.MaxASTNodes {
		return invalidQueryLimitError("query is too complex", "Analytics query is too complex")
	}
	return nil
}

func invalidQueryLimitError(internal, public string) error {
	return fault.New(internal,
		fault.Code(codes.User.BadRequest.InvalidAnalyticsQuery.URN()),
		fault.Public(public),
	)
}
