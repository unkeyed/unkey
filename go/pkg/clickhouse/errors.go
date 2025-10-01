package clickhouse

import (
	"errors"
	"strings"

	ch "github.com/ClickHouse/clickhouse-go/v2"
)

// IsUserQueryError checks if the ClickHouse error is due to a bad query (user error)
// vs a system/infrastructure error.
//
// Returns true for errors like:
// - Unknown column/identifier
// - Type mismatches
// - Syntax errors
// - Division by zero
//
// Returns false for errors like:
// - Connection failures
// - Timeouts
// - Infrastructure issues
func IsUserQueryError(err error) bool {
	if err == nil {
		return false
	}

	errMsg := strings.ToLower(err.Error())

	// Common user error patterns in ClickHouse
	userErrorPatterns := []string{
		"unknown identifier",
		"unknown column",
		"unknown table",
		"missing columns",
		"there is no column",
		"type mismatch",
		"cannot convert",
		"syntax error",
		"expected",
		"illegal type",
		"ambiguous column",
		"not an aggregate function",
		"division by zero",
		"aggregate function",
		"window function",
	}

	for _, pattern := range userErrorPatterns {
		if strings.Contains(errMsg, pattern) {
			return true
		}
	}

	// Check ClickHouse exception codes for user errors
	var chErr *ch.Exception
	if errors.As(err, &chErr) {
		// Common user error codes in ClickHouse
		userErrorCodes := []int32{
			47,  // UNKNOWN_IDENTIFIER
			60,  // UNKNOWN_TABLE
			62,  // SYNTAX_ERROR
			386, // ILLEGAL_TYPE_OF_ARGUMENT
			43,  // ILLEGAL_COLUMN
			352, // AMBIGUOUS_COLUMN_NAME
		}

		for _, code := range userErrorCodes {
			if chErr.Code == code {
				return true
			}
		}
	}

	return false
}

// ExtractUserFriendlyError extracts a user-friendly error message from ClickHouse error.
// It cleans up internal ClickHouse formatting and truncates long messages.
func ExtractUserFriendlyError(err error) string {
	if err == nil {
		return "Query failed"
	}

	errMsg := err.Error()

	// Try to extract the meaningful part of the error message
	// ClickHouse errors often have format: "code: XXX, message: actual message, ..."
	if idx := strings.Index(errMsg, "message: "); idx != -1 {
		errMsg = errMsg[idx+9:] // Skip "message: "
		// Find the end (usually a comma or newline)
		if endIdx := strings.Index(errMsg, ","); endIdx != -1 {
			errMsg = errMsg[:endIdx]
		}
	}

	// Clean up common prefixes
	errMsg = strings.TrimPrefix(errMsg, "clickhouse: ")
	errMsg = strings.TrimPrefix(errMsg, "code: ")

	// If the message is too long, truncate it
	if len(errMsg) > 200 {
		errMsg = errMsg[:200] + "..."
	}

	return errMsg
}
