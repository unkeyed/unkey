// Package create_deployment implements tier-1.3 of the preflight plan:
// exercising the ctrl API's control-plane RPC surface for deploys.
// It creates a deployment with a docker-image source, confirms the
// response carries a non-empty ID in PENDING status, and round-trips
// the ID through GetDeployment.
//
// The probe does NOT wait for READY. That is unachievable in dev (no
// Krane / Depot) and covered by later tier-1 probes in staging/prod.
// This probe's job is the RPC surface itself: what a bad token, a
// ctrl API regression, or a malformed protobuf would break first.
package create_deployment

import (
	"context"
	"errors"
	"time"

	"connectrpc.com/connect"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/svc/preflight/core"
	"github.com/unkeyed/unkey/svc/preflight/internal/ctrlclient"
	"github.com/unkeyed/unkey/svc/preflight/probes"
)

// Probe calls CreateDeployment + GetDeployment. See package comment.
type Probe struct{}

// Name implements probes.Probe.
func (Probe) Name() string { return "create_deployment" }

// Run implements probes.Probe.
func (Probe) Run(ctx context.Context, env *core.Env) core.Result {
	if env.CtrlBaseURL == "" {
		return core.Fail(errors.New("CtrlBaseURL is empty"))
	}

	if env.CtrlAuthToken == "" {
		return core.Fail(errors.New("CtrlAuthToken is empty"))
	}

	if env.PreflightProjectID == "" || env.PreflightAppID == "" || env.PreflightEnvironmentSlug == "" {
		return core.Fail(errors.New(
			"preflight project identifiers are unset; check PREFLIGHT_PROJECT_ID / PREFLIGHT_APP_ID / PREFLIGHT_ENVIRONMENT_SLUG",
		))
	}

	client := ctrlclient.NewDeployClient(env)
	phases := make([]core.Phase, 0, 2)

	// Phase 1: CreateDeployment.
	createStart := time.Now()

	//nolint:exhaustruct // only a subset of CreateDeploymentRequest fields are meaningful for a docker-source preflight deploy
	createReq := connect.NewRequest(&ctrlv1.CreateDeploymentRequest{
		ProjectId:       env.PreflightProjectID,
		AppId:           env.PreflightAppID,
		EnvironmentSlug: env.PreflightEnvironmentSlug,
		DockerImage:     "nginx:latest",
	})
	createReq.Header().Set("Authorization", ctrlclient.AuthHeader(env))

	createResp, err := client.CreateDeployment(ctx, createReq)
	phases = append(phases, core.Phase{Name: "create", Duration: time.Since(createStart), Err: err})
	if err != nil {
		return core.Failf("CreateDeployment RPC: %w", err).WithPhases(phases)
	}

	deploymentID := createResp.Msg.GetDeploymentId()
	if deploymentID == "" {
		return core.Fail(errors.New("CreateDeployment returned empty deployment_id")).WithPhases(phases)
	}

	if createResp.Msg.GetStatus() != ctrlv1.DeploymentStatus_DEPLOYMENT_STATUS_PENDING {
		return core.Failf("CreateDeployment returned status %v; want PENDING", createResp.Msg.GetStatus()).
			WithPhases(phases).
			WithDims(map[string]string{"deployment_id": deploymentID})
	}

	// Phase 2: GetDeployment round-trip. Catches regressions where
	// CreateDeployment returns OK but the row is not queryable (happens
	// more often than you'd expect when indexes drift or RLS rules
	// mis-match).
	getStart := time.Now()
	//nolint:exhaustruct // GetDeploymentRequest only has DeploymentId
	getReq := connect.NewRequest(&ctrlv1.GetDeploymentRequest{DeploymentId: deploymentID})
	getReq.Header().Set("Authorization", ctrlclient.AuthHeader(env))

	getResp, err := client.GetDeployment(ctx, getReq)
	phases = append(phases, core.Phase{Name: "get", Duration: time.Since(getStart), Err: err})
	if err != nil {
		return core.Failf("GetDeployment RPC: %w", err).
			WithPhases(phases).
			WithDims(map[string]string{"deployment_id": deploymentID})
	}

	if got := getResp.Msg.GetDeployment().GetId(); got != deploymentID {
		return core.Failf("GetDeployment returned id %q; want %q", got, deploymentID).
			WithPhases(phases).
			WithDims(map[string]string{"deployment_id": deploymentID})
	}

	return core.Pass().
		WithPhases(phases).
		WithDims(map[string]string{"source_type": "docker"})
}

func init() { probes.Register(Probe{}) }
