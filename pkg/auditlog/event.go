package auditlog

import (
	"slices"

	"github.com/unkeyed/unkey/pkg/assert"
)

// Event is the canonical envelope JSON-encoded into the clickhouse_outbox
// payload column. Both the writer (internal/services/auditlogs) and the
// drainer (svc/ctrl/worker/cron RunAuditLogExport) marshal/unmarshal this shape.
// Additive changes are safe (json.Unmarshal ignores unknown fields).
// Breaking changes (rename, type change, removed field) MUST bump
// OutboxVersionV1 below to a new value and ship a drainer that handles
// both before the writer starts emitting it.
//
// Source distinguishes platform-emitted events ("platform" — Unkey's own
// mutations against customer resources) from customer-emitted events
// ("customer" — once that surface ships). The drainer copies this into the
// CH `source` column unchanged.
type Event struct {
	EventID     string         `json:"event_id"`
	Time        int64          `json:"time"`
	WorkspaceID string         `json:"workspace_id"`
	Bucket      string         `json:"bucket"`
	Source      string         `json:"source"`
	Event       string         `json:"event"`
	Description string         `json:"description"`
	Actor       EventActor     `json:"actor"`
	RemoteIP    string         `json:"remote_ip,omitempty"`
	UserAgent   string         `json:"user_agent,omitempty"`
	Meta        map[string]any `json:"meta,omitempty"`
	Targets     []EventTarget  `json:"targets,omitempty"`
	// CorrelationID groups events emitted by one logical user action. The
	// audit log Insert service auto-mints one for any batched call with
	// >1 events; opt-in via WithCorrelation(ctx, ...) for flows that
	// emit events from multiple Insert calls. Empty for single-event
	// flows that don't need grouping.
	CorrelationID string `json:"correlation_id,omitempty"`
}

// EventActor is the actor sub-shape of Event. Kept separate so the JSON shape
// nests one level (matches how dashboards visualize "who").
type EventActor struct {
	Type string         `json:"type"`
	ID   string         `json:"id"`
	Name string         `json:"name,omitempty"`
	Meta map[string]any `json:"meta,omitempty"`
}

// EventTarget is one resource affected by the audit event. Multiple targets
// per event are common (e.g. a permission grant that touches a role + a key).
type EventTarget struct {
	Type string         `json:"type"`
	ID   string         `json:"id"`
	Name string         `json:"name,omitempty"`
	Meta map[string]any `json:"meta,omitempty"`
}

// Source values for Event.Source.
const (
	EventSourcePlatform = "platform"
	EventSourceCustomer = "customer"
)

// Bucket values for Event.Bucket. BucketBackoffice holds staff actions
// taken on a workspace from the back office. Dashboard and log drain readers
// must restrict to DashboardBuckets so those rows never leak.
const (
	BucketUnkeyMutations = "unkey_mutations"
	BucketBackoffice     = "unkey_backoffice"
)

// DashboardBuckets lists the buckets a workspace member may read
// through the dashboard or export through a log drain.
var DashboardBuckets = []string{BucketUnkeyMutations}

// KnownBuckets lists every bucket a writer may emit.
var KnownBuckets = []string{BucketUnkeyMutations, BucketBackoffice}

// IsKnownBucket reports whether bucket is one of KnownBuckets.
func IsKnownBucket(bucket string) bool {
	return slices.Contains(KnownBuckets, bucket)
}

// ResolveBucket returns the bucket a writer must stamp on an event. An
// empty bucket resolves to BucketUnkeyMutations. Any other value outside
// KnownBuckets is a programmer error and returns an assertion error, so a
// typo can never create a bucket that no reader filters on.
func ResolveBucket(bucket string) (string, error) {
	if bucket == "" {
		return BucketUnkeyMutations, nil
	}
	if err := assert.True(IsKnownBucket(bucket), "unknown audit log bucket"); err != nil {
		return "", err
	}
	return bucket, nil
}

// OutboxVersionV1 is the `version` value the writer puts on every
// clickhouse_outbox row that holds an Event. Drainers must include this
// in their known-versions list to pick up audit-log events. Bumping it
// requires shipping a drainer that handles the new version BEFORE the
// writer starts emitting it, otherwise rows pile up unread.
//
// Versions are namespaced per producer: `audit_log.*` is owned by the
// audit log writer/drainer pair. Future producers (e.g. customer-emitted
// audit logs, key lifecycle events) should pick their own prefix.
const OutboxVersionV1 = "audit_log.v1"
