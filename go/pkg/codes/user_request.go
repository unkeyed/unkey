package codes

// userBadRequest defines errors related to invalid user input and bad requests.
type userBadRequest struct {
	// PermissionsQuerySyntaxError indicates a syntax or lexical error in verifyKey permissions query parsing.
	PermissionsQuerySyntaxError Code
	// RequestBodyTooLarge indicates the request body exceeds the maximum allowed size.
	RequestBodyTooLarge Code
	// RequestTimeout indicates the request took too long to process.
	RequestTimeout Code
	// ClientClosedRequest indicates the client closed the connection before the request completed.
	ClientClosedRequest Code
	// InvalidAnalyticsQuery indicates the analytics SQL query is invalid or has syntax errors.
	InvalidAnalyticsQuery Code
	// InvalidTable indicates the table referenced in the query is not allowed or does not exist.
	InvalidTable Code
	// InvalidFunction indicates a disallowed function was used in the query.
	InvalidFunction Code
	// QueryNotSupported indicates the query type or operation is not supported (e.g., INSERT, UPDATE, DELETE).
	QueryNotSupported Code
	// QueryExecutionTimeout indicates the query exceeded the maximum execution time limit.
	QueryExecutionTimeout Code
	// QueryMemoryLimitExceeded indicates the query exceeded the maximum memory usage limit.
	QueryMemoryLimitExceeded Code
	// QueryRowsLimitExceeded indicates the query exceeded the maximum rows to read limit.
	QueryRowsLimitExceeded Code
	// QueryResultRowsLimitExceeded indicates the query exceeded the maximum result rows limit.
	QueryResultRowsLimitExceeded Code
	// QueryQuotaExceeded indicates the workspace has exceeded their query quota for the current window.
	QueryQuotaExceeded Code
}

// UserErrors defines all user-related errors in the Unkey system.
// These errors are caused by invalid user inputs or client behavior.
type UserErrors struct {
	// BadRequest contains errors related to invalid user input.
	BadRequest userBadRequest
}

// User contains all predefined user error codes.
// These errors can be referenced directly (e.g., codes.User.BadRequest.QueryEmpty)
// for consistent error handling throughout the application.
var User = UserErrors{
	BadRequest: userBadRequest{
		PermissionsQuerySyntaxError:  Code{SystemUser, CategoryUserBadRequest, "permissions_query_syntax_error"},
		RequestBodyTooLarge:          Code{SystemUser, CategoryUserBadRequest, "request_body_too_large"},
		RequestTimeout:               Code{SystemUser, CategoryUserBadRequest, "request_timeout"},
		ClientClosedRequest:          Code{SystemUser, CategoryUserBadRequest, "client_closed_request"},
		InvalidAnalyticsQuery:        Code{SystemUser, CategoryUserBadRequest, "invalid_analytics_query"},
		InvalidTable:                 Code{SystemUser, CategoryUserBadRequest, "invalid_table"},
		InvalidFunction:              Code{SystemUser, CategoryUserBadRequest, "invalid_function"},
		QueryNotSupported:            Code{SystemUser, CategoryUserBadRequest, "query_not_supported"},
		QueryExecutionTimeout:        Code{SystemUser, CategoryUserBadRequest, "query_execution_timeout"},
		QueryMemoryLimitExceeded:     Code{SystemUser, CategoryUserBadRequest, "query_memory_limit_exceeded"},
		QueryRowsLimitExceeded:       Code{SystemUser, CategoryUserBadRequest, "query_rows_limit_exceeded"},
		QueryResultRowsLimitExceeded: Code{SystemUser, CategoryUserBadRequest, "query_result_rows_limit_exceeded"},
		QueryQuotaExceeded:           Code{SystemUser, CategoryUserBadRequest, "query_quota_exceeded"},
	},
}
