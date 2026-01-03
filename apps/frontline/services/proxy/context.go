package proxy

import (
	"context"
	"time"

	"github.com/unkeyed/unkey/pkg/zen"
)

// requestStartTimeKey is the context key for storing the request start time.
// This is used to track timing across the request lifecycle without passing
// startTime as a parameter through multiple function calls.
var requestStartTimeKey = zen.NewContextKey[time.Time]("request_start_time")

// WithRequestStartTime stores the request start time in the context.
func WithRequestStartTime(ctx context.Context, startTime time.Time) context.Context {
	return requestStartTimeKey.WithValue(ctx, startTime)
}

// RequestStartTimeFromContext retrieves the request start time from the context.
// Returns the start time and true if found, or zero time and false if not found.
func RequestStartTimeFromContext(ctx context.Context) (time.Time, bool) {
	return requestStartTimeKey.FromContext(ctx)
}
