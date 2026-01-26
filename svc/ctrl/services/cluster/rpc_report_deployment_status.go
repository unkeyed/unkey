package cluster

import (
	"context"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
)

// ReportDeploymentStatus reconciles the observed deployment state reported by a krane agent.
// This is the feedback loop for convergence: agents report what's actually running so the
// control plane can track instance health and detect drift from desired state.
//
// The request contains either an Update or Delete change. For Update, the method upserts
// all reported instances and garbage-collects any instances in the database that were not
// included in the report. For Delete, all instances for the deployment in that region are
// removed. Both operations run within a retryable transaction to handle transient database
// errors using [db.TxRetry].
//
// Instance status is mapped from proto values to database enums via [ctrlDeploymentStatusToDbStatus].
// Unspecified or unknown statuses default to inactive.
//
// Returns CodeUnauthenticated if bearer token is invalid. Database errors during the
// transaction are returned as-is (not wrapped in Connect error codes).
func (s *Service) ReportDeploymentStatus(ctx context.Context, req *connect.Request[ctrlv1.ReportDeploymentStatusRequest]) (*connect.Response[ctrlv1.ReportDeploymentStatusResponse], error) {
	s.logger.Info("reporting deployment status", "req", req.Msg)

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

	err = db.TxRetry(ctx, s.db.RW(), func(txCtx context.Context, tx db.DBTX) error {

		switch msg := req.Msg.GetChange().(type) {
		case *ctrlv1.ReportDeploymentStatusRequest_Update_:
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

				wantInstanceNames := map[string]*ctrlv1.ReportDeploymentStatusRequest_Update_Instance{}
				for _, instance := range msg.Update.GetInstances() {
					wantInstanceNames[instance.GetK8SName()] = instance
				}

				for _, staleInstance := range staleInstances {
					if _, ok := wantInstanceNames[staleInstance.K8sName]; !ok {
						err = db.Query.DeleteInstance(ctx, tx, db.DeleteInstanceParams{
							K8sName: staleInstance.K8sName,
							Region:  region,
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

		case *ctrlv1.ReportDeploymentStatusRequest_Delete_:
			{

				deployment, err := db.Query.FindDeploymentByK8sName(ctx, tx, msg.Delete.GetK8SName())
				if err != nil {
					return err
				}

				err = db.Query.DeleteDeploymentInstances(ctx, tx, db.DeleteDeploymentInstancesParams{
					DeploymentID: deployment.ID,
					Region:       region,
				})
				if err != nil {
					return err
				}
			}

		}
		return nil
	})

	return connect.NewResponse(&ctrlv1.ReportDeploymentStatusResponse{}), err

}

// ctrlDeploymentStatusToDbStatus maps proto instance status values to database enum values.
// STATUS_PENDING maps to InstancesStatusPending, STATUS_RUNNING to InstancesStatusRunning,
// STATUS_FAILED to InstancesStatusFailed. STATUS_UNSPECIFIED and any unknown values default
// to InstancesStatusInactive.
func ctrlDeploymentStatusToDbStatus(status ctrlv1.ReportDeploymentStatusRequest_Update_Instance_Status) db.InstancesStatus {
	switch status {
	case ctrlv1.ReportDeploymentStatusRequest_Update_Instance_STATUS_UNSPECIFIED:
		return db.InstancesStatusInactive
	case ctrlv1.ReportDeploymentStatusRequest_Update_Instance_STATUS_PENDING:
		return db.InstancesStatusPending
	case ctrlv1.ReportDeploymentStatusRequest_Update_Instance_STATUS_RUNNING:
		return db.InstancesStatusRunning
	case ctrlv1.ReportDeploymentStatusRequest_Update_Instance_STATUS_FAILED:
		return db.InstancesStatusFailed
	default:
		return db.InstancesStatusInactive
	}
}
