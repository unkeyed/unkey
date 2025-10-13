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
	"unknown identifier":               true,
	"unknown expression":               true,
	"unknown function":                 true,
	"unknown column":                   true,
	"unknown table":                    true,
	"missing columns":                  true,
	"there is no column":               true,
	"type mismatch":                    true,
	"cannot convert":                   true,
	"syntax error":                     true,
	"expected":                         true,
	"illegal type":                     true,
	"ambiguous column":                 true,
	"not an aggregate function":        true,
	"division by zero":                 true,
	"aggregate function":               true,
	"window function":                  true,
	"unknown_identifier":               true, // ClickHouse error code name
	"db::exception":                    true, // Treat all DB exceptions as user errors
	"maybe you meant":                  true, // ClickHouse suggestions
	"no such column":                   true,
	"doesn't exist":                    true,
	"does not exist":                   true,
	"failed at position":               true,
	"unexpected token":                 true,
	"invalid expression":               true,
	"invalid number of arguments":      true,
	"wrong number of arguments":        true,
	"cannot parse":                     true,
	"unrecognized token":               true,
	"no matching signature":            true,
	"incompatible types":               true,
	"illegal aggregation":              true,
	"cannot find column":               true,
	"not allowed in this context":      true,
	"not supported":                    true,
	"invalid combination":              true,
	"invalid or illegal":               true,
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
// It preserves the key information like unknown identifiers, suggestions, and error context.
func ExtractUserFriendlyError(err error) string {
	if err == nil {
		return "Query failed"
	}

	errMsg := err.Error()

	// ClickHouse errors from HTTP interface often contain the actual DB::Exception message
	// Format: "Code: 47. DB::Exception: <actual error message>. (ERROR_NAME)"
	if idx := strings.Index(errMsg, "DB::Exception: "); idx != -1 {
		errMsg = errMsg[idx+15:] // Skip "DB::Exception: "

		// Find the end marker (usually the error code in parentheses at the end)
		if endIdx := strings.LastIndex(errMsg, " (version "); endIdx != -1 {
			errMsg = errMsg[:endIdx]
		}

		// Remove the final error code if present like ". (UNKNOWN_IDENTIFIER)"
		if endIdx := strings.LastIndex(errMsg, ". ("); endIdx != -1 {
			errMsg = errMsg[:endIdx]
		}

		return strings.TrimSpace(errMsg)
	}

	// Try to extract from exception object
	var chErr *ch.Exception
	if errors.As(err, &chErr) {
		return chErr.Message
	}

	// Clean up common prefixes for other formats
	errMsg = strings.TrimPrefix(errMsg, "clickhouse: ")
	errMsg = strings.TrimPrefix(errMsg, "sendQuery: ")
	errMsg = strings.TrimPrefix(errMsg, "[HTTP 404] response body: ")
	errMsg = strings.Trim(errMsg, "\"")

	// If the message is too long, try to extract the first sentence
	if len(errMsg) > 500 {
		if idx := strings.Index(errMsg, ". "); idx != -1 && idx < 500 {
			errMsg = errMsg[:idx+1]
		} else {
			errMsg = errMsg[:500] + "..."
		}
	}

	return strings.TrimSpace(errMsg)
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

	// All other ClickHouse errors are treated as user query errors (400)
	// This ensures we never return 500 for query execution issues
	return fault.Wrap(err,
		fault.Code(codes.User.BadRequest.InvalidAnalyticsQuery.URN()),
		fault.Public(ExtractUserFriendlyError(err)),
	)
}
