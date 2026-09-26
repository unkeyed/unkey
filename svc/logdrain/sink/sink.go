// Package sink defines the destination-side contract of the logdrain
// engine and its implementations (generic HTTP and Axiom).
//
// The engine (svc/logdrain/internal/engine) reads a batch of events from the
// source, calls Deliver exactly once per attempt, and commits the drain's
// offset only after Deliver returns an acknowledged result. Sinks therefore
// MUST NOT report acknowledgment unless the destination accepted the data: a
// false positive here converts the system from at-least-once to at-most-once.
//
// Delivery is at-least-once end to end. Destinations may receive duplicates
// after retries; every event carries EventID so consumers can dedupe.
package sink

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// diagnosticBodyBytesMax caps how much of a destination response is kept for
// diagnostics. Destinations control the response, so reads must be bounded.
const diagnosticBodyBytesMax = 4 << 10

// ReadDiagnostic reads up to diagnosticBodyBytesMax of the response body for
// error diagnostics, then drains and closes the body so the underlying HTTP
// connection can be reused. The boolean reports whether the body was truncated.
func ReadDiagnostic(body io.ReadCloser) ([]byte, bool, error) {
	diagnostic, readErr := io.ReadAll(io.LimitReader(body, diagnosticBodyBytesMax+1))
	_, drainErr := io.Copy(io.Discard, body)
	closeErr := body.Close()
	if readErr != nil {
		return nil, false, fmt.Errorf("read response body: %w", readErr)
	}
	if drainErr != nil {
		return nil, false, fmt.Errorf("drain response body: %w", drainErr)
	}
	if closeErr != nil {
		return nil, false, fmt.Errorf("close response body: %w", closeErr)
	}
	truncated := len(diagnostic) > diagnosticBodyBytesMax
	return diagnostic[:min(len(diagnostic), diagnosticBodyBytesMax)], truncated, nil
}

// Payload is the stream-specific event body. Each stream defines one
// concrete payload type so sinks can reshape fields for their
// destination without re-parsing JSON.
type Payload interface{ isPayload() }

// AuditLogActor identifies who performed the audited action.
type AuditLogActor struct {
	ID       string          `json:"id"`
	Type     string          `json:"type"`
	Name     string          `json:"name"`
	Metadata json.RawMessage `json:"metadata"`
}

// AuditLogTarget identifies one resource the audited action affected.
type AuditLogTarget struct {
	ID       string          `json:"id"`
	Type     string          `json:"type"`
	Name     string          `json:"name"`
	Metadata json.RawMessage `json:"metadata"`
}

// AuditLogContext carries request-level context of the audited action.
type AuditLogContext struct {
	Location  string `json:"location"`
	UserAgent string `json:"user_agent"`
}

// AuditLogPayload carries the data for the "audit_logs" stream.
type AuditLogPayload struct {
	ID     string `json:"id"`
	Action string `json:"action"`
	// OccurredAt is the event time in RFC3339 UTC with millisecond precision.
	OccurredAt    string           `json:"occurred_at"`
	Actor         AuditLogActor    `json:"actor"`
	Targets       []AuditLogTarget `json:"targets"`
	Context       AuditLogContext  `json:"context"`
	Metadata      json.RawMessage  `json:"metadata"`
	Description   string           `json:"description"`
	CorrelationID string           `json:"correlation_id"`
}

func (AuditLogPayload) isPayload() {}

// Event carries one log record for a destination to encode.
type Event struct {
	// EventID uniquely identifies the event within its stream. Consumers use
	// it to deduplicate redeliveries.
	EventID string

	// Stream names the source, e.g. "audit_logs".
	Stream string

	// Time is the event time in unix milliseconds.
	Time int64

	// Payload is the typed stream-specific event body. For
	// audit_logs this is the audit event envelope (actor, action, resources).
	Payload Payload
}

// Batch is what one Deliver call ships.
type Batch struct {
	// SchemaVersion versions the export envelope, starting at "v1".
	SchemaVersion string

	// DrainID identifies the logdrain configuration this delivery belongs to.
	DrainID string

	// WorkspaceID owning the drain and all events in the batch.
	WorkspaceID string

	Events []Event
}

// Result describes a completed delivery attempt. It keeps expected destination
// rejections separate from operational errors that produce no valid response.
type Result struct {
	// Acknowledged reports whether the destination accepted the complete batch.
	Acknowledged bool
	// HTTPStatus is the destination response status. Zero means no HTTP response exists.
	HTTPStatus int
	// ResponseBody contains a bounded response body for diagnostics.
	ResponseBody string
	// RequestBodyBytes is the uncompressed encoded request body size. It excludes headers.
	RequestBodyBytes int64
	// RetryAfter is a sink-specific minimum delay requested by the destination.
	RetryAfter time.Duration
}

// Sink delivers batches to one destination kind.
type Sink interface {
	// Deliver returns structured metadata for each attempt. It returns an error
	// for encoding, transport, or response-read failures. When available, the
	// result still contains metadata such as the request body size and status.
	Deliver(ctx context.Context, batch Batch) (Result, error)
}

// ParseRetryAfter parses the standard Retry-After delta-seconds and HTTP-date
// formats. An expired date or a negative or overflowing delta is invalid.
func ParseRetryAfter(value string, now time.Time) (time.Duration, bool) {
	value = strings.TrimSpace(value)
	seconds, secondsErr := strconv.ParseInt(value, 10, 64)
	if secondsErr == nil {
		if seconds < 0 || seconds > int64((time.Duration(1<<63-1))/time.Second) {
			return 0, false
		}
		return time.Duration(seconds) * time.Second, true
	}
	retryAt, dateErr := http.ParseTime(value)
	if dateErr != nil {
		return 0, false
	}
	delay := retryAt.Sub(now)
	if delay <= 0 {
		return 0, false
	}
	return delay, true
}
