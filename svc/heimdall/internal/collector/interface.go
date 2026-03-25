package collector

import (
	"context"
	"time"
)

// ResourceCollector scrapes resource usage metrics from the node's kubelet
// and buffers them for writing to ClickHouse.
type ResourceCollector interface {
	// Run starts the collection loop at the given interval and blocks until
	// the context is cancelled.
	Run(ctx context.Context, interval time.Duration) error

}
