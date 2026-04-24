// Package rollback asserts the rollback primitive end-to-end:
// create a new deployment, wait for it to supersede the current
// live, roll back to the previous, verify the env-sticky hostname
// returns to the target. Exercises DeployService.Rollback plus the
// route-reassignment workflow inside RoutingService.
//
// The probe mutates live state. It runs last in DefaultSolo so
// earlier probes are not affected, and it leaves the environment
// back on its original deployment (plus one extra superseded row
// that the nightly GC cleans up).
package rollback

import (
	"context"
	"errors"
	"fmt"
	"time"

	"connectrpc.com/connect"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/httpx"
	"github.com/unkeyed/unkey/svc/preflight/core"
	"github.com/unkeyed/unkey/svc/preflight/internal/ctrlclient"
	"github.com/unkeyed/unkey/svc/preflight/internal/wait"
	"github.com/unkeyed/unkey/svc/preflight/probes"
)

const (
	// supersedeTimeout covers new-deployment build + ready + route
	// reassignment. Docker-source (nginx:latest) warm hits ~60s on
	// minikube; staging needs more runway for cold Depot pulls.
	supersedeTimeout = 8 * time.Minute

	// rollbackTimeout covers the Restate rollback workflow + route
	// flip. No build or pod reschedule involved, so 60s is generous.
	rollbackTimeout = 60 * time.Second

	// pollPeriod matches the other live-flip probes.
	pollPeriod = 5 * time.Second

	// probeImage is what CreateDeployment gets pointed at. Same image
	// the create_deployment probe uses; deliberate — we want to flip
	// route stickiness, not test image pulls.
	probeImage = "nginx:latest"
)

type Probe struct{}

func (Probe) Name() string { return "rollback" }

type metaResponse struct {
	UnkeyDeploymentID string `json:"unkey_deployment_id"`
}

func (Probe) Run(ctx context.Context, env *core.Env) core.Result {
	if err := validatePrereqs(env); err != nil {
		return core.Fail(err)
	}

	hostname := envStickyHostname(env)
	dims := map[string]string{"hostname": hostname}
	phases := make([]core.Phase, 0, 4)

	targetID, err := timed(&phases, "read_current_live", func() (string, error) {
		return readLiveDeploymentID(ctx, hostname)
	})
	if err != nil {
		return core.Failf("read current live via %s: %w", hostname, err).WithPhases(phases).WithDims(dims)
	}
	dims["target_id"] = targetID

	sourceID, err := timed(&phases, "create_source", func() (string, error) {
		return createDeployment(ctx, env)
	})
	if err != nil {
		return core.Failf("create supersede deployment: %w", err).WithPhases(phases).WithDims(dims)
	}
	dims["source_id"] = sourceID

	_, err = timed(&phases, "await_supersede", func() (struct{}, error) {
		return struct{}{}, pollForLive(ctx, hostname, sourceID, supersedeTimeout)
	})
	if err != nil {
		return core.Failf("source %s never became live: %w", sourceID, err).WithPhases(phases).WithDims(dims)
	}

	_, err = timed(&phases, "rollback_and_verify", func() (struct{}, error) {
		if rbErr := callRollback(ctx, env, sourceID, targetID); rbErr != nil {
			return struct{}{}, rbErr
		}
		return struct{}{}, pollForLive(ctx, hostname, targetID, rollbackTimeout)
	})
	if err != nil {
		return core.Failf("rollback to %s: %w", targetID, err).WithPhases(phases).WithDims(dims)
	}

	return core.Pass().WithPhases(phases).WithDims(dims)
}

func validatePrereqs(env *core.Env) error {
	switch {
	case env.CtrlBaseURL == "":
		return errors.New("env.CtrlBaseURL is empty")
	case env.CtrlAuthToken == "":
		return errors.New("env.CtrlAuthToken is empty")
	case env.PreflightProjectID == "" || env.PreflightAppID == "" || env.PreflightEnvironmentSlug == "":
		return errors.New("preflight project identifiers are unset")
	case env.PreflightProjectSlug == "", env.PreflightAppSlug == "",
		env.PreflightWorkspaceSlug == "", env.PreflightApex == "":
		return errors.New("preflight slug fields are incomplete")
	}
	return nil
}

