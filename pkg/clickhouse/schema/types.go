package schema

import "encoding/json"

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
	// ExpiresAt is the unix-milli TTL stamp for this row. Writers compute
	// it as `Time + workspace.LogsRetentionDays * 86400000` via
	// internal/services/quotaretention. If left zero, the writer falls
	// back to the table's CH-side DEFAULT (the table's historical static
	// retention window — 90d for verifications) so a missing stamp does
	// not delete the row immediately.
	ExpiresAt int64 `ch:"expires_at" json:"expires_at"`
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
	// ExpiresAt is the unix-milli TTL stamp for this row. See
	// KeyVerification.ExpiresAt for semantics; default fallback for this
	// table is the historical 30d window.
	ExpiresAt int64 `ch:"expires_at" json:"expires_at"`
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
	// ExpiresAt is the unix-milli TTL stamp for this row. See
	// KeyVerification.ExpiresAt for semantics; default fallback for this
	// table is the historical 30d window.
	ExpiresAt int64 `ch:"expires_at" json:"expires_at"`
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

// BuildStepV1 represents the v1 build step raw table structure.
// This tracks individual build steps within a deployment process
// including timing, caching, and error information.
type BuildStepV1 struct {
	StartedAt    int64  `ch:"started_at" json:"started_at"`
	CompletedAt  int64  `ch:"completed_at" json:"completed_at"`
	WorkspaceID  string `ch:"workspace_id" json:"workspace_id"`
	ProjectID    string `ch:"project_id" json:"project_id"`
	DeploymentID string `ch:"deployment_id" json:"deployment_id"`
	StepID       string `ch:"step_id" json:"step_id"`
	Name         string `ch:"name" json:"name"`
	Cached       bool   `ch:"cached" json:"cached"`
	Error        string `ch:"error" json:"error"`
	HasLogs      bool   `ch:"has_logs" json:"has_logs"`
}

// BuildStepLogV1 represents the v1 build step log raw table structure.
// This stores log messages generated during build step execution
// for debugging and monitoring purposes.
type BuildStepLogV1 struct {
	Time         int64  `ch:"time" json:"time"`
	WorkspaceID  string `ch:"workspace_id" json:"workspace_id"`
	ProjectID    string `ch:"project_id" json:"project_id"`
	DeploymentID string `ch:"deployment_id" json:"deployment_id"`
	StepID       string `ch:"step_id" json:"step_id"`
	Message      string `ch:"message" json:"message"`
}

// SentinelRequest represents the v1 sentinel request raw table structure.
// This tracks requests routed through sentinel proxy to deployment instances
// with deployment routing, performance breakdown, and error categorization.
type SentinelRequest struct {
	RequestID       string              `ch:"request_id" json:"request_id"`
	Time            int64               `ch:"time" json:"time"`
	WorkspaceID     string              `ch:"workspace_id" json:"workspace_id"`
	EnvironmentID   string              `ch:"environment_id" json:"environment_id"`
	ProjectID       string              `ch:"project_id" json:"project_id"`
	SentinelID      string              `ch:"sentinel_id" json:"sentinel_id"`
	DeploymentID    string              `ch:"deployment_id" json:"deployment_id"`
	InstanceID      string              `ch:"instance_id" json:"instance_id"`
	InstanceAddress string              `ch:"instance_address" json:"instance_address"`
	Region          string              `ch:"region" json:"region"`
	Platform        string              `ch:"platform" json:"platform"`
	Method          string              `ch:"method" json:"method"`
	Host            string              `ch:"host" json:"host"`
	Path            string              `ch:"path" json:"path"`
	QueryString     string              `ch:"query_string" json:"query_string"`
	QueryParams     map[string][]string `ch:"query_params" json:"query_params"`
	RequestHeaders  []string            `ch:"request_headers" json:"request_headers"`
	RequestBody     string              `ch:"request_body" json:"request_body"`
	ResponseStatus  int32               `ch:"response_status" json:"response_status"`
	ResponseHeaders []string            `ch:"response_headers" json:"response_headers"`
	ResponseBody    string              `ch:"response_body" json:"response_body"`
	UserAgent       string              `ch:"user_agent" json:"user_agent"`
	IPAddress       string              `ch:"ip_address" json:"ip_address"`
	TotalLatency    int64               `ch:"total_latency" json:"total_latency"`
	InstanceLatency int64               `ch:"instance_latency" json:"instance_latency"`
	SentinelLatency int64               `ch:"sentinel_latency" json:"sentinel_latency"`
	// ExpiresAt is the unix-milli TTL stamp. See KeyVerification.ExpiresAt
	// for semantics; default fallback for this table is the historical 30d
	// window.
	ExpiresAt int64 `ch:"expires_at" json:"expires_at"`
}

// AuditLogV1 represents one logical audit event in audit_logs_raw_v1.
// Targets are stored as parallel Nested arrays (TargetTypes[i] pairs with
// TargetIDs[i] / TargetNames[i] / TargetMetas[i]). All four slices MUST be
// the same length.
//
// Source distinguishes platform-emitted events ("platform", default) from
// customer-emitted events ("customer", once that surface ships).
//
// Time fields are unix-milli (Int64), matching the rest of Unkey's CH
// tables. Meta fields are json.RawMessage so the writer can pass already-
// encoded JSON bytes through without re-marshaling, and so the JSON column
// type in CH stores them natively (not as escaped strings).
type AuditLogV1 struct {
	EventID     string `ch:"event_id" json:"event_id"`
	Time        int64  `ch:"time" json:"time"`
	InsertedAt  int64  `ch:"inserted_at" json:"inserted_at"`
	WorkspaceID string `ch:"workspace_id" json:"workspace_id"`
	Bucket      string `ch:"bucket" json:"bucket"`
	Source      string `ch:"source" json:"source"`

	Event       string `ch:"event" json:"event"`
	Description string `ch:"description" json:"description"`

	ActorType string          `ch:"actor_type" json:"actor_type"`
	ActorID   string          `ch:"actor_id" json:"actor_id"`
	ActorName string          `ch:"actor_name" json:"actor_name"`
	ActorMeta json.RawMessage `ch:"actor_meta" json:"actor_meta"`

	RemoteIP  string          `ch:"remote_ip" json:"remote_ip"`
	UserAgent string          `ch:"user_agent" json:"user_agent"`
	Meta      json.RawMessage `ch:"meta" json:"meta"`

	TargetTypes []string          `ch:"targets.type" json:"targets.type"`
	TargetIDs   []string          `ch:"targets.id" json:"targets.id"`
	TargetNames []string          `ch:"targets.name" json:"targets.name"`
	TargetMetas []json.RawMessage `ch:"targets.meta" json:"targets.meta"`

	// ExpiresAt drives the per-row TTL. Set at insert time so workspace
	// retention quotas apply without ALTER TABLE. Stored as unix-milli.
	ExpiresAt int64 `ch:"expires_at" json:"expires_at"`
}
