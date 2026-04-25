package schema

import (
	"encoding/json"
)

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
	// Tokens is the number of tokens the caller charged against the
	// limit on this decision. Recorded for both passed and rejected
	// requests so dashboards can break spend down by outcome.
	Tokens uint64 `ch:"tokens" json:"tokens"`
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
	Time         int64  `ch:"time" json:"time"`
	WorkspaceID  string `ch:"workspace_id" json:"workspace_id"`
	NamespaceID  string `ch:"namespace_id" json:"namespace_id"`
	Identifier   string `ch:"identifier" json:"identifier"`
	Passed       int64  `ch:"passed" json:"passed"`
	Total        int64  `ch:"total" json:"total"`
	PassedTokens int64  `ch:"passed_tokens" json:"passed_tokens"`
	TotalTokens  int64  `ch:"total_tokens" json:"total_tokens"`
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

// InstanceCheckpoint is a single counter reading for one container, written
// by heimdall. Billing math is done at query time as max(counter)-min(counter)
// over a window, which is monotone and idempotent on replay. We never store
// rates, only raw counters.
//
// Retry safety: retries of the same write carry identical values in every
// column, so ReplacingMergeTree's last-inserted-wins is correct regardless
// of which copy survives.
type InstanceCheckpoint struct {
	NodeID        string `ch:"node_id" json:"node_id"`
	WorkspaceID   string `ch:"workspace_id" json:"workspace_id"`
	ProjectID     string `ch:"project_id" json:"project_id"`
	EnvironmentID string `ch:"environment_id" json:"environment_id"`
	ResourceType  string `ch:"resource_type" json:"resource_type"`
	ResourceID    string `ch:"resource_id" json:"resource_id"`
	PodUID        string `ch:"pod_uid" json:"pod_uid"`
	InstanceID    string `ch:"instance_id" json:"instance_id"`
	ContainerUID  string `ch:"container_uid" json:"container_uid"`
	RestartCount  uint32 `ch:"restart_count" json:"restart_count"`
	Ts            int64  `ch:"ts" json:"ts"`
	EventKind     string `ch:"event_kind" json:"event_kind"`
	CPUUsageUsec  int64  `ch:"cpu_usage_usec" json:"cpu_usage_usec"`
	MemoryBytes   int64  `ch:"memory_bytes" json:"memory_bytes"`
	// Allocated from pod spec. Observational (utilization%), not billed.
	// Captured every tick so resize events appear as value changes over time.
	CPUAllocatedMillicores int32 `ch:"cpu_allocated_millicores" json:"cpu_allocated_millicores"`
	MemoryAllocatedBytes   int64 `ch:"memory_allocated_bytes" json:"memory_allocated_bytes"`
	DiskAllocatedBytes     int64 `ch:"disk_allocated_bytes" json:"disk_allocated_bytes"`
	DiskUsedBytes          int64 `ch:"disk_used_bytes" json:"disk_used_bytes"`
	// Network byte counters, monotonic per container_uid. Populated from
	// the per-pod eBPF TCX programs in svc/heimdall/internal/network: the
	// public/private split classifies on destination IP (RFC1918 +
	// link-local + loopback = private). Same billing shape as
	// cpu_usage_usec — max(counter)-min(counter) over a window. Zero on
	// platforms without eBPF (the macOS dev stub) or when attach hasn't
	// completed yet — see InstanceCheckpointAttributes.NetworkAttached
	// to distinguish "no traffic" from "not measured".
	NetworkEgressPublicBytes   int64  `ch:"network_egress_public_bytes" json:"network_egress_public_bytes"`
	NetworkEgressPrivateBytes  int64  `ch:"network_egress_private_bytes" json:"network_egress_private_bytes"`
	NetworkIngressPublicBytes  int64  `ch:"network_ingress_public_bytes" json:"network_ingress_public_bytes"`
	NetworkIngressPrivateBytes int64  `ch:"network_ingress_private_bytes" json:"network_ingress_private_bytes"`
	Region                     string `ch:"region" json:"region"`
	Platform                   string `ch:"platform" json:"platform"`
	// Attributes is open-schema diagnostic metadata serialised as a JSON
	// object string. The wire format is `string` because the ClickHouse
	// driver's JSON column path expects a marshaled string; producers
	// should construct via the typed InstanceCheckpointAttributes struct
	// and call Marshal so field names are compile-time checked.
	Attributes string `ch:"attributes" json:"attributes"`
}

