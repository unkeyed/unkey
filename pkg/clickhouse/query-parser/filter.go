package queryparser

import (
	clickhouse "github.com/AfterShip/clickhouse-sql-parser/parser"
)

// parenthesize wraps an expression in parentheses so a caller-supplied WHERE
// cannot defeat an injected filter through operator precedence. AND binds
// tighter than OR, so prepending `<injected> AND <where>` to a where like
// `x = 'a' OR 1=1` would parse as `(<injected> AND x = 'a') OR 1=1` and bypass
// the filter entirely. Wrapping yields `<injected> AND (x = 'a' OR 1=1)`.
//
// This parser has no dedicated grouping node; it represents `( ... )` as a
// single-item ParamExprList, which is what we construct here.
func parenthesize(expr clickhouse.Expr) clickhouse.Expr {
	return &clickhouse.ParamExprList{
		Items: &clickhouse.ColumnExprList{
			Items: []clickhouse.Expr{expr},
		},
	}
}

func (p *Parser) injectWorkspaceFilter() {
	walkQueryIncludingExcept(p.stmt, func(node clickhouse.Expr) bool {
		if selectQuery, ok := node.(*clickhouse.SelectQuery); ok {
			for _, source := range p.directTableSources(selectQuery) {
				p.injectWorkspaceFilterOnSelect(selectQuery, source)
			}
		}

		return true
	})

}

// injectWorkspaceFilterOnSelect injects the workspace filter on a single SELECT statement
func (p *Parser) injectWorkspaceFilterOnSelect(stmt *clickhouse.SelectQuery, source clickhouse.Ident) {
	// Create: workspace_id = 'ws_xxx'
	filter := &clickhouse.BinaryOperation{
		LeftExpr: &clickhouse.NestedIdentifier{
			Ident:    cloneIdentifier(source),
			DotIdent: &clickhouse.Ident{Name: "workspace_id"},
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
		RightExpr: parenthesize(stmt.Where.Expr),
	}
}

func (p *Parser) injectSecurityFilters() {
	for _, securityFilter := range p.config.SecurityFilters {
		// A security filter that is present but has no allowed values means the
		// principal may see nothing for this column, so we fail closed by
		// injecting a constant-false predicate (returning zero rows) rather than
		// skipping the filter, which would leak all rows the other filters allow.
		// This must not be a `continue`: dropping the filter is fail-open.

		walkQueryIncludingExcept(p.stmt, func(node clickhouse.Expr) bool {
			if selectQuery, ok := node.(*clickhouse.SelectQuery); ok {
				for _, source := range p.directTableSources(selectQuery) {
					p.injectSecurityFilterOnSelect(selectQuery, source, securityFilter)
				}
			}
			return true
		})
	}

}

// directTableSources returns the qualifiers for physical tables in this SELECT's
// FROM clause. It deliberately does not descend into subqueries, which are
// visited and filtered as independent SELECT nodes by the caller's AST walk.
func (p *Parser) directTableSources(stmt *clickhouse.SelectQuery) []clickhouse.Ident {
	if stmt.From == nil {
		return nil
	}
	sources := make([]clickhouse.Ident, 0, 1)
	var visit func(clickhouse.Expr)
	visit = func(expr clickhouse.Expr) {
		switch node := expr.(type) {
		case *clickhouse.JoinExpr:
			visit(node.Left)
			if node.Right != nil {
				visit(node.Right)
			}
		case *clickhouse.JoinTableExpr:
			visit(node.Table)
		case *clickhouse.TableExpr:
			tableExpr := node.Expr
			var alias *clickhouse.Ident
			if aliased, ok := tableExpr.(*clickhouse.AliasExpr); ok {
				tableExpr = aliased.Expr
				alias, _ = aliased.Alias.(*clickhouse.Ident)
			}
			table, ok := tableExpr.(*clickhouse.TableIdentifier)
			// CTE references are always unqualified. A qualified physical table may
			// have the same name as a CTE, especially after alias rewriting, and
			// must still receive row-level filters.
			if !ok || (table.Database == nil && p.isCTE(table.Table.Name)) {
				return
			}
			qualifier := *table.Table
			if alias != nil {
				qualifier = *alias
			}
			sources = append(sources, qualifier)
		}
	}
	visit(stmt.From.Expr)
	return sources
}

// cloneIdentifier returns an independent identifier so generated predicates
// retain source quoting without sharing mutable AST pointers.
func cloneIdentifier(ident clickhouse.Ident) *clickhouse.Ident {
	clone := ident
	return &clone
}

// injectSecurityFilterOnSelect injects a security filter on a single SELECT statement
func (p *Parser) injectSecurityFilterOnSelect(stmt *clickhouse.SelectQuery, source clickhouse.Ident, securityFilter SecurityFilter) {
	var filter clickhouse.Expr
	if len(securityFilter.AllowedValues) == 0 {
		// Fail closed: no allowed values means no rows are visible. We cannot
		// emit `{column} IN ()` since an empty IN list is a syntax error in
		// ClickHouse, so we use a constant-false predicate instead.
		filter = &clickhouse.NumberLiteral{Literal: "0"}
	} else {
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
		filter = &clickhouse.BinaryOperation{
			LeftExpr: &clickhouse.NestedIdentifier{
				Ident:    cloneIdentifier(source),
				DotIdent: &clickhouse.Ident{Name: securityFilter.Column},
			},
			Operation: "IN",
			RightExpr: &clickhouse.ParamExprList{
				Items: &clickhouse.ColumnExprList{Items: items},
			},
		}
	}

	// Add to WHERE clause
	if stmt.Where == nil {
		stmt.Where = &clickhouse.WhereClause{Expr: filter}
	} else {
		stmt.Where.Expr = &clickhouse.BinaryOperation{
			LeftExpr:  filter,
			Operation: "AND",
			RightExpr: parenthesize(stmt.Where.Expr),
		}
	}
}
