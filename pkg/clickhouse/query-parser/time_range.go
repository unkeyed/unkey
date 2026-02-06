package queryparser

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	clickhouse "github.com/AfterShip/clickhouse-sql-parser/parser"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger"
)

// validateTimeRange ensures the query doesn't access data older than MaxQueryRangeDays
func (p *Parser) validateTimeRange() error {
	if p.config.MaxQueryRangeDays <= 0 {
		// No restriction configured
		return nil
	}

	// Calculate the earliest allowed timestamp
	earliestAllowed := time.Now().AddDate(0, 0, -int(p.config.MaxQueryRangeDays))

	// Walk the query to find time-based WHERE conditions
	hasTimeFilter := false
	var validationErr error

	clickhouse.Walk(p.stmt, func(node clickhouse.Expr) bool {
		selectQuery, ok := node.(*clickhouse.SelectQuery)
		if !ok {
			return true
		}

		if selectQuery.Where == nil {
			return true
		}

		err := p.validateWhereClause(selectQuery.Where.Expr, earliestAllowed, &hasTimeFilter)
		if err != nil {
			validationErr = err
			return false
		}

		return true
	})

	if validationErr != nil {
		return validationErr
	}

	// If querying tables with time columns but no time filter, auto-add one for the full retention period
	if !hasTimeFilter && p.queryAccessesTimeBasedTables() {
		return p.injectDefaultTimeFilter()
	}

	return nil
}

// validateWhereClause recursively validates time conditions in WHERE clause
func (p *Parser) validateWhereClause(expr clickhouse.Expr, earliestAllowed time.Time, hasLowerBoundTimeFilter *bool) error {
	switch e := expr.(type) {
	case *clickhouse.BinaryOperation:
		// Check if this is a time comparison
		info := p.analyzeTimeComparison(e)
		if info.isTimeComparison {
			hasLowerBound, err := p.validateTimeComparison(e, earliestAllowed, info)
			if err != nil {
				return err
			}
			// Only set hasLowerBoundTimeFilter if this comparison establishes a lower bound
			if hasLowerBound {
				*hasLowerBoundTimeFilter = true
			}
		}

		// Recursively check left and right expressions
		if e.LeftExpr != nil {
			if err := p.validateWhereClause(e.LeftExpr, earliestAllowed, hasLowerBoundTimeFilter); err != nil {
				return err
			}
		}

		if e.RightExpr != nil {
			if err := p.validateWhereClause(e.RightExpr, earliestAllowed, hasLowerBoundTimeFilter); err != nil {
				return err
			}
		}

	case *clickhouse.BetweenClause:
		// Check if this is a time BETWEEN comparison
		// BETWEEN always establishes a lower bound
		if p.isTimeColumn(e.Expr) {
			*hasLowerBoundTimeFilter = true
			if err := p.validateBetweenClause(e, earliestAllowed); err != nil {
				return err
			}
		}
	}

	return nil
}

// timeComparisonInfo holds information about a time comparison
type timeComparisonInfo struct {
	isTimeComparison bool
	timeOnLeft       bool // true if time column is on left side, false if on right
}

// analyzeTimeComparison checks if a binary operation is comparing the 'time' column
// and determines which side the time column is on
func (p *Parser) analyzeTimeComparison(op *clickhouse.BinaryOperation) timeComparisonInfo {
	// Check if left side is 'time' column
	if p.isTimeColumn(op.LeftExpr) {
		return timeComparisonInfo{isTimeComparison: true, timeOnLeft: true}
	}

	// Check if right side is 'time' column (for reversed comparisons like '123456 < time')
	if p.isTimeColumn(op.RightExpr) {
		return timeComparisonInfo{isTimeComparison: true, timeOnLeft: false}
	}

	return timeComparisonInfo{isTimeComparison: false, timeOnLeft: false}
}

