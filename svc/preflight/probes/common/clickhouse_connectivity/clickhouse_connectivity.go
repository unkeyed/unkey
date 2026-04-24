// Package clickhouse_connectivity verifies the preflight runner can
// reach ClickHouse and get a timely pong back. Serves as a prereq
// check so later CH-dependent probes (request_logs,
// reconciliation_drift) fail behind it instead of with cryptic errors.
//
// Added when the MySQL/ClickHouse split landed; see
// docs/preflight/roadmap.md decision 2026-04-24.
package clickhouse_connectivity

import (
	"context"
	"errors"
	"time"

	"github.com/unkeyed/unkey/svc/preflight/core"
	"github.com/unkeyed/unkey/svc/preflight/probes"
)

// Probe pings ClickHouse and asserts it responds within 2s.
//
// Behaviour:
//   - env.ClickHouse nil (no URL configured): fail loudly. If CH is
//     genuinely not available yet in an environment, shadow-list the
//     probe rather than swallow the failure silently.
//   - Ping returns within 2s: pass.
//   - Ping errors or times out: fail.
type Probe struct{}

// Name implements probes.Probe.
func (Probe) Name() string { return "clickhouse_connectivity" }

// Run implements probes.Probe.
func (Probe) Run(ctx context.Context, env *core.Env) core.Result {
	if env.ClickHouse == nil {
		return core.Fail(errors.New("env.ClickHouse is nil; set PREFLIGHT_CLICKHOUSE_URL"))
	}

	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	start := time.Now()
	err := env.ClickHouse.Ping(pingCtx)
	phases := []core.Phase{{Name: "ping", Duration: time.Since(start), Err: err}}

	if err != nil {
		return core.Failf("ClickHouse ping: %w", err).WithPhases(phases)
	}
	return core.Pass().WithPhases(phases)
}

func init() { probes.Register(Probe{}) }
