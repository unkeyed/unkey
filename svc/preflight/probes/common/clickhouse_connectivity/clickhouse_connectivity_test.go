package clickhouse_connectivity_test

import (
	"context"
	"testing"
	"time"

	"github.com/unkeyed/unkey/svc/preflight/core"
	"github.com/unkeyed/unkey/svc/preflight/harness"
	"github.com/unkeyed/unkey/svc/preflight/probes/common/clickhouse_connectivity"
)

func TestProbe_RunsAgainstHarness(t *testing.T) {
	// Sequential by convention; see dev_test.go.
	h := harness.Start(t, harness.Config{
		SeedPreflightProject: false,
		MockRestateServices:  harness.NullRestateServices(),
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res := clickhouse_connectivity.Probe{}.Run(ctx, h.Env())
	if !res.OK {
		t.Fatalf("probe failed: err=%v phases=%+v", res.Err, res.Phases)
	}
	if len(res.Phases) != 1 || res.Phases[0].Name != "ping" {
		t.Errorf("expected single 'ping' phase, got %+v", res.Phases)
	}
}

func TestProbe_FailsWhenUnconfigured(t *testing.T) {
	// Do not spin up the harness for this; we want env.ClickHouse nil.
	env := &core.Env{
		Target:                   core.TargetStaging,
		Region:                   "test",
		RunID:                    "pflt_test",
		CtrlBaseURL:              "",
		CtrlAuthToken:            "",
		GitHubWebhookSecret:      "",
		PreflightProjectID:       "",
		PreflightAppID:           "",
		PreflightEnvironmentSlug: "",
		PreflightProjectSlug:     "",
		PreflightAppSlug:         "",
		PreflightWorkspaceSlug:   "",
		PreflightApex:            "",
		GitHubAppID:              0,
		GitHubInstallationID:     0,
		GitHubPrivateKeyPEM:      "",
		PreflightTestRepo:        "",
		DB:                       nil,
		ClickHouse:               nil,
	}

	res := clickhouse_connectivity.Probe{}.Run(context.Background(), env)
	if res.OK {
		t.Fatal("probe should fail when env.ClickHouse is nil")
	}
	if res.Err == nil {
		t.Fatal("expected Err to be populated on failure")
	}
}
