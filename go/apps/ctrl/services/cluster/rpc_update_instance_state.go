package cluster

import (
	"context"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/pkg/assert"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func (s *Service) UpdateInstanceState(ctx context.Context, req *connect.Request[ctrlv1.UpdateInstanceStateRequest]) (*connect.Response[ctrlv1.UpdateInstanceStateResponse], error) {

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
	case *ctrlv1.UpdateInstanceStateRequest_Upsert_:
		{
			deployment, err := db.Query.FindDeploymentById(ctx, s.db.RO(), msg.Upsert.GetDeploymentId())
			if err != nil {
				return nil, err
			}

			err = db.Query.UpsertInstance(ctx, s.db.RW(), db.UpsertInstanceParams{
				ID:            uid.New(uid.InstancePrefix),
				DeploymentID:  deployment.ID,
				WorkspaceID:   deployment.WorkspaceID,
				ProjectID:     deployment.ProjectID,
				Region:        region,
				Shard:         shard,
				K8sName:       msg.Upsert.GetK8SName(),
				Address:       msg.Upsert.GetAddress(),
				CpuMillicores: int32(msg.Upsert.GetCpuMillicores()),
				MemoryMib:     int32(msg.Upsert.GetMemoryMib()),
				Status:        ctrlStatusToDbStatus(msg.Upsert.GetStatus()),
			})
			if err != nil {
				return nil, err
			}

		}

	case *ctrlv1.UpdateInstanceStateRequest_Delete_:
		{

			err = db.Query.DeleteInstance(ctx, s.db.RW(), db.DeleteInstanceParams{
				K8sName: msg.Delete.GetK8SName(),
				Region:  region,
				Shard:   shard,
			})
			if err != nil {
				return nil, err
			}
		}

	}

	return connect.NewResponse(&ctrlv1.UpdateInstanceStateResponse{}), nil

}

func ctrlStatusToDbStatus(status ctrlv1.UpdateInstanceStateRequest_Status) db.InstancesStatus {
	switch status {
	case ctrlv1.UpdateInstanceStateRequest_STATUS_UNSPECIFIED:
		return db.InstancesStatusInactive
	case ctrlv1.UpdateInstanceStateRequest_STATUS_PENDING:
		return db.InstancesStatusPending
	case ctrlv1.UpdateInstanceStateRequest_STATUS_RUNNING:
		return db.InstancesStatusRunning
	case ctrlv1.UpdateInstanceStateRequest_STATUS_FAILED:
		return db.InstancesStatusFailed
	default:
		return db.InstancesStatusInactive
	}
}
