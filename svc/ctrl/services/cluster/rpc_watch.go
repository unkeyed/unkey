package cluster

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/proto"
)

func (s *Service) Watch(ctx context.Context, req *connect.Request[ctrlv1.WatchRequest], stream *connect.ServerStream[ctrlv1.State]) error {

	region := req.Msg.GetRegion()
	clusterID := req.Msg.GetClusterId()
	sequence := req.Msg.GetSequenceLastSeen()

	err := assert.All(
		assert.NotEmpty(region, "region must not be empty"),
		assert.NotEmpty(clusterID, "clusterID must not be empty"),
		assert.Greater(sequence, 0, "sequence must be greater than 0"),
	)
	if err != nil {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}

	s.logger.Info("watch request received",
		"region", region,
		"clusterID", clusterID,
		"sequence", sequence,
	)

	changes, err := db.Query.FindStateChangesByClusterAfterSequence(ctx, s.db.RW(), db.FindStateChangesByClusterAfterSequenceParams{
		ClusterID:     clusterID,
		AfterSequence: sequence,
	})
	if err != nil {
		return connect.NewError(connect.CodeInternal, err)
	}

	for _, change := range changes {

		msg := &ctrlv1.State{
			Sequence: change.Sequence,
			Kind:     nil,
		}

		switch change.ResourceType {
		case db.StateChangesResourceTypeSentinel:
			sentinel := &ctrlv1.SentinelState{}
			err = proto.Unmarshal(change.State, sentinel)
			if err != nil {
				return connect.NewError(connect.CodeInternal, err)

			}

			msg.Kind = &ctrlv1.State_Sentinel{
				Sentinel: sentinel,
			}
		case db.StateChangesResourceTypeDeployment:
			deployment := &ctrlv1.DeploymentState{}
			err = proto.Unmarshal(change.State, deployment)
			if err != nil {
				return connect.NewError(connect.CodeInternal, err)

			}

			msg.Kind = &ctrlv1.State_Deployment{
				Deployment: deployment,
			}
		default:
			return connect.NewError(connect.CodeInternal, fmt.Errorf("unexpected resource type %T", change.ResourceType))
		}

		err = stream.Send(msg)
		if err != nil {
			return connect.NewError(connect.CodeInternal, err)
		}

	}

	<-ctx.Done()
	return ctx.Err()

}
