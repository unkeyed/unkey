package cluster

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auth"
)

// ReportSentinelStatus records observed sentinel state from a krane agent and
// triggers NotifyReady when a pending deploy has converged.
//
// Health here means "is the sentinel serving traffic" (at least one pod
// available), not "did the rollout succeed". Rollout convergence is detected
// separately by comparing the reported running_image against the desired
// image. This lets frontline keep routing to a sentinel whose last deploy
// failed but whose old pods are still serving.
func (s *Service) ReportSentinelStatus(ctx context.Context, req *connect.Request[ctrlv1.ReportSentinelStatusRequest]) (*connect.Response[ctrlv1.ReportSentinelStatusResponse], error) {

	if err := auth.Authenticate(req, s.bearer); err != nil {
		return nil, err
	}

	region := req.Header().Get("X-Krane-Region")
	platform := req.Header().Get("X-Krane-Platform")

	if err := assert.All(
		assert.NotEmpty(region, "region is required"),
		assert.NotEmpty(platform, "platform is required"),
	); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	var health db.SentinelsHealth
	switch req.Msg.GetHealth() {
	case ctrlv1.Health_HEALTH_HEALTHY:
		health = db.SentinelsHealthHealthy
	case ctrlv1.Health_HEALTH_UNHEALTHY:
		health = db.SentinelsHealthUnhealthy
	case ctrlv1.Health_HEALTH_PAUSED:
		health = db.SentinelsHealthPaused
	case ctrlv1.Health_HEALTH_UNSPECIFIED:
		health = db.SentinelsHealthUnknown
	default:
		health = db.SentinelsHealthUnknown
	}

	err := db.Query.UpdateSentinelObservedState(ctx, s.db.RW(), db.UpdateSentinelObservedStateParams{
		K8sName:           req.Msg.GetK8SName(),
		AvailableReplicas: req.Msg.GetAvailableReplicas(),
		Health:            health,
		RunningImage:      req.Msg.GetRunningImage(),
		UpdatedAt:         sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// When a deploy is in progress and the desired image is actually running,
	// resolve the pending awakeable so Deploy can complete.
	if s.restate != nil {
		sentinel, err := db.Query.FindSentinelDeployContextByK8sName(ctx, s.db.RO(), req.Msg.GetK8SName())
		if errors.Is(err, sql.ErrNoRows) {
			// Sentinel row not found — nothing to notify. Krane may be reporting
			// on a sentinel that has just been deleted.
			return connect.NewResponse(&ctrlv1.ReportSentinelStatusResponse{}), nil
		}
		if err != nil {
			// Return the error so krane retries. UpdateSentinelObservedState
			// above is idempotent, so the retry is safe.
			return nil, connect.NewError(connect.CodeInternal, err)
		}
		// Convergence: deploy_status=progressing AND the observed state matches
		// the desired state on every dimension (image, replicas, health).
		// Previously this only checked image, which caused two real bugs:
		//   - Replica scale-ups flipped to ready prematurely on the first
		//     report (pre-scale), because RunningImage already matched.
		//   - Orphaned awakeables (e.g. Deploy worker killed between "mark
		//     progressing" and "create awakeable") left sentinels stuck at
		//     progressing until the 10-minute Deploy timeout.
		//
		// Fix: this handler is now the authoritative state-machine driver.
		// On convergence it flips deploy_status=ready directly AND fires
		// NotifyReady. The awakeable is now an optimization: if the Deploy
		// handler is alive, it resolves quickly; if it's dead, Deploy on
		// the next invocation reads deploy_status=ready and short-circuits
		// through its noConfigChange+Ready early-return.
		converged := sentinel.DeployStatus == db.SentinelsDeployStatusProgressing &&
			health == db.SentinelsHealthHealthy &&
			sentinel.RunningImage != "" &&
			sentinel.RunningImage == sentinel.DesiredImage &&
			req.Msg.GetAvailableReplicas() >= sentinel.DesiredReplicas

		if converged {
			if err := db.Query.UpdateSentinelDeployStatus(ctx, s.db.RW(), db.UpdateSentinelDeployStatusParams{
				ID:           sentinel.ID,
				DeployStatus: db.SentinelsDeployStatusReady,
				UpdatedAt:    sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
			}); err != nil {
				// Log and continue — krane will retry the whole report,
				// at which point we'll retry the flip. Don't fail the RPC
				// because the observed-state write above already succeeded.
				logger.Error("failed to flip sentinel to ready on convergence",
					"sentinel_id", sentinel.ID,
					"error", err,
				)
			}

			// Belt-and-suspenders: fire NotifyReady so a Deploy handler
			// currently parked on the awakeable resolves without waiting
			// for its next poll. ResolveAwakeable is idempotent on
			// already-resolved awakeables and a no-op when no awakeable
			// is stored.
			_, err := hydrav1.NewSentinelServiceIngressClient(s.restate, sentinel.ID).
				NotifyReady().
				Send(ctx, &hydrav1.SentinelServiceNotifyReadyRequest{})
			if err != nil {
				logger.Error("failed to notify sentinel ready", "sentinel_id", sentinel.ID, "error", err)
			}
		}
	}

	return connect.NewResponse(&ctrlv1.ReportSentinelStatusResponse{}), nil
}