// validateTimeComparison validates that a time comparison doesn't access data beyond retention.
// Returns (hasLowerBound, error) - hasLowerBound indicates if this comparison establishes a lower bound on time.
func (p *Parser) validateTimeComparison(op *clickhouse.BinaryOperation, earliestAllowed time.Time, info timeComparisonInfo) (bool, error) {
	// Extract the timestamp being compared
	timestamp, err := p.extractTimestamp(op)
	if err != nil {
		// If we can't parse the timestamp, allow it and let ClickHouse handle it
		// But we don't know if it's a lower bound, so return false to be safe
		// (this will cause default time filter injection if no other lower bound exists)
		return false, nil
	}

	// Normalize the operator to always be from the perspective of "time <op> value"
	// If time is on the right side, we need to flip the operator:
	//   value <= time  is equivalent to  time >= value
	//   value < time   is equivalent to  time > value
	//   value >= time  is equivalent to  time <= value
	//   value > time   is equivalent to  time < value
	//   value = time   is equivalent to  time = value
	operation := strings.ToUpper(string(op.Operation))
	if !info.timeOnLeft {
		operation = flipOperator(operation)
	}

	// Now check from the perspective of "time <op> value"
	// Lower bounds: time >= X, time > X, time = X
	// Upper bounds: time <= X, time < X (these don't establish a lower bound)
	switch operation {
	case ">=", ">":
		// time >= X or time > X - this establishes a lower bound
		if timestamp.Before(earliestAllowed) {
			return true, p.retentionExceededError(timestamp, earliestAllowed)
		}
		return true, nil
	case "=":
		// time = X - this establishes both bounds (exact match)
		if timestamp.Before(earliestAllowed) {
			return true, p.retentionExceededError(timestamp, earliestAllowed)
		}
		return true, nil
	case "<=", "<":
		// time <= X or time < X - upper bound only, does NOT establish a lower bound
		return false, nil
	}

	// Unknown operator - don't treat as lower bound
	return false, nil
}

// flipOperator flips a comparison operator (for when time is on the right side)
func flipOperator(op string) string {
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
		return op // = and other operators are symmetric
	}
}

// validateBetweenClause validates that a BETWEEN clause doesn't access data beyond retention
func (p *Parser) validateBetweenClause(between *clickhouse.BetweenClause, earliestAllowed time.Time) error {
	// Extract the lower bound (the "Between" field) from the BETWEEN clause
	// For "time BETWEEN X AND Y", we validate that X is not before earliestAllowed
	lowerBound, err := p.extractTimestampFromExpr(between.Between)
	if err != nil {
		// If we can't parse the timestamp, allow it and let ClickHouse handle it
		return nil
	}

	// Check if the lower bound is before the earliest allowed time
	if lowerBound.Before(earliestAllowed) {
		return p.retentionExceededError(lowerBound, earliestAllowed)
	}

	return nil
}

// retentionExceededError creates a standardized error for when a query exceeds retention
func (p *Parser) retentionExceededError(queriedTime, earliestAllowed time.Time) error {
	return fault.New(
		fmt.Sprintf("query time range exceeds retention period of %d days", p.config.MaxQueryRangeDays),
		fault.Code(codes.User.BadRequest.QueryRangeExceedsRetention.URN()),
		fault.Public(fmt.Sprintf("Cannot query data older than %d days. Your query attempts to access data from %s, but only data from %s onwards is available.",
			p.config.MaxQueryRangeDays,
			queriedTime.Format("2006-01-02"),
			earliestAllowed.Format("2006-01-02"),
		)),
	)
}

// extractTimestamp extracts a timestamp from a comparison expression
func (p *Parser) extractTimestamp(op *clickhouse.BinaryOperation) (time.Time, error) {
	var valueExpr clickhouse.Expr

	// Determine which side has the value (not the 'time' column)
	if p.isTimeColumn(op.LeftExpr) {
		valueExpr = op.RightExpr
	} else {
		valueExpr = op.LeftExpr
	}

	return p.extractTimestampFromExpr(valueExpr)
}

