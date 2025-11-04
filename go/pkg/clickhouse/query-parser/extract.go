package queryparser

import (
	"strings"

	clickhouse "github.com/AfterShip/clickhouse-sql-parser/parser"
)

// ExtractColumn extracts all string literal values for a given column name from WHERE and HAVING clauses.
// Only extracts from positive assertions (= and IN operators), ignores negative conditions (!=, NOT IN, <, >, etc).
// Returns a deduplicated slice of values found for the column. Returns empty slice if no values found.
// Must be called after Parse().
func (p *Parser) ExtractColumn(columnName string) []string {
	uniqueValues := make(map[string]bool)

	extractFunc := func(node clickhouse.Expr) bool {
		binOp, ok := node.(*clickhouse.BinaryOperation)
		if !ok {
			return true
		}

		// Check if left side is our column
		leftIdent, ok := binOp.LeftExpr.(*clickhouse.Ident)
		if !ok || !strings.EqualFold(leftIdent.Name, columnName) {
			return true
		}

		// Only extract from positive assertions (= or IN)
		// Ignore negative operators: !=, NOT IN, <, >, <=, >=
		if binOp.Operation == clickhouse.TokenKindSingleEQ || strings.EqualFold(string(binOp.Operation), "IN") {
			extractValues(binOp.RightExpr, uniqueValues)
		}

		return true
	}

	if p.stmt.Where != nil {
		clickhouse.Walk(p.stmt.Where.Expr, extractFunc)
	}

	if p.stmt.Having != nil {
		clickhouse.Walk(p.stmt.Having.Expr, extractFunc)
	}

	if len(uniqueValues) == 0 {
		return []string{}
	}

	// Convert map to slice
	result := make([]string, 0, len(uniqueValues))
	for value := range uniqueValues {
		result = append(result, value)
	}

	return result
}

func extractValues(expr clickhouse.Expr, values map[string]bool) {
	// Handle single string literal (for = operator)
	strLit, ok := expr.(*clickhouse.StringLiteral)
	if ok {
		values[strLit.Literal] = true
		return
	}

	// Handle IN operator: IN ('val1', 'val2', 'val3')
	paramList, ok := expr.(*clickhouse.ParamExprList)
	if !ok {
		return
	}

	if paramList.Items == nil {
		return
	}

	for _, item := range paramList.Items.Items {
		// Each item is wrapped in a ColumnExpr
		colExpr, ok := item.(*clickhouse.ColumnExpr)
		if !ok {
			continue
		}

		strLit, ok := colExpr.Expr.(*clickhouse.StringLiteral)
		if !ok {
			continue
		}

		values[strLit.Literal] = true
	}
}
