package clickhouse

import (
	"context"
	"errors"
	"strings"

	ch "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
)

// Common user error patterns in ClickHouse error messages
var userErrorPatterns = map[string]bool{
	"unknown identifier":          true,
	"unknown expression":          true,
	"unknown function":            true,
	"unknown column":              true,
	"unknown table":               true,
	"missing columns":             true,
	"there is no column":          true,
	"type mismatch":               true,
	"cannot convert":              true,
	"syntax error":                true,
	"expected":                    true,
	"illegal type":                true,
	"ambiguous column":            true,
	"not an aggregate function":   true,
	"division by zero":            true,
	"aggregate function":          true,
	"window function":             true,
	"unknown_identifier":          true, // ClickHouse error code name
	"db::exception":               true, // Treat all DB exceptions as user errors
	"maybe you meant":             true, // ClickHouse suggestions
	"no such column":              true,
	"doesn't exist":               true,
	"does not exist":              true,
	"failed at position":          true,
	"unexpected token":            true,
	"invalid expression":          true,
	"invalid number of arguments": true,
	"wrong number of arguments":   true,
	"cannot parse":                true,
	"unrecognized token":          true,
	"no matching signature":       true,
	"incompatible types":          true,
	"illegal aggregation":         true,
	"cannot find column":          true,
	"not allowed in this context": true,
	"not supported":               true,
	"invalid combination":         true,
	"invalid or illegal":          true,
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

// ExtractUserFriendlyError maps ClickHouse errors to stable public messages
// without exposing rewritten SQL, physical table names, or injected filters.
func ExtractUserFriendlyError(err error) string {
	if err == nil {
		return "Invalid analytics query"
	}

	errMsg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(errMsg, "not enough privileges"):
		return "The query reads a column that is not available. Select only the documented columns instead of *"
	case strings.Contains(errMsg, "unknown function"),
		strings.Contains(errMsg, "no matching signature"):
		return "Unknown function in analytics query"
	case strings.Contains(errMsg, "unknown identifier"),
		strings.Contains(errMsg, "unknown expression"),
		strings.Contains(errMsg, "unknown column"),
		strings.Contains(errMsg, "missing columns"),
		strings.Contains(errMsg, "there is no column"),
		strings.Contains(errMsg, "no such column"),
		strings.Contains(errMsg, "cannot find column"):
		return "Unknown identifier in analytics query"
	case strings.Contains(errMsg, "syntax error"),
		strings.Contains(errMsg, "unexpected token"),
		strings.Contains(errMsg, "unrecognized token"),
		strings.Contains(errMsg, "cannot parse"):
		return "Invalid SQL syntax"
	default:
		return "Invalid analytics query"
	}
}

// errorResponse defines a structured error response with code and message
type errorResponse struct {
	code    codes.URN
	message string
}

// resourceLimitPatterns maps error message patterns to error responses
var resourceLimitPatterns = map[string]errorResponse{
	"max bytes": {
		code:    codes.User.UnprocessableEntity.QueryMemoryLimitExceeded.URN(),
		message: "Query result exceeds the maximum response size.",
	},
	"max rows": {
		code:    codes.User.UnprocessableEntity.QueryRowsLimitExceeded.URN(),
		message: "Query result exceeds the maximum row count.",
	},
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
	"limit for rows": {
		code:    codes.User.UnprocessableEntity.QueryRowsLimitExceeded.URN(),
		message: "Query attempted to read too many rows. Try adding more filters or reducing the time range.",
	},
	"quota": {
		code:    codes.User.TooManyRequests.QueryQuotaExceeded.URN(),
		message: "Query quota exceeded for the current time window. Please try again later.",
	},
}

