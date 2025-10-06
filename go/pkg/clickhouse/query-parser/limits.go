package queryparser

import (
	"fmt"

	clickhouse "github.com/AfterShip/clickhouse-sql-parser/parser"
)

// EnforceLimit enforces the limit configuration on the query statement.
func (p *Parser) enforceLimit() error {
	if p.config.Limit == 0 {
		return nil
	}

	// Check if there's an existing limit
	if p.stmt.Limit == nil || p.stmt.Limit.Limit == nil {
		p.stmt.Limit = &clickhouse.LimitClause{
			Limit: &clickhouse.NumberLiteral{
				Literal: fmt.Sprintf("%d", p.config.Limit),
			},
		}
		return nil
	}

	numLit, ok := p.stmt.Limit.Limit.(*clickhouse.NumberLiteral)
	if !ok {
		// Not a number literal (e.g., LIMIT ALL), enforce max limit
		p.stmt.Limit.Limit = &clickhouse.NumberLiteral{
			Literal: fmt.Sprintf("%d", p.config.Limit),
		}
		return nil
	}

	var existingLimit int
	_, err := fmt.Sscanf(numLit.Literal, "%d", &existingLimit)
	if err != nil {
		// Can't parse the number, enforce max limit for safety
		p.stmt.Limit.Limit = &clickhouse.NumberLiteral{
			Literal: fmt.Sprintf("%d", p.config.Limit),
		}
		return nil
	}

	// Enforce max limit if existing is greater OR if it's negative/invalid
	if existingLimit > p.config.Limit || existingLimit <= 0 {
		p.stmt.Limit.Limit = &clickhouse.NumberLiteral{
			Literal: fmt.Sprintf("%d", p.config.Limit),
		}
	}

	return nil
}
