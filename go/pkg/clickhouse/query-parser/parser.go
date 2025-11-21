package queryparser

import (
	"context"
	"fmt"

	clickhouse "github.com/AfterShip/clickhouse-sql-parser/parser"
	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/fault"
)

// NewParser creates a new parser
func NewParser(config Config) *Parser {
	if config.Logger == nil {
		panic("query parser requires a logger - use logging.NewNoop() for tests")
	}
	return &Parser{
		stmt:     nil,
		config:   config,
		logger:   config.Logger,
		cteNames: make(map[string]bool),
	}
}

// Parse parses and rewrites a query
func (p *Parser) Parse(ctx context.Context, query string) (string, error) {
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

	// Build CTE registry FIRST so we know which table references are CTEs
	p.buildCTERegistry()

	// Inject security filters
	if err := p.injectSecurityFilters(); err != nil {
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

	if err := p.validateTimeRange(); err != nil {
		return "", err
	}

	return p.stmt.String(), nil
}