// extractTimestampFromExpr extracts a timestamp from any expression type
func (p *Parser) extractTimestampFromExpr(valueExpr clickhouse.Expr) (time.Time, error) {
	// This is a best effort attempt to extract a timestamp from the expression
	switch v := valueExpr.(type) {
	case *clickhouse.NumberLiteral:
		// Unix timestamp in milliseconds (Int64)
		timestamp, err := strconv.ParseInt(v.Literal, 10, 64)
		if err != nil {
			return time.Time{}, err
		}

		return time.UnixMilli(timestamp), nil
	case *clickhouse.StringLiteral:
		// DateTime or Date string
		// Try common formats
		formats := []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02"}

		for _, format := range formats {
			if t, err := time.Parse(format, v.Literal); err == nil {
				return t, nil
			}
		}

		return time.Time{}, fmt.Errorf("unsupported date format: %s", v.Literal)
	case *clickhouse.FunctionExpr:
		// Handle time functions
		funcName := strings.ToLower(v.Name.Name)
		switch funcName {
		case "now", "now64", "today":
			// These are always current time, so they're within retention
			return time.Now(), nil
		case "tostartofhour", "tostartofday", "tostartofweek", "tostartofmonth",
			"tostartofquarter", "tostartofyear", "tostartofminute",
			"todate", "todatetime", "todatetime64",
			"formatdatetime", "tounixtimestamp64milli":
			// These functions wrap a time expression - extract and evaluate the argument
			if v.Params != nil && v.Params.Items != nil && len(v.Params.Items.Items) > 0 {
				// Get the first argument and recursively evaluate it
				argExpr := v.Params.Items.Items[0]
				return p.extractTimestampFromExpr(argExpr)
			}

			return time.Time{}, fmt.Errorf("time function has no arguments")
		case "date_trunc":
			// date_trunc takes 2 arguments: unit (string) and timestamp
			// We need to extract the second argument (the timestamp)
			if v.Params != nil && v.Params.Items != nil && len(v.Params.Items.Items) >= 2 {
				// Get the second argument (timestamp)
				argExpr := v.Params.Items.Items[1]
				return p.extractTimestampFromExpr(argExpr)
			}

			return time.Time{}, fmt.Errorf("date_trunc requires 2 arguments")
		case "fromunixtimestamp64milli":
			// Converts unix timestamp (milliseconds) to DateTime
			// Extract the numeric argument and convert it
			if v.Params != nil && v.Params.Items != nil && len(v.Params.Items.Items) > 0 {
				argExpr := v.Params.Items.Items[0]
				// Try to extract as a number
				if numLit, ok := argExpr.(*clickhouse.NumberLiteral); ok {
					timestamp, err := strconv.ParseInt(numLit.Literal, 10, 64)
					if err != nil {
						return time.Time{}, err
					}

					return time.UnixMilli(timestamp), nil
				}

				// Could be a more complex expression, recursively evaluate
				return p.extractTimestampFromExpr(argExpr)
			}

			return time.Time{}, fmt.Errorf("fromunixtimestamp64milli has no arguments")
		default:
			// Unsupported time function - log and return current time as safe default
			// RLS policies at database level will enforce retention regardless
			logger.Warn("unsupported time function in retention validation, using current time as safe default",
				"function", funcName,
				"workspace_id", p.config.WorkspaceID,
			)

			return time.Now(), nil
		}
	case *clickhouse.BinaryOperation:
		// Handle expressions like: now() - INTERVAL 60 DAY
		return p.evaluateIntervalExpression(v)
	case *clickhouse.ColumnExpr:
		// ColumnExpr is a wrapper - unwrap and recursively evaluate
		return p.extractTimestampFromExpr(v.Expr)
	default:
		// Unsupported expression type - log and return current time as safe default
		// RLS policies at database level will enforce retention regardless
		logger.Warn(
			"unsupported timestamp expression type in retention validation, using current time as safe default",
			"type", fmt.Sprintf("%T", valueExpr),
			"workspace_id", p.config.WorkspaceID,
		)

		return time.Now(), nil
	}
}