// envStickyHostname mirrors ctrl's per-environment domain:
// <prefix>-<environment>-<workspace>.<apex>. The prefix omits the
// "default" app slug, same as every other route type.
func envStickyHostname(env *core.Env) string {
	prefix := env.PreflightProjectSlug
	if env.PreflightAppSlug != "default" {
		prefix = env.PreflightProjectSlug + "-" + env.PreflightAppSlug
	}
	return fmt.Sprintf("%s-%s-%s.%s",
		prefix,
		env.PreflightEnvironmentSlug,
		env.PreflightWorkspaceSlug,
		env.PreflightApex,
	)
}

// readLiveDeploymentID returns the UNKEY_DEPLOYMENT_ID the live
// hostname currently serves. Empty means something served /meta but
// Krane did not inject the env var, which is itself a failure.
func readLiveDeploymentID(ctx context.Context, hostname string) (string, error) {
	meta, err := httpx.Get[metaResponse](ctx, "https://"+hostname+"/meta")
	if err != nil {
		return "", err
	}
	if meta.UnkeyDeploymentID == "" {
		return "", errors.New("/meta returned empty UNKEY_DEPLOYMENT_ID")
	}
	return meta.UnkeyDeploymentID, nil
}

// createDeployment fires CreateDeployment with a docker source so we
// skip the build path and just exercise route reassignment.
func createDeployment(ctx context.Context, env *core.Env) (string, error) {
	client := ctrlclient.NewDeployClient(env)
	//nolint:exhaustruct // only a subset of CreateDeploymentRequest fields are meaningful here
	req := connect.NewRequest(&ctrlv1.CreateDeploymentRequest{
		ProjectId:       env.PreflightProjectID,
		AppId:           env.PreflightAppID,
		EnvironmentSlug: env.PreflightEnvironmentSlug,
		DockerImage:     probeImage,
	})
	req.Header().Set("Authorization", ctrlclient.AuthHeader(env))

	resp, err := client.CreateDeployment(ctx, req)
	if err != nil {
		return "", fmt.Errorf("CreateDeployment RPC: %w", err)
	}
	id := resp.Msg.GetDeploymentId()
	if id == "" {
		return "", errors.New("CreateDeployment returned empty deployment_id")
	}
	return id, nil
}

// callRollback invokes DeployService.Rollback. The server returns
// only after the Restate workflow finishes (routes already flipped).
func callRollback(ctx context.Context, env *core.Env, sourceID, targetID string) error {
	client := ctrlclient.NewDeployClient(env)
	req := connect.NewRequest(&ctrlv1.RollbackRequest{
		SourceDeploymentId: sourceID,
		TargetDeploymentId: targetID,
	})
	req.Header().Set("Authorization", ctrlclient.AuthHeader(env))

	if _, err := client.Rollback(ctx, req); err != nil {
		return fmt.Errorf("Rollback RPC: %w", err)
	}
	return nil
}

// pollForLive waits for /meta's UNKEY_DEPLOYMENT_ID to match wantID.
func pollForLive(ctx context.Context, hostname, wantID string, timeout time.Duration) error {
	pollCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	_, err := wait.Poll(pollCtx, pollPeriod, func(pctx context.Context) (struct{}, bool, error) {
		got, err := readLiveDeploymentID(pctx, hostname)
		if err != nil {
			return struct{}{}, false, nil
		}
		return struct{}{}, got == wantID, nil
	})
	return err
}

// timed runs fn and appends a Phase. Keeps the top-level Run readable
// by hiding the 3-line time/append/return dance each step would need.
func timed[T any](phases *[]core.Phase, name string, fn func() (T, error)) (T, error) {
	start := time.Now()
	v, err := fn()
	*phases = append(*phases, core.Phase{Name: name, Duration: time.Since(start), Err: err})
	return v, err
}

func init() { probes.Register(Probe{}) }
