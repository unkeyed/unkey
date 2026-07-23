package queryparser

import (
	"context"
	"fmt"

	clickhouse "github.com/AfterShip/clickhouse-sql-parser/parser"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
)

const (
	queryBytesMax       = 16 << 10
	projectedColumnsMax = 64
	astNodesMax         = 2_000
)

// NewParser creates a new parser
func NewParser(config Config) *Parser {
	return &Parser{
		stmt:     nil,
		config:   config,
		cteNames: make(map[string]bool),
	}
}

// walkQueryIncludingExcept traverses the complete query tree, including the
// right side of EXCEPT expressions omitted by the upstream Walk function.
func walkQueryIncludingExcept(node clickhouse.Expr, visit clickhouse.WalkFunc) bool {
	exceptBranches := make([]*clickhouse.SelectQuery, 0)
	if !clickhouse.Walk(node, func(current clickhouse.Expr) bool {
		if !visit(current) {
			return false
		}
		if query, ok := current.(*clickhouse.SelectQuery); ok && query.Except != nil {
			exceptBranches = append(exceptBranches, query.Except)
		}
		return true
	}) {
		return false
	}

	for _, branch := range exceptBranches {
		if !walkQueryIncludingExcept(branch, visit) {
			return false
		}
	}
	return true
}

// Parse parses and rewrites a query
func (p *Parser) Parse(ctx context.Context, query string) (string, error) {
	if len(query) > queryBytesMax {
		return "", newQueryLimitError("query exceeds maximum length", "Analytics query exceeds the maximum length")
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
	if len(stmts) != 1 {
		return "", fault.New("multiple statements are not allowed",
			fault.Code(codes.User.BadRequest.InvalidAnalyticsQuery.URN()),
			fault.Public("Analytics queries must contain exactly one statement"),
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
	if err := p.validateCTEAliases(); err != nil {
		return "", err
	}

	if err := p.validateSetOperands(); err != nil {
		return "", err
	}

	if err := p.rewriteTables(); err != nil {
		return "", err
	}

	// Qualify every injected predicate with its rewritten physical source.
	p.injectSecurityFilters()
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
	astNodesCount := 0
	projectedColumnsCount := 0
	var errLimit error
	walkQueryIncludingExcept(p.stmt, func(node clickhouse.Expr) bool {
		astNodesCount++
		if astNodesCount > astNodesMax {
			errLimit = newQueryLimitError("query is too complex", "Analytics query is too complex")
			return false
		}
		if selectQuery, ok := node.(*clickhouse.SelectQuery); ok {
			projectedColumnsCount += len(selectQuery.SelectItems)
			if projectedColumnsCount > projectedColumnsMax {
				errLimit = newQueryLimitError("too many projected columns", "Analytics query projects too many columns")
				return false
			}
		}
		return true
	})
	return errLimit
}

func newQueryLimitError(messageInternal, messagePublic string) error {
	return fault.New(messageInternal,
		fault.Code(codes.User.BadRequest.InvalidAnalyticsQuery.URN()),
		fault.Public(messagePublic),
	)
}
