package queryparser

import (
	"fmt"
	"strings"

	clickhouse "github.com/AfterShip/clickhouse-sql-parser/parser"
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

	// Date/Time functions
	"now":             true,
	"today":           true,
	"todate":          true,
	"todatetime":      true,
	"tostartofday":    true,
	"tostartofweek":   true,
	"tostartofmonth":  true,
	"tostartofyear":   true,
	"tostartofhour":   true,
	"tostartofminute": true,
	"date_trunc":      true,
	"formatdatetime":  true,

	// String functions
	"lower":     true,
	"upper":     true,
	"substring": true,
	"concat":    true,
	"length":    true,
	"trim":      true,

	// Math functions
	"round": true,
	"floor": true,
	"ceil":  true,
	"abs":   true,

	// Conditional functions
	"if":       true,
	"case":     true,
	"coalesce": true,

	// Type conversion
	"tostring":  true,
	"toint32":   true,
	"toint64":   true,
	"tofloat64": true,
}

func (p *Parser) validateFunctions() error {
	var validateErr error
	clickhouse.Walk(p.stmt, func(node clickhouse.Expr) bool {
		funcExpr, ok := node.(*clickhouse.FunctionExpr)
		if !ok {
			return true
		}

		if funcExpr.Name == nil {
			return true
		}

		funcName := strings.ToLower(funcExpr.Name.Name)
		if funcName == "" {
			return true
		}

		if allowedFunctions[funcName] {
			return true
		}

		validateErr = fault.New(fmt.Sprintf("function '%s' not allowed", funcName), fault.Public(fmt.Sprintf("Function '%s' is not allowed", funcName)))
		return false
	})

	return validateErr
}
