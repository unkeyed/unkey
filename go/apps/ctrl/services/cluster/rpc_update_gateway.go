package cluster

import (
	"context"
	"database/sql"
	"time"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/pkg/assert"
	"github.com/unkeyed/unkey/go/pkg/db"
)

func (s *Service) UpdateGateway(ctx context.Context, req *connect.Request[ctrlv1.UpdateGatewayRequest]) (*connect.Response[ctrlv1.UpdateGatewayResponse], error) {

	if err := s.authenticate(req); err != nil {
		return nil, err
	}

	region := req.Header().Get("X-Krane-Region")
	shard := req.Header().Get("X-Krane-Shard")

	err := assert.All(
		assert.NotEmpty(region, "region is required"),
		assert.NotEmpty(shard, "shard is required"),
	)
	if err != nil {
		return nil, err
	}

	switch msg := req.Msg.GetChange().(type) {
	case *ctrlv1.UpdateGatewayRequest_Create_:
		{

			health := db.GatewaysHealthUnknown
			if msg.Create.GetRunningReplicas() > 0 {
				health = db.GatewaysHealthHealthy
			}
			err = db.Query.UpdateGatewayReplicasAndHealth(ctx, s.db.RW(), db.UpdateGatewayReplicasAndHealthParams{
				ID:        msg.Create.GetGatewayId(),
				Replicas:  msg.Create.GetRunningReplicas(),
				Health:    health,
				UpdatedAt: sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
			})
			if err != nil {
				if db.IsDuplicateKeyError(err) {
					// This is expected, cause kubernetes frequently replays existing gateways
					return connect.NewResponse(&ctrlv1.UpdateGatewayResponse{}), nil
				}
				return nil, err
			}

		}
	case *ctrlv1.UpdateGatewayRequest_Update_:
		{

			health := db.GatewaysHealthUnknown
			if msg.Update.GetRunningReplicas() > 0 {
				health = db.GatewaysHealthHealthy
			}
			err = db.Query.UpdateGatewayReplicasAndHealth(ctx, s.db.RW(), db.UpdateGatewayReplicasAndHealthParams{
				ID:        msg.Update.GetGatewayId(),
				Replicas:  msg.Update.GetRunningReplicas(),
				Health:    health,
				UpdatedAt: sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
			})
			if err != nil {
				return nil, err
			}

		}
	case *ctrlv1.UpdateGatewayRequest_Delete_:
		{

			err = db.Query.UpdateGatewayReplicasAndHealth(ctx, s.db.RW(), db.UpdateGatewayReplicasAndHealthParams{
				ID:        msg.Delete.GetGatewayId(),
				Replicas:  0,
				Health:    db.GatewaysHealthPaused,
				UpdatedAt: sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
			})
			if err != nil {
				return nil, err
			}
		}

	}

	return connect.NewResponse(&ctrlv1.UpdateGatewayResponse{}), nil

}
