package runner

import (
	"context"
	"runtime/debug"
	"time"

	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/preflight/core"
)

// diagnoser mirrors svc/preflight/probes.Diagnoser so the Runner does
// not import the probes package (which would invert the dependency).
// Go's structural typing means a probe that implements
// probes.Diagnoser also satisfies this interface.
type diagnoser interface {
	Diagnose(ctx context.Context, env *core.Env, failure core.Result) []core.Artifact
}

// diagnoseBudget bounds how long the Runner will wait for a probe's
// Diagnose call. Diagnostics are best-effort: if they take too long
// the oncall still gets the probe's own artifacts + error message.
const diagnoseBudget = 5 * time.Second

// safeDiagnose invokes a probe's optional Diagnose method with a
// bounded deadline and a recover(), so a broken diagnostic path
// never masks the probe failure it was meant to explain.
func (r *Runner) safeDiagnose(ctx context.Context, p Probe, env *core.Env, failure core.Result) (artifacts []core.Artifact) {
	d, ok := any(p).(diagnoser)
	if !ok {
		return nil
	}

	defer func() {
		if rec := recover(); rec != nil {
			logger.Error("preflight: Diagnose panicked",
				"probe", p.Name(),
				"panic", rec,
				"stack", string(debug.Stack()),
			)
			artifacts = nil
		}
	}()

	diagCtx, cancel := context.WithTimeout(ctx, diagnoseBudget)
	defer cancel()
	return d.Diagnose(diagCtx, env, failure)
}
