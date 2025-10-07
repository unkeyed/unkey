package clickhouse

import (
	"errors"
	"strings"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/fault"
)

// Common user error patterns in ClickHouse error messages
var userErrorPatterns = map[string]bool{
	"unknown identifier":        true,
	"unknown column":            true,
	"unknown table":             true,
	"missing columns":           true,
	"there is no column":        true,
	"type mismatch":             true,
	"cannot convert":            true,
	"syntax error":              true,
	"expected":                  true,
	"illegal type":              true,
	"ambiguous column":          true,
	"not an aggregate function": true,
	"division by zero":          true,
	"aggregate function":        true,
	"window function":           true,
}

// ClickHouse exception codes that indicate user query errors
var userErrorCodes = map[int32]bool{
	47:  true, // UNKNOWN_IDENTIFIER
	60:  true, // UNKNOWN_TABLE
	62:  true, // SYNTAX_ERROR
	386: true, // ILLEGAL_TYPE_OF_ARGUMENT
	43:  true, // ILLEGAL_COLUMN
	352: true, // AMBIGUOUS_COLUMN_NAME
}

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

	// Check error message patterns
	for pattern := range userErrorPatterns {
		if strings.Contains(errMsg, pattern) {
			return true
		}
	}

	// Check ClickHouse exception codes
	var chErr *ch.Exception
	if errors.As(err, &chErr) {
		return userErrorCodes[chErr.Code]
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

// errorResponse defines a structured error response with code and message
type errorResponse struct {
	code    codes.URN
	message string
}

// resourceLimitPatterns maps error message patterns to error responses
var resourceLimitPatterns = map[string]errorResponse{
	"timeout": {
		code:    codes.User.UnprocessableEntity.QueryExecutionTimeout.URN(),
		message: "Query execution time limit exceeded. Try simplifying your query or reducing the time range.",
	},
	"execution time": {
		code:    codes.User.UnprocessableEntity.QueryExecutionTimeout.URN(),
		message: "Query execution time limit exceeded. Try simplifying your query or reducing the time range.",
	},
	"memory": {
		code:    codes.User.UnprocessableEntity.QueryMemoryLimitExceeded.URN(),
		message: "Query memory limit exceeded. Try simplifying your query or reducing the result set size.",
	},
	"too many rows": {
		code:    codes.User.UnprocessableEntity.QueryRowsLimitExceeded.URN(),
		message: "Query attempted to read too many rows. Try adding more filters or reducing the time range.",
	},
	"limit for rows_to_read": {
		code:    codes.User.UnprocessableEntity.QueryRowsLimitExceeded.URN(),
		message: "Query attempted to read too many rows. Try adding more filters or reducing the time range.",
	},
	"result rows": {
		code:    codes.User.UnprocessableEntity.QueryResultRowsLimitExceeded.URN(),
		message: "Query result set too large. Try adding LIMIT clause or aggregating the data.",
	},
	"max_result_rows": {
		code:    codes.User.UnprocessableEntity.QueryResultRowsLimitExceeded.URN(),
		message: "Query result set too large. Try adding LIMIT clause or aggregating the data.",
	},
	"quota": {
		code:    codes.User.TooManyRequests.QueryQuotaExceeded.URN(),
		message: "Query quota exceeded for the current time window. Please try again later.",
	},
}

// resourceLimitCodes maps ClickHouse exception codes to error responses
var resourceLimitCodes = map[int32]errorResponse{
	159: { // TIMEOUT_EXCEEDED
		code:    codes.User.UnprocessableEntity.QueryExecutionTimeout.URN(),
		message: "Query execution time limit exceeded. Try simplifying your query or reducing the time range.",
	},
	241: { // MEMORY_LIMIT_EXCEEDED
		code:    codes.User.UnprocessableEntity.QueryMemoryLimitExceeded.URN(),
		message: "Query memory limit exceeded. Try simplifying your query or reducing the result set size.",
	},
	396: { // QUERY_WAS_CANCELLED
		code:    codes.User.UnprocessableEntity.QueryExecutionTimeout.URN(),
		message: "Query was cancelled due to resource limits.",
	},
	198: { // TOO_MANY_ROWS
		code:    codes.User.UnprocessableEntity.QueryRowsLimitExceeded.URN(),
		message: "Query attempted to read too many rows. Try adding more filters or reducing the time range.",
	},
	202: { // TOO_MANY_SIMULTANEOUS_QUERIES / QUOTA_EXCEEDED
		code:    codes.User.TooManyRequests.QueryQuotaExceeded.URN(),
		message: "Query quota exceeded for the current time window. Please try again later.",
	},
}

// WrapClickHouseError wraps a ClickHouse error with appropriate error codes and user-friendly messages.
// It detects resource limit violations and other user errors and tags them with specific error codes.
func WrapClickHouseError(err error) error {
	if err == nil {
		return nil
	}

	errMsg := strings.ToLower(err.Error())

	// Check for resource limit violations via message patterns
	for pattern, response := range resourceLimitPatterns {
		if strings.Contains(errMsg, pattern) {
			return fault.Wrap(err,
				fault.Code(response.code),
				fault.Public(response.message),
			)
		}
	}

	// Check ClickHouse exception codes for resource errors
	var chErr *ch.Exception
	if errors.As(err, &chErr) {
		if response, ok := resourceLimitCodes[chErr.Code]; ok {
			return fault.Wrap(err,
				fault.Code(response.code),
				fault.Public(response.message),
			)
		}
	}

	// For other errors, check if it's a user error or system error
	if IsUserQueryError(err) {
		return fault.Wrap(err,
			fault.Code(codes.User.BadRequest.InvalidAnalyticsQuery.URN()),
			fault.Public(ExtractUserFriendlyError(err)),
		)
	}

	// System/infrastructure error - don't expose details
	return fault.Wrap(err,
		fault.Public("Query execution failed. Please try again."),
	)
}
