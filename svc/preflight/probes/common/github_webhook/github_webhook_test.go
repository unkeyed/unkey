package github_webhook_test

import (
	"context"
	"testing"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/svc/preflight/harness"
	"github.com/unkeyed/unkey/svc/preflight/probes/common/github_webhook"
)

// capturingService forwards every HandlePushRequest to a channel so the
// test can assert on what the ctrl API forwarded.
type capturingService struct {
	hydrav1.UnimplementedGitHubWebhookServiceServer
	requests chan *hydrav1.HandlePushRequest
}

func (c *capturingService) HandlePush(_ restate.ObjectContext, req *hydrav1.HandlePushRequest) (*hydrav1.HandlePushResponse, error) {
	c.requests <- req
	//nolint:exhaustruct // empty response body is what the real service returns
	return &hydrav1.HandlePushResponse{}, nil
}

func TestProbe_RunsAgainstHarness(t *testing.T) {
	// Harness tests are sequential by convention; see dev_test.go for
	// the reason.

	received := make(chan *hydrav1.HandlePushRequest, 1)
	mock := &capturingService{requests: received} //nolint:exhaustruct

	h := harness.Start(t, harness.Config{
		SeedPreflightProject: false,
		MockRestateServices: []restate.ServiceDefinition{
			hydrav1.NewGitHubWebhookServiceServer(mock),
		},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	res := github_webhook.Probe{}.Run(ctx, h.Env())

	if !res.OK {
		t.Fatalf("probe failed: err=%v, phases=%+v", res.Err, res.Phases)
	}
	if len(res.Phases) != 2 {
		t.Fatalf("expected 2 phases (accept_valid, reject_invalid), got %d", len(res.Phases))
	}
	if res.Phases[0].Name != "accept_valid" || res.Phases[1].Name != "reject_invalid" {
		t.Errorf("phase names wrong: %+v", res.Phases)
	}

	// The accept_valid phase must have resulted in a Restate invocation.
	// The reject_invalid phase must NOT have; the ctrl API's verifier
	// should have blocked it before forwarding.
	select {
	case req := <-received:
		if req.GetRepositoryFullName() != "unkeyed/preflight-test-app" {
			t.Errorf("forwarded request targets unexpected repo: %q", req.GetRepositoryFullName())
		}
		if req.GetBranch() != "preflight-test" {
			t.Errorf("forwarded request targets unexpected branch: %q", req.GetBranch())
		}
	case <-time.After(5 * time.Second):
		t.Fatal("expected HandlePush invocation from accept_valid phase; none arrived in 5s")
	}

	// Drain any lingering invocation to detect if reject_invalid leaked
	// through the verifier. A real leak is a tier-1 security finding.
	select {
	case extra := <-received:
		t.Fatalf("unexpected second HandlePush invocation; the invalid-signature phase should have been rejected by the ctrl API. Got: %+v", extra)
	case <-time.After(500 * time.Millisecond):
	}
}
