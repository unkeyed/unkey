package schema

// KeyVerification represents the v2 key verification raw table structure.
// This matches the key_verifications_raw_v2 table schema with additional
// fields like spent_credits and latency compared to v1.
type KeyVerification struct {
	RequestID    string   `ch:"request_id" json:"request_id"`
	Time         int64    `ch:"time" json:"time"`
	WorkspaceID  string   `ch:"workspace_id" json:"workspace_id"`
	KeySpaceID   string   `ch:"key_space_id" json:"key_space_id"`
	IdentityID   string   `ch:"identity_id" json:"identity_id"`
	ExternalID   string   `ch:"external_id" json:"external_id"`
	KeyID        string   `ch:"key_id" json:"key_id"`
	Region       string   `ch:"region" json:"region"`
	Outcome      string   `ch:"outcome" json:"outcome"`
	Tags         []string `ch:"tags" json:"tags"`
	SpentCredits int64    `ch:"spent_credits" json:"spent_credits"`
	Latency      float64  `ch:"latency" json:"latency"`
}

// Ratelimit represents the v2 ratelimit raw table structure.
// This matches the ratelimits_raw_v2 table schema with additional
// latency field compared to v1.
type Ratelimit struct {
	RequestID   string  `ch:"request_id" json:"request_id"`
	Time        int64   `ch:"time" json:"time"`
	WorkspaceID string  `ch:"workspace_id" json:"workspace_id"`
	NamespaceID string  `ch:"namespace_id" json:"namespace_id"`
	Identifier  string  `ch:"identifier" json:"identifier"`
	Passed      bool    `ch:"passed" json:"passed"`
	Latency     float64 `ch:"latency" json:"latency"`
	OverrideID  string  `ch:"override_id" json:"override_id"`
	Limit       uint64  `ch:"limit" json:"limit"`
	Remaining   uint64  `ch:"remaining" json:"remaining"`
	ResetAt     int64   `ch:"reset_at" json:"reset_at"`
}

// ApiRequest represents the v2 API request raw table structure.
// This matches the api_requests_raw_v2 table schema with query parameters
// and region field compared to v1.
type ApiRequest struct {
	RequestID       string              `ch:"request_id" json:"request_id"`
	Time            int64               `ch:"time" json:"time"`
	WorkspaceID     string              `ch:"workspace_id" json:"workspace_id"`
	Host            string              `ch:"host" json:"host"`
	Method          string              `ch:"method" json:"method"`
	Path            string              `ch:"path" json:"path"`
	QueryString     string              `ch:"query_string" json:"query_string"`
	QueryParams     map[string][]string `ch:"query_params" json:"query_params"`
	RequestHeaders  []string            `ch:"request_headers" json:"request_headers"`
	RequestBody     string              `ch:"request_body" json:"request_body"`
	ResponseStatus  int32               `ch:"response_status" json:"response_status"`
	ResponseHeaders []string            `ch:"response_headers" json:"response_headers"`
	ResponseBody    string              `ch:"response_body" json:"response_body"`
	Error           string              `ch:"error" json:"error"`
	ServiceLatency  int64               `ch:"service_latency" json:"service_latency"`
	UserAgent       string              `ch:"user_agent" json:"user_agent"`
	IpAddress       string              `ch:"ip_address" json:"ip_address"`
	Region          string              `ch:"region" json:"region"`
}

// KeyVerificationAggregated represents aggregated key verification data
// from the per-minute/hour/day/month materialized views.
type KeyVerificationAggregated struct {
	Time        int64    `ch:"time" json:"time"`
	WorkspaceID string   `ch:"workspace_id" json:"workspace_id"`
	KeySpaceID  string   `ch:"key_space_id" json:"key_space_id"`
	IdentityID  string   `ch:"identity_id" json:"identity_id"`
	KeyID       string   `ch:"key_id" json:"key_id"`
	Outcome     string   `ch:"outcome" json:"outcome"`
	Tags        []string `ch:"tags" json:"tags"`
	Count       int64    `ch:"count" json:"count"`
}

// RatelimitAggregated represents aggregated ratelimit data
// from the per-minute/hour/day/month materialized views.
type RatelimitAggregated struct {
	Time        int64  `ch:"time" json:"time"`
	WorkspaceID string `ch:"workspace_id" json:"workspace_id"`
	NamespaceID string `ch:"namespace_id" json:"namespace_id"`
	Identifier  string `ch:"identifier" json:"identifier"`
	Passed      int64  `ch:"passed" json:"passed"`
	Total       int64  `ch:"total" json:"total"`
}

// ApiRequestAggregated represents aggregated API request data
// from the per-minute/hour/day/month materialized views.
type ApiRequestAggregated struct {
	Time           int64  `ch:"time" json:"time"`
	WorkspaceID    string `ch:"workspace_id" json:"workspace_id"`
	Path           string `ch:"path" json:"path"`
	ResponseStatus int32  `ch:"response_status" json:"response_status"`
	Host           string `ch:"host" json:"host"`
	Method         string `ch:"method" json:"method"`
	Count          int64  `ch:"count" json:"count"`
}
