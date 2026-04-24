// Package noop implements a placeholder probe used during rollout to
// verify end-to-end Runner wiring before any real tier-1 probe lands.
// It is intentionally trivial and should be removed once real probes
// are stable.
package noop

import (
	"context"

	"github.com/unkeyed/unkey/svc/preflight/core"
	"github.com/unkeyed/unkey/svc/preflight/probes"
)

// Probe always passes; its output only proves that the probe registry,
// the Runner, and the observability pipeline are all wired.
type Probe struct{}

// Name implements probes.Probe.
func (Probe) Name() string { return "noop" }

// Run implements probes.Probe. Always returns OK with one Dim echoing
// the target, so you can tell staging noop apart from prod noop in logs.
func (Probe) Run(_ context.Context, env *core.Env) core.Result {
	return core.Pass().WithDims(map[string]string{"target": string(env.Target)})
}

func init() { probes.Register(Probe{}) }
