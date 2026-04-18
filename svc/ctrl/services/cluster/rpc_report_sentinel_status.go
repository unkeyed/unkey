package cluster

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
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

	if err := validateRegionKey(req.Msg.GetRegion()); err != nil {
		return nil, err
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
		if sentinel.DeployStatus == db.SentinelsDeployStatusProgressing &&
			health == db.SentinelsHealthHealthy &&
			sentinel.RunningImage != "" &&
			sentinel.RunningImage == sentinel.DesiredImage {
			if s.notifiedReady.AddIfAbsent(sentinel.ID) {
				_, err := hydrav1.NewSentinelServiceIngressClient(s.restate, sentinel.ID).
					NotifyReady().
					Send(ctx, &hydrav1.SentinelServiceNotifyReadyRequest{})
				if err != nil {
					logger.Error("failed to notify sentinel ready", "sentinel_id", sentinel.ID, "error", err)
				}
			}
		}
	}

	return connect.NewResponse(&ctrlv1.ReportSentinelStatusResponse{}), nil
}
