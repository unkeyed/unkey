// Package probes holds the preflight probe library.
//
// A probe asserts one property of the deploy pipeline and returns a
// typed Result. Probes are composed into suites by the registry below and
// invoked by the Runner in cmd/preflight.
//
// Contract for probes:
//   - MUST NOT import prometheus, OTEL, slog, the artifact uploader, Slack,
//     or incident.io. The Runner handles every cross-cutting concern.
//   - MUST NOT open their own clients. Use *preflight.Env.
//   - MUST be idempotent and safe to retry.
//
// Adding a new probe:
//  1. Create a file in this package implementing the Probe interface.
//  2. Register it in registry.go.
//  3. Write a runbook at docs/preflight/probes/<name>.md.
//
// See docs/preflight/adding-a-probe.md for the canonical worked example.
package probes

import (
	"context"

	"github.com/unkeyed/unkey/svc/preflight/core"
)

// Probe is the preflight probe interface. Implementations live in this
// package (one file per probe) and are registered via Register().
type Probe interface {
	// Name returns a stable, lowercase, underscore-separated identifier used
	// as the probe label on every metric and log line. It is also used to
	// look up the probe's runbook at docs/preflight/probes/<name>.md.
	Name() string

	// Run performs the assertion. It must return quickly relative to its
	// caller's context deadline; the Runner wraps the call in tracing and
	// metric emission.
	Run(ctx context.Context, env *core.Env) core.Result
}

// Diagnoser is the optional probe interface for on-failure enrichment.
// The Runner calls Diagnose only when Run returns Result.OK = false,
// appends the returned artifacts to the bundle it uploads, and never
// lets Diagnose's own failures mask the underlying probe failure.
//
// Diagnose is explicitly allowed to read from env.DB when non-nil:
// diagnostics are not assertions, so the "primary assertions go through
// the customer surface" rule does not apply. A failed probe is the
// worst possible moment to restrict operators to guessing; pull
// whatever context helps an oncall at 3am into the bundle.
//
// Implementations should:
//   - Return quickly. The Runner grants ~5s on top of the probe's
//     existing deadline; honour that.
//   - Return nil (not an error) when no artifacts can be collected.
//     A probe that only has trivial failure modes simply does not
//     implement Diagnoser.
//   - Never mutate env or external state.
type Diagnoser interface {
	Diagnose(ctx context.Context, env *core.Env, failure core.Result) []core.Artifact
}
