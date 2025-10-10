package queryparser

import (
	clickhouse "github.com/AfterShip/clickhouse-sql-parser/parser"
)

func (p *Parser) injectWorkspaceFilter() error {
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

	if p.stmt.Where == nil {
		p.stmt.Where = &clickhouse.WhereClause{Expr: filter}
		return nil
	}

	p.stmt.Where.Expr = &clickhouse.BinaryOperation{
		LeftExpr:  filter,
		Operation: "AND",
		RightExpr: p.stmt.Where.Expr,
	}

	return nil
}

func (p *Parser) injectSecurityFilters() error {
	for _, securityFilter := range p.config.SecurityFilters {
		if len(securityFilter.AllowedValues) == 0 {
			continue
		}

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
		if p.stmt.Where == nil {
			p.stmt.Where = &clickhouse.WhereClause{Expr: filter}
		} else {
			p.stmt.Where.Expr = &clickhouse.BinaryOperation{
				LeftExpr:  filter,
				Operation: "AND",
				RightExpr: p.stmt.Where.Expr,
			}
		}
	}

	return nil
}
