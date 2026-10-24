package lazy

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

// state tracks where lazy metrics register. Only first-use registration and
// SetRegistry take the lock, never Inc or Observe.
var state struct {
	mu           sync.Mutex
	registry     *prometheus.Registry
	explicit     bool
	collectors   []prometheus.Collector
	declarations []declaration
}

// SetRegistry sets the registry that all lazy metrics register to, and
// creates every declared counter and histogram series at 0, so a series
// exists before its first event. See [Values] and [Dynamic].
//
// Metrics used before SetRegistry go to a fallback registry so that tests
// which exercise metric-using code without a service's Run() do not panic.
// SetRegistry moves those collectors to reg, keeping the same collector
// objects, so a metric touched during startup is still scraped. It panics if
// a moved collector conflicts with one already in reg.
//
// First call wins: subsequent calls are no-ops. Production binaries only call
// this once per process, so the idempotence is there to support in-process
// multi-node integration tests (e.g. svc/api/integration), where several
// api.Run goroutines share a single process and therefore a single lazy
// registry. Metric values from those nodes are aggregated rather than
// isolated, which is fine because those tests don't assert on metric state.
func SetRegistry(reg *prometheus.Registry) {
	if reg == nil {
		panic("lazy.SetRegistry: nil registry")
	}
	if promote(reg) {
		primeDeclared()
	}
}

// promote makes reg the registry and reports whether it was the first
// explicit call.
func promote(reg *prometheus.Registry) bool {
	state.mu.Lock()
	defer state.mu.Unlock()

	if state.explicit {
		return false
	}

	for _, c := range state.collectors {
		reg.MustRegister(c)
		if state.registry != nil {
			state.registry.Unregister(c)
		}
	}
	state.registry = reg
	state.explicit = true
	return true
}

// registered adds c to the current registry and returns it. Until SetRegistry
// runs, it remembers c so SetRegistry can move it out of the fallback.
func registered[C prometheus.Collector](c C) C {
	state.mu.Lock()
	defer state.mu.Unlock()

	if state.registry == nil {
		state.registry = prometheus.NewRegistry()
	}
	state.registry.MustRegister(c)
	if !state.explicit {
		state.collectors = append(state.collectors, c)
	}
	return c
}
