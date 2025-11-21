package queryparser

import (
	"fmt"
	"strings"

	clickhouse "github.com/AfterShip/clickhouse-sql-parser/parser"
	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/fault"
)

// Whitelist of allowed ClickHouse functions for analytics queries
var allowedFunctions = map[string]bool{
	// Aggregate functions
	"count":          true,
	"sum":            true,
	"avg":            true,
	"min":            true,
	"max":            true,
	"any":            true,
	"grouparray":     true,
	"groupuniqarray": true,
	"uniq":           true,
	"uniqexact":      true,
	"quantile":       true,

	// Date/Time functions
	// IMPORTANT: When adding new time-related functions here, you must also update
	// pkg/clickhouse/query-parser/time_range.go to handle them in the retention validation logic.
	// See extractTimestampFromExpr() function to add support for new time functions.
	"now":                      true,
	"now64":                    true,
	"today":                    true,
	"todate":                   true,
	"todatetime":               true,
	"todatetime64":             true,
	"tostartofday":             true,
	"tostartofweek":            true,
	"tostartofmonth":           true,
	"tostartofyear":            true,
	"tostartofhour":            true,
	"tostartofminute":          true,
	"date_trunc":               true,
	"formatdatetime":           true,
	"fromunixtimestamp64milli": true,
	"tounixtimestamp64milli":   true,
	"tointervalday":            true,
	"tointervalweek":           true,
	"tointervalmonth":          true,
	"tointervalyear":           true,
	"tointervalhour":           true,
	"tointervalminute":         true,
	"tointervalsecond":         true,
	"tointervalmillisecond":    true,
	"tointervalmicrosecond":    true,
	"tointervalnanosecond":     true,
	"tointervalquarter":        true,

	// String functions
	"lower":      true,
	"upper":      true,
	"substring":  true,
	"concat":     true,
	"length":     true,
	"trim":       true,
	"startswith": true,
	"endswith":   true,

	// Math functions
	"round": true,
	"floor": true,
	"ceil":  true,
	"abs":   true,

	// Conditional functions
	"if":       true,
	"sumif":    true,
	"case":     true,
	"coalesce": true,
	"countif":  true,

	// Type conversion
	"tostring":  true,
	"toint32":   true,
	"toint64":   true,
	"tofloat64": true,

	// Array functions
	"has":         true,
	"hasany":      true,
	"hasall":      true,
	"arrayjoin":   true,
	"arrayfilter": true,
}

// Whitelist of allowed table functions
// Table functions are used in FROM clause and can access external data sources
// Most are blocked by default for security
var allowedTableFunctions = map[string]bool{
	// Currently no table functions are whitelisted
	// If needed, safe ones could be added here, e.g.:
	// "numbers": true, // generates sequence of numbers
}

func (p *Parser) validateFunctions() error {
	var validateErr error

	clickhouse.WalkWithBreak(p.stmt, func(node clickhouse.Expr) bool {
		// Check regular functions
		funcExpr, isFuncExpr := node.(*clickhouse.FunctionExpr)
		if isFuncExpr {
			if funcExpr.Name == nil || funcExpr.Name.Name == "" {
				return true
			}

			funcName := strings.ToLower(funcExpr.Name.Name)
			if allowedFunctions[funcName] {
				return true
			}

			validateErr = fault.New(fmt.Sprintf("function '%s' not allowed", funcName),
				fault.Code(codes.User.BadRequest.InvalidAnalyticsFunction.URN()),
				fault.Public(fmt.Sprintf("Function '%s' is not allowed", funcName)),
			)
			return false
		}

		// Check table functions
		tableFuncExpr, isTableFuncExpr := node.(*clickhouse.TableFunctionExpr)
		if !isTableFuncExpr {
			return true
		}

		if tableFuncExpr.Name == nil {
			return true
		}

		funcName := strings.ToLower(tableFuncExpr.Name.String())
		if funcName == "" {
			return true
		}

		if allowedTableFunctions[funcName] {
			return true
		}

		validateErr = fault.New(fmt.Sprintf("table function '%s' not allowed", funcName),
			fault.Code(codes.User.BadRequest.InvalidAnalyticsFunction.URN()),
			fault.Public(fmt.Sprintf("Table function '%s' is not allowed for security reasons", funcName)),
		)
		return false
	})

	return validateErr
}
