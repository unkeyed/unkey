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

func (s *Service) PushSentinelState(ctx context.Context, req *connect.Request[ctrlv1.PushSentinelStateRequest]) (*connect.Response[ctrlv1.PushSentinelStateResponse], error) {

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
	case *ctrlv1.PushSentinelStateRequest_Upsert_:
		{

			health := db.SentinelsHealthUnknown
			if msg.Upsert.GetRunningReplicas() > 0 {
				health = db.SentinelsHealthHealthy
			}
			err = db.Query.UpdateSentinelReplicasAndHealth(ctx, s.db.RW(), db.UpdateSentinelReplicasAndHealthParams{
				K8sCrdName: msg.Upsert.GetK8SCrdName(),
				Replicas:   msg.Upsert.GetRunningReplicas(),
				Health:     health,
				UpdatedAt:  sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
			})
			if err != nil {
				if db.IsDuplicateKeyError(err) {
					// This is expected, cause kubernetes frequently replays existing sentinels
					return connect.NewResponse(&ctrlv1.PushSentinelStateResponse{}), nil
				}
				return nil, err
			}

		}

	case *ctrlv1.PushSentinelStateRequest_Delete_:
		{

			err = db.Query.UpdateSentinelReplicasAndHealth(ctx, s.db.RW(), db.UpdateSentinelReplicasAndHealthParams{
				K8sCrdName: msg.Delete.GetK8SCrdName(),
				Replicas:   0,
				Health:     db.SentinelsHealthPaused,
				UpdatedAt:  sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
			})
			if err != nil {
				return nil, err
			}
		}

	}

	return connect.NewResponse(&ctrlv1.PushSentinelStateResponse{}), nil

}
