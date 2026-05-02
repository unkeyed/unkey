// Package sinks contains the per-provider HTTP forwarders used by logdrain
// to push records out to third-party log providers.
//
// All sinks consume the same internal Record type and convert it to the
// provider's wire format. Batching and retry are the sink's responsibility.
// Backoff and auto-pause live in the worker layer that calls Send.
package sinks

import "context"

// RecordKind distinguishes runtime stdout/stderr lines from sentinel HTTP
// access log entries. Both flow through the same Sink, but the wire-format
// conversion attaches different attribute keys.
type RecordKind int

const (
	// RecordRuntime is a parsed stdout/stderr line from a customer pod.
	RecordRuntime RecordKind = iota
	// RecordRequest is a sentinel-observed HTTP request to a deployment.
	RecordRequest
)

// Record is the in-memory shape every Sink receives. It maps cleanly onto
// an OTLP LogRecord plus per-tenant identifiers.
//
// Attributes is the parsed structured payload (JSON or logfmt for runtime,
// HTTP request fields for request logs). Body is the human-readable
// message; for request logs it is set to a synthetic "method path status"
// summary so the provider has a useful default to display.
type Record struct {
	Kind         RecordKind
	TimeMs       int64
	SeverityText string

	// Tenant identifiers — always populated, propagated as attributes.
	WorkspaceID   string
	ProjectID     string
	EnvironmentID string
	AppID         string
	DeploymentID  string
	Region        string
	Platform      string
	K8sPodName    string

	Body       string
	Attributes map[string]any

	// CursorTimeMs is the per-record cursor watermark — `inserted_at` for
	// runtime rows, `time` for request rows. Together with LastID it
	// forms the (cursor_time, last_id) tuple the per-drain fan-out
	// compares against each drain's individual cursor to decide which
	// records the drain still owes its provider. Distinct from TimeMs
	// because runtime rows have a separate `inserted_at` column that is
	// monotonically stable in a way `time` (the log emission time) is
	// not.
	CursorTimeMs int64

	// LastID is the source row's stable per-row identifier and the
	// second component of the cursor tuple — `log_id` for runtime
	// (Vector-minted "log_<16 hex chars>" UUID-v7-shaped id), and
	// `request_id` for sentinel requests. Stored as a string so the
	// cursor predicate compares ClickHouse columns directly without an
	// inline cityHash64 fingerprint that would block sort-key prune.
	// Doubles as a stable Idempotency-Key for providers that support
	// per-event dedup.
	LastID string
}

// Sink is the contract every provider implementation satisfies. Send is
// called with a batch the worker has already filtered. The sink is
// responsible for chunking the batch to honour the provider's request size
// limits, but it is not responsible for cross-batch retry — that lives in
// the worker so consecutive_failures and auto-pause are uniform across
// providers.
//
// HealthCheck is invoked on the dashboard "test push" path during drain
// create/update; it should make a single, low-cost call that surfaces
// auth errors verbatim.
type Sink interface {
	Send(ctx context.Context, batch []Record) error
	HealthCheck(ctx context.Context) error
}

// SeverityNumber maps the parsed severity_text on a record into the OTLP
// severity_number scale (0..24). Unknown values default to INFO (9), which
// matches Vector's pipeline behaviour.
func SeverityNumber(text string) int32 {
	switch text {
	case "trace":
		return 1
	case "debug":
		return 5
	case "info", "":
		return 9
	case "notice":
		return 10
	case "warn", "warning":
		return 13
	case "error":
		return 17
	case "critical", "alert":
		return 21
	case "fatal", "emergency", "panic":
		return 24
	default:
		return 9
	}
}
