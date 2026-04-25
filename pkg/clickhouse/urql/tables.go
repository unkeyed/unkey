package urql

import (
	"strconv"
	"strings"
	"time"

	clickhouse "github.com/AfterShip/clickhouse-sql-parser/parser"
)

// collectCTENames scans the WITH clause and registers every CTE name so they
// won't be treated as logical-table candidates during ownership detection.
func (c *compiler) collectCTENames() {
	if c.stmt.With == nil {
		return
	}
	for _, cte := range c.stmt.With.CTEs {
		ident, ok := cte.Expr.(*clickhouse.Ident)
		if !ok {
			continue
		}
		c.cteNames[strings.ToLower(ident.Name)] = true
	}
}

// detectOwnership walks every TableIdentifier (including those inside JOINs,
// subqueries, and CTE bodies) and decides whether URQL owns this query.
//
// URQL owns the query if any TableIdentifier matches a logical table name in
// the schema. CTE-bound names are excluded from the check.
func (c *compiler) detectOwnership() (bool, error) {
	var owned bool
	var found *LogicalTable

	clickhouse.Walk(c.stmt, func(node clickhouse.Expr) bool {
		ti, ok := node.(*clickhouse.TableIdentifier)
		if !ok {
			return true
		}
		if ti.Database != nil {
			// Database-qualified names (e.g. default.foo) are never logical.
			return true
		}
		name := ti.Table.Name
		if c.cteNames[strings.ToLower(name)] {
			return true
		}
		if t := c.schema.Lookup(name); t != nil {
			owned = true
			found = t
		}
		return true
	})

	c.table = found
	return owned, nil
}

// validateAllTablesAreLogical, given that URQL has decided to own the query,
// rejects any non-CTE table reference that isn't a known logical table.
// This catches "mixed" queries like `key_verifications JOIN default.legacy_x`
// where the user accidentally combined URQL and physical names.
func (c *compiler) validateAllTablesAreLogical() error {
	var firstErr error

	clickhouse.WalkWithBreak(c.stmt, func(node clickhouse.Expr) bool {
		ti, ok := node.(*clickhouse.TableIdentifier)
		if !ok {
			return true
		}

		fullName := ti.Table.Name
		if ti.Database != nil {
			fullName = ti.Database.Name + "." + ti.Table.Name
		}

		if c.cteNames[strings.ToLower(ti.Table.Name)] && ti.Database == nil {
			return true
		}

		if ti.Database == nil && c.schema.Lookup(ti.Table.Name) != nil {
			return true
		}

		firstErr = urqlErrorf("table '%s' is not a URQL logical table; URQL queries must reference only logical tables (got mixed legacy/URQL usage)", fullName)
		return false
	})

	return firstErr
}

// detectTimeBucket walks the AST and sets useTimeBucket if any timeBucket()
// call is present anywhere in the query.
func (c *compiler) detectTimeBucket() {
	clickhouse.Walk(c.stmt, func(node clickhouse.Expr) bool {
		fe, ok := node.(*clickhouse.FunctionExpr)
		if !ok || fe.Name == nil {
			return true
		}
		if strings.EqualFold(fe.Name.Name, "timeBucket") {
			c.useTimeBucket = true
		}
		return true
	})
}

// resolveVariant picks the physical variant for the URQL logical table.
//
// Rules:
//   - No timeBucket(): use raw.
//   - timeBucket() present: use the variant whose native granularity covers
//     the WHERE-clause lower-bound time range. Falls back to per_hour when
//     no time filter is present (the legacy parser will inject one later
//     for retention enforcement).
func (c *compiler) resolveVariant() error {
	if !c.useTimeBucket {
		c.granularity = GranularityRaw
		return nil
	}

	rangeDuration, ok := c.detectTimeRange()
	if !ok {
		c.granularity = GranularityPerHour
		return nil
	}

	// Round to the nearest hour so a query like `time > now() - INTERVAL 7 DAY`
	// — which by the time we evaluate it has accumulated a few microseconds
	// past 7 days due to repeated calls to time.Now() — still picks the
	// granularity the user intended.
	rounded := rangeDuration.Round(time.Hour)
	switch {
	case rounded <= 6*time.Hour:
		c.granularity = GranularityPerMinute
	case rounded <= 7*24*time.Hour:
		c.granularity = GranularityPerHour
	case rounded <= 90*24*time.Hour:
		c.granularity = GranularityPerDay
	default:
		c.granularity = GranularityPerMonth
	}
	return nil
}

