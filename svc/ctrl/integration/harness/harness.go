package harness

import (
	"testing"

	workerharness "github.com/unkeyed/unkey/svc/ctrl/worker/harness"
)

// Harness is re-exported for backwards compatibility.
type Harness = workerharness.Harness

// New proxies to the worker-scoped harness implementation.
func New(t *testing.T) *Harness {
	return workerharness.New(t)
}
