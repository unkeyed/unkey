package cluster

import (
	"context"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/pkg/assert"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func (s *Service) UpdateInstance(ctx context.Context, req *connect.Request[ctrlv1.UpdateInstanceRequest]) (*connect.Response[ctrlv1.UpdateInstanceResponse], error) {

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
	case *ctrlv1.UpdateInstanceRequest_Create_:
		{
			deployment, err := db.Query.FindDeploymentById(ctx, s.db.RO(), msg.Create.GetDeploymentId())
			if err != nil {
				return nil, err
			}

			err = db.Query.InsertInstance(ctx, s.db.RW(), db.InsertInstanceParams{
				ID:            uid.New(uid.InstancePrefix),
				DeploymentID:  deployment.ID,
				WorkspaceID:   deployment.WorkspaceID,
				ProjectID:     deployment.ProjectID,
				Region:        region,
				Shard:         shard,
				PodName:       msg.Create.GetPodName(),
				Address:       msg.Create.GetAddress(),
				CpuMillicores: msg.Create.GetCpuMillicores(),
				MemoryMib:     msg.Create.GetMemoryMib(),
				Status:        db.InstancesStatusPending,
			})
			if err != nil {
				if db.IsDuplicateKeyError(err) {
					// This is expected, cause kubernetes frequently replays existing instances
					return connect.NewResponse(&ctrlv1.UpdateInstanceResponse{}), nil
				}
				return nil, err
			}

		}
	case *ctrlv1.UpdateInstanceRequest_Update_:
		{

			err = db.Query.UpdateInstanceStatus(ctx, s.db.RW(), db.UpdateInstanceStatusParams{
				PodName: msg.Update.GetPodName(),
				Region:  region,
				Shard:   shard,
				Status:  ctrlStatusToDbStatus(msg.Update.GetStatus()),
			})
			if err != nil {
				return nil, err
			}

		}
	case *ctrlv1.UpdateInstanceRequest_Delete_:
		{

			err = db.Query.DeleteInstance(ctx, s.db.RW(), db.DeleteInstanceParams{
				PodName: msg.Delete.GetPodName(),
				Region:  region,
				Shard:   shard,
			})
			if err != nil {
				return nil, err
			}
		}

	}

	return connect.NewResponse(&ctrlv1.UpdateInstanceResponse{}), nil

}

func ctrlStatusToDbStatus(status ctrlv1.UpdateInstanceRequest_Status) db.InstancesStatus {
	switch status {
	case ctrlv1.UpdateInstanceRequest_STATUS_UNSPECIFIED:
		return db.InstancesStatusInactive
	case ctrlv1.UpdateInstanceRequest_STATUS_PENDING:
		return db.InstancesStatusPending
	case ctrlv1.UpdateInstanceRequest_STATUS_RUNNING:
		return db.InstancesStatusRunning
	case ctrlv1.UpdateInstanceRequest_STATUS_FAILED:
		return db.InstancesStatusFailed
	default:
		return db.InstancesStatusInactive
	}
}
