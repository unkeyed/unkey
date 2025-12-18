package cluster

import (
	"context"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/pkg/assert"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func (s *Service) UpdateDeploymentState(ctx context.Context, req *connect.Request[ctrlv1.UpdateDeploymentStateRequest]) (*connect.Response[ctrlv1.UpdateDeploymentStateResponse], error) {

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
	case *ctrlv1.UpdateDeploymentStateRequest_Update_:
		{
			deployment, err := db.Query.FindDeploymentByK8sName(ctx, s.db.RO(), msg.Update.GetK8SName())
			if err != nil {
				return nil, err
			}

			staleInstances, err := db.Query.FindInstancesByDeploymentIdAndRegion(ctx, s.db.RO(), db.FindInstancesByDeploymentIdAndRegionParams{
				Deploymentid: deployment.ID,
				Region:       region,
			})
			if err != nil {
				return nil, err
			}

			wantInstanceNames := map[string]*ctrlv1.UpdateDeploymentStateRequest_Update_Instance{}
			for _, instance := range msg.Update.GetInstances() {
				wantInstanceNames[instance.GetK8SName()] = instance
			}

			for _, staleInstance := range staleInstances {
				if _, ok := wantInstanceNames[staleInstance.K8sName]; !ok {
					err = db.Query.DeleteInstance(ctx, s.db.RW(), db.DeleteInstanceParams{
						K8sName: staleInstance.K8sName,
						Region:  region,
						Shard:   shard,
					})
					if err != nil {
						return nil, err
					}
				}
			}

			for _, instance := range msg.Update.GetInstances() {
				err = db.Query.UpsertInstance(ctx, s.db.RW(), db.UpsertInstanceParams{
					ID:            uid.New(uid.InstancePrefix),
					DeploymentID:  deployment.ID,
					WorkspaceID:   deployment.WorkspaceID,
					ProjectID:     deployment.ProjectID,
					Region:        region,
					Shard:         shard,
					K8sName:       instance.GetK8SName(),
					Address:       instance.GetAddress(),
					CpuMillicores: int32(instance.GetCpuMillicores()),
					MemoryMib:     int32(instance.GetMemoryMib()),
					Status:        ctrlDeploymentStatusToDbStatus(instance.GetStatus()),
				})
				if err != nil {
					return nil, err
				}
			}
		}

	case *ctrlv1.UpdateDeploymentStateRequest_Delete_:
		{

			deployment, err := db.Query.FindDeploymentByK8sName(ctx, s.db.RO(), msg.Delete.GetK8SName())
			if err != nil {
				return nil, err
			}

			err = db.Query.DeleteDeploymentInstances(ctx, s.db.RW(), db.DeleteDeploymentInstancesParams{
				DeploymentID: deployment.ID,
				Region:       region,
				Shard:        shard,
			})
			if err != nil {
				return nil, err
			}
		}

	}

	return connect.NewResponse(&ctrlv1.UpdateDeploymentStateResponse{}), nil

}

func ctrlDeploymentStatusToDbStatus(status ctrlv1.UpdateDeploymentStateRequest_Update_Instance_Status) db.InstancesStatus {
	switch status {
	case ctrlv1.UpdateDeploymentStateRequest_Update_Instance_STATUS_UNSPECIFIED:
		return db.InstancesStatusInactive
	case ctrlv1.UpdateDeploymentStateRequest_Update_Instance_STATUS_PENDING:
		return db.InstancesStatusPending
	case ctrlv1.UpdateDeploymentStateRequest_Update_Instance_STATUS_RUNNING:
		return db.InstancesStatusRunning
	case ctrlv1.UpdateDeploymentStateRequest_Update_Instance_STATUS_FAILED:
		return db.InstancesStatusFailed
	default:
		return db.InstancesStatusInactive
	}
}
