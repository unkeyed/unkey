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
	"github.com/unkeyed/unkey/svc/ctrl/pkg/metrics"
)

// ReportSentinelStatus records observed sentinel state from a krane agent and
// triggers NotifyReady when a pending deploy has converged.
//
// Health here means "is the sentinel serving traffic" (at least one pod
// available), not "did the rollout succeed". Rollout convergence is detected
// separately by comparing the reported running_image against the desired
// image. This lets frontline keep routing to a sentinel whose last deploy
// failed but whose old pods are still serving.
func (s *Service) ReportSentinelStatus(ctx context.Context, req *connect.Request[ctrlv1.ReportSentinelStatusRequest]) (response *connect.Response[ctrlv1.ReportSentinelStatusResponse], retErr error) {
	start := time.Now()
	defer func() {
		elapsed := time.Since(start)
		result := "success"
		if retErr != nil {
			result = "error"
		}
		metrics.ReportSentinelStatusDurationSeconds.WithLabelValues(result).Observe(elapsed.Seconds())
		logger.Info("report sentinel status: handled",
			"k8s_name", req.Msg.GetK8SName(),
			"duration_ms", elapsed.Milliseconds(),
			"result", result,
		)
	}()

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

	s.maybeNotifySentinelReady(ctx, req.Msg, health)

	return connect.NewResponse(&ctrlv1.ReportSentinelStatusResponse{}), nil
}

// maybeNotifySentinelReady evaluates whether the reported observed state
// means the pending deploy has converged, and if so flips deploy_status to
// ready and fires NotifyReady on the Sentinel workflow. Best-effort: errors
// are logged but not returned, mirroring ReportDeploymentStatus's
// maybeNotifyInstancesReady.
func (s *Service) maybeNotifySentinelReady(ctx context.Context, req *ctrlv1.ReportSentinelStatusRequest, health db.SentinelsHealth) {
	if s.restate == nil {
		metrics.NotifySentinelReadyTotal.WithLabelValues("restate_disabled").Inc()
		return
	}

	// RW avoids misclassifying as not_converged under replica lag: we just
	// wrote observed state to RW a few lines up, and the convergence check
	// depends on those fields (running_image, deploy_status).
	sentinel, err := db.Query.FindSentinelDeployContextByK8sName(ctx, s.db.RW(), req.GetK8SName())
	if errors.Is(err, sql.ErrNoRows) {
		// Sentinel row not found — nothing to notify. Krane may be reporting
		// on a sentinel that has just been deleted.
		metrics.NotifySentinelReadyTotal.WithLabelValues("notfound").Inc()
		logger.Info("notify sentinel ready: skipped",
			"k8s_name", req.GetK8SName(),
			"outcome", "notfound",
		)
		return
	}
	if err != nil {
		metrics.NotifySentinelReadyTotal.WithLabelValues("lookup_error").Inc()
		logger.Error("notify sentinel ready: lookup failed",
			"k8s_name", req.GetK8SName(),
			"outcome", "lookup_error",
			"error", err,
		)
		return
	}

	if !sentinelConverged(sentinel, health, req.GetAvailableReplicas()) {
		metrics.NotifySentinelReadyTotal.WithLabelValues("not_converged").Inc()
		logger.Info("notify sentinel ready: not converged",
			"sentinel_id", sentinel.ID,
			"k8s_name", req.GetK8SName(),
			"outcome", "not_converged",
			"deploy_status", sentinel.DeployStatus,
			"health", health,
			"running_image", sentinel.RunningImage,
			"desired_image", sentinel.DesiredImage,
			"available_replicas", req.GetAvailableReplicas(),
			"desired_replicas", sentinel.DesiredReplicas,
		)
		return
	}

	// This handler is the authoritative state-machine driver on convergence:
	// flip deploy_status=ready directly AND fire NotifyReady. The awakeable
	// is an optimization: if the Deploy handler is alive it resolves quickly,
	// and if dead the next Deploy invocation reads deploy_status=ready and
	// short-circuits through its noConfigChange+Ready early-return.
	//
	// The flip is a CAS (only applies when current status is still
	// 'progressing') so a concurrent writer — e.g. the Deploy worker marking
	// 'failed' on timeout — wins deterministically and we don't overwrite
	// its terminal status with 'ready' based on our stale read above.
	affected, err := db.Query.FlipSentinelDeployStatusIfProgressing(ctx, s.db.RW(), db.FlipSentinelDeployStatusIfProgressingParams{
		ID:           sentinel.ID,
		DeployStatus: db.SentinelsDeployStatusReady,
		UpdatedAt:    sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
	})
	if err != nil {
		// Log and continue — krane will retry the whole report, at which
		// point we'll retry the flip. The observed-state write already
		// succeeded, so don't fail the RPC on this.
		metrics.NotifySentinelReadyTotal.WithLabelValues("flip_error").Inc()
		logger.Error("notify sentinel ready: flip to ready failed",
			"sentinel_id", sentinel.ID,
			"outcome", "flip_error",
			"error", err,
		)
		return
	}
	if affected == 0 {
		// Someone else already moved the sentinel out of 'progressing'
		// (Deploy worker timeout, rollback, delete). Skip NotifyReady —
		// whoever flipped it is responsible for the terminal side effects.
		metrics.NotifySentinelReadyTotal.WithLabelValues("flip_superseded").Inc()
		logger.Info("notify sentinel ready: flip superseded",
			"sentinel_id", sentinel.ID,
			"outcome", "flip_superseded",
		)
		return
	}

	// Belt-and-suspenders: fire NotifyReady so a Deploy handler currently
	// parked on the awakeable resolves without waiting for its next poll.
	// ResolveAwakeable is idempotent on already-resolved awakeables and a
	// no-op when no awakeable is stored.
	if _, err := hydrav1.NewSentinelServiceIngressClient(s.restate, sentinel.ID).
		NotifyReady().
		Send(ctx, &hydrav1.SentinelServiceNotifyReadyRequest{}); err != nil {
		metrics.NotifySentinelReadyTotal.WithLabelValues("restate_error").Inc()
		logger.Error("notify sentinel ready: restate call failed",
			"sentinel_id", sentinel.ID,
			"outcome", "restate_error",
			"error", err,
		)
		return
	}

	metrics.NotifySentinelReadyTotal.WithLabelValues("notified").Inc()
	logger.Info("notify sentinel ready: sent",
		"sentinel_id", sentinel.ID,
		"outcome", "notified",
	)
}

// sentinelConverged reports whether the sentinel's observed state matches
// desired on every dimension that gates a deploy completion: deploy_status
// must be progressing, health must be healthy, the running image must be
// set and equal to the desired image, and the reported available replica
// count must meet or exceed the desired replica count.
//
// Previously this check only compared images, which caused two bugs:
//   - Replica scale-ups flipped to ready prematurely on the first report
//     (pre-scale), because RunningImage already matched.
//   - Orphaned awakeables (e.g. Deploy worker killed between "mark
//     progressing" and "create awakeable") left sentinels stuck at
//     progressing until the 10-minute Deploy timeout.
func sentinelConverged(sentinel db.FindSentinelDeployContextByK8sNameRow, health db.SentinelsHealth, availableReplicas int32) bool {
	return sentinel.DeployStatus == db.SentinelsDeployStatusProgressing &&
		health == db.SentinelsHealthHealthy &&
		sentinel.RunningImage != "" &&
		sentinel.RunningImage == sentinel.DesiredImage &&
		availableReplicas >= sentinel.DesiredReplicas
}