// InstanceCheckpointAttributes is the typed payload for InstanceCheckpoint.Attributes.
//
// Adding a key is one struct field — no schema migration, no rollup propagation,
// no impact on billing math. omitempty keeps the on-disk JSON compact when a
// field is absent (e.g. image_id before kubelet has caught up with the pod).
type InstanceCheckpointAttributes struct {
	// Image is the container image string (e.g. "registry.io/foo:abc123") of
	// the primary container, taken from Status.ContainerStatuses at sample
	// time. Pairs with ImageID for "this OOM correlates with this rebuild"
	// debugging without joining live pod state.
	Image string `json:"image,omitempty"`
	// ImageID is the immutable image digest reported by kubelet. Stable
	// across pulls of the same tag, so it's the right key for tracking
	// "did the binary change between these two checkpoints?"
	ImageID string `json:"image_id,omitempty"`
	// QOSClass is the pod's QoS class (Guaranteed | Burstable | BestEffort).
	// Determines OOM-kill ordering, so it's high-signal context when a
	// memory column shows a sudden zero (process killed) vs flatline.
	QOSClass string `json:"qos_class,omitempty"`
	// EBPFProgramVersion is heimdall's git revision (from buildinfo). The
	// bundled TCX programs are rebuilt with the binary, so this maps 1:1
	// to the network counter implementation that produced the row.
	EBPFProgramVersion string `json:"ebpf_program_version,omitempty"`
	// EBPFPinDir is the bpffs path heimdall pins its maps under. Bumped
	// whenever the map ABI changes (e.g. v1→v2 when the key size changed
	// from u32 ifindex to u64 netns cookie). Lets checkpoints from old
	// vs new pin generations be distinguished at query time without a
	// node_id join against deploy history.
	EBPFPinDir string `json:"ebpf_pin_dir,omitempty"`
	// NetworkAttached is true when the eBPF TCX counters were actually
	// read for this checkpoint. False means the network_*_bytes columns
	// are zero by fail-open (host-network pod, attach queue full, pod
	// not Running yet, eBPF reader unavailable on macOS dev) — lets us
	// tell "real zero traffic" apart from "we couldn't read it" without
	// running a separate Prometheus correlation.
	NetworkAttached bool `json:"network_attached,omitempty"`
	// Collectors is the list of metric kinds this heimdall agent had
	// enabled when the row was written (subset of cpu / memory / disk /
	// network). Disabled kinds write zero into the corresponding numeric
	// columns; this attribute lets query-time consumers tell "0 because
	// disabled" from "0 because measured zero". Empty means all four
	// were enabled (the unset/default case).
	Collectors []string `json:"enabled_collectors,omitempty"`
}

// Marshal renders the attributes payload as the JSON-string wire form expected
// by the instance_checkpoints_v1.attributes column. json.Marshal of a struct
// of scalar fields cannot fail.
func (a InstanceCheckpointAttributes) Marshal() string {
	b, _ := json.Marshal(a)
	return string(b)
}

// InstanceEventV1 represents the v1 instance event raw table structure.
// Captured by krane's pod watch on container running, termination, and
// waiting (CrashLoopBackOff, ImagePullBackOff, …) transitions. Mirrors
// corev1.ContainerState; the proto wire format uses a oneof, the CH table
// is the flat materialized view ctrl writes from that oneof.
type InstanceEventV1 struct {
	Time          int64  `ch:"time" json:"time"`
	WorkspaceID   string `ch:"workspace_id" json:"workspace_id"`
	ProjectID     string `ch:"project_id" json:"project_id"`
	AppID         string `ch:"app_id" json:"app_id"`
	EnvironmentID string `ch:"environment_id" json:"environment_id"`
	DeploymentID  string `ch:"deployment_id" json:"deployment_id"`

	PodUID        string `ch:"pod_uid" json:"pod_uid"`
	PodName       string `ch:"pod_name" json:"pod_name"`
	NodeName      string `ch:"node_name" json:"node_name"`
	ContainerName string `ch:"container_name" json:"container_name"`
	ContainerID   string `ch:"container_id" json:"container_id"`
	RestartCount  int32  `ch:"restart_count" json:"restart_count"`

	EventKind string `ch:"event_kind" json:"event_kind"`

	ExitCode int32  `ch:"exit_code" json:"exit_code"`
	Signal   int32  `ch:"signal" json:"signal"`
	Reason   string `ch:"reason" json:"reason"`
	Message  string `ch:"message" json:"message"`

	Region   string `ch:"region" json:"region"`
	Platform string `ch:"platform" json:"platform"`

	EventFingerprint string `ch:"event_fingerprint" json:"event_fingerprint"`

	// Attributes carries selected k8s metadata for the event row (image,
	// resource limits, build_id, …) as a JSON-encoded string. ctrl marshals
	// the proto map<string,string> into JSON before queuing the row so the
	// CH JSON column receives valid JSON (empty map → "{}"). The string
	// shape sidesteps a clickhouse-go quirk where AppendStruct doesn't
	// reliably auto-serialize Go maps into JSON columns.
	Attributes string `ch:"attributes" json:"attributes"`
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

	// CorrelationID groups rows that came out of one logical user action.
	// Empty for single-event flows; auto-minted by the audit log Insert
	// service when the caller batches >1 events; settable via
	// auditlog.WithCorrelation(ctx, ...) for flows that fan out across
	// multiple Insert calls.
	CorrelationID string `ch:"correlation_id" json:"correlation_id"`
}
