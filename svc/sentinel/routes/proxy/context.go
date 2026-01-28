package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/unkeyed/unkey/pkg/zen"
)

// SentinelRequestTracking holds data collected during request processing
// for ClickHouse logging. Shared via context, mutated by handler/proxy callbacks.
type SentinelRequestTracking struct {
	// Set by handler before proxy
	RequestID    string
	StartTime    time.Time
	DeploymentID string
	Deployment   *DeploymentInfo
	Instance     *InstanceInfo
	RequestBody  []byte

	// Set by proxy Director callback
	InstanceStart time.Time

	// Set by proxy ModifyResponse/ErrorHandler callbacks
	InstanceEnd     time.Time
	ResponseStatus  int32
	ResponseHeaders http.Header
	ResponseBody    []byte
}

type DeploymentInfo struct {
	WorkspaceID   string
	EnvironmentID string
	ProjectID     string
}

type InstanceInfo struct {
	ID      string
	Address string
}

var sentinelTrackingKey = zen.NewContextKey[*SentinelRequestTracking]("sentinel_tracking")

func WithSentinelTracking(ctx context.Context, tracking *SentinelRequestTracking) context.Context {
	return sentinelTrackingKey.WithValue(ctx, tracking)
}

func SentinelTrackingFromContext(ctx context.Context) (*SentinelRequestTracking, bool) {
	return sentinelTrackingKey.FromContext(ctx)
}
