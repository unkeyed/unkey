package cluster

import (
	"context"
	"database/sql"
	"time"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auth"
)

// deploymentActiveStatuses are the non-terminal statuses where a Deploy
// handler may be parked on the instances-ready awakeable. If the deployment
// is outside this set (ready, failed, cancelled, superseded, skipped,
// stopped, awaiting_approval), there's nothing to notify.
var deploymentActiveStatuses = map[db.DeploymentsStatus]bool{
	db.DeploymentsStatusStarting:   true,
	db.DeploymentsStatusBuilding:   true,
	db.DeploymentsStatusDeploying:  true,
	db.DeploymentsStatusNetwork:    true,
	db.DeploymentsStatusFinalizing: true,
}

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
	logger.Info("reporting deployment status", "req", req.Msg)

	if err := auth.Authenticate(req, s.bearer); err != nil {
		return nil, err
	}
	regionName := req.Header().Get("X-Krane-Region")
	platform := req.Header().Get("X-Krane-Platform")

	if err := assert.All(
		assert.NotEmpty(regionName, "region is required"),
		assert.NotEmpty(platform, "platform is required"),
	); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	// TODO: cache this lookup to avoid hitting the database on every status report
	region, err := db.Query.FindRegionByPlatformAndName(ctx, s.db.RO(), db.FindRegionByPlatformAndNameParams{
		Platform: platform,
		Name:     regionName,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	// Captured from the Update transaction so we can call NotifyInstancesReady
	// on the Deploy workflow after the tx commits (and the new state is
	// visible to the health-check query).
	var updatedDeployment *db.Deployment

	err = db.TxRetry(ctx, s.db.RW(), func(txCtx context.Context, tx db.DBTX) error {
		switch msg := req.Msg.GetChange().(type) {
		case *ctrlv1.ReportDeploymentStatusRequest_Update_:
			{
				deployment, err := db.Query.FindDeploymentByK8sName(ctx, tx, msg.Update.GetK8SName())
				if err != nil {
					return err
				}
				updatedDeployment = &deployment

				staleInstances, err := db.Query.FindInstancesByDeploymentIdAndRegionID(ctx, tx, db.FindInstancesByDeploymentIdAndRegionIDParams{
					DeploymentID: deployment.ID,
					RegionID:     region.ID,
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
							K8sName:  staleInstance.K8sName,
							RegionID: region.ID,
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
						AppID:         deployment.AppID,
						RegionID:      region.ID,
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

				if err := db.Query.DeleteDeploymentInstances(ctx, tx, db.DeleteDeploymentInstancesParams{
					DeploymentID: deployment.ID,
					RegionID:     region.ID,
				}); err != nil {
					return err
				}

				if deployment.DesiredState == db.DeploymentsDesiredStateStandby || deployment.DesiredState == db.DeploymentsDesiredStateArchived {
					if err := db.Query.StopDeploymentIfNoInstances(ctx, tx, db.StopDeploymentIfNoInstancesParams{
						ID:        deployment.ID,
						UpdatedAt: sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
					}); err != nil {
						return err
					}
				}
			}
		}
		return nil
	})
	if err != nil {
		return connect.NewResponse(&ctrlv1.ReportDeploymentStatusResponse{}), err
	}

	// After the tx commits, if an Update just upserted instances for an
	// active deployment, check whether the per-region healthy threshold is
	// met and notify the suspended Deploy workflow. This is the feedback
	// loop that unblocks waitForDeployments's awakeable. Any errors here
	// are logged but don't fail the RPC — krane retrying wouldn't help, and
	// the Deploy workflow will eventually hit its own timeout if nobody
	// ever notifies it.
	if updatedDeployment != nil && s.restate != nil {
		s.maybeNotifyInstancesReady(ctx, *updatedDeployment)
	}

	return connect.NewResponse(&ctrlv1.ReportDeploymentStatusResponse{}), nil
}

// maybeNotifyInstancesReady checks whether enough regions are healthy for
// the given deployment and, if so, calls DeployService.NotifyInstancesReady
// to unblock the suspended Deploy workflow. Best-effort: errors are logged
// but not returned, mirroring ReportSentinelStatus's thundering-herd gate.
func (s *Service) maybeNotifyInstancesReady(ctx context.Context, deployment db.Deployment) {
	if !deploymentActiveStatuses[deployment.Status] {
		return
	}

	// Per-region minimum replica requirements.
	minReplicaRows, err := db.Query.FindDeploymentTopologyMinReplicas(ctx, s.db.RO(), deployment.ID)
	if err != nil {
		logger.Error("failed to load deployment topology min replicas",
			"deployment_id", deployment.ID,
			"error", err,
		)
		return
	}
	if len(minReplicaRows) == 0 {
		// No topology rows yet — nothing to check.
		return
	}

	regionMinReplicas := make(map[string]uint32, len(minReplicaRows))
	for _, row := range minReplicaRows {
		regionMinReplicas[row.RegionID] = row.AutoscalingReplicasMin
	}
	// Mirrors waitForDeployments: requires (numRegions - 1) healthy regions,
	// minimum 1. Tolerates one full regional outage.
	requiredRegions := max(len(regionMinReplicas)-1, 1)

	instances, err := db.Query.FindInstancesByDeploymentId(ctx, s.db.RO(), deployment.ID)
	if err != nil {
		logger.Error("failed to load deployment instances for readiness check",
			"deployment_id", deployment.ID,
			"error", err,
		)
		return
	}

	runningPerRegion := make(map[string]uint32)
	for _, instance := range instances {
		if instance.Status == db.InstancesStatusRunning {
			runningPerRegion[instance.RegionID]++
		}
	}

	healthyRegions := 0
	for regionID, minReplicas := range regionMinReplicas {
		if runningPerRegion[regionID] >= minReplicas {
			healthyRegions++
		}
	}

	if healthyRegions < requiredRegions {
		return
	}

	if !s.notifiedReady.AddIfAbsent("deployment:" + deployment.ID) {
		return
	}

	_, err = hydrav1.NewDeployServiceIngressClient(s.restate, deployment.ID).
		NotifyInstancesReady().
		Send(ctx, &hydrav1.NotifyInstancesReadyRequest{
			DeploymentId: deployment.ID,
		})
	if err != nil {
		logger.Error("failed to notify deploy workflow of instance readiness",
			"deployment_id", deployment.ID,
			"error", err,
		)
	}
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
