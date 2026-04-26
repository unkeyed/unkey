package harness_test

import (
	"context"
	"testing"
	"time"

	"github.com/unkeyed/unkey/svc/preflight"
	"github.com/unkeyed/unkey/svc/preflight/harness"
	"github.com/unkeyed/unkey/svc/preflight/runner"
)

// TestDev is the flagship end-to-end test for the preflight probe suite
// against the in-process harness. It is the dev equivalent of running
// `preflight --target=staging` in CI: same Runner, same suite
// composition, same probe registry; only the target clients differ.
//
// Running:
//
//	make test                                 # full repo (container orchestration included)
//	go test -run TestDev ./svc/preflight/...  # preflight-only, assumes containers running
//
// This test is the reason the dev target is NOT exposed as a CLI flag:
// here we already have testing.T so the harness wraps pkg/dockertest
// cleanly. Trying to call the harness from a binary would force us to
// shim testing.T, which is not worth the complexity.
func TestDev(t *testing.T) {
	// Deliberately NOT Parallel. Harness tests all hit the same shared
	// Restate container and compete for its admin endpoint during
	// worker registration; parallel runs occasionally time out.
	h := harness.Start(t, harness.Config{
		SeedPreflightProject: true,
		MockRestateServices:  harness.NullRestateServices(),
	})

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	r := runner.New("solo", h.Region, nil, nil)
	env := h.Env()

	// Seed a sentinel_request row for request_logs to find. In
	// staging the git_push probe writes this for real via sentinel;
	// in the harness we stand in. DevSuite excludes request_logs by
	// default (because outside the harness there is no traffic
	// source); we opt back in here because we just seeded.
	h.SeedSentinelRequest(t, ctx, env.RunID)

	suite := preflight.DevSuite()
	suite.Probes = append(suite.Probes, "request_logs")

	results := preflight.RunSuite(ctx, r, env, suite)

	if len(results) == 0 {
		t.Fatal("preflight: RunSuite returned no results; suite composition is wrong")
	}
	for _, r := range results {
		if !r.OK {
			t.Fatalf("preflight probe failed: err=%v", r.Err)
		}
	}
}
