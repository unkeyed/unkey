package cluster

import (
	"context"
	"database/sql"
	"time"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/db"
)

func (s *Service) UpdateSentinelState(ctx context.Context, req *connect.Request[ctrlv1.UpdateSentinelStateRequest]) (*connect.Response[ctrlv1.UpdateSentinelStateResponse], error) {

	if err := s.authenticate(req); err != nil {
		return nil, err
	}

	region := req.Header().Get("X-Krane-Region")

	err := assert.All(
		assert.NotEmpty(region, "region is required"),
	)
	if err != nil {
		return nil, err
	}

	health := db.SentinelsHealthUnknown
	switch req.Msg.GetHealth() {
	case ctrlv1.Health_HEALTH_HEALTHY:
		health = db.SentinelsHealthHealthy
	case ctrlv1.Health_HEALTH_UNHEALTHY:
		health = db.SentinelsHealthUnhealthy
	case ctrlv1.Health_HEALTH_PAUSED:
		health = db.SentinelsHealthPaused
	default:
		health = db.SentinelsHealthUnknown
	}
	err = db.Query.UpdateSentinelAvailableReplicasAndHealth(ctx, s.db.RW(), db.UpdateSentinelAvailableReplicasAndHealthParams{
		K8sName:           req.Msg.GetK8SName(),
		AvailableReplicas: req.Msg.GetAvailableReplicas(),
		Health:            health,
		UpdatedAt:         sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&ctrlv1.UpdateSentinelStateResponse{}), nil

}
