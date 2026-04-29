package proxy

import (
	"context"
	"net/http"
	"time"

	"github.com/unkeyed/unkey/pkg/zen"
)

// RequestTracking holds data collected during a local-instance request for
// ClickHouse logging. The handler initializes the tracking, the proxy
// callbacks (Director, ModifyResponse, ErrorHandler) mutate it as the
// request progresses. Cross-region requests do not populate tracking — the
// peer frontline writes its own ClickHouse row.
type RequestTracking struct {
	// Set by handler before proxy
	RequestID     string
	StartTime     time.Time
	DeploymentID  string
	WorkspaceID   string
	EnvironmentID string
	ProjectID     string
	InstanceID    string
	Address       string
	RequestBody   []byte

	// Set by proxy Director callback
	InstanceStart time.Time

	// Set by proxy ModifyResponse / ErrorHandler callbacks
	InstanceEnd     time.Time
	ResponseStatus  int32
	ResponseHeaders http.Header
	ResponseBody    []byte
}

var requestTrackingKey = zen.NewContextKey[*RequestTracking]("frontline_request_tracking")

// WithRequestTracking attaches a tracking record to the context.
func WithRequestTracking(ctx context.Context, t *RequestTracking) context.Context {
	return requestTrackingKey.WithValue(ctx, t)
}

// RequestTrackingFromContext returns the tracking record on the context, if
// one was attached.
func RequestTrackingFromContext(ctx context.Context) (*RequestTracking, bool) {
	return requestTrackingKey.FromContext(ctx)
}
