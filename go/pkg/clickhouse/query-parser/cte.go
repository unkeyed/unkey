package queryparser

import (
	clickhouse "github.com/AfterShip/clickhouse-sql-parser/parser"
)

// buildCTERegistry scans the WITH clause and registers all CTE names
func (p *Parser) buildCTERegistry() {
	if p.stmt.With == nil || len(p.stmt.With.CTEs) == 0 {
		return
	}

	for _, cte := range p.stmt.With.CTEs {
		// CTE Expr is the name we reference it by
		if ident, ok := cte.Expr.(*clickhouse.Ident); ok {
			p.cteNames[ident.Name] = true
		}
	}
}

// isCTE checks if a table name is a CTE
func (p *Parser) isCTE(tableName string) bool {
	return p.cteNames[tableName]
}
