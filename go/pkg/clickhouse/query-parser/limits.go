package queryparser

import (
	"fmt"

	clickhouse "github.com/AfterShip/clickhouse-sql-parser/parser"
)

// EnforceLimit enforces the limit configuration on the query statement.
// It walks the entire AST to enforce limits on all SELECT statements including subqueries.
func (p *Parser) enforceLimit() error {
	if p.config.Limit == 0 {
		return nil
	}

	// Walk the AST to enforce limits on all SELECT statements including subqueries
	clickhouse.Walk(p.stmt, func(node clickhouse.Expr) bool {
		// Check if this is a SELECT query
		if selectQuery, ok := node.(*clickhouse.SelectQuery); ok {
			p.enforceLimitOnSelect(selectQuery)
		}
		return true
	})

	return nil
}

// enforceLimitOnSelect enforces the limit on a single SELECT statement
func (p *Parser) enforceLimitOnSelect(stmt *clickhouse.SelectQuery) {
	// Check if there's an existing limit
	if stmt.Limit == nil || stmt.Limit.Limit == nil {
		stmt.Limit = &clickhouse.LimitClause{
			Limit: &clickhouse.NumberLiteral{
				Literal: fmt.Sprintf("%d", p.config.Limit),
			},
		}
		return
	}

	numLit, ok := stmt.Limit.Limit.(*clickhouse.NumberLiteral)
	if !ok {
		// Not a number literal (e.g., LIMIT ALL), enforce max limit
		stmt.Limit.Limit = &clickhouse.NumberLiteral{
			Literal: fmt.Sprintf("%d", p.config.Limit),
		}
		return
	}

	var existingLimit int
	_, err := fmt.Sscanf(numLit.Literal, "%d", &existingLimit)
	if err != nil {
		// Can't parse the number, enforce max limit for safety
		stmt.Limit.Limit = &clickhouse.NumberLiteral{
			Literal: fmt.Sprintf("%d", p.config.Limit),
		}
		return
	}

	// Enforce max limit if existing is greater OR if it's negative/invalid
	if existingLimit > p.config.Limit || existingLimit < 0 {
		stmt.Limit.Limit = &clickhouse.NumberLiteral{
			Literal: fmt.Sprintf("%d", p.config.Limit),
		}
	}
}
