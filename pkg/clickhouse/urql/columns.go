package urql

import (
	"regexp"
	"slices"
	"strings"

	clickhouse "github.com/AfterShip/clickhouse-sql-parser/parser"
)

// isKnownColumnName reports whether the given name resolves to a column on
// the resolved variant of the URQL logical table.
func (c *compiler) isKnownColumnName(name string) bool {
	if c.table == nil {
		return false
	}
	col, ok := c.lookupColumn(name)
	if !ok {
		return false
	}
	return col.availableOn(c.granularity)
}

// lookupColumn finds a column by case-insensitive name on the URQL logical
// table.
func (c *compiler) lookupColumn(name string) (Column, bool) {
	if c.table == nil {
		return newColumn(), false
	}
	if col, ok := c.table.Columns[name]; ok {
		return col, true
	}
	lower := strings.ToLower(name)
	for k, col := range c.table.Columns {
		if strings.ToLower(k) == lower {
			return col, true
		}
	}
	return newColumn(), false
}

// isSimpleQuery returns true for queries with a single table reference, no
// JOIN, no UNION, no CTE, no subquery. Phase 1 only validates column refs on
// simple queries — JOIN-aware per-scope validation is Phase 2.
func (c *compiler) isSimpleQuery() bool {
	if c.stmt.With != nil && len(c.stmt.With.CTEs) > 0 {
		return false
	}
	if c.stmt.UnionAll != nil || c.stmt.UnionDistinct != nil || c.stmt.Except != nil {
		return false
	}
	if c.stmt.From == nil || c.stmt.From.Expr == nil {
		return false
	}

	simple := true
	clickhouse.Walk(c.stmt.From, func(node clickhouse.Expr) bool {
		switch node.(type) {
		case *clickhouse.JoinExpr, *clickhouse.SelectQuery:
			simple = false
			return false
		}
		return true
	})
	return simple
}

// validateColumnReferences walks the SELECT items, WHERE/HAVING expressions,
// GROUP BY and ORDER BY clauses, and rejects column references that aren't
// in the resolved variant's column set.
func (c *compiler) validateColumnReferences() error {
	if c.table == nil {
		return nil
	}
	if !c.isSimpleQuery() {
		return nil
	}

	var firstErr error
	check := func(ident *clickhouse.Ident) {
		if firstErr != nil {
			return
		}
		if c.isKnownColumnName(ident.Name) {
			return
		}
		firstErr = urqlErrorf("unknown column '%s' on URQL logical table '%s'", ident.Name, c.table.Name)
	}

	for _, item := range c.stmt.SelectItems {
		if item == nil {
			continue
		}
		c.walkColumnRefs(item.Expr, check)
	}
	if c.stmt.Where != nil {
		c.walkColumnRefs(c.stmt.Where.Expr, check)
	}
	if c.stmt.Having != nil {
		c.walkColumnRefs(c.stmt.Having.Expr, check)
	}
	if c.stmt.GroupBy != nil {
		c.walkColumnRefs(c.stmt.GroupBy.Expr, check)
	}
	if c.stmt.OrderBy != nil {
		for _, item := range c.stmt.OrderBy.Items {
			c.walkColumnRefs(item, check)
		}
	}
	return firstErr
}

// walkColumnRefs recursively walks an expression and invokes fn for each
// identifier that is in a column-reference position. Function names, table
// aliases, and column-output aliases are intentionally skipped.
//
// Star expressions (`SELECT *`), wildcards, and literals are no-ops.
func (c *compiler) walkColumnRefs(expr clickhouse.Expr, fn func(*clickhouse.Ident)) {
	if expr == nil {
		return
	}
	switch e := expr.(type) {
	case *clickhouse.Ident:
		// SELECT *, COUNT(*), etc.: asterisk is parsed as an Ident with name "*".
		if e.Name == "*" {
			return
		}
		// timeBucket as bare ident in GROUP BY is allowed; it'll be replaced
		// at the SQL string level. Treat it as a known column for validation.
		if strings.EqualFold(e.Name, "timeBucket") {
			return
		}
		fn(e)
	case *clickhouse.NestedIdentifier:
		// `t.col` — validate the column part. The table-alias part is opaque.
		if e.DotIdent != nil {
			fn(e.DotIdent)
			return
		}
		if e.Ident != nil {
			fn(e.Ident)
		}
	case *clickhouse.Path:
		// `a.request_id` becomes Path{Fields: [a, request_id]}. The last
		// field is the column reference; preceding fields are table aliases.
		if len(e.Fields) > 0 {
			fn(e.Fields[len(e.Fields)-1])
		}
	case *clickhouse.ColumnExpr:
		c.walkColumnRefs(e.Expr, fn)
		// Alias intentionally skipped.
	case *clickhouse.FunctionExpr:
		// Function name skipped. Walk into params.
		if e.Params != nil && e.Params.Items != nil {
			for _, arg := range e.Params.Items.Items {
				c.walkColumnRefs(arg, fn)
			}
		}
	case *clickhouse.BinaryOperation:
		c.walkColumnRefs(e.LeftExpr, fn)
		c.walkColumnRefs(e.RightExpr, fn)
	case *clickhouse.BetweenClause:
		c.walkColumnRefs(e.Expr, fn)
		c.walkColumnRefs(e.Between, fn)
		c.walkColumnRefs(e.And, fn)
	case *clickhouse.ParamExprList:
		if e.Items != nil {
			for _, item := range e.Items.Items {
				c.walkColumnRefs(item, fn)
			}
		}
	}
}

