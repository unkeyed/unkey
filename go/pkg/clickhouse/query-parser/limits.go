package queryparser

import (
	"fmt"

	clickhouse "github.com/AfterShip/clickhouse-sql-parser/parser"
)

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
		return nil
	}

	var existingLimit int
	_, err := fmt.Sscanf(numLit.Literal, "%d", &existingLimit)
	if err != nil {
		return nil
	}

	if existingLimit > p.config.Limit {
		p.stmt.Limit.Limit = &clickhouse.NumberLiteral{
			Literal: fmt.Sprintf("%d", p.config.Limit),
		}
	}

	return nil
}
