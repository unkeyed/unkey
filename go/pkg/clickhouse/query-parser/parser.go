package queryparser

import (
	"context"
	"fmt"

	clickhouse "github.com/AfterShip/clickhouse-sql-parser/parser"
	resulttransformer "github.com/unkeyed/unkey/go/pkg/clickhouse/result-transformer"
	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/fault"
)

// NewParser creates a new parser
func NewParser(config Config) *Parser {
	return &Parser{
		config:         config,
		columnMappings: make(map[string]string),
	}
}

// Parse parses and rewrites a query
func (p *Parser) Parse(ctx context.Context, query string) (ParseResult, error) {
	// Parse SQL
	parser := clickhouse.NewParser(query)
	stmts, err := parser.ParseStmts()
	if err != nil {
		return ParseResult{}, fault.Wrap(err,
			fault.Code(codes.User.BadRequest.InvalidAnalyticsQuery.URN()),
			fault.Public(fmt.Sprintf("Invalid SQL syntax: %v", err)),
		)
	}

	if len(stmts) == 0 {
		return ParseResult{}, fault.New("no statements found",
			fault.Code(codes.User.BadRequest.InvalidAnalyticsQuery.URN()),
			fault.Public("No SQL statements found"),
		)
	}

	// Only allow SELECT
	stmt, ok := stmts[0].(*clickhouse.SelectQuery)
	if !ok {
		return ParseResult{}, fault.New("only SELECT queries allowed",
			fault.Code(codes.User.BadRequest.InvalidAnalyticsQueryType.URN()),
			fault.Public("Only SELECT queries are allowed"),
		)
	}

	p.stmt = stmt

	// Build alias lookup map
	p.buildAliasMap()

	// Inject security filters BEFORE virtual column rewriting so they get resolved too
	if err := p.injectSecurityFilters(); err != nil {
		return ParseResult{}, err
	}

	// Rewrite SELECT clause for virtual columns
	if err := p.rewriteSelectColumns(); err != nil {
		return ParseResult{}, err
	}

	// Do all the rewriting
	if err := p.rewriteVirtualColumns(ctx); err != nil {
		return ParseResult{}, err
	}

	if err := p.rewriteTables(); err != nil {
		return ParseResult{}, err
	}

	if err := p.injectWorkspaceFilter(); err != nil {
		return ParseResult{}, err
	}

	if err := p.enforceLimit(); err != nil {
		return ParseResult{}, err
	}

	if err := p.validateFunctions(); err != nil {
		return ParseResult{}, err
	}

	// Collect column mappings
	mappings := make([]resulttransformer.ColumnMapping, 0, len(p.columnMappings))
	for resultCol, actualCol := range p.columnMappings {
		mappings = append(mappings, resulttransformer.ColumnMapping{
			ResultColumn: resultCol,
			ActualColumn: actualCol,
		})
	}

	return ParseResult{
		Query:          p.stmt.String(),
		ColumnMappings: mappings,
	}, nil
}
