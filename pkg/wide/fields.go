package wide

// Standard field names for wide events.
// All field names use snake_case for consistency.
//
// Usage:
//
//	ev.Set(wide.FieldRequestID, requestID)
//	ev.Set(wide.FieldWorkspaceID, workspaceID)
const (
	// --- Request fields ---

	// FieldRequestID is the unique identifier for the request.
	FieldRequestID = "request_id"

	// FieldMethod is the HTTP method (GET, POST, etc.).
	FieldMethod = "method"

	// FieldPath is the URL path (e.g., "/v2/keys.verifyKey").
	FieldPath = "path"

	// FieldHost is the HTTP Host header value.
	FieldHost = "host"

	// FieldUserAgent is the User-Agent header value.
	FieldUserAgent = "user_agent"

	// FieldIPAddress is the client's IP address.
	FieldIPAddress = "ip_address"

	// FieldContentLength is the request Content-Length header value.
	FieldContentLength = "content_length"

	// --- Response fields ---

	// FieldStatusCode is the HTTP response status code.
	FieldStatusCode = "status_code"

	// FieldDurationMs is the request duration in milliseconds.
	FieldDurationMs = "duration_ms"

	// FieldResponseSize is the response body size in bytes.
	FieldResponseSize = "response_size"

	// --- Business context fields ---

	// FieldWorkspaceID is the workspace making the request.
	FieldWorkspaceID = "workspace_id"

	// FieldAPIID is the API identifier.
	FieldAPIID = "api_id"

	// FieldKeyID is the key identifier (for key operations).
	FieldKeyID = "key_id"

	// FieldIdentityID is the identity identifier.
	FieldIdentityID = "identity_id"

	// FieldEnvironmentID is the environment identifier (for sentinel).
	FieldEnvironmentID = "environment_id"

	// FieldDeploymentID is the deployment identifier.
	FieldDeploymentID = "deployment_id"

	// FieldInstanceID is the instance identifier.
	FieldInstanceID = "instance_id"

	// --- Error fields ---

	// FieldError is the error message (user-facing or sanitized).
	FieldError = "error"

	// FieldErrorCode is the structured error code (URN format).
	FieldErrorCode = "error_code"

	// FieldErrorInternal is the internal error message (not user-facing).
	FieldErrorInternal = "error_internal"

	// FieldErrorType categorizes the error source (none, user, customer, platform).
	FieldErrorType = "error_type"

	// FieldErrorLocations is the chain of file:line locations where the error was wrapped.
	FieldErrorLocations = "error_locations"

	// --- API-specific fields ---

	// FieldCacheHit indicates whether the response was served from cache.
	FieldCacheHit = "cache_hit"

	// FieldCacheLatencyMs is the cache operation latency in milliseconds.
	FieldCacheLatencyMs = "cache_latency_ms"

	// FieldRateLimitExceeded indicates if rate limit was exceeded.
	FieldRateLimitExceeded = "rate_limit_exceeded"

	// FieldRateLimitRemaining is the remaining rate limit quota.
	FieldRateLimitRemaining = "rate_limit_remaining"

	// FieldDBQueries is the number of database queries executed.
	FieldDBQueries = "db_queries"

	// FieldDBLatencyMs is the total database latency in milliseconds.
	FieldDBLatencyMs = "db_latency_ms"

	// FieldDBErrorType is the classification of a database error (not_found, duplicate_key, deadlock, etc.).
	FieldDBErrorType = "db_error_type"

	// FieldDBRetryAttempt is the current retry attempt number (1, 2, 3).
	FieldDBRetryAttempt = "db_retry_attempt"

	// FieldDBRetryReason explains why a retry was needed (deadlock, connection, etc.).
	FieldDBRetryReason = "db_retry_reason"

	// --- Infrastructure fields ---

	// FieldRegion is the geographic region where the request was processed.
	FieldRegion = "region"

	// FieldPlatform is the cloud platform (aws, gcp, etc.).
	FieldPlatform = "platform"

	// FieldServiceName is the name of the service handling the request.
	FieldServiceName = "service_name"

	// FieldServiceVersion is the version/image of the service.
	FieldServiceVersion = "service_version"

	// --- Sampling fields ---

	// FieldSampleReason explains why this event was logged.
	FieldSampleReason = "sample_reason"

	// --- Verification-specific fields ---

	// FieldKeyValid indicates if the key was valid.
	FieldKeyValid = "key_valid"

	// FieldKeyDisabled indicates if the key was disabled.
	FieldKeyDisabled = "key_disabled"

	// FieldKeyExpired indicates if the key was expired.
	FieldKeyExpired = "key_expired"

	// FieldKeyRateLimited indicates if the key was rate limited.
	FieldKeyRateLimited = "key_rate_limited"

	// FieldKeyUsageLimited indicates if the key exceeded usage limits.
	FieldKeyUsageLimited = "key_usage_limited"

	// --- Proxy fields (frontline/sentinel) ---

	// FieldUpstreamURL is the full upstream URL for proxy requests (e.g., "http://sentinel:8040").
	FieldUpstreamURL = "upstream_url"

	// FieldUpstreamHost is the upstream host for proxy requests.
	FieldUpstreamHost = "upstream_host"

	// FieldUpstreamLatencyMs is the upstream request latency in milliseconds.
	FieldUpstreamLatencyMs = "upstream_latency_ms"

	// FieldUpstreamStatusCode is the status code from the upstream service.
	FieldUpstreamStatusCode = "upstream_status_code"

	// FieldProxyHops is the number of proxy hops for the request.
	FieldProxyHops = "proxy_hops"
)

// Error types for FieldErrorType.
const (
	// ErrorTypeNone indicates no error occurred.
	ErrorTypeNone = "none"

	// ErrorTypeUser indicates a user/client error (4xx except rate limits).
	ErrorTypeUser = "user"

	// ErrorTypeCustomer indicates an error with the customer's upstream service.
	ErrorTypeCustomer = "customer"

	// ErrorTypePlatform indicates an Unkey platform error (5xx).
	ErrorTypePlatform = "platform"

	// ErrorTypeUnknown indicates an uncategorized error.
	ErrorTypeUnknown = "unknown"
)
