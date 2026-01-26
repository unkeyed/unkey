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

// ReportSentinelStatus records the observed replica count and health for a sentinel as
// reported by a krane agent. This updates the available_replicas, health, and updated_at
// fields in the database, enabling the control plane to track which sentinels are actually
// running and their current health state.
//
// The health proto value is mapped to database enums: HEALTH_HEALTHY to SentinelsHealthHealthy,
// HEALTH_UNHEALTHY to SentinelsHealthUnhealthy, HEALTH_PAUSED to SentinelsHealthPaused, and
// HEALTH_UNSPECIFIED to SentinelsHealthUnknown.
//
// Returns CodeUnauthenticated if bearer token is invalid, or CodeInternal if the database
// update fails.
func (s *Service) ReportSentinelStatus(ctx context.Context, req *connect.Request[ctrlv1.ReportSentinelStatusRequest]) (*connect.Response[ctrlv1.ReportSentinelStatusResponse], error) {

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

	return connect.NewResponse(&ctrlv1.ReportSentinelStatusResponse{}), nil

}
