package schema

//go:generate go tool github.com/mailru/easyjson/easyjson -all requests.go

// Slice types for efficient JSON marshaling/unmarshaling
//
//easyjson:json
type ApiRequestV1Slice []ApiRequestV1

//easyjson:json
type KeyVerificationRequestV1Slice []KeyVerificationRequestV1

//easyjson:json
type RatelimitRequestV1Slice []RatelimitRequestV1

// ApiRequestV1 represents an HTTP API request with its associated metadata,
// request details, and response information. This structure is used to log
// and analyze API usage patterns, performance metrics, and error rates.
//
// Fields are mapped to ClickHouse columns using the `ch` struct tags.
type ApiRequestV1 struct {
	// WorkspaceID identifies the workspace that the request belongs to
	WorkspaceID string `ch:"workspace_id" json:"workspace_id"`

	// RequestID is a unique identifier for this request
	RequestID string `ch:"request_id" json:"request_id"`

	// Time is the Unix timestamp in milliseconds when the request was received
	Time int64 `ch:"time" json:"time"`

	// Host is the hostname from the request (e.g., "api.unkey.dev")
	Host string `ch:"host" json:"host"`

	// Method is the HTTP method (GET, POST, etc.)
	Method string `ch:"method" json:"method"`

	// Path is the request URI path
	Path string `ch:"path" json:"path"`

	// RequestHeaders contains the HTTP request headers
	RequestHeaders []string `ch:"request_headers" json:"request_headers"`

	// RequestBody contains the HTTP request body (sanitized of sensitive data)
	RequestBody string `ch:"request_body" json:"request_body"`

	// ResponseStatus is the HTTP status code returned
	ResponseStatus int `ch:"response_status" json:"response_status"`

	// ResponseHeaders contains the HTTP response headers
	ResponseHeaders []string `ch:"response_headers" json:"response_headers"`

	// ResponseBody contains the HTTP response body (sanitized of sensitive data)
	ResponseBody string `ch:"response_body" json:"response_body"`

	// Error contains any error message if the request failed
	Error string `ch:"error" json:"error"`

	// ServiceLatency is the time in milliseconds it took to process the request
	ServiceLatency int64 `ch:"service_latency" json:"service_latency"`

	UserAgent string `ch:"user_agent" json:"user_agent"`

	IpAddress string `ch:"ip_address" json:"ip_address"`

	Country string `ch:"country" json:"country"`

	City string `ch:"city" json:"city"`

	Colo      string `ch:"colo" json:"colo"`
	Continent string `ch:"continent" json:"continent"`
}

// KeyVerificationRequestV1 represents a key verification operation, tracking
// when and how API keys are validated. This structure is used to analyze
// key usage patterns, identify unauthorized access attempts, and track
// verification performance.
//
// Fields are mapped to ClickHouse columns using the `ch` struct tags.
type KeyVerificationRequestV1 struct {
	// RequestID is a unique identifier for this verification request
	RequestID string `ch:"request_id" json:"request_id"`

	// Time is the Unix timestamp in milliseconds when the verification occurred
	Time int64 `ch:"time" json:"time"`

	// WorkspaceID identifies the workspace that the key belongs to
	WorkspaceID string `ch:"workspace_id" json:"workspace_id"`

	// KeySpaceID identifies the key space that the key belongs to
	KeySpaceID string `ch:"key_space_id" json:"key_space_id"`

	// KeyID is the unique identifier of the key being verified
	KeyID string `ch:"key_id" json:"key_id"`

	// Region indicates the geographic region where the verification occurred
	Region string `ch:"region" json:"region"`

	// Outcome is the result of the verification (e.g., "success", "invalid", "expired")
	Outcome string `ch:"outcome" json:"outcome"`

	// IdentityID links the key to a specific identity, if applicable
	IdentityID string `ch:"identity_id" json:"identity_id"`

	Tags []string `ch:"tags" json:"tags"`
}

type RatelimitRequestV1 struct {
	RequestID   string `ch:"request_id" json:"request_id"`
	Time        int64  `ch:"time" json:"time"`
	WorkspaceID string `ch:"workspace_id" json:"workspace_id"`
	NamespaceID string `ch:"namespace_id" json:"namespace_id"`
	Identifier  string `ch:"identifier" json:"identifier"`
	Passed      bool   `ch:"passed" json:"passed"`
}
