package runner

import (
	"context"
	"errors"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/unkeyed/unkey/svc/preflight/core"
)

// stubProbe is a minimal probe for exercising the Runner without
// pulling in a real probe package. Keeps the test self-contained.
type stubProbe struct {
	name      string
	result    core.Result
	callCount int
}

func (s *stubProbe) Name() string { return s.name }

func (s *stubProbe) Run(_ context.Context, _ *core.Env) core.Result {
	s.callCount++
	return s.result
}

// recordingUploader captures Upload calls so tests can assert the
// Runner invokes it with the right args on failure and skips it on success.
type recordingUploader struct {
	calls []uploadCall
}

type uploadCall struct {
	runID     string
	probeName string
	artifacts []core.Artifact
}

func (r *recordingUploader) Upload(_ context.Context, runID, probeName string, artifacts []core.Artifact) error {
	r.calls = append(r.calls, uploadCall{runID: runID, probeName: probeName, artifacts: artifacts})
	return nil
}

func TestInvoke_Success_EmitsSuccessCounter(t *testing.T) {
	probe := &stubProbe{name: "stub_ok", result: core.Pass()}
	uploader := &recordingUploader{}
	r := New("solo", "us-east-1", uploader, nil)

	env := &core.Env{Target: core.TargetDev, Region: "us-east-1", RunID: "pflt_test"}

	res := r.Invoke(context.Background(), probe, env)

	if !res.OK {
		t.Fatalf("expected OK result, got %+v", res)
	}
	if probe.callCount != 1 {
		t.Fatalf("expected probe to be invoked once, got %d", probe.callCount)
	}
	if got := testutil.ToFloat64(runTotal.WithLabelValues("solo", "stub_ok", "success", "us-east-1")); got != 1 {
		t.Fatalf("expected preflight_run_total{result=success}=1, got %v", got)
	}
	if len(uploader.calls) != 0 {
		t.Fatalf("expected no artifact uploads on success, got %d", len(uploader.calls))
	}
}

func TestInvoke_Failure_EmitsFailCounterAndUploads(t *testing.T) {
	artifact := core.Artifact{Name: "stub.txt", ContentType: "text/plain", Body: []byte("diag")}
	probe := &stubProbe{
		name:   "stub_fail",
		result: core.Fail(errors.New("boom")).WithArtifacts([]core.Artifact{artifact}),
	}
	uploader := &recordingUploader{}
	r := New("solo", "us-east-1", uploader, nil)

	env := &core.Env{Target: core.TargetDev, Region: "us-east-1", RunID: "pflt_fail"}
	_ = r.Invoke(context.Background(), probe, env)

	if got := testutil.ToFloat64(runTotal.WithLabelValues("solo", "stub_fail", "fail", "us-east-1")); got != 1 {
		t.Fatalf("expected preflight_run_total{result=fail}=1, got %v", got)
	}
	if len(uploader.calls) != 1 {
		t.Fatalf("expected 1 artifact upload, got %d", len(uploader.calls))
	}
	call := uploader.calls[0]
	if call.runID != "pflt_fail" {
		t.Errorf("expected runID=pflt_fail, got %q", call.runID)
	}
	if call.probeName != "stub_fail" {
		t.Errorf("expected probeName=stub_fail, got %q", call.probeName)
	}
	if len(call.artifacts) != 1 || call.artifacts[0].Name != "stub.txt" {
		t.Errorf("expected 1 artifact named stub.txt, got %+v", call.artifacts)
	}
}

func TestInvoke_ShadowFailure_UsesShadowLabel(t *testing.T) {
	probe := &stubProbe{
		name:   "stub_shadow",
		result: core.Fail(errors.New("boom")),
	}
	r := New("solo", "us-east-1", nil, []string{"stub_shadow"})

	env := &core.Env{Target: core.TargetDev, Region: "us-east-1", RunID: "pflt_shadow"}
	_ = r.Invoke(context.Background(), probe, env)

	if got := testutil.ToFloat64(runTotal.WithLabelValues("solo", "stub_shadow", "shadow_fail", "us-east-1")); got != 1 {
		t.Fatalf("expected preflight_run_total{result=shadow_fail}=1, got %v", got)
	}
	if got := testutil.ToFloat64(runTotal.WithLabelValues("solo", "stub_shadow", "fail", "us-east-1")); got != 0 {
		t.Fatalf("shadowed probe should not emit result=fail, got %v", got)
	}
}

func TestInvoke_EmitsPhaseBuckets(t *testing.T) {
	probe := &stubProbe{
		name: "stub_phases",
		result: core.Pass().WithPhases([]core.Phase{
			{Name: "create", Duration: 10},
			{Name: "poll", Duration: 20},
		}),
	}
	r := New("solo", "us-east-1", nil, nil)
	env := &core.Env{Target: core.TargetDev, Region: "us-east-1", RunID: "pflt_phase"}
	_ = r.Invoke(context.Background(), probe, env)

	if got := testutil.CollectAndCount(phaseDuration); got < 2 {
		t.Fatalf("expected at least 2 phase histograms after probe emitted 2 phases, got %d", got)
	}
}
