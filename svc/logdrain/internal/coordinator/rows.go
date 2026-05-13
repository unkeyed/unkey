package coordinator

import "encoding/json"

// runtimeRow is the projected ClickHouse row used for cursor pagination
// of runtime_logs_raw_v2. It is a trimmed copy of pkg/clickhouse/schema's
// RuntimeLog — the dashboard write path stores body and headers, but the
// coordinator's SELECT only reads the columns the cursor query projects.
// Keeping a local subset stops the driver from complaining about the
// schema's `expires_at` field that the read path never wants.
//
// log_id is Vector's stable per-row id ("log_<16 hex chars>",
// UUID-v7-shaped). The (inserted_at, log_id) tuple is a strict total
// order — log_id is globally unique by construction, so unlike the v1
// cityHash64 fingerprint it can't theoretically collide.
type runtimeRow struct {
	InsertedAt    int64  `ch:"inserted_at"`
	LogID         string `ch:"log_id"`
	Time          int64  `ch:"time"`
	Severity      string `ch:"severity"`
	Message       string `ch:"message"`
	WorkspaceID   string `ch:"workspace_id"`
	ProjectID     string `ch:"project_id"`
	EnvironmentID string `ch:"environment_id"`
	AppID         string `ch:"app_id"`
	DeploymentID  string `ch:"deployment_id"`
	K8sPodName    string `ch:"k8s_pod_name"`
	Region        string `ch:"region"`
	Platform      string `ch:"platform"`
	Attributes    string `ch:"attributes"`
}

// requestRow is the projected ClickHouse row used for cursor pagination of
// sentinel_requests_raw_v1. `inserted_at` (CH-side ingest timestamp, added
// in the 20260513 migration) drives the watermark. Producer-set `time` is
// corrupted by sentinel pod clock skew and any retransmits, so a cursor
// on `time` would silently drop rows that land with a `time` value behind
// the cursor. `request_id` is a stored String column already unique per
// row; the cursor compares it directly without an inline cityHash64.
type requestRow struct {
	InsertedAt      int64  `ch:"inserted_at"`
	Time            int64  `ch:"time"`
	RequestID       string `ch:"request_id"`
	WorkspaceID     string `ch:"workspace_id"`
	ProjectID       string `ch:"project_id"`
	EnvironmentID   string `ch:"environment_id"`
	DeploymentID    string `ch:"deployment_id"`
	InstanceID      string `ch:"instance_id"`
	InstanceAddress string `ch:"instance_address"`
	Region          string `ch:"region"`
	Platform        string `ch:"platform"`
	Method          string `ch:"method"`
	Host            string `ch:"host"`
	Path            string `ch:"path"`
	ResponseStatus  int32  `ch:"response_status"`
	UserAgent       string `ch:"user_agent"`
	IPAddress       string `ch:"ip_address"`
	TotalLatency    int64  `ch:"total_latency"`
}

// parseAttrs decodes the JSON attributes blob shipped on every runtime
// row. Vector's attributes column is `JSON` in CH and arrives here
// already serialized as a string; we decode lazily so plain-text logs
// (empty / "null") cost nothing.
func parseAttrs(s string) map[string]any {
	if s == "" || s == "null" {
		return nil
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		return nil
	}
	return out
}

// classifyCHError reduces a raw CH driver error to a low-cardinality
// label for ClickHouseQueryErrors. Stub today; the production breakdown
// is timeout / authn / dns / unknown — wire that in once we have real
// CH outage signal to bucket against.
func classifyCHError(_ error) string {
	return "unknown"
}
