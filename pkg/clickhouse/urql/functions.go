package urql

import (
	"regexp"
	"strings"

	clickhouse "github.com/AfterShip/clickhouse-sql-parser/parser"
)

// allowedPrettyFormats are the format hints accepted by prettyFormat().
// These are returned alongside query results so consumers (SDK, dashboard)
// know how to render the column.
var allowedPrettyFormats = map[string]bool{
	"duration":        true,
	"durationSeconds": true,
	"bytes":           true,
	"decimalBytes":    true,
	"quantity":        true,
	"percent":         true,
}

// validateTimeBucketArgs ensures every `timeBucket()` call has zero arguments.
// Phase 1 picks the bucket size automatically from the WHERE-clause time
// range, so user-supplied bucket sizes aren't supported yet.
func (c *compiler) validateTimeBucketArgs() error {
	if !c.useTimeBucket {
		return nil
	}

	var argErr error
	clickhouse.WalkWithBreak(c.stmt, func(node clickhouse.Expr) bool {
		fe, ok := node.(*clickhouse.FunctionExpr)
		if !ok || fe.Name == nil {
			return true
		}
		if !strings.EqualFold(fe.Name.Name, "timeBucket") {
			return true
		}
		if fe.Params != nil && fe.Params.Items != nil && len(fe.Params.Items.Items) > 0 {
			argErr = urqlError(
				"timeBucket() does not accept arguments in URQL Phase 1",
				"timeBucket() does not accept arguments yet. Call it without arguments and the bucket size will be chosen automatically.",
			)
			return false
		}
		return true
	})
	if argErr != nil {
		return argErr
	}
	return nil
}

// timeBucketCallRe matches `timeBucket()` (with optional whitespace).
var timeBucketCallRe = regexp.MustCompile(`(?i)\btimeBucket\s*\(\s*\)`)

// timeBucketIdentRe matches a bare `timeBucket` identifier with word boundaries
// — for the common pattern `GROUP BY timeBucket` where users treat the function
// as an alias.
var timeBucketIdentRe = regexp.MustCompile(`(?i)\btimeBucket\b`)

// replaceTimeBucket rewrites both `timeBucket()` and bare `timeBucket` in the
// emitted SQL to the table's time column name. Doing this at the SQL string
// level avoids the parent-replacement gymnastics that AST-level mutation
// would require, and the result still flows through the legacy parser for
// re-parse + validation.
func replaceTimeBucket(sql, timeColumn string) string {
	sql = timeBucketCallRe.ReplaceAllString(sql, timeColumn)
	sql = timeBucketIdentRe.ReplaceAllString(sql, timeColumn)
	return sql
}

// extractPrettyFormat finds every `prettyFormat(expr, 'format')` call appearing
// as a top-level SELECT item (`SELECT prettyFormat(...) AS x FROM ...`),
// records the format hint keyed by the alias, and replaces the SelectItem's
// expression with the inner expression so the emitted SQL contains just
// `expr` (preserving the alias).
//
// The walk descends into nested SelectQueries so subquery prettyFormat()
// calls in the projection are also processed.
func (c *compiler) extractPrettyFormat() (map[string]string, error) {
	var formats map[string]string
	var firstErr error

	clickhouse.WalkWithBreak(c.stmt, func(node clickhouse.Expr) bool {
		if firstErr != nil {
			return false
		}
		sq, ok := node.(*clickhouse.SelectQuery)
		if !ok {
			return true
		}
		for _, item := range sq.SelectItems {
			if item == nil {
				continue
			}
			fe, ok := item.Expr.(*clickhouse.FunctionExpr)
			if !ok || fe.Name == nil {
				continue
			}
			if !strings.EqualFold(fe.Name.Name, "prettyFormat") {
				continue
			}

			if fe.Params == nil || fe.Params.Items == nil || len(fe.Params.Items.Items) != 2 {
				firstErr = urqlError(
					"prettyFormat() requires exactly two arguments: an expression and a format string",
					"prettyFormat(expr, 'format') requires exactly two arguments.",
				)
				return false
			}

			innerExpr, formatStr, err := splitPrettyFormatArgs(fe.Params.Items.Items)
			if err != nil {
				firstErr = err
				return false
			}

			if !allowedPrettyFormats[formatStr] {
				firstErr = urqlErrorf("prettyFormat: unknown format '%s'; allowed formats are duration, durationSeconds, bytes, decimalBytes, quantity, percent", formatStr)
				return false
			}

			if item.Alias == nil || item.Alias.Name == "" {
				firstErr = urqlError(
					"prettyFormat() must be aliased",
					"prettyFormat(...) calls must be aliased: prettyFormat(avg(latency), 'duration') AS avg_latency",
				)
				return false
			}

			if formats == nil {
				formats = make(map[string]string)
			}
			formats[item.Alias.Name] = formatStr

			item.Expr = innerExpr
		}
		return true
	})

	if firstErr != nil {
		return nil, firstErr
	}
	return formats, nil
}

func splitPrettyFormatArgs(items []clickhouse.Expr) (clickhouse.Expr, string, error) {
	innerWrapped := items[0]
	formatWrapped := items[1]

	innerExpr := unwrapColumnExpr(innerWrapped)
	formatExpr := unwrapColumnExpr(formatWrapped)

	strLit, ok := formatExpr.(*clickhouse.StringLiteral)
	if !ok {
		return nil, "", urqlError(
			"prettyFormat: second argument must be a string literal",
			"prettyFormat(expr, 'format') second argument must be a string literal like 'duration'.",
		)
	}
	return innerExpr, strLit.Literal, nil
}

func unwrapColumnExpr(expr clickhouse.Expr) clickhouse.Expr {
	if col, ok := expr.(*clickhouse.ColumnExpr); ok {
		return col.Expr
	}
	return expr
}
