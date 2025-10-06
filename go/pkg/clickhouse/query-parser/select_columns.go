package queryparser

import (
	clickhouse "github.com/AfterShip/clickhouse-sql-parser/parser"
)

// rewriteSelectColumns rewrites virtual columns in the SELECT clause to actual columns
// and tracks which virtual columns are used
func (p *Parser) rewriteSelectColumns() error {
	if p.stmt.SelectItems == nil {
		return nil
	}

	for _, item := range p.stmt.SelectItems {
		p.rewriteSelectItem(item)
	}

	// Also check GROUP BY clause
	if p.stmt.GroupBy != nil && p.stmt.GroupBy.Expr != nil {
		p.rewriteExpr(p.stmt.GroupBy.Expr) // GROUP BY doesn't have aliases
	}

	return nil
}

func (p *Parser) rewriteSelectItem(item *clickhouse.SelectItem) {
	if item == nil || item.Expr == nil {
		return
	}

	// Get the original column name before rewriting
	originalColumn := p.getColumnName(item.Expr)

	// Get the result column name (alias if present)
	resultColumn := ""
	if item.Alias != nil {
		resultColumn = item.Alias.Name
	}

	// Track the actual column after rewriting
	actualColumn := p.rewriteExpr(item.Expr)

	// If a virtual column was rewritten and there's no alias, add one to preserve the virtual name
	if actualColumn != "" && actualColumn != originalColumn && resultColumn == "" {
		// Add alias to preserve the original virtual column name
		item.Alias = &clickhouse.Ident{Name: originalColumn}
		resultColumn = originalColumn
	}

	// If no alias and no rewriting, the result column is the actual column
	if resultColumn == "" {
		resultColumn = actualColumn
	}

	// Only track mappings when the column was actually rewritten (virtual column -> actual column)
	if actualColumn != "" && actualColumn != originalColumn {
		p.columnMappings[resultColumn] = actualColumn
	}
}

func (p *Parser) getColumnName(expr clickhouse.Expr) string {
	if expr == nil {
		return ""
	}

	// Check if it's directly an Ident
	if ident, ok := expr.(*clickhouse.Ident); ok {
		return ident.Name
	}

	// For ColumnExpr with Ident
	if colExpr, ok := expr.(*clickhouse.ColumnExpr); ok {
		if ident, ok := colExpr.Expr.(*clickhouse.Ident); ok {
			return ident.Name
		}
	}

	return ""
}

// resolveVirtualColumn checks if an identifier matches a virtual column (canonical name or alias),
// updates the identifier to the actual column name, and returns the actual column name.
// Returns empty string if the identifier is not a virtual column.
func (p *Parser) resolveVirtualColumn(ident *clickhouse.Ident) string {
	if ident == nil {
		return ""
	}

	// Check if this ident matches any virtual column
	for canonicalName, vcConfig := range p.config.VirtualColumns {
		if ident.Name == canonicalName {
			ident.Name = vcConfig.ActualColumn
			return vcConfig.ActualColumn
		}
		// Check aliases
		for _, alias := range vcConfig.Aliases {
			if ident.Name == alias {
				ident.Name = vcConfig.ActualColumn
				return vcConfig.ActualColumn
			}
		}
	}

	return ""
}

func (p *Parser) rewriteExpr(expr clickhouse.Expr) string {
	if expr == nil {
		return ""
	}

	// Check if it's directly an Ident
	if ident, ok := expr.(*clickhouse.Ident); ok {
		if actualColumn := p.resolveVirtualColumn(ident); actualColumn != "" {
			return actualColumn
		}
		// Not a virtual column, return the name as-is
		return ident.Name
	}

	// For ColumnExpr with Ident (for complex expressions), check and rewrite
	if colExpr, ok := expr.(*clickhouse.ColumnExpr); ok {
		if ident, ok := colExpr.Expr.(*clickhouse.Ident); ok {
			if actualColumn := p.resolveVirtualColumn(ident); actualColumn != "" {
				return actualColumn
			}
			return ident.Name
		}
	}

	return ""
}

func (p *Parser) rewriteColumnRef(expr clickhouse.Expr, virtualName, actualName string) {
	// If this is a ColumnExpr with an Ident that matches, rewrite it
	if colExpr, ok := expr.(*clickhouse.ColumnExpr); ok {
		if ident, ok := colExpr.Expr.(*clickhouse.Ident); ok {
			if ident.Name == virtualName {
				ident.Name = actualName
			}
		}
	}

	// Handle nested ColumnExpr
	switch e := expr.(type) {
	case *clickhouse.ColumnExpr:
		if e.Expr != nil {
			p.rewriteColumnRef(e.Expr, virtualName, actualName)
		}
	}
}