// detectTimeRange returns the duration between the earliest lower-bound time
// found in the outermost WHERE/HAVING clause and now. Returns (0, false) if
// no time bound is found.
func (c *compiler) detectTimeRange() (time.Duration, bool) {
	if c.stmt.Where == nil {
		return 0, false
	}
	earliest, ok := c.findEarliestTimeBound(c.stmt.Where.Expr)
	if !ok {
		return 0, false
	}
	d := time.Since(earliest)
	if d < 0 {
		return 0, false
	}
	return d, true
}

func (c *compiler) findEarliestTimeBound(expr clickhouse.Expr) (time.Time, bool) {
	switch e := expr.(type) {
	case *clickhouse.BinaryOperation:
		if t, ok := c.timeBoundFromBinaryOp(e); ok {
			return t, true
		}
		// Recurse into AND/OR
		var earliest time.Time
		var found bool
		if e.LeftExpr != nil {
			if t, ok := c.findEarliestTimeBound(e.LeftExpr); ok {
				earliest = t
				found = true
			}
		}
		if e.RightExpr != nil {
			if t, ok := c.findEarliestTimeBound(e.RightExpr); ok {
				if !found || t.Before(earliest) {
					earliest = t
					found = true
				}
			}
		}
		return earliest, found
	case *clickhouse.BetweenClause:
		if !c.isTimeColumn(e.Expr) {
			return time.Time{}, false
		}
		if t, err := evalTimeExpr(e.Between); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// timeBoundFromBinaryOp interprets `time op value` (or its mirror) and
// returns the lower-bound timestamp implied by the comparison. Only `>=`,
// `>`, and `=` operations on the time column produce a lower bound.
func (c *compiler) timeBoundFromBinaryOp(op *clickhouse.BinaryOperation) (time.Time, bool) {
	timeOnLeft := c.isTimeColumn(op.LeftExpr)
	timeOnRight := c.isTimeColumn(op.RightExpr)
	if !timeOnLeft && !timeOnRight {
		return time.Time{}, false
	}

	operation := strings.ToUpper(string(op.Operation))
	if !timeOnLeft {
		operation = flipTimeOp(operation)
	}

	var valueExpr clickhouse.Expr
	if timeOnLeft {
		valueExpr = op.RightExpr
	} else {
		valueExpr = op.LeftExpr
	}

	switch operation {
	case ">=", ">", "=":
		t, err := evalTimeExpr(valueExpr)
		if err != nil {
			return time.Time{}, false
		}
		return t, true
	}
	return time.Time{}, false
}

func flipTimeOp(op string) string {
	switch op {
	case ">=":
		return "<="
	case ">":
		return "<"
	case "<=":
		return ">="
	case "<":
		return ">"
	default:
		return op
	}
}

func (c *compiler) isTimeColumn(expr clickhouse.Expr) bool {
	if c.table == nil {
		return false
	}
	name := c.table.TimeColumn
	switch e := expr.(type) {
	case *clickhouse.Ident:
		return strings.EqualFold(e.Name, name)
	case *clickhouse.NestedIdentifier:
		return e.Ident != nil && strings.EqualFold(e.Ident.Name, name)
	}
	return false
}

// evalTimeExpr is a best-effort time-expression evaluator. It handles the
// shapes URQL needs for variant picking: now()/today() arithmetic with
// INTERVAL, raw numbers (treated as unix milli), and quoted timestamps.
func evalTimeExpr(expr clickhouse.Expr) (time.Time, error) {
	switch v := expr.(type) {
	case *clickhouse.NumberLiteral:
		ts, err := strconv.ParseInt(v.Literal, 10, 64)
		if err != nil {
			return time.Time{}, err
		}
		return time.UnixMilli(ts), nil
	case *clickhouse.StringLiteral:
		for _, format := range []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02"} {
			if t, err := time.Parse(format, v.Literal); err == nil {
				return t, nil
			}
		}
		return time.Time{}, errUnparsableTime
	case *clickhouse.FunctionExpr:
		switch strings.ToLower(v.Name.Name) {
		case "now", "now64", "today":
			return time.Now(), nil
		case "fromunixtimestamp64milli":
			if v.Params != nil && v.Params.Items != nil && len(v.Params.Items.Items) > 0 {
				return evalTimeExpr(v.Params.Items.Items[0])
			}
		}
		return time.Time{}, errUnparsableTime
	case *clickhouse.BinaryOperation:
		base, err := evalTimeExpr(v.LeftExpr)
		if err != nil {
			return time.Time{}, err
		}
		interval, err := evalIntervalExpr(v.RightExpr)
		if err != nil {
			return time.Time{}, err
		}
		operation := strings.ToUpper(string(v.Operation))
		switch operation {
		case "-":
			return base.Add(-interval), nil
		case "+":
			return base.Add(interval), nil
		}
		return time.Time{}, errUnparsableTime
	case *clickhouse.ColumnExpr:
		return evalTimeExpr(v.Expr)
	}
	return time.Time{}, errUnparsableTime
}

func evalIntervalExpr(expr clickhouse.Expr) (time.Duration, error) {
	interval, ok := expr.(*clickhouse.IntervalExpr)
	if !ok {
		return 0, errUnparsableTime
	}
	num, ok := interval.Expr.(*clickhouse.NumberLiteral)
	if !ok {
		return 0, errUnparsableTime
	}
	value, err := strconv.ParseInt(num.Literal, 10, 64)
	if err != nil {
		return 0, err
	}
	switch strings.ToUpper(interval.Unit.String()) {
	case "SECOND":
		return time.Duration(value) * time.Second, nil
	case "MINUTE":
		return time.Duration(value) * time.Minute, nil
	case "HOUR":
		return time.Duration(value) * time.Hour, nil
	case "DAY":
		return time.Duration(value) * 24 * time.Hour, nil
	case "WEEK":
		return time.Duration(value) * 7 * 24 * time.Hour, nil
	case "MONTH":
		return time.Duration(value) * 30 * 24 * time.Hour, nil
	case "YEAR":
		return time.Duration(value) * 365 * 24 * time.Hour, nil
	}
	return 0, errUnparsableTime
}

// errUnparsableTime is a sentinel for time expressions we don't know how to
// evaluate. Callers fall back to safe defaults rather than rejecting the
// query.
var errUnparsableTime = stringErr("urql: unparsable time expression")

type stringErr string

func (e stringErr) Error() string { return string(e) }

// rewriteTables replaces every TableIdentifier referencing the URQL logical
// table with the resolved physical variant. CTE-bound names are left alone.
func (c *compiler) rewriteTables() error {
	if c.table == nil {
		return nil
	}
	physical, ok := c.table.Variants[c.granularity]
	if !ok {
		return urqlErrorf("urql: logical table '%s' has no variant for granularity %d", c.table.Name, c.granularity)
	}
	parts := strings.Split(physical, ".")

	clickhouse.Walk(c.stmt, func(node clickhouse.Expr) bool {
		ti, ok := node.(*clickhouse.TableIdentifier)
		if !ok {
			return true
		}
		if ti.Database != nil {
			return true
		}
		if !strings.EqualFold(ti.Table.Name, c.table.Name) {
			return true
		}
		if c.cteNames[strings.ToLower(ti.Table.Name)] {
			return true
		}
		switch len(parts) {
		case 2:
			ti.Database = &clickhouse.Ident{Name: parts[0]}
			ti.Table = &clickhouse.Ident{Name: parts[1]}
		case 1:
			ti.Database = nil
			ti.Table = &clickhouse.Ident{Name: parts[0]}
		}
		return true
	})
	return nil
}
