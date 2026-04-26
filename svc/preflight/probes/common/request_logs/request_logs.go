// Package request_logs implements tier-1.11 of the preflight plan.
// It asserts that sentinel request-log rows reach ClickHouse.
//
// The probe does not send the request itself: that is tier-1.10's job
// (frontline_routing). This probe only checks the logging side of the
// pipeline. In dev the harness seeds a row with Path
// "/preflight-<runID>" so the query has something to find; in
// staging, tier-1.10's traffic writes the row for real and this probe
// reads it back within wait.Poll's deadline.
//
// Splitting the "generate traffic" and "confirm log" concerns across
// two probes keeps each one narrow enough that an alert points at
// exactly one broken component: a request_logs failure when 1.10 is
// green means the sentinel to ClickHouse ingest pipeline is broken,
// not the request path.
package request_logs

import (
	"context"
	"errors"
	"time"

	"github.com/unkeyed/unkey/svc/preflight/core"
	"github.com/unkeyed/unkey/svc/preflight/internal/wait"
	"github.com/unkeyed/unkey/svc/preflight/probes"
)

// Default deadlines. 30s matches the tier-1.11 description in the
// plan; bump via env var if a specific environment has a slower
// ingest path (ClickHouse batches can buffer up to ~10s).
const (
	defaultTimeout = 30 * time.Second
	pollPeriod     = 1 * time.Second
)

// Probe asserts a sentinel_requests_raw_v1 row tagged with env.RunID
// lands in ClickHouse within defaultTimeout.
type Probe struct{}

// Name implements probes.Probe.
func (Probe) Name() string { return "request_logs" }

// Run implements probes.Probe.
func (Probe) Run(ctx context.Context, env *core.Env) core.Result {
	if env.ClickHouse == nil {
		return core.Fail(errors.New("env.ClickHouse is nil; set PREFLIGHT_CLICKHOUSE_URL"))
	}
	if env.RunID == "" {
		return core.Fail(errors.New("env.RunID is empty; cannot correlate row"))
	}

	path := "/preflight-" + env.RunID

	queryCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	start := time.Now()
	_, err := wait.Poll(queryCtx, pollPeriod, func(ctx context.Context) (struct{}, bool, error) {
		count, err := env.ClickHouse.CountSentinelRequestsByPath(ctx, path)
		if err != nil {
			// Transient CH errors should not abort; keep polling until
			// ctx deadline. A schema-level error will show up as
			// repeated failures and eventually surface as ErrDeadline.
			return struct{}{}, false, nil
		}
		return struct{}{}, count > 0, nil
	})

	phases := []core.Phase{{Name: "poll_clickhouse", Duration: time.Since(start), Err: err}}

	if err != nil {
		return core.Failf("no sentinel_requests row for path=%s within %s: %w",
			path, defaultTimeout, err).
			WithPhases(phases).
			WithDims(map[string]string{"path": path})
	}

	return core.Pass().
		WithPhases(phases).
		WithDims(map[string]string{"path": path})
}

func init() { probes.Register(Probe{}) }
