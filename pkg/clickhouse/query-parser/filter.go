package queryparser

import (
	clickhouse "github.com/AfterShip/clickhouse-sql-parser/parser"
)

func (p *Parser) injectWorkspaceFilter() {
	// Walk the AST to inject workspace filter only on SELECT statements that directly access tables
	clickhouse.Walk(p.stmt, func(node clickhouse.Expr) bool {
		// Check if this is a SELECT query
		if selectQuery, ok := node.(*clickhouse.SelectQuery); ok {
			// Only inject if this SELECT directly references a table (not a subquery)
			if p.selectReferencesTable(selectQuery) {
				p.injectWorkspaceFilterOnSelect(selectQuery)
			}
		}

		return true
	})

}

// injectWorkspaceFilterOnSelect injects the workspace filter on a single SELECT statement
func (p *Parser) injectWorkspaceFilterOnSelect(stmt *clickhouse.SelectQuery) {
	// Create: workspace_id = 'ws_xxx'
	filter := &clickhouse.BinaryOperation{
		LeftExpr: &clickhouse.NestedIdentifier{
			Ident: &clickhouse.Ident{Name: "workspace_id"},
		},
		Operation: "=",
		RightExpr: &clickhouse.StringLiteral{
			Literal: p.config.WorkspaceID,
		},
	}

	if stmt.Where == nil {
		stmt.Where = &clickhouse.WhereClause{Expr: filter}
		return
	}

	stmt.Where.Expr = &clickhouse.BinaryOperation{
		LeftExpr:  filter,
		Operation: "AND",
		RightExpr: stmt.Where.Expr,
	}
}

func (p *Parser) injectSecurityFilters() {
	for _, securityFilter := range p.config.SecurityFilters {
		if len(securityFilter.AllowedValues) == 0 {
			continue
		}

		// Walk the AST to inject security filter only on SELECT statements that directly access tables
		clickhouse.Walk(p.stmt, func(node clickhouse.Expr) bool {
			// Check if this is a SELECT query
			if selectQuery, ok := node.(*clickhouse.SelectQuery); ok {
				// Only inject if this SELECT directly references a table (not a subquery)
				if p.selectReferencesTable(selectQuery) {
					p.injectSecurityFilterOnSelect(selectQuery, securityFilter)
				}
			}
			return true
		})
	}

}

// selectReferencesTable checks if a SELECT statement directly references a
// real table in its FROM clause. Returns true only when at least one
// concrete (non-CTE, non-subquery) table reference is present, so the
// workspace_id and security filters get injected exactly where they're
// useful: on SELECTs that read from physical analytics tables.
//
// CTE-name references are treated like subqueries here: the CTE body
// already had filters injected on its own table reference, so injecting
// again on the outer SELECT (which reads from the CTE name) would
// reference columns the CTE doesn't expose and produce ClickHouse
// "Unknown identifier" errors.
func (p *Parser) selectReferencesTable(stmt *clickhouse.SelectQuery) bool {
	if stmt.From == nil {
		return false
	}

	// If the FROM clause directly contains a subquery, the inner SELECT
	// gets its own injection pass; don't double-inject on the outer.
	hasSubquery := false
	clickhouse.Walk(stmt.From, func(node clickhouse.Expr) bool {
		if _, ok := node.(*clickhouse.SelectQuery); ok {
			hasSubquery = true
			return false
		}
		return true
	})
	if hasSubquery {
		return false
	}

	// Inject only when at least one TableIdentifier in FROM is a real
	// (non-CTE) table. Database-qualified names are always real tables;
	// bare names that match the CTE registry are CTE references and are
	// skipped.
	hasRealTable := false
	clickhouse.Walk(stmt.From, func(node clickhouse.Expr) bool {
		ti, ok := node.(*clickhouse.TableIdentifier)
		if !ok {
			return true
		}
		if ti.Database != nil {
			hasRealTable = true
			return false
		}
		if !p.isCTE(ti.Table.Name) {
			hasRealTable = true
			return false
		}
		return true
	})

	return hasRealTable
}

// injectSecurityFilterOnSelect injects a security filter on a single SELECT statement
func (p *Parser) injectSecurityFilterOnSelect(stmt *clickhouse.SelectQuery, securityFilter SecurityFilter) {
	// Build IN list: {column} IN ('val1', 'val2', ...)
	items := make([]clickhouse.Expr, len(securityFilter.AllowedValues))
	for i, value := range securityFilter.AllowedValues {
		items[i] = &clickhouse.ColumnExpr{
			Expr: &clickhouse.StringLiteral{
				Literal: value,
			},
		}
	}

	// Create filter using column name
	filter := &clickhouse.BinaryOperation{
		LeftExpr:  &clickhouse.Ident{Name: securityFilter.Column},
		Operation: "IN",
		RightExpr: &clickhouse.ParamExprList{
			Items: &clickhouse.ColumnExprList{Items: items},
		},
	}

	// Add to WHERE clause
	if stmt.Where == nil {
		stmt.Where = &clickhouse.WhereClause{Expr: filter}
	} else {
		stmt.Where.Expr = &clickhouse.BinaryOperation{
			LeftExpr:  filter,
			Operation: "AND",
			RightExpr: stmt.Where.Expr,
		}
	}
}
