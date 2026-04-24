package core

import "fmt"

// Result is the sole contract between a Probe and the Runner.
// Probes construct a Result and return it; the Runner reads the Result
// and emits metrics, logs, traces, and artifact bundles from it.
//
// Probes MUST NOT emit their own metrics, open their own spans, or upload
// their own artifacts. Everything cross-cutting lives in the Runner so that
// adding a new output (PagerDuty, Sentry, etc.) is a one-file change.
type Result struct {
	// OK is true when the probe passed. Any other value means the Runner
	// counts the run as a failure and uploads Artifacts.
	OK bool

	// Err is populated on failure. Prefer wrapping with fmt.Errorf("%w", ...)
	// over bare strings so callers can errors.Is/As later.
	Err error

	// Phases captures per-step timings inside a probe. The Runner emits one
	// preflight_phase_duration_seconds histogram bucket per phase. Leave
	// empty for probes that are a single assertion.
	Phases []Phase

	// Dims adds extra labels to metrics and log lines. Use for dimensions
	// that the standard {suite,probe,result,region} label set does not cover
	// (e.g. {"source_type":"git"}, {"protocol":"h2c"}).
	Dims map[string]string

	// Artifacts are uploaded to the artifact bucket only when OK is false.
	// Keep them small: kubectl describes, log tails, JSON snapshots.
	Artifacts []Artifact
}

// Pass returns an OK Result with every optional field nil. Probes that
// want to attach phases / dims / artifacts chain WithPhases / WithDims
// / WithArtifacts on top.
//
//	return core.Pass().WithPhases(phases)
//	return core.Pass().WithDims(map[string]string{"protocol": "h2c"})
func Pass() Result {
	return Result{
		OK:        true,
		Err:       nil,
		Phases:    nil,
		Dims:      nil,
		Artifacts: nil,
	}
}

// Fail returns a failed Result carrying err. Like Pass, optional fields
// are nil; probes add them via chainers.
//
//	return core.Fail(err)
//	return core.Fail(err).WithPhases(phases).WithArtifacts(bundle)
func Fail(err error) Result {
	return Result{
		OK:        false,
		Err:       err,
		Phases:    nil,
		Dims:      nil,
		Artifacts: nil,
	}
}

// Failf is the fmt.Errorf shorthand for Fail. Preserves %w wrapping
// because %w goes through fmt.Errorf unchanged.
//
//	return core.Failf("list deployments: %w", err)
func Failf(format string, args ...any) Result {
	return Fail(fmt.Errorf(format, args...))
}

// WithPhases returns r with the supplied phases attached. Phases feed
// the per-phase histogram the Runner emits.
func (r Result) WithPhases(phases []Phase) Result {
	r.Phases = phases
	return r
}

// WithDims returns r with the supplied extra Prometheus labels
// attached. Keep cardinality low; values become part of every metric
// series the Runner emits for the enclosing probe.
func (r Result) WithDims(dims map[string]string) Result {
	r.Dims = dims
	return r
}

// WithArtifacts returns r with the supplied diagnostic bundles
// attached. Only uploaded on failure; keep artifacts small.
func (r Result) WithArtifacts(artifacts []Artifact) Result {
	r.Artifacts = artifacts
	return r
}
