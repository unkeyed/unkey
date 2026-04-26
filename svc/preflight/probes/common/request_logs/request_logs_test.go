package request_logs_test

import (
	"context"
	"testing"
	"time"

	"github.com/unkeyed/unkey/svc/preflight/harness"
	"github.com/unkeyed/unkey/svc/preflight/probes/common/request_logs"
)

func TestProbe_FindsSeededRow(t *testing.T) {
	// Harness tests are sequential; see dev_test.go.

	h := harness.Start(t, harness.Config{
		SeedPreflightProject: true,
		MockRestateServices:  harness.NullRestateServices(),
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	env := h.Env()
	// Simulate what real sentinel would write during a tier-1.10 probe.
	h.SeedSentinelRequest(t, ctx, env.RunID)

	res := request_logs.Probe{}.Run(ctx, env)
	if !res.OK {
		t.Fatalf("probe failed: err=%v phases=%+v", res.Err, res.Phases)
	}
	if len(res.Phases) != 1 || res.Phases[0].Name != "poll_clickhouse" {
		t.Errorf("expected single 'poll_clickhouse' phase, got %+v", res.Phases)
	}
}

func TestProbe_FailsWhenNoRow(t *testing.T) {
	// No SeedSentinelRequest: probe should time out and fail.

	h := harness.Start(t, harness.Config{
		SeedPreflightProject: false,
		MockRestateServices:  harness.NullRestateServices(),
	})

	// Short deadline via ctx so the test finishes quickly; the probe's
	// own 30s timeout would otherwise block us for the full duration.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res := request_logs.Probe{}.Run(ctx, h.Env())
	if res.OK {
		t.Fatal("probe should fail when no row has been written")
	}
	if res.Err == nil {
		t.Fatal("expected Err to be populated on failure")
	}
}
