package probes

import (
	"fmt"
	"sort"
	"sync"
)

// registry is the package-level probe registry. Suites are built from
// these entries. Registration order does not determine run order; see
// suite composition in cmd/preflight/flow.go.
var (
	registryMu sync.RWMutex
	registry   = map[string]Probe{}
)

// Register adds a probe to the global registry. Duplicate names panic
// because a name collision means two probes share a metric label set,
// which would silently corrupt the dashboards.
func Register(p Probe) {
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, exists := registry[p.Name()]; exists {
		panic(fmt.Sprintf("probes: duplicate registration for %q", p.Name()))
	}
	registry[p.Name()] = p
}

// All returns every registered probe, sorted by name for deterministic
// iteration in tests and runbook-check CI.
func All() []Probe {
	registryMu.RLock()
	defer registryMu.RUnlock()
	out := make([]Probe, 0, len(registry))
	for _, p := range registry {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name() < out[j].Name() })
	return out
}

// ByName looks up a single probe. Returns (nil, false) when unknown.
// Used by flow.go when a suite opts a probe in by name.
func ByName(name string) (Probe, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	p, ok := registry[name]
	return p, ok
}

// Names returns the sorted list of registered probe names. Used by the
// runbook-check CI to match probes against docs/preflight/probes/<name>.md.
func Names() []string {
	probes := All()
	names := make([]string, len(probes))
	for i, p := range probes {
		names[i] = p.Name()
	}
	return names
}