// resourceLimitCodes maps ClickHouse exception codes to error responses
var resourceLimitCodes = map[int32]errorResponse{
	158: { // TOO_MANY_ROWS_OR_BYTES in older ClickHouse versions.
		code:    codes.User.UnprocessableEntity.QueryRowsLimitExceeded.URN(),
		message: "Query result exceeds the maximum row count.",
	},
	159: { // TIMEOUT_EXCEEDED
		code:    codes.User.UnprocessableEntity.QueryExecutionTimeout.URN(),
		message: "Query execution time limit exceeded. Try simplifying your query or reducing the time range.",
	},
	241: { // MEMORY_LIMIT_EXCEEDED
		code:    codes.User.UnprocessableEntity.QueryMemoryLimitExceeded.URN(),
		message: "Query memory limit exceeded. Try simplifying your query or reducing the result set size.",
	},
	198: { // TOO_MANY_ROWS
		code:    codes.User.UnprocessableEntity.QueryRowsLimitExceeded.URN(),
		message: "Query attempted to read too many rows. Try adding more filters or reducing the time range.",
	},
	202: { // TOO_MANY_SIMULTANEOUS_QUERIES / QUOTA_EXCEEDED
		code:    codes.User.TooManyRequests.QueryQuotaExceeded.URN(),
		message: "Query quota exceeded for the current time window. Please try again later.",
	},
	394: { // QUERY_WAS_CANCELLED
		code:    codes.User.UnprocessableEntity.QueryExecutionTimeout.URN(),
		message: "Query was cancelled due to resource limits.",
	},
}

// resourceLimitNames maps unambiguous ClickHouse exception names to error responses.
var resourceLimitNames = map[string]errorResponse{
	"TOO_MANY_ROWS": {
		code:    codes.User.UnprocessableEntity.QueryRowsLimitExceeded.URN(),
		message: "Query result exceeds the maximum row count.",
	},
	"QUERY_WAS_CANCELLED": {
		code:    codes.User.UnprocessableEntity.QueryExecutionTimeout.URN(),
		message: "Query was cancelled due to resource limits.",
	},
}

// WrapClickHouseError wraps a ClickHouse error with appropriate error codes and user-friendly messages.
// It detects resource limit violations and other user errors and tags them with specific error codes.
func WrapClickHouseError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return fault.Wrap(
			err,
			fault.Code(codes.User.UnprocessableEntity.QueryExecutionTimeout.URN()),
			fault.Public("Query execution was canceled or timed out."),
		)
	}

	errMsg := strings.ToLower(err.Error())

	// Check for resource limit violations via message patterns
	for pattern, response := range resourceLimitPatterns {
		if strings.Contains(errMsg, pattern) {
			return fault.Wrap(
				err,
				fault.Code(response.code),
				fault.Public(response.message),
			)
		}
	}

	// Check ClickHouse exception codes for resource errors
	var chErr *ch.Exception
	if errors.As(err, &chErr) {
		if response, ok := resourceLimitCodes[chErr.Code]; ok {
			return fault.Wrap(
				err,
				fault.Code(response.code),
				fault.Public(response.message),
			)
		}
		if response, ok := resourceLimitNames[chErr.Name]; ok {
			return fault.Wrap(
				err,
				fault.Code(response.code),
				fault.Public(response.message),
			)
		}
		// ClickHouse uses one exception for row and byte result overflow, so standard
		// messages are classified above before these version-specific fallbacks.
		if chErr.Code == 396 {
			return fault.Wrap(
				err,
				fault.Code(codes.User.UnprocessableEntity.QueryMemoryLimitExceeded.URN()),
				fault.Public("Query result exceeds the maximum response size."),
			)
		}
		if chErr.Name == "TOO_MANY_ROWS_OR_BYTES" {
			return fault.Wrap(
				err,
				fault.Code(codes.User.UnprocessableEntity.QueryRowsLimitExceeded.URN()),
				fault.Public("Query result exceeds the maximum row count."),
			)
		}
	}

	// All other ClickHouse errors are treated as user query errors (400)
	// This ensures we never return 500 for query execution issues
	return fault.Wrap(
		err,
		fault.Code(codes.User.BadRequest.InvalidAnalyticsQuery.URN()),
		fault.Public(ExtractUserFriendlyError(err)),
	)
}