// validateAllowedValues walks WHERE and HAVING for `column = literal` and
// `column IN (literals)` patterns and rejects any literal that isn't in the
// AllowedValues list of its column.
//
// Negative comparisons (!=, NOT IN) are not validated — there's no security
// or correctness reason to restrict what values users can exclude.
func (c *compiler) validateAllowedValues() error {
	if c.table == nil {
		return nil
	}

	var firstErr error
	walk := func(expr clickhouse.Expr) {
		if expr == nil || firstErr != nil {
			return
		}
		clickhouse.WalkWithBreak(expr, func(node clickhouse.Expr) bool {
			if firstErr != nil {
				return false
			}
			binOp, ok := node.(*clickhouse.BinaryOperation)
			if !ok {
				return true
			}

			columnName := identNameFromExpr(binOp.LeftExpr)
			if columnName == "" {
				return true
			}
			column, ok := c.lookupColumn(columnName)
			if !ok || len(column.AllowedValues) == 0 {
				return true
			}

			operation := strings.ToUpper(string(binOp.Operation))
			if operation != "=" && operation != "IN" {
				return true
			}

			values := extractStringValues(binOp.RightExpr)
			for _, v := range values {
				if !slices.Contains(column.AllowedValues, v) {
					firstErr = urqlErrorf("invalid value '%s' for column '%s'; allowed values: %s",
						v, columnName, strings.Join(column.AllowedValues, ", "))
					return false
				}
			}
			return true
		})
	}

	if c.stmt.Where != nil {
		walk(c.stmt.Where.Expr)
	}
	if c.stmt.Having != nil {
		walk(c.stmt.Having.Expr)
	}
	return firstErr
}

func identNameFromExpr(expr clickhouse.Expr) string {
	switch e := expr.(type) {
	case *clickhouse.Ident:
		return e.Name
	case *clickhouse.NestedIdentifier:
		if e.DotIdent != nil {
			return e.DotIdent.Name
		}
		if e.Ident != nil {
			return e.Ident.Name
		}
	case *clickhouse.Path:
		if len(e.Fields) > 0 {
			return e.Fields[len(e.Fields)-1].Name
		}
	}
	return ""
}

func extractStringValues(expr clickhouse.Expr) []string {
	switch e := expr.(type) {
	case *clickhouse.StringLiteral:
		return []string{e.Literal}
	case *clickhouse.ColumnExpr:
		return extractStringValues(e.Expr)
	case *clickhouse.ParamExprList:
		if e.Items == nil {
			return nil
		}
		var values []string
		for _, item := range e.Items.Items {
			values = append(values, extractStringValues(item)...)
		}
		return values
	}
	return nil
}

// replaceVirtualColumns substitutes virtual-column references in the emitted
// SQL with their physical expressions. Done at the SQL string level after
// AST emission to avoid the parent-replacement gymnastics that AST-level
// mutation would require for every Expr-bearing node.
//
// Word boundaries protect against partial matches (`is_successful_count`
// stays intact). Aliases (`AS is_successful`) are not specially protected;
// virtual column names should be chosen to be unlikely user-facing aliases.
// The output flows back through the legacy parser, so any syntax issue
// from a corner case surfaces there.
func replaceVirtualColumns(sql string, table *LogicalTable, g Granularity) string {
	if table == nil {
		return sql
	}
	for name, col := range table.Columns {
		if col.Expression == "" {
			continue
		}
		if !col.availableOn(g) {
			continue
		}
		re := regexp.MustCompile(`\b` + regexp.QuoteMeta(name) + `\b`)
		sql = re.ReplaceAllString(sql, "("+col.Expression+")")
	}
	return sql
}
