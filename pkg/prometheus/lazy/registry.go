package lazy

import (
	"sync/atomic"

	"github.com/prometheus/client_golang/prometheus"
)

// registry is the target registry for lazy metrics. Set once at startup.
var registry atomic.Pointer[prometheus.Registry]

// SetRegistry sets the registry that all lazy metrics will register to on
// first use. Must be called before any metric is used.
//
// First call wins: subsequent calls are silent no-ops. Production binaries
// only call this once per process, so the idempotence is there to support
// in-process multi-node integration tests (e.g. svc/api/integration), where
// several api.Run goroutines share a single process and therefore a single
// lazy registry. Metric values from those nodes are aggregated rather than
// isolated, which is fine because those tests don't assert on metric state.
func SetRegistry(reg *prometheus.Registry) {
	if reg == nil {
		panic("lazy.SetRegistry: nil registry")
	}
	registry.CompareAndSwap(nil, reg)
}

func getRegistry() prometheus.Registerer {
	if r := registry.Load(); r != nil {
		return r
	}
	// No explicit SetRegistry — install a throwaway default so tests that
	// touch metric-using code directly (without going through a service's
	// Run()) don't panic. Production binaries call SetRegistry in Run()
	// before any metric use, so the default is never installed there.
	registry.CompareAndSwap(nil, prometheus.NewRegistry())
	return registry.Load()
}
