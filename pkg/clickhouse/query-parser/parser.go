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
	if p.config.QueryBytesMax > 0 && len(query) > p.config.QueryBytesMax {
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
	astNodesCount := 0
	projectedColumnsCount := 0
	expressionsPending := []clickhouse.Expr{p.stmt}
	for len(expressionsPending) > 0 {
		expressionsPendingIndexLast := len(expressionsPending) - 1
		expression := expressionsPending[expressionsPendingIndexLast]
		expressionsPending = expressionsPending[:expressionsPendingIndexLast]
		var errLimit error
		clickhouse.Walk(expression, func(node clickhouse.Expr) bool {
			astNodesCount++
			if p.config.ASTNodesMax > 0 && astNodesCount > p.config.ASTNodesMax {
				errLimit = invalidQueryLimitError("query is too complex", "Analytics query is too complex")
				return false
			}
			if selectQuery, ok := node.(*clickhouse.SelectQuery); ok {
				projectedColumnsCount += len(selectQuery.SelectItems)
				if p.config.ProjectedColumnsMax > 0 && projectedColumnsCount > p.config.ProjectedColumnsMax {
					errLimit = invalidQueryLimitError("too many projected columns", "Analytics query projects too many columns")
					return false
				}
				// AfterShip's walker omits EXCEPT, so count that branch explicitly.
				if selectQuery.Except != nil {
					expressionsPending = append(expressionsPending, selectQuery.Except)
				}
			}
			return true
		})
		if errLimit != nil {
			return errLimit
		}
	}

	return nil
}

func invalidQueryLimitError(messageInternal, messagePublic string) error {
	return fault.New(messageInternal,
		fault.Code(codes.User.BadRequest.InvalidAnalyticsQuery.URN()),
		fault.Public(messagePublic),
	)
}
