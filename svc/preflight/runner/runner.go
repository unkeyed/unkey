// Package runner executes preflight probes and emits every
// cross-cutting signal the plan requires: Prometheus metrics, OTEL
// spans, structured logs, diagnostic bundles on failure, and
// shadow-mode alert suppression.
//
// Probes are forbidden from touching any of those concerns directly.
// A probe returns a core.Result and the Runner does the rest. This
// split is the most important design invariant in the preflight
// codebase: if a probe could emit metrics on its own, we would end up
// with N copies of the same boilerplate and one of them would drift.
package runner

import (
	"context"
	"time"

	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/otel/tracing"
	"github.com/unkeyed/unkey/svc/preflight/core"
)

// Probe is the minimal contract Runner needs. Mirrors
// svc/preflight/probes.Probe structurally so the Runner does not
// import the probes package (which would invert the dependency).
type Probe interface {
	Name() string
	Run(ctx context.Context, env *core.Env) core.Result
}

// Runner wraps every probe invocation with tracing, metric emission,
// logging, diagnostic enrichment, and shadow-mode alert suppression.
//
// One Runner is constructed per suite per run.
type Runner struct {
	suite     string
	region    string
	artifacts ArtifactUploader

	// shadow contains probe names whose failures should NOT count
	// toward burn-rate alerts. They still emit metrics (so dashboards
	// stay honest) but log at warn rather than error and carry
	// result="shadow_fail" on the counter, which the PrometheusRule
	// ignores.
	shadow map[string]bool
}

// New constructs a Runner scoped to a single suite and region.
// artifacts may be nil; nil simply skips upload.
func New(suite, region string, artifacts ArtifactUploader, shadowed []string) *Runner {
	shadow := make(map[string]bool, len(shadowed))
	for _, name := range shadowed {
		shadow[name] = true
	}

	return &Runner{
		suite:     suite,
		region:    region,
		artifacts: artifacts,
		shadow:    shadow,
	}
}

// Invoke runs a single probe and emits every cross-cutting signal.
// The returned Result is the same one the probe produced, unmodified,
// so callers can make flow-control decisions (e.g. abort the suite on
// a tier-1 failure).
func (r *Runner) Invoke(ctx context.Context, p Probe, env *core.Env) core.Result {
	ctx, span := tracing.Start(ctx, "preflight."+p.Name())
	defer span.End()

	logger.Info("preflight probe starting",
		"suite", r.suite,
		"probe", p.Name(),
		"region", r.region,
	)

	t0 := time.Now()
	res := p.Run(ctx, env)
	dur := time.Since(t0)

	r.emitMetrics(p.Name(), res, dur)
	r.logResult(p.Name(), res, dur)
	tracing.RecordError(span, res.Err)

	if !res.OK {
		// Enrich with probe-supplied Diagnose output before uploading.
		// Everything after the probe's own Run is best-effort: a broken
		// Diagnose must not mask the underlying failure.
		if extra := r.safeDiagnose(ctx, p, env, res); len(extra) > 0 {
			res.Artifacts = append(res.Artifacts, extra...)
		}

		if r.artifacts != nil && len(res.Artifacts) > 0 {
			r.upload(ctx, p.Name(), env.RunID, res.Artifacts)
		}
	}

	return res
}
