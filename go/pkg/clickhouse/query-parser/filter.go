package queryparser

import (
	clickhouse "github.com/AfterShip/clickhouse-sql-parser/parser"
)

func (p *Parser) injectWorkspaceFilter() error {
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

	return nil
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

func (p *Parser) injectSecurityFilters() error {
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

	return nil
}

// selectReferencesTable checks if a SELECT statement directly references a table in its FROM clause
// Returns true if the FROM clause contains a table, false if it contains only a subquery
func (p *Parser) selectReferencesTable(stmt *clickhouse.SelectQuery) bool {
	if stmt.From == nil {
		return false
	}

	// Check if the FROM clause directly contains a subquery
	// If it does, we should NOT inject filters here
	hasSubquery := false
	clickhouse.Walk(stmt.From, func(node clickhouse.Expr) bool {
		if _, ok := node.(*clickhouse.SelectQuery); ok {
			hasSubquery = true
			return false // Stop walking
		}
		return true
	})

	// If there's a subquery in the FROM, don't inject filters
	if hasSubquery {
		return false
	}

	// Otherwise, check if there's a table reference
	hasTable := false
	clickhouse.Walk(stmt.From, func(node clickhouse.Expr) bool {
		if _, ok := node.(*clickhouse.TableExpr); ok {
			hasTable = true
			return false // Stop walking
		}
		return true
	})

	return hasTable
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
