package create_deployment_test

import (
	"context"
	"testing"
	"time"

	"github.com/unkeyed/unkey/svc/preflight/harness"
	"github.com/unkeyed/unkey/svc/preflight/probes/common/create_deployment"
)

func TestProbe_RunsAgainstHarness(t *testing.T) {
	// Harness tests are sequential by convention; see dev_test.go.

	h := harness.Start(t, harness.Config{
		SeedPreflightProject: true,
		MockRestateServices:  harness.NullRestateServices(),
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	res := create_deployment.Probe{}.Run(ctx, h.Env())

	if !res.OK {
		t.Fatalf("probe failed: err=%v, phases=%+v", res.Err, res.Phases)
	}
	if len(res.Phases) != 2 {
		t.Fatalf("expected 2 phases (create, get), got %d", len(res.Phases))
	}
	if res.Phases[0].Name != "create" || res.Phases[1].Name != "get" {
		t.Errorf("phase names wrong: %+v", res.Phases)
	}
}
