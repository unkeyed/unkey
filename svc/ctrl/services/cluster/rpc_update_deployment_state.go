package cluster

import (
	"context"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
)

func (s *Service) UpdateDeploymentState(ctx context.Context, req *connect.Request[ctrlv1.UpdateDeploymentStateRequest]) (*connect.Response[ctrlv1.UpdateDeploymentStateResponse], error) {
	s.logger.Info("updating deployment state", "req", req.Msg)
	//"update:{k8s_name:\"pgeywtmuengq\" instances:{k8s_name:\"pgeywtmuengq-kdfvj\" address:\"192-168-194-33.uzapavou.pod.cluster.local\" status:STATUS_RUNNING}}"

	if err := s.authenticate(req); err != nil {
		return nil, err
	}
	region := req.Header().Get("X-Krane-Region")
	clusterID := req.Header().Get("X-Krane-Cluster-Id")

	err := assert.All(
		assert.NotEmpty(region, "region is required"),
		assert.NotEmpty(clusterID, "clusterID is required"),
	)
	if err != nil {
		return nil, err
	}

	err = db.TxRetry(ctx, s.db.RW(), func(txCtx context.Context, tx db.DBTX) error {

		switch msg := req.Msg.GetChange().(type) {
		case *ctrlv1.UpdateDeploymentStateRequest_Update_:
			{
				deployment, err := db.Query.FindDeploymentByK8sName(ctx, tx, msg.Update.GetK8SName())
				if err != nil {
					return err
				}

				staleInstances, err := db.Query.FindInstancesByDeploymentIdAndRegion(ctx, tx, db.FindInstancesByDeploymentIdAndRegionParams{
					Deploymentid: deployment.ID,
					Region:       region,
				})
				if err != nil {
					return err
				}

				wantInstanceNames := map[string]*ctrlv1.UpdateDeploymentStateRequest_Update_Instance{}
				for _, instance := range msg.Update.GetInstances() {
					wantInstanceNames[instance.GetK8SName()] = instance
				}

				for _, staleInstance := range staleInstances {
					if _, ok := wantInstanceNames[staleInstance.K8sName]; !ok {
						err = db.Query.DeleteInstance(ctx, tx, db.DeleteInstanceParams{
							K8sName:   staleInstance.K8sName,
							Region:    region,
							ClusterID: clusterID,
						})
						if err != nil {
							return err
						}
					}
				}

				for _, instance := range msg.Update.GetInstances() {
					err = db.Query.UpsertInstance(ctx, tx, db.UpsertInstanceParams{
						ID:            uid.New(uid.InstancePrefix),
						DeploymentID:  deployment.ID,
						WorkspaceID:   deployment.WorkspaceID,
						ProjectID:     deployment.ProjectID,
						Region:        region,
						ClusterID:     clusterID,
						K8sName:       instance.GetK8SName(),
						Address:       instance.GetAddress(),
						CpuMillicores: int32(instance.GetCpuMillicores()),
						MemoryMib:     int32(instance.GetMemoryMib()),
						Status:        ctrlDeploymentStatusToDbStatus(instance.GetStatus()),
					})
					if err != nil {
						return err
					}
				}
			}

		case *ctrlv1.UpdateDeploymentStateRequest_Delete_:
			{

				deployment, err := db.Query.FindDeploymentByK8sName(ctx, tx, msg.Delete.GetK8SName())
				if err != nil {
					return err
				}

				err = db.Query.DeleteDeploymentInstances(ctx, tx, db.DeleteDeploymentInstancesParams{
					DeploymentID: deployment.ID,
					ClusterID:    clusterID,
				})
				if err != nil {
					return err
				}
			}

		}
		return nil
	})

	return connect.NewResponse(&ctrlv1.UpdateDeploymentStateResponse{}), err

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
