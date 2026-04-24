package preflight

import (
	"context"
	"fmt"

	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/preflight/core"
	"github.com/unkeyed/unkey/svc/preflight/probes"
	"github.com/unkeyed/unkey/svc/preflight/runner"
)

// RunSuite invokes every probe in the suite with the same Runner and
// returns the collected results. A probe's failure does not abort the
// suite unless the caller decides to, so tier-2/3/4 probes can be
// non-gating without special treatment.
func RunSuite(ctx context.Context, r *runner.Runner, env *core.Env, s Suite) []core.Result {
	logger.Info("preflight: suite started",
		"suite", s.Name,
		"region", env.Region,
		"run_id", env.RunID,
		"probe_count", len(s.Probes),
	)

	out := make([]core.Result, 0, len(s.Probes))
	for _, name := range s.Probes {
		p, ok := probes.ByName(name)
		if !ok {
			logger.Error("preflight: probe not registered", "probe", name)
			out = append(out, core.Fail(fmt.Errorf("probe %q not registered", name)))
			continue
		}
		out = append(out, r.Invoke(ctx, p, env))
	}

	summary := summarise(out)
	logger.Info("preflight: suite finished",
		"suite", s.Name,
		"region", env.Region,
		"run_id", env.RunID,
		"pass", summary.pass,
		"fail", summary.fail,
	)
	return out
}

type suiteSummary struct {
	pass int
	fail int
}

func summarise(results []core.Result) suiteSummary {
	var s suiteSummary
	for _, r := range results {
		if r.OK {
			s.pass++
		} else {
			s.fail++
		}
	}

	return s
}
