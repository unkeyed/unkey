package codes

// userBadRequest defines errors related to invalid user input and bad requests.
type userBadRequest struct {
	// MissingRequiredHeader indicates a required HTTP header is missing from the request.
	MissingRequiredHeader Code
	// PermissionsQuerySyntaxError indicates a syntax or lexical error in verifyKey permissions query parsing.
	PermissionsQuerySyntaxError Code
	// RequestBodyTooLarge indicates the request body exceeds the maximum allowed size.
	RequestBodyTooLarge Code
	// RequestBodyUnreadable indicates the request body could not be read due to malformed request or connection issues.
	RequestBodyUnreadable Code
	// RequestTimeout indicates the request took too long to process.
	RequestTimeout Code
	// ClientClosedRequest indicates the client closed the connection before the request completed.
	ClientClosedRequest Code
	// InvalidAnalyticsQuery indicates the analytics SQL query is invalid or has syntax errors.
	InvalidAnalyticsQuery Code
	// InvalidAnalyticsTable indicates the table referenced in the analytics query is not allowed or does not exist.
	InvalidAnalyticsTable Code
	// InvalidAnalyticsFunction indicates a disallowed function was used in the analytics query.
	InvalidAnalyticsFunction Code
	// InvalidAnalyticsQueryType indicates the query type or operation is not supported (e.g., INSERT, UPDATE, DELETE).
	InvalidAnalyticsQueryType Code
	// QueryRangeExceedsRetention indicates the query attempts to access data older than the workspace's retention period.
	QueryRangeExceedsRetention Code
}

// userUnprocessableEntity defines errors for requests that are syntactically correct but cannot be processed.
type userUnprocessableEntity struct {
	// QueryExecutionTimeout indicates the query exceeded the maximum execution time limit.
	QueryExecutionTimeout Code
	// QueryMemoryLimitExceeded indicates the query exceeded the maximum memory usage limit.
	QueryMemoryLimitExceeded Code
	// QueryRowsLimitExceeded indicates the query exceeded the maximum rows to read limit.
	QueryRowsLimitExceeded Code
}

// userTooManyRequests defines errors related to rate limiting and quota exceeded.
type userTooManyRequests struct {
	// QueryQuotaExceeded indicates the workspace has exceeded their query quota for the current window.
	QueryQuotaExceeded Code
	// WorkspaceRateLimited indicates the workspace has exceeded its API rate limit for the current window.
	WorkspaceRateLimited Code
}

// UserErrors defines all user-related errors in the Unkey system.
// These errors are caused by invalid user inputs or client behavior.
type UserErrors struct {
	// BadRequest contains errors related to invalid user input.
	BadRequest userBadRequest
	// UnprocessableEntity contains errors for syntactically valid requests that cannot be processed.
	UnprocessableEntity userUnprocessableEntity
	// TooManyRequests contains errors related to rate limiting.
	TooManyRequests userTooManyRequests
}

// User contains all predefined user error codes.
// These errors can be referenced directly (e.g., codes.User.BadRequest.QueryEmpty)
// for consistent error handling throughout the application.
var User = UserErrors{
	BadRequest: userBadRequest{
		MissingRequiredHeader:       Code{SystemUser, CategoryUserBadRequest, "missing_required_header", KindUnknown},
		PermissionsQuerySyntaxError: Code{SystemUser, CategoryUserBadRequest, "permissions_query_syntax_error", KindUnknown},
		RequestBodyTooLarge:         Code{SystemUser, CategoryUserBadRequest, "request_body_too_large", KindRequestEntityTooLarge},
		RequestBodyUnreadable:       Code{SystemUser, CategoryUserBadRequest, "request_body_unreadable", KindUnknown},
		RequestTimeout:              Code{SystemUser, CategoryUserBadRequest, "request_timeout", KindRequestTimeout},
		ClientClosedRequest:         Code{SystemUser, CategoryUserBadRequest, "client_closed_request", KindClientClosedRequest},
		InvalidAnalyticsQuery:       Code{SystemUser, CategoryUserBadRequest, "invalid_analytics_query", KindUnknown},
		InvalidAnalyticsTable:       Code{SystemUser, CategoryUserBadRequest, "invalid_analytics_table", KindUnknown},
		InvalidAnalyticsFunction:    Code{SystemUser, CategoryUserBadRequest, "invalid_analytics_function", KindUnknown},
		InvalidAnalyticsQueryType:   Code{SystemUser, CategoryUserBadRequest, "invalid_analytics_query_type", KindUnknown},
		QueryRangeExceedsRetention:  Code{SystemUser, CategoryUserBadRequest, "query_range_exceeds_retention", KindUnknown},
	},
	UnprocessableEntity: userUnprocessableEntity{
		QueryExecutionTimeout:    Code{SystemUser, CategoryUserUnprocessableEntity, "query_execution_timeout", KindUnknown},
		QueryMemoryLimitExceeded: Code{SystemUser, CategoryUserUnprocessableEntity, "query_memory_limit_exceeded", KindUnknown},
		QueryRowsLimitExceeded:   Code{SystemUser, CategoryUserUnprocessableEntity, "query_rows_limit_exceeded", KindUnknown},
	},
	TooManyRequests: userTooManyRequests{
		QueryQuotaExceeded:   Code{SystemUser, CategoryUserTooManyRequests, "query_quota_exceeded", KindUnknown},
		WorkspaceRateLimited: Code{SystemUser, CategoryUserTooManyRequests, "workspace_rate_limited", KindUnknown},
	},
}
