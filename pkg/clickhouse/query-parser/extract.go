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
	result := make([]string, 0)

	extractFunc := func(node clickhouse.Expr) bool {
		binOp, ok := node.(*clickhouse.BinaryOperation)
		if !ok {
			return true
		}

		// Match both `column` and `qualifier.column`. Only the terminal
		// identifier names the column in a qualified reference.
		leftIdent, ok := terminalIdentifier(binOp.LeftExpr)
		if !ok || !strings.EqualFold(leftIdent.Name, columnName) {
			return true
		}

		// Only extract from positive assertions (= or IN)
		// Ignore negative operators: !=, NOT IN, <, >, <=, >=
		if binOp.Operation == clickhouse.TokenKindSingleEQ || strings.EqualFold(string(binOp.Operation), "IN") {
			extractValues(binOp.RightExpr, uniqueValues, &result)
		}

		return true
	}

	// Walk all branches and nested SELECTs. Route policy applies to every
	// physical source, so literals hidden in a subquery must also be authorized.
	walkQueryIncludingExcept(p.stmt, extractFunc)

	if len(uniqueValues) == 0 {
		return []string{}
	}

	return result
}

func terminalIdentifier(expr clickhouse.Expr) (*clickhouse.Ident, bool) {
	switch ident := expr.(type) {
	case *clickhouse.Ident:
		return ident, true
	case *clickhouse.NestedIdentifier:
		if ident.DotIdent != nil {
			return ident.DotIdent, true
		}
		return ident.Ident, ident.Ident != nil
	case *clickhouse.Path:
		if len(ident.Fields) == 0 {
			return nil, false
		}
		return ident.Fields[len(ident.Fields)-1], true
	default:
		return nil, false
	}
}

func extractValues(expr clickhouse.Expr, values map[string]bool, result *[]string) {
	// Handle single string literal (for = operator)
	strLit, ok := expr.(*clickhouse.StringLiteral)
	if ok {
		if !values[strLit.Literal] {
			values[strLit.Literal] = true
			*result = append(*result, strLit.Literal)
		}
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

		if !values[strLit.Literal] {
			values[strLit.Literal] = true
			*result = append(*result, strLit.Literal)
		}
	}
}