// evaluateIntervalExpression evaluates time arithmetic expressions like "now() - INTERVAL 60 DAY"
func (p *Parser) evaluateIntervalExpression(expr *clickhouse.BinaryOperation) (time.Time, error) {
	operation := strings.ToUpper(string(expr.Operation))

	// Get the base time by recursively evaluating the left expression
	// This handles cases like: now(), today(), or even nested expressions
	baseTime, err := p.extractTimestampFromExpr(expr.LeftExpr)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to evaluate base time: %w", err)
	}

	// Parse the INTERVAL expression on the right side
	switch operation {
	case "-":
		// This should be: INTERVAL N DAY/HOUR/MONTH
		interval, err := p.parseIntervalExpression(expr.RightExpr)
		if err != nil {
			return time.Time{}, err
		}
		return baseTime.Add(-interval), nil
	case "+":
		interval, err := p.parseIntervalExpression(expr.RightExpr)
		if err != nil {
			return time.Time{}, err
		}

		return baseTime.Add(interval), nil
	default:
		return time.Time{}, fmt.Errorf("unsupported operation in interval expression: %s", operation)
	}
}

// parseIntervalExpression parses an INTERVAL expression like "INTERVAL 60 DAY"
func (p *Parser) parseIntervalExpression(expr clickhouse.Expr) (time.Duration, error) {
	// ClickHouse parser represents "INTERVAL 60 DAY" as an IntervalExpr
	interval, ok := expr.(*clickhouse.IntervalExpr)

	if !ok {
		return 0, fmt.Errorf("expected interval expression")

	}

	// Extract the numeric value
	var value int64
	switch v := interval.Expr.(type) {
	case *clickhouse.NumberLiteral:
		var err error
		value, err = strconv.ParseInt(v.Literal, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid interval value: %s", v.Literal)
		}
	default:
		return 0, fmt.Errorf("unsupported interval value type")
	}

	// Convert to duration based on unit
	unit := strings.ToUpper(interval.Unit.String())
	switch unit {
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
		// Approximate: 30 days per month
		return time.Duration(value) * 30 * 24 * time.Hour, nil
	case "YEAR":
		// Approximate: 365 days per year
		return time.Duration(value) * 365 * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("unsupported interval unit: %s", unit)
	}
}

// isTimeColumn checks if an expression is the 'time' column
func (p *Parser) isTimeColumn(expr clickhouse.Expr) bool {
	if ident, ok := expr.(*clickhouse.NestedIdentifier); ok {
		return ident.Ident != nil && strings.EqualFold(ident.Ident.Name, "time")
	}

	if ident, ok := expr.(*clickhouse.Ident); ok {
		return strings.EqualFold(ident.Name, "time")
	}

	return false
}

// queryAccessesTimeBasedTables checks if the query accesses tables that have time columns
func (p *Parser) queryAccessesTimeBasedTables() bool {
	// All analytics tables in the allowed list have time columns
	// We can check if any table is being accessed
	return len(p.config.AllowedTables) > 0
}

// injectDefaultTimeFilter adds a default time filter for the full retention period
func (p *Parser) injectDefaultTimeFilter() error {
	// Create the time filter: time >= now() - INTERVAL N DAY
	timeFilter := &clickhouse.BinaryOperation{
		LeftExpr:  &clickhouse.Ident{Name: "time"},
		Operation: clickhouse.TokenKindGE,
		RightExpr: &clickhouse.BinaryOperation{
			LeftExpr: &clickhouse.FunctionExpr{
				Name: &clickhouse.Ident{Name: "now"},
				Params: &clickhouse.ParamExprList{
					Items: &clickhouse.ColumnExprList{
						Items: []clickhouse.Expr{},
					},
				},
			},
			Operation: clickhouse.TokenKindMinus,
			RightExpr: &clickhouse.IntervalExpr{
				IntervalPos: 1, // Non-zero value required for String() to output "INTERVAL"
				Expr: &clickhouse.NumberLiteral{
					Literal: fmt.Sprintf("%d", p.config.MaxQueryRangeDays),
				},
				Unit: &clickhouse.Ident{Name: "DAY"},
			},
		},
	}

	// Add to WHERE clause or create new one
	if p.stmt.Where == nil {
		p.stmt.Where = &clickhouse.WhereClause{
			Expr: timeFilter,
		}
	} else {
		// Combine with existing WHERE using AND
		p.stmt.Where.Expr = &clickhouse.BinaryOperation{
			LeftExpr:  p.stmt.Where.Expr,
			Operation: "AND",
			RightExpr: timeFilter,
		}
	}

	return nil
}
