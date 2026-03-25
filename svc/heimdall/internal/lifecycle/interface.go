package lifecycle

import "context"

// EventTracker watches Kubernetes pod lifecycle events and emits billing
// events with millisecond-precise timestamps.
type EventTracker interface {
	// Run starts the pod informer and blocks until the context is cancelled.
	Run(ctx context.Context) error
}
