package queryparser

import (
	"strings"

	clickhouse "github.com/AfterShip/clickhouse-sql-parser/parser"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
)

// ExtractColumnValues parses bounded query input and extracts its original column assertions.
func ExtractColumnValues(query string, columnName string) ([]string, error) {
	stmts, err := parseBoundedStatements(query)
	if err != nil {
		return nil, err
	}
	if len(stmts) == 0 {
		return nil, fault.New("no statements found",
			fault.Code(codes.User.BadRequest.InvalidAnalyticsQuery.URN()),
			fault.Public("No SQL statements found"),
		)
	}

	stmt, ok := stmts[0].(*clickhouse.SelectQuery)
	if !ok {
		return nil, fault.New("only SELECT queries allowed",
			fault.Code(codes.User.BadRequest.InvalidAnalyticsQueryType.URN()),
			fault.Public("Only SELECT queries are allowed"),
		)
	}

	parser := NewParser(Config{
		WorkspaceID:       "",
		TableAliases:      nil,
		AllowedTables:     nil,
		SecurityFilters:   nil,
		Limit:             0,
		MaxQueryRangeDays: 0,
	})
	parser.stmt = stmt
	parser.collectExtractedColumns()
	return parser.ExtractColumn(columnName), nil
}

// ExtractColumn extracts string literals asserted against a column throughout the original query AST.
// It handles =, ==, IN, and GLOBAL IN while ignoring negative conditions.
// Returns a deduplicated slice of values found for the column. Returns empty slice if no values found.
// Must be called after Parse().
func (p *Parser) ExtractColumn(columnName string) []string {
	uniqueValues := p.extractedColumns[strings.ToLower(columnName)]

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

func (p *Parser) collectExtractedColumns() {
	p.extractedColumns = make(map[string]map[string]struct{})
	clickhouse.Walk(p.stmt, func(node clickhouse.Expr) bool {
		binOp, ok := node.(*clickhouse.BinaryOperation)
		if !ok {
			return true
		}
		switch string(binOp.Operation) {
		case string(clickhouse.TokenKindSingleEQ), string(clickhouse.TokenKindDoubleEQ):
			if !p.collectBinaryOperation(binOp.LeftExpr, binOp.RightExpr) {
				p.collectBinaryOperation(binOp.RightExpr, binOp.LeftExpr)
			}
		case "IN", "GLOBAL IN":
			p.collectBinaryOperation(binOp.LeftExpr, binOp.RightExpr)
		}
		return true
	})
}

func (p *Parser) collectBinaryOperation(columnExpr clickhouse.Expr, valueExpr clickhouse.Expr) bool {
	columnName, ok := extractedColumnName(columnExpr)
	if !ok {
		return false
	}

	columnName = strings.ToLower(columnName)
	values, ok := p.extractedColumns[columnName]
	if !ok {
		values = make(map[string]struct{})
		p.extractedColumns[columnName] = values
	}
	extractValues(valueExpr, values)
	return true
}

func extractedColumnName(expr clickhouse.Expr) (string, bool) {
	switch column := expr.(type) {
	case *clickhouse.Ident:
		return column.Name, true
	case *clickhouse.NestedIdentifier:
		if column.DotIdent != nil {
			return column.DotIdent.Name, true
		}
		return column.Ident.Name, true
	case *clickhouse.Path:
		if len(column.Fields) > 0 {
			return column.Fields[len(column.Fields)-1].Name, true
		}
		return "", false
	default:
		return "", false
	}
}

func extractValues(expr clickhouse.Expr, values map[string]struct{}) {
	// Handle single string literal (for = operator)
	strLit, ok := expr.(*clickhouse.StringLiteral)
	if ok {
		values[strLit.Literal] = struct{}{}
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

		values[strLit.Literal] = struct{}{}
	}
}
